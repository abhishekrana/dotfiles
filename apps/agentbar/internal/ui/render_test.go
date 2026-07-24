package ui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/abhishekrana/agentbar/internal/model"
)

// A single-band list (all active) shows no dividers; a pinned+active+dormant
// list shows the pinned label, a bare rule before active, then the dormant label.
func TestBuildBlocksDividers(t *testing.T) {
	single := model.Snapshot{Sessions: []model.Session{
		{Name: "a", Agents: []model.Agent{{PaneID: "%1"}}},
		{Name: "b", Agents: []model.Agent{{PaneID: "%2"}}},
	}}
	for _, bl := range buildBlocks(single) {
		if bl.kind == blockSection {
			t.Fatalf("single band must have no section dividers, got label %q", bl.label)
		}
	}

	three := model.Snapshot{Sessions: model.Arrange([]model.Session{
		{Name: "act", Agents: []model.Agent{{PaneID: "%2"}}},
		{Name: "dead"}, // dormant
		{Name: "pin", Agents: []model.Agent{{PaneID: "%1"}}},
	}, map[string]bool{"pin": true})}
	var labels []string
	for _, bl := range buildBlocks(three) {
		if bl.kind == blockSection {
			labels = append(labels, bl.label)
		}
	}
	want := []string{"★ pinned ·1", "", "dormant ·1"}
	if !slices.Equal(labels, want) {
		t.Errorf("dividers = %q, want %q", labels, want)
	}
}

// A dormant session renders as a single dimmed line (no spacer, no "no agents"
// tag); an active session keeps its two-line block.
func TestDormantSessionIsOneDimLine(t *testing.T) {
	r := testRenderer()
	dormant := r.sessionBlock(model.Session{Name: "dead"}, true, false, false)
	if len(dormant) != 1 {
		t.Fatalf("dormant session must be 1 line, got %d: %q", len(dormant), dormant)
	}
	if strings.Contains(dormant[0], "no agents") {
		t.Errorf("dormant row must drop the 'no agents' tag: %q", dormant[0])
	}
	active := r.sessionBlock(model.Session{Name: "live", Agents: []model.Agent{{}}}, false, false, false)
	if len(active) != 2 {
		t.Errorf("active session must be 2 lines (spacer + name), got %d", len(active))
	}
}

func testRenderer() renderer {
	return renderer{theme: SolarizedLight(), width: 36, nameW: 6}
}

// The branch is the block's headline (first line); the status line follows.
func TestAgentBlockBranchIsHeadline(t *testing.T) {
	r := testRenderer()
	sess := model.Session{Name: "api", Agents: []model.Agent{
		{Command: "claude", Branch: "feat/x", State: model.StateWorking},
	}}
	lines := r.agentBlock(sess, 0, false, false, 0, time.Now())
	if len(lines) < 2 {
		t.Fatalf("want branch + status lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "feat/x") {
		t.Errorf("branch should be the headline (line 0), got %q", lines[0])
	}
	if !strings.Contains(lines[1], "claude") {
		t.Errorf("status line should follow the branch, got %q", lines[1])
	}
}

// Consecutive Claudes on one branch draw that branch once; a differing branch
// in the same session keeps its own headline (a session can span worktrees).
func TestBranchHeadlineCollapsesOnlySameBranch(t *testing.T) {
	same := model.Session{Agents: []model.Agent{
		{Command: "claude", Branch: "b", State: model.StateWorking},
		{Command: "claude", Branch: "b", State: model.StatePermission},
	}}
	if !agentShowsBranch(same, 0) {
		t.Error("first agent should show the branch")
	}
	if agentShowsBranch(same, 1) {
		t.Error("second agent on the same branch should not repeat it")
	}
	diff := model.Session{Agents: []model.Agent{
		{Command: "claude", Branch: "b1", State: model.StateWorking},
		{Command: "claude", Branch: "b2", State: model.StateWorking},
	}}
	if !agentShowsBranch(diff, 1) {
		t.Error("a different branch in the same session must show its own headline")
	}
}

// A branch shared by several Claudes takes the color of its most-urgent one.
func TestGroupColorIsMostUrgent(t *testing.T) {
	r := testRenderer()
	sess := model.Session{Agents: []model.Agent{
		{Branch: "b", State: model.StateWorking},
		{Branch: "b", State: model.StatePermission},
		{Branch: "b", State: model.StateDone},
	}}
	if got := r.groupColor(sess, 0); got != r.theme.Blocked {
		t.Errorf("shared branch should take the most-urgent color %v, got %v", r.theme.Blocked, got)
	}
}
