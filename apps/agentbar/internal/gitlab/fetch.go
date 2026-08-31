package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// How much is asked for at once, and the paging backstop.
//
// A manifest node is two scalars, so a page of it is free and 100 is the ceiling GitLab
// allows. A detail node carries descriptions, discussion threads and a merge check, and
// costs about 0.4s of server time on its own - so those go out in small chunks, several
// at a time. Measured on a real queue: 52 merge requests in one call took 29s and the
// same 52 in four concurrent chunks took 9s. GitLab does the work either way; asking for
// it down one connection is what made it slow.
const (
	manifestPage = 100
	detailChunk  = 13
	detailAtOnce = 4
	maxPages     = 40
	// Threads per merge request, taken with `last:` not `first:`. GitLab returns them
	// oldest-first, and on a busy merge request the oldest are all system notes
	// ("added 3 commits"); the human argument is at the tail.
	threadsPerMR = 30
)

// stampFields is a manifest node: which rows are open, and a token that changes when one
// does. GitLab's updatedAt is that token - it has no content hash to offer, and its
// GraphQL endpoint sends an ETag but ignores If-None-Match, so this is the only
// conditional fetch on offer here.
const stampFields = `iid updatedAt`

// mrFields is the selection every merge request view is built from.
//
// reviewState answers "is anyone actually reviewing this" and approvalState.rules
// exposes which rule is unsatisfied - neither exists on the REST list, which is why
// this is GraphQL. mergeabilityChecks is the one that answers "can I merge it":
// detailedMergeStatus names only one blocker and is computed lazily, so it reads
// UNCHECKED for much of any real queue.
const mrFields = `
        iid title draft conflicts detailedMergeStatus description
        autoMergeEnabled autoMergeStrategy
        approved approvalsRequired approvalsLeft
        resolvableDiscussionsCount resolvedDiscussionsCount
        sourceBranch targetBranch commitCount createdAt updatedAt webUrl
        discussions(last: %d) {
          nodes { resolved notes(last: 8) { nodes { body system author { username } } } }
        }
        diffStatsSummary { additions deletions fileCount }
        mergeabilityChecks { identifier status }
        headPipeline { status detailedStatus { label } stages { nodes { name status } } }
        approvalState { rules { name approvalsRequired approved approvedBy { nodes { username } } } }
        labels { nodes { title } }
        reviewers { nodes { username mergeRequestInteraction { reviewState } } }`

// id is fetched because a status or iteration move addresses the work item behind the
// issue; status and iteration because they are what the issue view bands and marks by.
// The lifecycle that orders those bands is a separate call - see Statuses.
//
// description and discussions are the ticket itself. Without them a preview could say
// where the work sat and never what it was, which is a row with a URL on it. Threads take
// `last:` for the same reason the merge request selection does: GitLab returns them
// oldest-first, and the oldest are the system notes.
const issueFields = `
        id iid title description updatedAt webUrl
        labels { nodes { title } }
        assignees { nodes { username } }
        status { id name category }
        iteration { id title startDate dueDate iterationCadence { title } }
        discussions(last: %d) {
          nodes { resolved notes(last: 8) { nodes { body system author { username } } } }
        }`

// Threads per issue. Fewer than a merge request gets: an issue's argument is one
// conversation, where a merge request's is one per line of the diff.
const threadsPerIssue = 20

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// page is one GraphQL response. Project is a pointer so a null - the not-visible case -
// is distinguishable from a project with no merge requests.
type page[T any] struct {
	Data struct {
		Project *struct {
			MergeRequests *struct {
				PageInfo pageInfo          `json:"pageInfo"`
				Nodes    []json.RawMessage `json:"nodes"`
			} `json:"mergeRequests"`
			Issues *struct {
				PageInfo pageInfo          `json:"pageInfo"`
				Nodes    []json.RawMessage `json:"nodes"`
			} `json:"issues"`
		} `json:"project"`
	} `json:"data"`
}

// MergeRequestStamps lists every open merge request that is yours as an iid and an
// updatedAt, and nothing else. One call per account per relation, for a whole queue.
func (c *Client) MergeRequestStamps(ctx context.Context, project string, users []string) ([]json.RawMessage, error) {
	return c.stamps(ctx, "mergeRequests", project, users)
}

