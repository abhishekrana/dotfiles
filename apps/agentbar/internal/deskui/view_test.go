package deskui

import (
	"os"
	"strings"
	"testing"

	"github.com/abhishekrana/agentbar/internal/workdesk"
)

func osWriteFile(path string, b []byte) error { return os.WriteFile(path, b, 0o644) }

// stripANSI removes styling so a test can assert on what a person reads rather than on
// escape sequences.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// The view has to show the things the design is about: where you are, how stale the
// mirror is, how much is asking for you, the bands, and the line where they stop.
func TestViewShowsTheThingsItIsFor(t *testing.T) {
	t.Parallel()
	out := stripANSI(testModel(t).View())
	for _, want := range []string{
		"1 inbox", "2 issues", "3 mrs", "4 agents", // where you are, in lifecycle order
		"synced",                             // how old the snapshot is
		"⚑",                                  // how much is asking for you
		"gitlab says it is you",              // a band GitLab vouches for
		"returned to you",                    // a band in GitLab's own words
		"no reviewer yet",                    // the band this tool exists for
		"nothing below asks anything of you", // the active/inactive line
		"!412", "#121",                       // rows
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the view does not show %q", want)
		}
	}
}

// Nothing may run off the edge: a line wider than the terminal wraps and shears the
// whole layout.
func TestViewNeverExceedsItsWidth(t *testing.T) {
	t.Parallel()
	for _, width := range []int{80, 100, 140, 200} {
		m := testModel(t)
		m.resize(width, 40)
		for _, v := range workdesk.Views() {
			m.setView(v)
			for i, line := range strings.Split(m.View(), "\n") {
				if w := len([]rune(stripANSI(line))); w > width {
					t.Errorf("width %d, view %s, line %d is %d cells:\n  %q",
						width, v, i+1, w, stripANSI(line))
				}
			}
		}
	}
}

// The preview is built from the snapshot, so it must answer the question the sheet exists
// for and name the rule that explains a stuck merge request.
func TestPreviewAnswersCanIMergeIt(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	m.setView(workdesk.ViewMRs)
	m.cursor = 0
	m.syncPreview()
	out := stripANSI(m.preview.View())

	for _, want := range []string{"Can I merge it?", "no · 1 blocker(s)", "not enough approvals"} {
		if !strings.Contains(out, want) {
			t.Errorf("the preview does not contain %q\n--- got ---\n%s", want, out)
		}
	}
}

// The unassigned case is the most common reason work in this queue stops moving, so it has
// to be words rather than an empty section.
func TestPreviewNamesTheUnassignedCase(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	m.setView(workdesk.ViewMRs)
	for i, r := range m.rows {
		if r.Label == "no reviewer yet" {
			m.cursor = i
			break
		}
	}
	m.syncPreview()
	if out := stripANSI(m.preview.View()); !strings.Contains(out, "nobody is assigned") {
		t.Errorf("the preview does not call out that nobody is assigned\n--- got ---\n%s", out)
	}
}

// An agent that finished with nothing to show for it is the row no forge view can produce.
func TestPreviewNamesWorkThatNeverReachedGitLab(t *testing.T) {
	t.Parallel()
	m := testModel(t)
	m.setView(workdesk.ViewAgents)
	for i, r := range m.rows {
		if strings.Contains(r.Label, "no merge request") {
			m.cursor = i
			break
		}
	}
	m.syncPreview()
	if out := stripANSI(m.preview.View()); !strings.Contains(out, "no merge request for this branch") {
		t.Errorf("the preview does not name the missing merge request\n--- got ---\n%s", out)
	}
}

func TestHelpOverlayListsEveryBinding(t *testing.T) {
	t.Parallel()
	m := press(testModel(t), "?")
	if !m.showHelp {
		t.Fatal("? did not open help")
	}
	out := stripANSI(m.View())
	for _, want := range []string{"assign a reviewer", "set auto-merge", "gate matrix", "promote to a pane"} {
		if !strings.Contains(out, want) {
			t.Errorf("help does not document %q", want)
		}
	}
	if m = press(m, "?"); m.showHelp {
		t.Error("? did not close help again")
	}
}

// A window narrower than the panes should degrade rather than panic.
func TestTinyTerminalDoesNotPanic(t *testing.T) {
	t.Parallel()
	for _, size := range [][2]int{{20, 6}, {1, 1}, {39, 10}} {
		m := testModel(t)
		m.resize(size[0], size[1])
		_ = m.View()
	}
}
