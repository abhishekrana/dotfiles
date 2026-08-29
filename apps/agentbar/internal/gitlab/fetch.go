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

const issueFields = `
        iid title updatedAt webUrl
        labels { nodes { title } }`

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

// MergeRequestStamps lists every open merge request you authored as an iid and an
// updatedAt, and nothing else. One call for a whole queue.
func (c *Client) MergeRequestStamps(ctx context.Context, project, who string) ([]json.RawMessage, error) {
	return c.walk(ctx, "mergeRequests", project, stampFields, authored(who), manifestPage)
}

// IssueStamps is the same for issues.
func (c *Client) IssueStamps(ctx context.Context, project, who string) ([]json.RawMessage, error) {
	return c.walk(ctx, "issues", project, stampFields, authored(who), manifestPage)
}

// MergeRequestsByIID fetches the full selection for the merge requests named, and only
// those. Nodes come back as raw JSON so this package stays free of the model types: the
// caller decodes them into whatever it needs.
func (c *Client) MergeRequestsByIID(ctx context.Context, project string, iids []string) ([]json.RawMessage, error) {
	return c.chunked(ctx, "mergeRequests", project, fmt.Sprintf(mrFields, threadsPerMR), iids)
}

// IssuesByIID is the same for issues.
func (c *Client) IssuesByIID(ctx context.Context, project string, iids []string) ([]json.RawMessage, error) {
	return c.chunked(ctx, "issues", project, issueFields, iids)
}

func authored(who string) string {
	return "state: opened, authorUsername: " + strconv.Quote(who)
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
