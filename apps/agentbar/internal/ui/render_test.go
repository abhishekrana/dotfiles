package ui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/abhishekrana/agentbar/internal/model"
)

// A single-band list (all active) shows no dividers; a pinned+active+dormant
// list labels all three, so no band is left as "the rest".
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
	want := []string{"pinned ·1", "active ·1", "dormant ·1"}
	if !slices.Equal(labels, want) {
		t.Errorf("dividers = %q, want %q", labels, want)
	}

	// Every band is labelled on the same rule - one non-empty neighbour is
	// enough. The active header used to appear only under a pinned band, so a
	// pin-free list left its first group nameless.
	noPins := model.Snapshot{Sessions: model.Arrange([]model.Session{
		{Name: "act", Agents: []model.Agent{{PaneID: "%1"}}},
		{Name: "dead"},
	}, nil)}
	labels = nil
	for _, bl := range buildBlocks(noPins) {
		if bl.kind == blockSection {
			labels = append(labels, bl.label)
		}
	}
	if want := []string{"active ·1", "dormant ·1"}; !slices.Equal(labels, want) {
		t.Errorf("dividers without a pinned band = %q, want %q", labels, want)
	}
}

// The top pinned divider stays tight under the header rule; every divider
// below it gets a blank line above, and the dormant one also gets a blank
// below (its sessions pack tight). Line counts: pinned = 1, active = 2, dormant = 3.
func TestBandDividerSpacing(t *testing.T) {
	snap := model.Snapshot{Sessions: model.Arrange([]model.Session{
		{Name: "act", Agents: []model.Agent{{PaneID: "%2"}}},
		{Name: "dead"}, // dormant
		{Name: "pin", Agents: []model.Agent{{PaneID: "%1"}}},
	}, map[string]bool{"pin": true})}
	var pinned, active, dormant block
	var havePinned, haveActive, haveDormant bool
	for _, b := range buildBlocks(snap) {
		if b.kind != blockSection {
			continue
		}
		switch {
		case strings.HasPrefix(b.label, "pinned"):
			pinned, havePinned = b, true
		case strings.HasPrefix(b.label, "dormant"):
			dormant, haveDormant = b, true
		case strings.HasPrefix(b.label, "active"):
			active, haveActive = b, true
		}
	}
	if !havePinned || !haveActive || !haveDormant {
		t.Fatal("expected pinned, active, and dormant dividers")
	}
	// The top pinned divider is tight; every divider below it has a blank above.
	if pinned.pad {
		t.Error("top pinned divider must stay tight (no blank above it)")
	}
	if !active.pad || !dormant.pad {
		t.Error("dividers below the top must have a blank line above them")
	}
	// Only the dormant divider adds a blank below itself.
	if pinned.gapAfter || active.gapAfter {
		t.Error("only the dormant divider adds a blank below itself")
	}
	if !dormant.gapAfter {
		t.Error("dormant divider must add a blank below itself")
	}
	if got := blockLineCount(pinned, snap, false); got != 1 {
		t.Errorf("pinned divider = %d lines, want 1 (tight rule)", got)
	}
	if got := blockLineCount(active, snap, false); got != 2 {
		t.Errorf("active divider = %d lines, want 2 (blank + rule)", got)
	}
	if got := blockLineCount(dormant, snap, false); got != 3 {
		t.Errorf("dormant divider = %d lines, want 3 (blank + rule + blank)", got)
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
	if !showsHeadline(same, 0, false) {
		t.Error("first agent should show the branch")
	}
	if showsHeadline(same, 1, false) {
		t.Error("second agent on the same branch should not repeat it")
	}
	diff := model.Session{Agents: []model.Agent{
		{Command: "claude", Branch: "b1", State: model.StateWorking},
		{Command: "claude", Branch: "b2", State: model.StateWorking},
	}}
	if !showsHeadline(diff, 1, false) {
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

// Title mode heads the block with Claude's own title instead of the branch;
// the block keeps its shape, so nothing moves when you flip.
func TestTitleModeHeadsWithTheTitle(t *testing.T) {
	sess := model.Session{Name: "api", Agents: []model.Agent{
		{Command: "claude", Branch: "4629-startup-fail-fast", Title: "Merge request review",
			State: model.StateWorking},
	}}
	byBranch := testRenderer().agentBlock(sess, 0, false, false, 0, time.Now())
	r := testRenderer()
	r.byTitle = true
	byTitle := r.agentBlock(sess, 0, false, false, 0, time.Now())
	if len(byTitle) != len(byBranch) {
		t.Fatalf("headline mode must not change the block height: %d vs %d", len(byTitle), len(byBranch))
	}
	if !strings.Contains(byBranch[0], "4629-startup-fail-fast") {
		t.Errorf("branch mode should head with the branch, got %q", byBranch[0])
	}
	if !strings.Contains(byTitle[0], "Merge request review") {
		t.Errorf("title mode should head with the title, got %q", byTitle[0])
	}
	if strings.Contains(byTitle[0], "4629") {
		t.Errorf("title mode should drop the branch, got %q", byTitle[0])
	}
}

// An agent Claude has not titled keeps its branch headline in title mode - a
// row must never lose its headline.
func TestTitleModeFallsBackToBranch(t *testing.T) {
	sess := model.Session{Agents: []model.Agent{
		{Command: "claude", Branch: "chore/flags", State: model.StateIdle},
	}}
	if !showsHeadline(sess, 0, true) {
		t.Fatal("an untitled agent must still show its branch in title mode")
	}
	r := testRenderer()
	r.byTitle = true
	if got := r.agentBlock(sess, 0, false, false, 0, time.Now()); !strings.Contains(got[0], "chore/flags") {
		t.Errorf("want the branch as fallback headline, got %q", got[0])
	}
	if got := blockLineCount(block{kind: blockAgent}, model.Snapshot{Sessions: []model.Session{sess}}, true); got != 2 {
		t.Errorf("fallback headline must be counted: got %d lines, want 2", got)
	}
}

// Claudes sharing a branch collapse to one headline; distinct titles do not,
// so title mode heads each agent in a shared worktree.
func TestTitleModeDoesNotCollapseDistinctTitles(t *testing.T) {
	sess := model.Session{Agents: []model.Agent{
		{Command: "claude", Branch: "b", Title: "First job", State: model.StateWorking},
		{Command: "claude", Branch: "b", Title: "Second job", State: model.StatePermission},
	}}
	if showsHeadline(sess, 1, false) {
		t.Error("branch mode should collapse the shared branch")
	}
	if !showsHeadline(sess, 1, true) {
		t.Error("title mode should head each differently-titled agent")
	}
}
