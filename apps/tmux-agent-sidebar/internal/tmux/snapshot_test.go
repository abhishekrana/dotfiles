package tmux

import (
	"strings"
	"testing"

	"github.com/abhishekrana/tmux-agent-sidebar/internal/model"
)

type fakeRunner struct {
	panes string
	calls []string
}

func (f *fakeRunner) Run(args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	if args[0] == "list-panes" {
		return f.panes, nil
	}
	return "", nil
}

// Lines as tmux emits them: tab-separated, empty user options at the end
// of the last line (must not be lost — regression for the TrimSpace bug).
const fixture = "beta\t1\t1\t1\t%9\t1\tbash\t/tmp\t\t\t\t\t\n" +
	"alpha\t1\t2\t1\t%3\t1\tclaude\t/tmp\t1\tdone\t1700000000\t\t2\n" +
	"alpha\t1\t1\t0\t%2\t0\tbash\t/tmp\t1\tworking\t1700000000\t\t\n" +
	"alpha\t1\t3\t0\t%5\t1\tnode\t/tmp\t1\tworking\t1700000000\t\t"

func TestSnapshotParsesFilters(t *testing.T) {
	r := &fakeRunner{panes: fixture}
	snap := Snapshot(r, NewBranchCache(), "alpha")

	if len(snap.Sessions) != 2 {
		t.Fatalf("want 2 sessions, got %+v", snap.Sessions)
	}
	if snap.Sessions[0].Name != "alpha" || !snap.Sessions[0].Current {
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
	snap := Snapshot(r, NewBranchCache(), "alpha")
	if !snap.Sessions[0].Agents[0].Seen {
		t.Error("visible done agent must be marked seen")
	}
	joined := strings.Join(r.calls, " | ")
	if !strings.Contains(joined, "set-option -pq -t %3 @agent_seen 1") {
		t.Errorf("must stamp @agent_seen on the pane: %s", joined)
	}
}
