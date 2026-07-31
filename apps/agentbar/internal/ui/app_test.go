package ui

import (
	"os"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/abhishekrana/agentbar/internal/model"
)

// fakeRunner records every tmux invocation and replies from a canned map
// keyed on the joined argument string.
type fakeRunner struct {
	calls   [][]string
	replies map[string]string
}

// Pins and trace lines land under XDG_STATE_HOME; point it at a tempdir so a
// direct `go test ./internal/ui/` never writes to the developer's state - a
// fixture session name matching a real one would pin it in the live sidebar.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "agentbar-ui-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_STATE_HOME", dir)
	code := m.Run()
	_ = os.RemoveAll(dir) // os.Exit skips defers
	os.Exit(code)
}

func (f *fakeRunner) Run(args ...string) (string, error) {
	f.calls = append(f.calls, args)
	return f.replies[strings.Join(args, " ")], nil
}

func twoSessionSnap() model.Snapshot {
	return model.Snapshot{Sessions: []model.Session{
		{Name: "api", Current: true, Agents: []model.Agent{
			{PaneID: "%0", WindowIndex: 1, Branch: "main", State: model.StateIdle},
		}},
		{Name: "blog", Agents: []model.Agent{
			{PaneID: "%6", WindowIndex: 1, Branch: "blog", State: model.StateIdle},
		}},
	}}
}

// testApp builds a live-mode App around a fake runner with the two-session
// snapshot already applied. Block layout: 0=api header, 1=%0 agent,
// 2=blog header, 3=%6 agent.
func testApp(r *fakeRunner) App {
	a := App{runner: r, current: "api"}
	a.setSnapshot(twoSessionSnap())
	return a
}

func TestSnapMsgAdoptsSharedSelection(t *testing.T) {
	a := testApp(&fakeRunner{})
	if a.cursor != 1 {
		t.Fatalf("initial cursor = %d, want 1 (first agent)", a.cursor)
	}
	m, _ := a.Update(snapMsg{snap: twoSessionSnap(), sel: "%6"})
	a = m.(App)
	if a.cursor != 3 {
		t.Errorf("cursor = %d after adopting %%6, want 3", a.cursor)
	}
	if a.lastSel != "%6" {
		t.Errorf("lastSel = %q, want %%6", a.lastSel)
	}
}

func TestSnapMsgUnchangedSelectionKeepsLocalCursor(t *testing.T) {
	a := testApp(&fakeRunner{})
	a.lastSel = "%6" // already adopted earlier
	a.cursor = 3
	a.moveCursor(-1) // k: onto blog's header (now selectable)
	a.moveCursor(-1) // k again: onto %0's agent block
	if a.cursor != 1 {
		t.Fatalf("moveCursor: cursor = %d, want 1", a.cursor)
	}
	m, _ := a.Update(snapMsg{snap: twoSessionSnap(), sel: "%6"})
	a = m.(App)
	if a.cursor != 1 {
		t.Errorf("unchanged shared selection stomped local cursor: got %d, want 1", a.cursor)
	}
}

func TestSnapMsgUnknownPaneKeepsCursor(t *testing.T) {
	a := testApp(&fakeRunner{})
	m, _ := a.Update(snapMsg{snap: twoSessionSnap(), sel: "%99"})
	a = m.(App)
	if a.cursor != 1 {
		t.Errorf("unknown pane moved cursor: got %d, want 1", a.cursor)
	}
}

func TestNotifyKeyTogglesOption(t *testing.T) {
	r := &fakeRunner{}
	a := testApp(r)
	if a.notify {
		t.Fatal("notify should start off")
	}
	press := func() App {
		m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		return m.(App)
	}
	a = press()
	if !a.notify {
		t.Error("first n did not turn notify on")
	}
	wrote := false
	for _, c := range r.calls {
		if strings.Join(c, " ") == "set-option -g @agent_notify on" {
			wrote = true
		}
	}
	if !wrote {
		t.Errorf("no `set-option -g @agent_notify on`; calls=%v", r.calls)
	}
	if a = press(); a.notify {
		t.Error("second n did not turn notify off")
	}
}

