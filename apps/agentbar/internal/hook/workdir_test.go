package hook

import (
	"strings"
	"testing"
)

func editEvent(name, tool, path string) Event {
	ev := Event{Name: name, ToolName: tool}
	ev.ToolInput.FilePath = path
	return ev
}

func TestEditedPath(t *testing.T) {
	notebook := Event{Name: "PostToolUse", ToolName: "NotebookEdit"}
	notebook.ToolInput.NotebookPath = "/wt/svc-b/nb.ipynb"

	cases := []struct {
		name string
		ev   Event
		want string
	}{
		{"write", editEvent("PostToolUse", "Write", "/wt/svc-b/a.go"), "/wt/svc-b/a.go"},
		{"edit pre", editEvent("PreToolUse", "Edit", "/wt/svc-b/a.go"), "/wt/svc-b/a.go"},
		{"notebook path", notebook, "/wt/svc-b/nb.ipynb"},
		// Read is not a write: it must not move the diff pane to whatever the
		// agent happened to look at.
		{"read ignored", editEvent("PostToolUse", "Read", "/wt/svc-b/a.go"), ""},
		{"bash ignored", editEvent("PostToolUse", "Bash", ""), ""},
		{"relative ignored", editEvent("PostToolUse", "Edit", "a.go"), ""},
		{"empty ignored", editEvent("PostToolUse", "Edit", ""), ""},
		{"cwd change", Event{Name: "CwdChanged", Cwd: "/wt/svc-c"}, "/wt/svc-c"},
		{"cwd change relative ignored", Event{Name: "CwdChanged", Cwd: "sub"}, ""},
		{"other event ignored", Event{Name: "Stop", Cwd: "/wt/svc-c"}, ""},
	}
	for _, c := range cases {
		if got := EditedPath(c.ev); got != c.want {
			t.Errorf("%s: EditedPath = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestWithin(t *testing.T) {
	cases := []struct {
		dir, path string
		want      bool
	}{
		{"/wt/svc-b", "/wt/svc-b/cloud/a.vue", true},
		{"/wt/svc-b", "/wt/svc-b", true},
		{"/wt/svc-b/", "/wt/svc-b/a.go", true},
		// The prefix trap: svc-a must not swallow svc-b, or an edit in a
		// sibling worktree would look like the same one and never re-point.
		{"/wt/svc-a", "/wt/svc-b/a.go", false},
		{"/wt/svc-b", "/wt/svc-c/a.go", false},
		{"", "/wt/svc-b/a.go", false},
	}
	for _, c := range cases {
		if got := Within(c.dir, c.path); got != c.want {
			t.Errorf("Within(%q, %q) = %v, want %v", c.dir, c.path, got, c.want)
		}
	}
}

func TestApplyWorkdirSkipsSameWorktree(t *testing.T) {
	f := &fakeRunner{options: map[string]string{"@agent_workdir": "/wt/svc-b"}}
	before, after := ApplyWorkdir(f, "%1", editEvent("PostToolUse", "Edit", "/wt/svc-b/deep/a.go"))
	if before != "" || after != "" {
		t.Fatalf("same worktree should be a no-op, got (%q, %q)", before, after)
	}
	for _, c := range f.calls {
		if strings.HasPrefix(c, "set-option") {
			t.Fatalf("same worktree wrote an option: %v", c)
		}
	}
}

func TestApplyWorkdirIgnoresNonWrites(t *testing.T) {
	f := &fakeRunner{options: map[string]string{}}
	if _, after := ApplyWorkdir(f, "%1", editEvent("PostToolUse", "Read", "/wt/svc-c/a.go")); after != "" {
		t.Fatalf("Read moved the workdir to %q", after)
	}
	if len(f.calls) != 0 {
		t.Fatalf("Read caused tmux calls: %v", f.calls)
	}
}

func TestApplyWorkdirStampsPaneAndWindow(t *testing.T) {
	// A real repo root is needed, so use the module's own checkout.
	root := repoRoot(".")
	if root == "" {
		t.Skip("not a git checkout")
	}
	f := &fakeRunner{options: map[string]string{}}
	before, after := ApplyWorkdir(f, "%1", editEvent("PostToolUse", "Write", root+"/go.mod"))
	if before != "" || after != root {
		t.Fatalf("got (%q, %q), want (%q, %q)", before, after, "", root)
	}
	joined := strings.Join(f.calls, "\n")
	// Both scopes: the pane copy is per agent, the window copy is what a status
	// format or pane border can compare against without forking.
	for _, want := range []string{"-pq", "-wq", "@agent_workdir", root} {
		if !strings.Contains(joined, want) {
			t.Errorf("tmux calls missing %q: %v", want, f.calls)
		}
	}
}

func TestPushWorkdir(t *testing.T) {
	cases := []struct{ name, list, dir, want string }{
		{"first", "", "/wt/a", "|/wt/a|"},
		{"newest first", "|/wt/a|", "/wt/b", "|/wt/b|/wt/a|"},
		// Revisiting a worktree moves it to the front rather than duplicating it,
		// so the picker's agent band never lists the same tree twice.
		{"revisit moves up", "|/wt/b|/wt/a|", "/wt/a", "|/wt/a|/wt/b|"},
		{"capped at five", "|/wt/5|/wt/4|/wt/3|/wt/2|/wt/1|", "/wt/6",
			"|/wt/6|/wt/5|/wt/4|/wt/3|/wt/2|"},
		{"empty dir is a no-op", "|/wt/a|", "", "|/wt/a|"},
	}
	for _, c := range cases {
		if got := PushWorkdir(c.list, c.dir); got != c.want {
			t.Errorf("%s: PushWorkdir(%q, %q) = %q, want %q", c.name, c.list, c.dir, got, c.want)
		}
	}
}

// The list is pipe-WRAPPED so a tmux format can test membership with
// #{m:*|dir|*,list} without a shorter name matching a longer one.
func TestPushWorkdirWrapsForMembership(t *testing.T) {
	list := PushWorkdir(PushWorkdir("", "/wt/svc-a"), "/wt/svc-b")
	if !strings.Contains(list, "|/wt/svc-a|") || !strings.Contains(list, "|/wt/svc-b|") {
		t.Fatalf("both entries must be individually delimited: %q", list)
	}
	if strings.Contains(list, "|/wt/svc-b|/wt/svc-b|") {
		t.Fatalf("duplicate entry: %q", list)
	}
}
