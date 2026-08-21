package tmux

import (
	"strings"
	"testing"

	"github.com/abhishekrana/agentbar/internal/model"
)

type fakeRunner struct {
	panes   string
	replies map[string]string // canned output keyed by the joined args
	calls   []string
}

func (f *fakeRunner) Run(args ...string) (string, error) {
	call := strings.Join(args, " ")
	f.calls = append(f.calls, call)
	if out, ok := f.replies[call]; ok {
		return out, nil
	}
	if args[0] == "list-panes" {
		return f.panes, nil
	}
	return "", nil
}

// Lines as tmux emits them: tab-separated, empty user options at the end
// of the last line (must not be lost - regression for the TrimSpace bug).
const fixture = "beta\t1\t1\t1\t%9\t1\tbash\t/tmp\t\t\t\t\t\t\tbox\tbox\n" +
	"app\t1\t2\t1\t%3\t1\tclaude\t/tmp\t1\tdone\t1700000000\t\t2\t/wt/other\t✳ Ship the parser\tbox\n" +
	"app\t1\t1\t0\t%2\t0\tbash\t/tmp\t1\tworking\t1700000000\t\t\t\tbox\tbox\n" +
	"app\t1\t3\t0\t%5\t1\tnode\t/tmp\t1\tworking\t1700000000\t\t\t\tbox\tbox"

func TestSnapshotParsesFilters(t *testing.T) {
	r := &fakeRunner{panes: fixture}
	snap := Snapshot(r, NewBranchCache(), "app")

	if len(snap.Sessions) != 2 {
		t.Fatalf("want 2 sessions, got %+v", snap.Sessions)
	}
	if snap.Sessions[0].Name != "app" || !snap.Sessions[0].Current {
		t.Errorf("alphabetical order / current flag wrong: %+v", snap.Sessions[0])
	}
	if len(snap.Sessions[1].Agents) != 0 {
		t.Errorf("beta must have no agents")
	}

	agents := snap.Sessions[0].Agents
	if len(agents) != 2 {
		t.Fatalf("zombie bash pane must be filtered; got %+v", agents)
	}
	// Sorted by window index: %3 (win 2) before %5 (win 3).
	if agents[0].PaneID != "%3" || agents[1].PaneID != "%5" {
		t.Errorf("agent order wrong: %+v", agents)
	}
	if agents[0].State != model.StateDone || agents[0].Subagents != 2 {
		t.Errorf("agent fields wrong: %+v", agents[0])
	}
	// Last line with trailing empty fields must survive parsing.
	if agents[1].State != model.StateWorking {
		t.Errorf("last-line agent lost: %+v", agents[1])
	}
}

func TestSnapshotMarksVisibleDoneAsSeen(t *testing.T) {
	r := &fakeRunner{panes: fixture}
	snap := Snapshot(r, NewBranchCache(), "app")
	if !snap.Sessions[0].Agents[0].Seen {
		t.Error("visible done agent must be marked seen")
	}
	joined := strings.Join(r.calls, " | ")
	if !strings.Contains(joined, "set-option -pq -t %3 @agent_seen 1") {
		t.Errorf("must stamp @agent_seen on the pane: %s", joined)
	}
}

// A TMUX_PANE the server doesn't have resolves to empty with no error - a
// server that inherited a stale one (it is fixed at server start, and tmux
// never re-stamps it for a run-shell child) would otherwise report no current
// session at all, and the session keys would walk from the wrong row.
func TestCurrentSessionFallsBackOffAStalePane(t *testing.T) {
	t.Setenv("TMUX_PANE", "%57") // a pane from some other tmux server
	r := &fakeRunner{replies: map[string]string{
		"display-message -p #S": "dotfiles",
	}}
	if got := CurrentSession(r); got != "dotfiles" {
		t.Errorf("CurrentSession = %q, want dotfiles from the client fallback", got)
	}
	// A pane that does resolve is still preferred: the sidebar's own pane may
	// sit in a different session than the client is looking at.
	own := &fakeRunner{replies: map[string]string{
		"display-message -p -t %57 #S": "payments",
		"display-message -p #S":        "dotfiles",
	}}
	if got := CurrentSession(own); got != "payments" {
		t.Errorf("CurrentSession = %q, want the pane's own session payments", got)
	}
}

// The title is Claude's own name for the session, read from the pane title.
func TestSnapshotReadsAgentTitle(t *testing.T) {
	snap := Snapshot(&fakeRunner{panes: fixture}, NewBranchCache(), "app")
	agents := snap.Sessions[0].Agents
	if len(agents) != 2 {
		t.Fatalf("want 2 agents, got %d", len(agents))
	}
	if got := agents[0].Title; got != "Ship the parser" {
		t.Errorf("title = %q, want %q (glyph stripped)", got, "Ship the parser")
	}
	// The second agent's pane title is still the hostname tmux seeded it with.
	if got := agents[1].Title; got != "" {
		t.Errorf("an untitled pane must read as no title, got %q", got)
	}
}

// The glyph Claude prefixes is optional; its pre-prompt placeholder and the
// hostname tmux seeds pane_title with both mean "not titled yet".
func TestAgentTitle(t *testing.T) {
	for in, want := range map[string]string{
		"✳ Ship the parser":     "Ship the parser",
		"Ship the parser":       "Ship the parser",
		titlePlaceholder:        "",
		"✳ " + titlePlaceholder: "",
		"box":                   "",
		"":                      "",
	} {
		if got := agentTitle(in, "box"); got != want {
			t.Errorf("agentTitle(%q) = %q, want %q", in, got, want)
		}
	}
}
