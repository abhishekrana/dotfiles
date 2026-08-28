package deskui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/abhishekrana/agentbar/internal/ui"
	"github.com/abhishekrana/agentbar/internal/workdesk"
)

// frozen keeps the rendered output stable: ages are relative to now, so a test that used
// the real clock would render differently every day.
var frozen = time.Date(2026, 8, 27, 21, 27, 32, 0, time.UTC)

func testModel(t *testing.T) Model {
	t.Helper()
	mirror, err := workdesk.FixtureMirror()
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	raw, err := workdesk.FixtureAgents()
	if err != nil {
		t.Fatalf("fixture agents: %v", err)
	}
	agents := parseAgents(t, raw)

	m := New(Deps{
		Mirror: mirror,
		Agents: func() []workdesk.Agent { return agents },
		Now:    func() time.Time { return frozen },
	}, ui.SolarizedLight(), workdesk.ViewInbox)
	m.resize(140, 40)
	return m
}

// parseAgents goes through the real file parser rather than hand-building structs, so the
// fixture format stays covered.
func parseAgents(t *testing.T, raw []byte) []workdesk.Agent {
	t.Helper()
	path := t.TempDir() + "/agents.tsv"
	if err := writeFile(path, raw); err != nil {
		t.Fatal(err)
	}
	agents, err := workdesk.LoadAgents(path)
	if err != nil {
		t.Fatalf("LoadAgents: %v", err)
	}
	return agents
}

