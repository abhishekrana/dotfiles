package gitlab

import (
	"context"
	"strings"
	"testing"
)

func TestWorkItemGID(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, want string }{
		// One record, two type names: the swap is the whole conversion.
		{"gid://gitlab/Issue/42", "gid://gitlab/WorkItem/42"},
		{"gid://gitlab/WorkItem/42", "gid://gitlab/WorkItem/42"},
		// Passed through, to fail at GitLab rather than quietly here.
		{"", ""},
		{"42", "42"},
	}
	for _, c := range cases {
		if got := WorkItemGID(c.in); got != c.want {
			t.Errorf("WorkItemGID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIssueMoves(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		write Write
		want  []string
		label string
	}{{
		name:  "a status move addresses the work item and the status by id",
		write: SetStatus("128", "Cold start is slow", "gid://gitlab/Issue/9128", "gid://gitlab/Status/2", "To do"),
		want:  []string{`id: "gid://gitlab/WorkItem/9128"`, `statusWidget: {status: "gid://gitlab/Status/2"}`},
		label: "Move #128 to To do",
	}, {
		name: "into the sprint",
		write: SetIteration("128", "Cold start is slow", "gid://gitlab/Issue/9128",
			"gid://gitlab/Iteration/7", "Aug 24 - Sep 6"),
		want:  []string{`iterationWidget: {iterationId: "gid://gitlab/Iteration/7"}`},
		label: "Put #128 in the sprint Aug 24 - Sep 6",
	}, {
		// A null iteration is how GitLab is told to take one off.
		name:  "out of the sprint",
		write: SetIteration("128", "Cold start is slow", "gid://gitlab/Issue/9128", "", "Aug 24 - Sep 6"),
		want:  []string{`iterationWidget: {iterationId: null}`},
		label: "Take #128 out of the sprint Aug 24 - Sep 6",
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := c.write.Command()
			for _, want := range c.want {
				if !strings.Contains(cmd, want) {
					t.Errorf("command %q does not carry %q", cmd, want)
				}
			}
			if !strings.HasPrefix(c.write.Label, c.label) {
				t.Errorf("label %q does not start with %q", c.write.Label, c.label)
			}
		})
	}
}

// fakeRunner answers one canned response, so a refusal can be replayed without a forge.
type fakeRunner struct{ out string }

func (f fakeRunner) Run(context.Context, ...string) ([]byte, error) { return []byte(f.out), nil }

// A GraphQL mutation reports its refusals in a 200 body, which glab does not read. Without
// this a status move GitLab declined would print as one that worked.
func TestDoReadsTheMutationPayload(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		out  string
		bad  string
	}{
		{"refused", `{"data":{"workItemUpdate":{"errors":["Status is not available"]}}}`, "Status is not available"},
		{"accepted", `{"data":{"workItemUpdate":{"errors":[]}}}`, ""},
		// glab's own subcommands print prose, which is not a payload to read.
		{"not graphql at all", "!412 merged", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out, err := (&Client{Runner: fakeRunner{out: c.out}}).Do(context.Background(), Write{})
			if c.bad == "" {
				if err != nil {
					t.Fatalf("Do reported %v on a payload that carried no complaint", err)
				}
				if out != c.out {
					t.Errorf("Do returned %q, want the runner's own output", out)
				}
				return
			}
			if err == nil {
				t.Fatal("a refused mutation reported success")
			}
			if !strings.Contains(err.Error(), c.bad) {
				t.Errorf("error %q does not carry GitLab's own words %q", err, c.bad)
			}
		})
	}
}