func TestSignalSnapAdoptsSelectionImmediately(t *testing.T) {
	a := testApp(&fakeRunner{})
	m, cmd := a.Update(snapMsg{snap: twoSessionSnap(), sel: "%6", signal: true})
	a = m.(App)
	if a.cursor != 3 {
		t.Errorf("cursor = %d after signal refresh, want 3", a.cursor)
	}
	if cmd == nil {
		t.Error("signal snapMsg must re-arm waitRefresh, got nil cmd")
	}
}

// A session switch made outside the sidebar moves the highlight to the
// newly attached session's focused agent.
func TestAttachedSessionChangeMovesHighlight(t *testing.T) {
	a := testApp(&fakeRunner{})
	snap := twoSessionSnap()
	snap.Sessions[0].Attached = true
	m, _ := a.Update(snapMsg{snap: snap, sel: ""})
	a = m.(App)
	if a.cursor != 1 {
		t.Fatalf("cursor = %d after api attach, want 1", a.cursor)
	}

	snap = twoSessionSnap()
	snap.Sessions[1].Attached = true
	snap.Sessions[1].Agents[0].Focused = true
	m, _ = a.Update(snapMsg{snap: snap, sel: ""})
	a = m.(App)
	if a.cursor != 3 {
		t.Errorf("cursor = %d after switch to blog, want 3", a.cursor)
	}

	// No change: local j/k position must survive the next tick. Two k's
	// step past blog's now-selectable header onto %0's agent block.
	a.moveCursor(-1)
	a.moveCursor(-1)
	m, _ = a.Update(snapMsg{snap: snap, sel: ""})
	a = m.(App)
	if a.cursor != 1 {
		t.Errorf("cursor = %d after unchanged tick, want 1", a.cursor)
	}
}

// Switching to a session before its agent exists must still hand the
// agent the highlight once it starts.
func TestAgentStartedAfterSwitchGetsHighlight(t *testing.T) {
	a := testApp(&fakeRunner{})
	snap := twoSessionSnap()
	snap.Sessions[1].Attached = true
	snap.Sessions[1].Agents = nil // switched here before claude started
	m, _ := a.Update(snapMsg{snap: snap, sel: ""})
	a = m.(App)
	// Two bands now, both labelled: 0 active-div, 1 api-hdr, 2 %0, 3 dormant-div.
	if a.cursor != 2 {
		t.Fatalf("cursor = %d while blog has no agents, want 2 (api's agent)", a.cursor)
	}

	snap = twoSessionSnap()
	snap.Sessions[1].Attached = true
	m, _ = a.Update(snapMsg{snap: snap, sel: ""})
	a = m.(App)
	if a.cursor != 3 {
		t.Errorf("cursor = %d after agent started in attached session, want 3", a.cursor)
	}
}

func TestActivatePublishesSelectionAndSignals(t *testing.T) {
	r := &fakeRunner{}
	a := testApp(r)
	a.cursor = 3 // blog's agent
	m, _ := a.activate()
	a = m.(App)

	if a.lastSel != "%6" {
		t.Errorf("lastSel = %q, want %%6 (own write must not be re-adopted)", a.lastSel)
	}
	if len(r.calls) == 0 {
		t.Fatal("activate issued no tmux command")
	}
	jump := strings.Join(r.calls[len(r.calls)-1], " ")
	for _, want := range []string{
		"switch-client",
		"-t blog",
		"select-window -t blog:1",
		"select-pane -t %6",
		"set-option -g @sidebar_selected %6",
		"wait-for -S " + refreshChannel,
	} {
		if !strings.Contains(jump, want) {
			t.Errorf("jump command missing %q:\n%s", want, jump)
		}
	}
}

