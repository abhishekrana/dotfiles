package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/abhishekrana/tmux-agent-sidebar/internal/model"
)

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