// IssueStamps is the same for issues.
func (c *Client) IssueStamps(ctx context.Context, project string, users []string) ([]json.RawMessage, error) {
	return c.stamps(ctx, "issues", project, users)
}

// MergeRequestsByIID fetches the full selection for the merge requests named, and only
// those. Nodes come back as raw JSON so this package stays free of the model types: the
// caller decodes them into whatever it needs.
func (c *Client) MergeRequestsByIID(ctx context.Context, project string, iids []string) ([]json.RawMessage, error) {
	return c.chunked(ctx, "mergeRequests", project, fmt.Sprintf(mrFields, threadsPerMR), iids)
}

// IssuesByIID is the same for issues.
func (c *Client) IssuesByIID(ctx context.Context, project string, iids []string) ([]json.RawMessage, error) {
	return c.chunked(ctx, "issues", project, fmt.Sprintf(issueFields, threadsPerIssue), iids)
}

// ownership is the two ways a row can be yours, asked for separately and unioned.
//
// Both, because the halves are different queues: an account that files the work is
// rarely the account it is assigned to, and either half alone reads as a complete board
// while hiding most of it. GitLab's own `or:` filter would do this in one call, but it
// answered with a fraction of what its parts return, so the union is done here where it
// can be seen.
var ownership = []string{"authorUsername", "assigneeUsername"}

func owned(relation, who string) string {
	return "state: opened, " + relation + ": " + strconv.Quote(who)
}

func authored(who string) string { return owned("authorUsername", who) }

// stamps walks the manifest once per account per relation, all of them in flight
// together, and concatenates. Duplicates are expected - a row you authored and were
// assigned is named twice - and are dropped by the caller, which decodes the iids anyway.
func (c *Client) stamps(ctx context.Context, field, project string, users []string) ([]json.RawMessage, error) {
	type call struct{ relation, who string }
	var calls []call
	for _, who := range users {
		for _, relation := range ownership {
			calls = append(calls, call{relation, who})
		}
	}
	got := make([][]json.RawMessage, len(calls))
	errs := make([]error, len(calls))

	slot := make(chan struct{}, detailAtOnce)
	var wg sync.WaitGroup
	for i, cl := range calls {
		wg.Go(func() {
			slot <- struct{}{}
			defer func() { <-slot }()
			got[i], errs[i] = c.walk(ctx, field, project, stampFields, owned(cl.relation, cl.who), manifestPage)
		})
	}
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	var all []json.RawMessage
	for _, nodes := range got {
		all = append(all, nodes...)
	}
	return all, nil
}

// chunked runs the detail fetches a few at a time. GitLab charges per node either way,
// so the point is to have several of them in flight rather than one long one.
func (c *Client) chunked(ctx context.Context, field, project, selection string,
	iids []string) ([]json.RawMessage, error) {
	if len(iids) == 0 {
		return nil, nil
	}
	batches := chunk(iids, detailChunk)
	got := make([][]json.RawMessage, len(batches))
	errs := make([]error, len(batches))

	slot := make(chan struct{}, detailAtOnce)
	var wg sync.WaitGroup
	for i, batch := range batches {
		wg.Go(func() {
			slot <- struct{}{}
			defer func() { <-slot }()
			quoted := make([]string, len(batch))
			for j, iid := range batch {
				quoted[j] = strconv.Quote(iid)
			}
			filter := "iids: [" + strings.Join(quoted, ", ") + "]"
			got[i], errs[i] = c.walk(ctx, field, project, selection, filter, manifestPage)
		})
	}
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	var all []json.RawMessage
	for _, nodes := range got {
		all = append(all, nodes...)
	}
	return all, nil
}

func chunk(ids []string, size int) [][]string {
	var out [][]string
	for len(ids) > size {
		out = append(out, ids[:size])
		ids = ids[size:]
	}
	return append(out, ids)
}

