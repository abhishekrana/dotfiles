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
