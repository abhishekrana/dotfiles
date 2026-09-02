package deskui

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/abhishekrana/agentbar/internal/workdesk"
)

// w only ever widens, and it survives being pressed past the end of the ring.
func TestWindowKeyWidensAndWraps(t *testing.T) {
	t.Parallel()
	m := testModelWindow(t, workdesk.DefaultWindow)
	want := []string{"30d", "all", "7d"}
	for _, w := range want {
		m = press(m, "w")
		if got := m.CurrentWindow().String(); got != w {
			t.Fatalf("w went to %q, want %q", got, w)
		}
	}
}

// Widening has to bring rows back, or the key reports a window it is not applying.
func TestWindowKeyChangesWhatIsListed(t *testing.T) {
	t.Parallel()
	m := testModelWindow(t, workdesk.DefaultWindow)
	week := len(m.rows)
	all := len(testModel(t).rows)
	if week >= all {
		t.Fatalf("a seven-day window shows %d rows of %d - it is not filtering", week, all)
	}
	if got := len(press(m, "w", "w").rows); got != all {
		t.Errorf("widening to all shows %d rows, want %d", got, all)
	}
}

// The window is the inbox's. A key that silently reshaped another view would be worse
// than one that does nothing.
func TestWindowKeyIsInertOffTheInbox(t *testing.T) {
	t.Parallel()
	for _, view := range []string{"2", "3", "4"} {
		m := press(testModelWindow(t, workdesk.DefaultWindow), view)
		before := len(m.rows)
		after := press(m, "w")
		if after.CurrentWindow() != workdesk.DefaultWindow {
			t.Errorf("view %s: w moved the window to %v", view, after.CurrentWindow())
		}
		if len(after.rows) != before {
			t.Errorf("view %s: w changed the list from %d rows to %d", view, before, len(after.rows))
		}
	}
}

// A list that quietly leaves rows out reads as complete when it is not - so the bar says
// where the window stands, at every stop including the widest.
func TestTabBarNamesTheWindow(t *testing.T) {
	t.Parallel()
	m := testModelWindow(t, workdesk.DefaultWindow)
	for _, want := range []string{"7d", "30d", "all"} {
		bar := stripANSI(m.tabBar())
		if !strings.Contains(bar, sep+want+sep) {
			t.Errorf("the tab bar at %s does not name the window: %q", want, bar)
		}
		m = press(m, "w")
	}
	// Not on the views that have none, where it would be a fact about somewhere else.
	off := stripANSI(press(testModelWindow(t, workdesk.DefaultWindow), "3").tabBar())
	if strings.Contains(off, sep+"7d"+sep) {
		t.Errorf("the merge request view claims a window: %q", off)
	}
}

// Widening is an offer, not a discovery: the foot of the list says how many rows are
// behind the window and which key reveals them.
func TestListFootOwnsWhatTheWindowHeldBack(t *testing.T) {
	t.Parallel()
	m := testModelWindow(t, workdesk.DefaultWindow)
	if m.older == 0 {
		t.Fatal("a seven-day window over the fixture held nothing back - the test proves nothing")
	}
	if got := len(testModel(t).rows) - len(m.rows); got != m.older {
		t.Errorf("the list offers %d older rows, but widening reveals %d", m.older, got)
	}
	pane := stripANSI(m.listPane(80))
	want := strconv.Itoa(m.older) + " older · w widens to 30d"
	if !strings.Contains(pane, want) {
		t.Errorf("the list foot does not say %q:\n%s", want, pane)
	}
	// At the widest window there is nothing behind it, so the line goes rather than
	// naming a narrower stop.
	if wide := stripANSI(press(m, "w", "w").listPane(80)); strings.Contains(wide, "older · w widens") {
		t.Errorf("the list still offers older rows at the widest window:\n%s", wide)
	}
}

// A window that empties the list has the most to explain, so the line has to survive the
// empty case rather than being lost with the rows.
func TestListFootSurvivesAnEmptyWindow(t *testing.T) {
	t.Parallel()
	m := testModelWindow(t, workdesk.Window(time.Minute))
	if len(m.rows) != 0 {
		t.Fatalf("a one-minute window still shows %d rows", len(m.rows))
	}
	pane := stripANSI(m.listPane(80))
	if !strings.Contains(pane, "nothing here") || !strings.Contains(pane, "older · w widens") {
		t.Errorf("an emptied list does not explain itself:\n%s", pane)
	}
}

// The line is pinned, so it costs the rows a line rather than scrolling away with them.
func TestListFootDoesNotOverflowThePane(t *testing.T) {
	t.Parallel()
	for _, m := range []Model{
		testModelWindow(t, workdesk.DefaultWindow),
		testModelWindow(t, workdesk.Window(time.Minute)),
		testModel(t),
	} {
		got := strings.Count(m.listPane(80), "\n") + 1
		if want := bodyHeight(m.height); got != want {
			t.Errorf("the list pane is %d lines, want %d", got, want)
		}
	}
}