func (c *Client) walk(ctx context.Context, field, project, selection, filter string,
	first int) ([]json.RawMessage, error) {
	var all []json.RawMessage
	cursor := ""
	for n := 0; n < maxPages; n++ {
		var p page[json.RawMessage]
		if err := c.graphQL(ctx, listQuery(field, project, selection, filter, first, cursor), &p); err != nil {
			return nil, fmt.Errorf("%s page %d: %w", field, n+1, err)
		}
		// A null project is not an empty project. Treating it as zero rows is what
		// once replaced a good board with a blank one and reported success.
		if p.Data.Project == nil {
			return nil, fmt.Errorf("%s: %w", project, ErrNotVisible)
		}
		var info pageInfo
		switch field {
		case "mergeRequests":
			if p.Data.Project.MergeRequests == nil {
				return all, nil
			}
			all = append(all, p.Data.Project.MergeRequests.Nodes...)
			info = p.Data.Project.MergeRequests.PageInfo
		default:
			if p.Data.Project.Issues == nil {
				return all, nil
			}
			all = append(all, p.Data.Project.Issues.Nodes...)
			info = p.Data.Project.Issues.PageInfo
		}
		if !info.HasNextPage || info.EndCursor == "" {
			return all, nil
		}
		cursor = info.EndCursor
	}
	// Hitting the cap means the queue outgrew the backstop. Say so rather than
	// silently returning a partial mirror that reads as complete.
	return all, fmt.Errorf("%s: stopped at the %d page cap", field, maxPages)
}

func listQuery(field, project, selection, filter string, first int, cursor string) string {
	after := ""
	if cursor != "" {
		after = `, after: ` + strconv.Quote(cursor)
	}
	return fmt.Sprintf(`query {
  project(fullPath: %s) {
    %s(%s, first: %d%s) {
      pageInfo { hasNextPage endCursor }
      nodes {%s
      }
    }
  }
}`, strconv.Quote(project), field, filter, first, after, selection)
}

// Todos fetches GitLab's own record of what is waiting on you, one call per action the
// caller cares about.
//
// Asking for the actions by name rather than filtering afterwards is most of the cost:
// the pending feed is an accumulating log GitLab never marks done, and on a real account
// it is overwhelmingly machine notifications the board already reports. Measured, one
// page each: the unfiltered feed took 2.4s a page and needed five of them; the four
// actions worth keeping took 0.4s together, three of them empty.
//
// Never fatal to a sync: this is REST rather than GraphQL and a token without the scope
// simply has none, in which case the inferred bands still work.
func (c *Client) Todos(ctx context.Context, project string, actions []string) ([]json.RawMessage, error) {
	got := make([][]json.RawMessage, len(actions))
	errs := make([]error, len(actions))
	var wg sync.WaitGroup
	for i, action := range actions {
		wg.Go(func() { got[i], errs[i] = c.todosFor(ctx, project, action) })
	}
	wg.Wait()

	var all []json.RawMessage
	for _, nodes := range got {
		all = append(all, nodes...)
	}
	return all, errors.Join(errs...)
}

func (c *Client) todosFor(ctx context.Context, project, action string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	for page := 1; page <= maxPages; page++ {
		path := fmt.Sprintf("todos?state=pending&action=%s&per_page=100&page=%d", action, page)
		raw, err := c.Runner.Run(ctx, "api", path)
		if err != nil {
			return all, err
		}
		var batch []json.RawMessage
		if err := json.Unmarshal(raw, &batch); err != nil {
			return all, fmt.Errorf("decode %s todos page %d: %w", action, page, err)
		}
		if len(batch) == 0 {
			break
		}
		// Still filtered by project: the action filter is GitLab's, this one is ours,
		// and a token that can see several projects would otherwise mix them.
		for _, td := range batch {
			if todoProject(td) == project {
				all = append(all, td)
			}
		}
		if len(batch) < 100 {
			break
		}
	}
	return all, nil
}

func todoProject(raw json.RawMessage) string {
	var td struct {
		Project struct {
			Path string `json:"path_with_namespace"`
		} `json:"project"`
	}
	if err := json.Unmarshal(raw, &td); err != nil {
		return ""
	}
	return td.Project.Path
}

