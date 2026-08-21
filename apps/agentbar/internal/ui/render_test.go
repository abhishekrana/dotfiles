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
	if got := blockLineCount(pinned, snap); got != 1 {
		t.Errorf("pinned divider = %d lines, want 1 (tight rule)", got)
	}
	if got := blockLineCount(active, snap); got != 2 {
		t.Errorf("active divider = %d lines, want 2 (blank + rule)", got)
	}
	if got := blockLineCount(dormant, snap); got != 3 {
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
	return renderer{theme: SolarizedLight(), width: 36}
}

// An agent block is its title then its state line, each a step deeper than the
// session above it: the title carries the identity, the state line the state.
func TestAgentBlockIsTitleThenState(t *testing.T) {
	r := testRenderer()
	sess := model.Session{Name: "api", Branch: "feat/x", Agents: []model.Agent{
		{Command: "claude", Branch: "feat/x", Title: "Rate limit rollout", State: model.StateWorking},
	}}
	lines := r.agentBlock(sess, 0, false, false, 0, time.Now())
	if len(lines) != 2 {
		t.Fatalf("want title + state lines, got %d: %q", len(lines), lines)
	}
	if !strings.Contains(lines[0], "Rate limit rollout") || !strings.HasPrefix(lines[0], agentIndent) {
		t.Errorf("the title should lead the block, indented, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "working") || !strings.HasPrefix(lines[1], stateIndent) {
		t.Errorf("the state line should sit a step deeper, got %q", lines[1])
	}
	// The command is the same word on every row, so it is no longer drawn.
	if strings.Contains(lines[1], "claude") {
		t.Errorf("the state line should not spell out the command, got %q", lines[1])
	}
	// The branch belongs to the session, never to the agent block.
	for i, l := range lines {
		if strings.Contains(l, "feat/x") {
			t.Errorf("line %d repeats the session's branch: %q", i, l)
		}
	}
}

// An untitled agent - Claude has not named it yet - is its state line alone, so
// the block shrinks instead of drawing an empty first line.
func TestUntitledAgentIsOneLine(t *testing.T) {
	r := testRenderer()
	sess := model.Session{Agents: []model.Agent{{Command: "claude", State: model.StateIdle}}}
	if got := r.agentBlock(sess, 0, false, false, 0, time.Now()); len(got) != 1 {
		t.Fatalf("want the state line alone, got %d: %q", len(got), got)
	}
	snap := model.Snapshot{Sessions: []model.Session{sess}}
	if got := blockLineCount(block{kind: blockAgent}, snap); got != 1 {
		t.Errorf("blockLineCount = %d, want 1", got)
	}
	sess.Agents[0].Title = "Something"
	snap = model.Snapshot{Sessions: []model.Session{sess}}
	if got := blockLineCount(block{kind: blockAgent}, snap); got != 2 {
		t.Errorf("a titled agent counts 2 lines, got %d", got)
	}
}

// Every agent draws its own title: unlike a branch, no two are the same, so
// there is nothing to collapse.
func TestEveryAgentDrawsItsOwnTitle(t *testing.T) {
	r := testRenderer()
	sess := model.Session{Branch: "b", Agents: []model.Agent{
		{Command: "claude", Branch: "b", Title: "First job", State: model.StateWorking},
		{Command: "claude", Branch: "b", Title: "Second job", State: model.StatePermission},
	}}
	for i, want := range []string{"First job", "Second job"} {
		got := r.agentBlock(sess, i, false, false, 0, time.Now())
		if len(got) != 2 || !strings.Contains(got[0], want) {
			t.Errorf("agent %d should head with %q, got %q", i, want, got)
		}
	}
}

// The session line carries its branch, dim, beside the name.
func TestSessionRowCarriesTheBranch(t *testing.T) {
	r := testRenderer()
	sess := model.Session{Name: "api-2", Branch: "4629-startup-fail-fast",
		Agents: []model.Agent{{Command: "claude", State: model.StateIdle}}}
	row := r.sessionRow(sess, false, false, false)
	if !strings.Contains(row, "api-2") || !strings.Contains(row, "4629-startup-fail-fast") {
		t.Errorf("session row should carry name and branch, got %q", row)
	}
	if !strings.Contains(row, "\u2387") {
		t.Errorf("the branch should carry the rail's glyph, got %q", row)
	}
	// A dormant session has no agent, so no worktree to read a branch from.
	dormant := model.Session{Name: "cold", Branch: "main"}
	if got := r.sessionRow(dormant, true, false, false); strings.Contains(got, "main") {
		t.Errorf("a dormant row should drop the branch, got %q", got)
	}
}

// A branch too long for what is left of the line is truncated; one with no room
// at all is dropped rather than mangled.
func TestBranchTagFits(t *testing.T) {
	if got := branchTag("4629-startup-fail-fast", 12); len([]rune(got)) > 12 {
		t.Errorf("branchTag overflowed its room: %q", got)
	}
	if got := branchTag("main", 4); got != "" {
		t.Errorf("no room should mean no tag, got %q", got)
	}
	if got := branchTag("", 20); got != "" {
		t.Errorf("no branch should mean no tag, got %q", got)
	}
}