// Terminals eat the press of a focusing click but deliver the release,
// so the jump must fire on release and ignore the press.
func TestClickJumpsOnReleaseNotPress(t *testing.T) {
	r := &fakeRunner{}
	a := testApp(r)
	a.width, a.height = 30, 20

	// Body rows: 0-1 api header, 2-3 agent %0, 4-5 blog header,
	// 6-7 agent %6; screen y = body row + 2 header lines, so %6 is at y=8.
	press := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, Y: 8}
	m, _ := a.Update(press)
	a = m.(App)
	if len(r.calls) != 0 {
		t.Fatalf("press must not jump, got %v", r.calls)
	}

	release := tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft, Y: 8}
	m, _ = a.Update(release)
	a = m.(App)
	if a.cursor != 3 {
		t.Errorf("cursor = %d after release on %%6's row, want 3", a.cursor)
	}
	if len(r.calls) == 0 || !strings.Contains(strings.Join(r.calls[len(r.calls)-1], " "), "switch-client") {
		t.Errorf("release did not jump: calls %v", r.calls)
	}
}

// The session header is two lines now: a leading spacer then the name.
// Clicking the blank spacer row must still select and switch to that
// session - the enlarged target is the whole point.
func TestClickSessionSpacerLineSwitches(t *testing.T) {
	r := &fakeRunner{}
	a := testApp(r)
	a.width, a.height = 30, 20

	// Body rows: 0-1 api header (spacer, name), 2-3 agent %0,
	// 4-5 blog header. blog's spacer line is body row 4 -> screen y = 6.
	release := tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft, Y: 6}
	m, _ := a.Update(release)
	a = m.(App)
	if a.cursor != 2 {
		t.Fatalf("cursor = %d after clicking blog's spacer line, want 2 (its header)", a.cursor)
	}
	if len(r.calls) == 0 || !strings.Contains(strings.Join(r.calls[len(r.calls)-1], " "), "-t blog") {
		t.Errorf("spacer-line click did not switch to blog: calls %v", r.calls)
	}
}

// Pointer motion sets the hovered block; motion off the body clears it.
func TestMotionSetsHover(t *testing.T) {
	a := testApp(&fakeRunner{})
	a.width, a.height = 30, 20

	// Body: 0-1 api header, 2-3 agent %0, 4-5 blog header,
	// 6-7 agent %6. Motion over %6's row is screen y=8.
	m, _ := a.Update(tea.MouseMsg{Action: tea.MouseActionMotion, Y: 8})
	a = m.(App)
	if a.hover != 3 {
		t.Errorf("hover = %d after motion over %%6, want 3", a.hover)
	}
	m, _ = a.Update(tea.MouseMsg{Action: tea.MouseActionMotion, Y: 99})
	a = m.(App)
	if a.hover != -1 {
		t.Errorf("hover = %d after motion off the body, want -1", a.hover)
	}
}

func TestSetSnapshotKeepsSelectionByPane(t *testing.T) {
	a := testApp(&fakeRunner{})
	a.cursor = 3
	// New snapshot with an extra agent shifting block indices.
	snap := twoSessionSnap()
	snap.Sessions[0].Agents = append(snap.Sessions[0].Agents,
		model.Agent{PaneID: "%2", WindowIndex: 2, State: model.StateWorking})
	a.setSnapshot(snap)
	b := a.blocks[a.cursor]
	if got := snap.Sessions[b.session].Agents[b.agent].PaneID; got != "%6" {
		t.Errorf("selection drifted to %s after snapshot refresh, want %%6", got)
	}
}

// A selected session header stays anchored by name across a refresh that
// shifts block indices, just as an agent selection anchors by pane.
func TestSetSnapshotKeepsSessionSelectionByName(t *testing.T) {
	a := testApp(&fakeRunner{})
	a.cursor = 2 // blog's header
	// A new agent under api shifts every later block index down by one.
	snap := twoSessionSnap()
	snap.Sessions[0].Agents = append(snap.Sessions[0].Agents,
		model.Agent{PaneID: "%2", WindowIndex: 2, State: model.StateWorking})
	a.setSnapshot(snap)
	b := a.blocks[a.cursor]
	if b.kind != blockSession || snap.Sessions[b.session].Name != "blog" {
		t.Errorf("session selection drifted after refresh: block %+v", b)
	}
}

