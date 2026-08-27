package gitlab

import (
	"context"
	"fmt"
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

// Do runs the write and returns glab's output. Callers are expected to have confirmed
// with the person first; nothing here asks.
func (c *Client) Do(ctx context.Context, w Write) (string, error) {
	out, err := c.Runner.Run(ctx, w.Args...)
	return strings.TrimSpace(string(out)), err
}