// Statuses is the project's work-item status lifecycle: every status, in the order
// GitLab declares it.
//
// This is where the issue bands and their order come from, so neither is written down
// here - a column added or moved upstream appears on the next sync. Read from the work
// item type rather than from a board: a project can carry dozens of boards, they disagree
// about which statuses to show and in what order, and picking one of them would be
// picking a workflow rather than reading it.
func (c *Client) Statuses(ctx context.Context, project string) ([]json.RawMessage, error) {
	const q = `query {
  project(fullPath: %s) {
    workItemTypes(first: 30) {
      nodes {
        name
        widgetDefinitions {
          type
          ... on WorkItemWidgetDefinitionStatus { allowedStatuses { id name category } }
        }
      }
    }
  }
}`
	var resp struct {
		Data struct {
			Project *struct {
				WorkItemTypes struct {
					Nodes []struct {
						Name              string `json:"name"`
						WidgetDefinitions []struct {
							Type            string            `json:"type"`
							AllowedStatuses []json.RawMessage `json:"allowedStatuses"`
						} `json:"widgetDefinitions"`
					} `json:"nodes"`
				} `json:"workItemTypes"`
			} `json:"project"`
		} `json:"data"`
	}
	if err := c.graphQL(ctx, fmt.Sprintf(q, strconv.Quote(project)), &resp); err != nil {
		return nil, err
	}
	if resp.Data.Project == nil {
		return nil, fmt.Errorf("%s: %w", project, ErrNotVisible)
	}
	// The Issue type's lifecycle, falling back to the first type that declares one: every
	// type on a project shares its lifecycle, and a project whose work is filed under
	// another type should still get bands.
	var fallback []json.RawMessage
	for _, t := range resp.Data.Project.WorkItemTypes.Nodes {
		for _, w := range t.WidgetDefinitions {
			if w.Type != "STATUS" || len(w.AllowedStatuses) == 0 {
				continue
			}
			if t.Name == "Issue" {
				return w.AllowedStatuses, nil
			}
			if fallback == nil {
				fallback = w.AllowedStatuses
			}
		}
	}
	return fallback, nil
}

// CurrentIteration is the sprint running now, or nil where the project has none.
//
// includeAncestors because iteration cadences are usually defined on the group rather
// than the project, and the project's own list is then empty.
func (c *Client) CurrentIteration(ctx context.Context, project string) (json.RawMessage, error) {
	const q = `query {
  project(fullPath: %s) {
    iterations(first: 1, state: current, includeAncestors: true) {
      nodes { id title startDate dueDate iterationCadence { title } }
    }
  }
}`
	var resp struct {
		Data struct {
			Project *struct {
				Iterations struct {
					Nodes []json.RawMessage `json:"nodes"`
				} `json:"iterations"`
			} `json:"project"`
		} `json:"data"`
	}
	if err := c.graphQL(ctx, fmt.Sprintf(q, strconv.Quote(project)), &resp); err != nil {
		return nil, err
	}
	if resp.Data.Project == nil {
		return nil, fmt.Errorf("%s: %w", project, ErrNotVisible)
	}
	if len(resp.Data.Project.Iterations.Nodes) == 0 {
		return nil, nil
	}
	return resp.Data.Project.Iterations.Nodes[0], nil
}

// SchemaCheck validates the merge-request selection against the live schema without
// reading any data, by asking about a project path that cannot exist. GraphQL validates
// a query before executing it, so a field GitLab has moved or removed comes back as an
// error while a valid query returns a null project.
//
// This turns a GitLab upgrade that breaks a field into one command rather than a
// mystery about why a view went blank.
func (c *Client) SchemaCheck(ctx context.Context) error {
	const probe = "workdesk/schema-probe-does-not-exist"
	var p page[json.RawMessage]
	q := listQuery("mergeRequests", probe, fmt.Sprintf(mrFields, threadsPerMR),
		authored("workdesk-probe"), manifestPage, "")
	if err := c.graphQL(ctx, q, &p); err != nil {
		return err
	}
	if p.Data.Project != nil {
		return fmt.Errorf("the probe path %q unexpectedly exists; schema check is inconclusive", probe)
	}
	return nil
}