// Activating a session header switches the client to it and, unlike an
// agent jump, leaves window/pane selection alone.
func TestActivateSessionSwitchesClient(t *testing.T) {
	r := &fakeRunner{}
	a := testApp(r)
	a.cursor = 2 // blog's header
	m, _ := a.activate()
	a = m.(App)

	if len(r.calls) == 0 {
		t.Fatal("activate issued no tmux command")
	}
	got := strings.Join(r.calls[len(r.calls)-1], " ")
	// Switches, and publishes the session token so every sidebar highlights
	// its row - but doesn't touch the target's window/pane selection.
	for _, want := range []string{
		"switch-client", "-t blog",
		"set-option -g @sidebar_selected =blog",
		"wait-for -S " + refreshChannel,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("switch command missing %q:\n%s", want, got)
		}
	}
	for _, absent := range []string{"select-window", "select-pane"} {
		if strings.Contains(got, absent) {
			t.Errorf("session switch should not run %q:\n%s", absent, got)
		}
	}
	if a.lastSel != "=blog" {
		t.Errorf("lastSel = %q, want =blog (own write must not be re-adopted)", a.lastSel)
	}
}

// A published session token ("=name") highlights that session's row in
// another sidebar, just as a pane id highlights an agent.
func TestSnapMsgAdoptsSessionToken(t *testing.T) {
	a := testApp(&fakeRunner{})
	m, _ := a.Update(snapMsg{snap: twoSessionSnap(), sel: "=blog"})
	a = m.(App)
	if b := a.blocks[a.cursor]; b.kind != blockSession || a.snap.Sessions[b.session].Name != "blog" {
		t.Errorf("cursor = %d (%+v) after adopting =blog, want blog's header", a.cursor, b)
	}
}

// Clicking the current session is a no-op: no client is switched.
func TestActivateCurrentSessionIsNoop(t *testing.T) {
	r := &fakeRunner{}
	a := testApp(r)
	a.cursor = 0 // api's header (current)
	if _, _ = a.activate(); len(r.calls) != 0 {
		t.Errorf("activating the current session issued %v", r.calls)
	}
}

// Pressing p pins the selected session: it floats above the active band
// (behind a divider), the cursor rides along with it, the divider is not
// selectable, j/k skip it, and the set persists to @agentbar-pins.
func TestPinFloatsRegroupsAndPersists(t *testing.T) {
	r := &fakeRunner{}
	a := testApp(r)
	a.cursor = 2 // blog's header (blocks: 0 api-hdr, 1 %0, 2 blog-hdr, 3 %6)

	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	a = m.(App)

	// Persisted to the shared option.
	var joined string
	for _, c := range r.calls {
		joined += strings.Join(c, " ") + " | "
	}
	if !strings.Contains(joined, "set-option -g @agentbar-pins blog") {
		t.Errorf("pin not written to @agentbar-pins: %s", joined)
	}
	// blog floated to the top band, pinned.
	if a.snap.Sessions[0].Name != "blog" || !a.snap.Sessions[0].Pinned {
		t.Fatalf("blog did not float pinned-first: %v", a.snap.Sessions)
	}
	// Cursor rode along to blog's header (now behind the pinned divider).
	if b := a.blocks[a.cursor]; b.kind != blockSession || a.snap.Sessions[b.session].Name != "blog" {
		t.Fatalf("cursor left blog after pin: block %+v", b)
	}
	// A divider now heads the list and cannot be selected or landed on.
	if a.blocks[0].kind != blockSection {
		t.Fatalf("want a section divider at block 0, got %+v", a.blocks[0])
	}
	if a.blockSelectable(0) {
		t.Error("section divider must not be selectable")
	}
	// Stepping down from blog's header skips the active-band divider onto api.
	a.moveCursor(1) // onto blog's %6 agent
	a.moveCursor(1) // skip the bare-rule divider, land on api's header
	if b := a.blocks[a.cursor]; b.kind != blockSession || a.snap.Sessions[b.session].Name != "api" {
		t.Errorf("j did not skip the divider onto api: block %+v cursor %d", b, a.cursor)
	}
}

