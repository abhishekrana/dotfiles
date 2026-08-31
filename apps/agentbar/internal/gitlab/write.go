package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Write is one pending change to GitLab: the command it would run, and a label a person
// can be asked to confirm.
//
// Modelled as a value rather than executed directly so the confirm, the dry run and the
// trace all see the same thing, and so a test can assert what would be run without
// running it.
type Write struct {
	Label string
	Args  []string
}

// Command is the shell line this would run, for the confirm prompt and for a dry run.
func (w Write) Command() string { return "glab " + strings.Join(w.Args, " ") }

// Assign puts a reviewer on a merge request. The one write that empties the band this
// whole tool exists to surface: finished work nobody was asked to look at.
func Assign(iid, title, reviewer string) Write {
	return Write{
		Label: fmt.Sprintf("Assign %s as reviewer on !%s  %s", reviewer, iid, title),
		Args:  []string{"mr", "update", iid, "--reviewer", reviewer},
	}
}

// AutoMerge asks GitLab to merge it once its gates go green, so a merge request that is
// only waiting on a pipeline needs nothing further from you.
func AutoMerge(iid, title string) Write {
	return Write{
		Label: fmt.Sprintf("Set auto-merge on !%s  %s", iid, title),
		Args:  []string{"mr", "merge", iid, "--auto-merge", "--yes"},
	}
}

// Merge merges now.
func Merge(iid, title string) Write {
	return Write{
		Label: fmt.Sprintf("Merge !%s  %s - now", iid, title),
		Args:  []string{"mr", "merge", iid, "--yes"},
	}
}

// WorkItemGID is the global ID a work-item mutation addresses, from the issue's own.
//
// GitLab hands out gid://gitlab/Issue/<n> and takes gid://gitlab/WorkItem/<n> for the
// same record - one row, two type names - so the swap is the whole conversion. Anything
// unrecognised is passed through, which fails loudly at GitLab rather than quietly here.
func WorkItemGID(issueGID string) string {
	const from = "gid://gitlab/Issue/"
	if n, ok := strings.CutPrefix(issueGID, from); ok {
		return "gid://gitlab/WorkItem/" + n
	}
	return issueGID
}

// SetStatus moves an issue to a status - the column move, done from here.
//
// The status is addressed by its global ID rather than its name: GitLab's input takes an
// ID, and the mirror already holds the lifecycle it came from.
func SetStatus(iid, title, issueGID, statusGID, statusName string) Write {
	return Write{
		Label: fmt.Sprintf("Move #%s to %s  %s", iid, statusName, title),
		Args: mutation(fmt.Sprintf(`workItemUpdate(input: {id: %s, statusWidget: {status: %s}})`,
			strconv.Quote(WorkItemGID(issueGID)), strconv.Quote(statusGID))),
	}
}

// SetIteration puts an issue in the current sprint, or takes it out - a null iteration is
// how GitLab is told to remove one.
func SetIteration(iid, title, issueGID, iterationGID, sprint string) Write {
	id, label := "null", fmt.Sprintf("Take #%s out of the sprint %s  %s", iid, sprint, title)
	if iterationGID != "" {
		id = strconv.Quote(iterationGID)
		label = fmt.Sprintf("Put #%s in the sprint %s  %s", iid, sprint, title)
	}
	return Write{
		Label: label,
		Args: mutation(fmt.Sprintf(`workItemUpdate(input: {id: %s, iterationWidget: {iterationId: %s}})`,
			strconv.Quote(WorkItemGID(issueGID)), id)),
	}
}

// mutation wraps one call as the glab invocation that runs it. GraphQL because neither
// status nor iteration has a glab subcommand, and because the mutation is the same shape
// for both.
func mutation(call string) []string {
	return []string{"api", "graphql", "-f", "query=mutation { " + call + " { errors } }"}
}

// Do runs the write and returns glab's output. Callers are expected to have confirmed
// with the person first; nothing here asks.
func (c *Client) Do(ctx context.Context, w Write) (string, error) {
	out, err := c.Runner.Run(ctx, w.Args...)
	if err != nil {
		return strings.TrimSpace(string(out)), err
	}
	return strings.TrimSpace(string(out)), payloadErrors(out)
}

// payloadErrors reads the complaints a GraphQL mutation returns in its own payload.
//
// A refused mutation is a 200 with an errors array inside the field it was asked for -
// glab reports the transport, not that - so without this a status move GitLab declined
// would print as one that worked. Silent on anything that is not a GraphQL response, so
// the glab subcommand writes are unaffected.
func payloadErrors(out []byte) error {
	var resp struct {
		Data map[string]struct {
			Errors []string `json:"errors"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil
	}
	for field, payload := range resp.Data {
		if len(payload.Errors) > 0 {
			return fmt.Errorf("%s: %s", field, strings.Join(payload.Errors, "; "))
		}
	}
	return nil
}
