package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// Page size and the paging backstop. Small because these nested selections are heavy
// per node - descriptions and discussion threads travel with every merge request.
const (
	pageSize = 25
	maxPages = 40
	// Threads per merge request, taken with `last:` not `first:`. GitLab returns them
	// oldest-first, and on a busy merge request the oldest are all system notes
	// ("added 3 commits"); the human argument is at the tail.
	threadsPerMR = 30
)

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

// MergeRequests fetches every open merge request authored by who, walking the cursor.
// Nodes come back as raw JSON so this package stays free of the model types: the caller
// decodes them into whatever it needs.
func (c *Client) MergeRequests(ctx context.Context, project, who string) ([]json.RawMessage, error) {
	return c.walk(ctx, project, who, "mergeRequests")
}

// Issues fetches every open issue authored by who.
func (c *Client) Issues(ctx context.Context, project, who string) ([]json.RawMessage, error) {
	return c.walk(ctx, project, who, "issues")
}

func (c *Client) walk(ctx context.Context, project, who, field string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	cursor := ""
	for n := 0; n < maxPages; n++ {
		var p page[json.RawMessage]
		if err := c.graphQL(ctx, listQuery(field, project, who, cursor), &p); err != nil {
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

func listQuery(field, project, who, cursor string) string {
	after := ""
	if cursor != "" {
		after = `, after: ` + strconv.Quote(cursor)
	}
	fields := issueFields
	if field == "mergeRequests" {
		fields = fmt.Sprintf(mrFields, threadsPerMR)
	}
	return fmt.Sprintf(`query {
  project(fullPath: %s) {
    %s(state: opened, authorUsername: %s, first: %d%s) {
      pageInfo { hasNextPage endCursor }
      nodes {%s
      }
    }
  }
}`, strconv.Quote(project), field, strconv.Quote(who), pageSize, after, fields)
}

// Todos fetches GitLab's own record of what is waiting on you, filtered to one project.
//
// Never fatal to a sync: this is REST rather than GraphQL and a token without the scope
// simply has none, in which case the inferred bands still work. Paged, because
// per_page alone would silently truncate a busy inbox.
func (c *Client) Todos(ctx context.Context, project string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	for page := 1; page <= maxPages; page++ {
		path := fmt.Sprintf("todos?state=pending&per_page=100&page=%d", page)
		raw, err := c.Runner.Run(ctx, "api", path)
		if err != nil {
			return all, err
		}
		var batch []json.RawMessage
		if err := json.Unmarshal(raw, &batch); err != nil {
			return all, fmt.Errorf("decode todos page %d: %w", page, err)
		}
		if len(batch) == 0 {
			break
		}
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
	if err := c.graphQL(ctx, listQuery("mergeRequests", probe, "workdesk-probe", ""), &p); err != nil {
		return err
	}
	if p.Data.Project != nil {
		return fmt.Errorf("the probe path %q unexpectedly exists; schema check is inconclusive", probe)
	}
	return nil
}