// A click that lands on a section divider is a no-op (dividers aren't jumps).
func TestClickOnDividerIsNoop(t *testing.T) {
	r := &fakeRunner{}
	// One pinned + one active session => a tight "pinned" divider rule at body
	// row 0 (the top divider gets no leading blank).
	a := App{runner: r, current: "api", pins: map[string]bool{"blog": true}}
	snap := twoSessionSnap()
	snap.Sessions = model.Arrange(snap.Sessions, a.pins)
	a.setSnapshot(snap)
	a.width, a.height = 30, 20

	// Body row 0 is the divider rule; screen y = 0 + 2 header lines = 2.
	release := tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft, Y: 2}
	m, _ := a.Update(release)
	a = m.(App)
	for _, c := range r.calls {
		if len(c) > 0 && c[0] == "switch-client" {
			t.Errorf("click on a divider switched sessions: %v", c)
		}
	}
}

// attnSnap: an idle agent, a permission agent, and an asking agent across
// three sessions. Blocks: 0=a, 1=%0(working), 2=b, 3=%2(idle),
// 4=%3(permission), 5=c, 6=%4(question).
func attnSnap() model.Snapshot {
	return model.Snapshot{Sessions: []model.Session{
		{Name: "a", Current: true, Agents: []model.Agent{
			{PaneID: "%0", WindowIndex: 1, State: model.StateWorking},
		}},
		{Name: "b", Agents: []model.Agent{
			{PaneID: "%2", WindowIndex: 1, State: model.StateIdle},
			{PaneID: "%3", WindowIndex: 2, State: model.StatePermission},
		}},
		{Name: "c", Agents: []model.Agent{
			{PaneID: "%4", WindowIndex: 1, State: model.StateQuestion},
		}},
	}}
}

func sawPane(r *fakeRunner, pane string) bool {
	for _, c := range r.calls {
		if slices.Contains(c, pane) {
			return true
		}
	}
	return false
}

// Tab is the work queue: it steps to the next agent waiting on the user
// (permission/asking), skipping idle/working ones, jumping to each, and
// wrapping around across sessions.
func TestTabStepsThroughAttention(t *testing.T) {
	r := &fakeRunner{}
	a := App{runner: r, current: "a"}
	a.setSnapshot(attnSnap())
	if a.cursor != 1 {
		t.Fatalf("initial cursor = %d, want 1 (first agent)", a.cursor)
	}

	// From the working agent, skip the idle one and land on permission (%3).
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(App)
	if a.cursor != 4 {
		t.Errorf("cursor = %d after tab, want 4 (permission %%3)", a.cursor)
	}
	if !sawPane(r, "%3") {
		t.Errorf("tab did not jump to %%3; calls: %v", r.calls)
	}

	// Next tab advances to the asking agent (%4).
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(App)
	if a.cursor != 6 {
		t.Errorf("cursor = %d after 2nd tab, want 6 (asking %%4)", a.cursor)
	}

	// Past the last, it wraps back to the permission agent.
	m, _ = a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(App)
	if a.cursor != 4 {
		t.Errorf("cursor = %d after 3rd tab, want 4 (wrapped)", a.cursor)
	}
}

// With nobody waiting, tab does nothing - no move, no jump.
func TestTabNoAttentionIsNoop(t *testing.T) {
	r := &fakeRunner{}
	a := testApp(r) // both agents idle
	before := a.cursor
	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	a = m.(App)
	if a.cursor != before {
		t.Errorf("cursor moved to %d on tab with nothing waiting, want %d", a.cursor, before)
	}
	for _, c := range r.calls {
		if len(c) > 0 && c[0] == "switch-client" {
			t.Errorf("tab jumped with nothing waiting: %v", c)
		}
	}
}