func press(m Model, keys ...string) Model {
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter", "tab", "esc", "up", "down":
			msg = tea.KeyMsg{Type: keyTypeFor(k)}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func keyTypeFor(k string) tea.KeyType {
	switch k {
	case "enter":
		return tea.KeyEnter
	case "tab":
		return tea.KeyTab
	case "esc":
		return tea.KeyEsc
	case "up":
		return tea.KeyUp
	default:
		return tea.KeyDown
	}
}

// The cursor is always on a real row. In the fzf version band headers were smuggled into
// the item list and the cursor had to be taught to step over them; here they are derived
// at render time, so walking the whole list can never land on one.
func TestCursorNeverLandsOnAHeader(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	for i := 0; i < len(m.rows)*2+3; i++ {
		row, ok := m.current()
		if !ok {
			t.Fatalf("no row under the cursor at step %d", i)
		}
		if row.Ref == "" {
			t.Fatalf("cursor landed on something with no reference: %+v", row)
		}
		m = press(m, "j")
	}
}

func TestCursorWraps(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	first, _ := m.current()
	m = press(m, "k") // up from the top
	last, _ := m.current()
	if first.Ref == last.Ref {
		t.Error("k at the top did not wrap to the end")
	}
	m = press(m, "j")
	back, _ := m.current()
	if back.Ref != first.Ref {
		t.Errorf("wrapping is not symmetric: landed on %s, want %s", back.Ref, first.Ref)
	}
}

func TestViewRing(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	for _, c := range []struct{ key, want string }{
		{"2", "issues"}, {"3", "mrs"}, {"4", "agents"}, {"1", "inbox"},
	} {
		m = press(m, c.key)
		if got := m.CurrentView().String(); got != c.want {
			t.Errorf("%q selected %q, want %q", c.key, got, c.want)
		}
	}
	// tab walks the same ring, so the two can never disagree.
	for _, want := range []string{"issues", "mrs", "agents", "inbox"} {
		m = press(m, "tab")
		if got := m.CurrentView().String(); got != want {
			t.Errorf("tab landed on %q, want %q", got, want)
		}
	}
}

func TestFilterNarrowsAndClears(t *testing.T) {
	t.Parallel()
	m := press(testModel(t), "3") // the merge request view
	before := len(m.rows)

	m = press(m, "/")
	if !m.filtering {
		t.Fatal("/ did not start filtering")
	}
	m = press(m, "m", "a", "n", "i") // "manifest"
	if len(m.rows) >= before {
		t.Errorf("filter did not narrow: %d rows, was %d", len(m.rows), before)
	}
	for _, r := range m.rows {
		hay := strings.ToLower(r.Ref + r.Title + r.Note + r.Branch)
		if !strings.Contains(hay, "mani") {
			t.Errorf("row %s does not match the filter", r.Ref)
		}
	}
	m = press(m, "esc")
	if len(m.rows) != before {
		t.Errorf("esc did not restore the list: %d rows, want %d", len(m.rows), before)
	}
}

// The UI records what to do and quits; it never acts. That is what keeps every action a
// plain function runnable with no terminal.
func TestActionsAreRequestedNotPerformed(t *testing.T) {
	t.Parallel()
	m := press(testModel(t), "3") // a merge request is selected
	for _, key := range []string{"o", "y", "c", "d", "a", "e", "M"} {
		got := press(m, key)
		if got.Pending == nil {
			t.Fatalf("%q recorded no action", key)
		}
		if got.Pending.Key != key {
			t.Errorf("%q recorded %q", key, got.Pending.Key)
		}
		if !strings.HasPrefix(got.Pending.Ref, "mrs:") {
			t.Errorf("%q recorded ref %q, want a mrs: reference", key, got.Pending.Ref)
		}
	}
}

// A write key on a row that is not a merge request must say so rather than quietly doing
// nothing - the shell silently ignored it.
func TestWriteKeysRefuseNonMergeRequests(t *testing.T) {
	t.Parallel()
	m := press(testModel(t), "2") // the issue view
	for _, key := range []string{"a", "e", "M"} {
		got := press(m, key)
		if got.Pending != nil {
			t.Errorf("%q on an issue recorded %+v", key, got.Pending)
		}
		if got.notice == "" {
			t.Errorf("%q on an issue said nothing", key)
		}
	}
}

func TestAgentRowsRequestAJump(t *testing.T) {
	t.Parallel()
	m := press(testModel(t), "4")
	row, ok := m.current()
	if !ok {
		t.Fatal("the agents view is empty")
	}
	if !strings.HasPrefix(row.Ref, "%") {
		t.Fatalf("agent row reference is %q, want a pane id", row.Ref)
	}
	got := press(m, "enter")
	if got.Pending == nil || !strings.HasPrefix(got.Pending.Ref, "agents:") {
		t.Errorf("enter on an agent recorded %+v, want an agents: reference", got.Pending)
	}
}

func writeFile(path string, b []byte) error {
	return osWriteFile(path, b)
}

// click sends the release, which is what the model acts on - terminals eat the press of a
// click that also focuses their window.
func click(m Model, x, y int) Model {
	next, _ := m.Update(tea.MouseMsg{
		X: x, Y: y, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft,
	})
	return next.(Model)
}

func wheel(m Model, x int, button tea.MouseButton) Model {
	next, _ := m.Update(tea.MouseMsg{X: x, Y: bodyTop, Action: tea.MouseActionPress, Button: button})
	return next.(Model)
}

// screenY is the line a list item is drawn on, or -1 when it is scrolled out of view.
func screenY(m Model, item int) int {
	items := m.listItems()
	height := bodyHeight(m.height)
	start := windowStart(len(items), cursorLine(items, m.cursor), height)
	if item < start || item >= start+height {
		return -1
	}
	return bodyTop + item - start
}

// A click is how you look at a row; the second click on it is how you act. That is the
// one place this differs from the sidebar, which jumps on the first click - here the
// preview beside the list is the reason to click at all.
func TestClickSelectsThenOpens(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	items := m.listItems()

	var target, y int
	for i, it := range items {
		if !it.header && it.row > m.cursor {
			target, y = it.row, screenY(m, i)
			break
		}
	}
	if y < 0 {
		t.Fatal("no second row on screen to click")
	}

	m = click(m, 2, y)
	if m.cursor != target {
		t.Fatalf("click selected row %d, want %d", m.cursor, target)
	}
	if m.Pending != nil {
		t.Fatalf("the first click acted: %+v", m.Pending)
	}

	m = click(m, 2, y)
	if m.Pending == nil || m.Pending.Key != "enter" {
		t.Errorf("the second click recorded %+v, want an enter", m.Pending)
	}
}

// Band headers are derived, not items, so a click on one has to land on a real row rather
// than on nothing.
func TestClickOnABandHeaderLandsOnARow(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	items := m.listItems()

	found := false
	for i, it := range items {
		if !it.header || it.row == m.cursor {
			continue
		}
		y := screenY(m, i)
		if y < 0 {
			continue
		}
		found = true
		got := click(m, 2, y)
		if got.cursor != it.row {
			t.Errorf("header click selected row %d, want the first row under it, %d", got.cursor, it.row)
		}
		if _, ok := got.current(); !ok {
			t.Error("header click left the cursor on no row at all")
		}
		break
	}
	if !found {
		t.Skip("no band header on screen below the cursor")
	}
}

// The divider is a rule, not a row: clicking it must change nothing.
func TestClickOnTheDividerDoesNothing(t *testing.T) {
	t.Parallel()
	m := press(testModel(t), "3")
	for i, it := range m.listItems() {
		if it.row >= 0 {
			continue
		}
		y := screenY(m, i)
		if y < 0 {
			t.Skip("the active/inactive line is off screen")
		}
		got := click(m, 2, y)
		if got.cursor != m.cursor || got.Pending != nil {
			t.Errorf("clicking the divider moved to %d / recorded %+v", got.cursor, got.Pending)
		}
		return
	}
	t.Skip("this view has no active/inactive line")
}

// The tab bar is clickable at the same columns it renders, so the two cannot drift.
func TestClickOnATabSwitchesView(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	for _, s := range tabSpans() {
		got := click(m, s.start+1, 0)
		if got.CurrentView() != s.view {
			t.Errorf("clicking %q selected %q, want %q", s.text, got.CurrentView(), s.view)
		}
	}
	// The gap between two labels is not a tab.
	first := tabSpans()[0]
	if got := click(m, first.end, 0); got.CurrentView() != m.CurrentView() {
		t.Errorf("clicking the gap after %q switched to %q", first.text, got.CurrentView())
	}
}

// The staleness marker is what you would click to refresh, so it re-syncs.
func TestClickOnSyncedRequestsASync(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	start, end, _, ok := m.rightSpans()
	if !ok {
		t.Fatal("the tab bar's right group does not fit")
	}
	got := click(m, (start+end)/2, 0)
	if got.Pending == nil || got.Pending.Key != "r" {
		t.Errorf("clicking the staleness recorded %+v, want a re-sync", got.Pending)
	}
}

// A popup swallows a click on the tmux chip that opened it, so the ✕ is the only pointer
// that can close this - it has to be on the hard-right cell and it has to quit.
func TestClickOnCloseQuits(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	_, _, closeStart, ok := m.rightSpans()
	if !ok {
		t.Fatal("the tab bar's right group does not fit")
	}
	if closeStart != m.width-1 {
		t.Errorf("✕ starts at %d, want the last cell %d", closeStart, m.width-1)
	}
	if !strings.Contains(stripANSI(m.View()), closeMark) {
		t.Error("the tab bar does not draw a ✕")
	}
	// The model quits by command, so assert on the command rather than on state.
	next, cmd := m.Update(tea.MouseMsg{
		X: m.width - 1, Y: 0, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft,
	})
	if cmd == nil {
		t.Fatal("clicking ✕ issued no command, want quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("clicking ✕ issued %T, want tea.QuitMsg", cmd())
	}
	if next.(Model).Pending != nil {
		t.Errorf("clicking ✕ recorded an action: %+v", next.(Model).Pending)
	}
}

// alt+n is the key tmux opens this with, so it has to close it too - that is the whole
// toggle, and the status chip cannot provide one.
func TestAltNClosesSoTheOpenerToggles(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n"), Alt: true})
	if cmd == nil {
		t.Fatal("alt+n issued no command, want quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("alt+n issued %T, want tea.QuitMsg", cmd())
	}
	if next.(Model).Pending != nil {
		t.Errorf("alt+n recorded an action: %+v", next.(Model).Pending)
	}
}

// A wheel is a scroll, not a walk: it stops at the ends rather than wrapping the way j/k
// deliberately do.
func TestWheelStopsAtTheEnds(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	if got := wheel(m, 2, tea.MouseButtonWheelUp); got.cursor != 0 {
		t.Errorf("the wheel wrapped off the top to %d", got.cursor)
	}
	m = wheel(m, 2, tea.MouseButtonWheelDown)
	if m.cursor != 1 {
		t.Fatalf("the wheel moved to %d, want 1", m.cursor)
	}
	m.cursor = len(m.rows) - 1
	if got := wheel(m, 2, tea.MouseButtonWheelDown); got.cursor != len(m.rows)-1 {
		t.Errorf("the wheel wrapped off the end to %d", got.cursor)
	}
}

// The wheel follows the pointer's pane: over the preview it scrolls the sheet and leaves
// the cursor alone.
func TestWheelOverThePreviewScrollsIt(t *testing.T) {
	t.Parallel()
	m := press(testModel(t), "3")
	lw, _ := paneWidths(m.width)
	before := m.cursor

	m = wheel(m, lw+2, tea.MouseButtonWheelDown)
	if m.cursor != before {
		t.Errorf("a wheel over the preview moved the cursor to %d", m.cursor)
	}
	if m.preview.YOffset == 0 {
		t.Error("a wheel over the preview did not scroll it")
	}
	m = wheel(m, lw+2, tea.MouseButtonWheelUp)
	if m.preview.YOffset != 0 {
		t.Errorf("the wheel did not scroll back up: offset %d", m.preview.YOffset)
	}
}

// The help overlay owns the whole window, so any click closes it - the same dismissal the
// next keypress gives a notice.
func TestClickClosesHelp(t *testing.T) {
	t.Parallel()
	m := press(testModel(t), "?")
	if !m.showHelp {
		t.Fatal("? did not open help")
	}
	if got := click(m, 5, 5); got.showHelp {
		t.Error("a click did not close the help overlay")
	}
}

// The gestures have to be on the help screen: an undocumented one is one nobody finds.
func TestHelpOverlayDocumentsTheMouse(t *testing.T) {
	t.Parallel()
	out := stripANSI(press(testModel(t), "?").View())
	for _, h := range mouseHints() {
		if !strings.Contains(out, h[1]) {
			t.Errorf("help does not document the %q gesture", h[0])
		}
	}
}
