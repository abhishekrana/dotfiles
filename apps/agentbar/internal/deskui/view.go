package deskui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/abhishekrana/agentbar/internal/workdesk"
)

// Chrome shared with the rest of this environment: the band marker is the sidebar's and
// the pane rail's, and the middot is the separator every status segment already uses.
const (
	bandMark = "▌"
	sep      = " · "
	// The list gets slightly less than half, because a merge request sheet is the wider
	// of the two things on screen.
	listShare = 0.46
)

func paneWidths(total int) (list, preview int) {
	if total < 40 {
		return total, 0
	}
	list = int(float64(total) * listShare)
	// One column for the divider between the panes.
	return list, total - list - 1
}

// bodyHeight is what is left after the tab bar, its rule, and the footer.
func bodyHeight(total int) int {
	h := total - 4
	if h < 3 {
		return 3
	}
	return h
}

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}
	if m.showHelp {
		return m.helpScreen()
	}
	lw, pw := paneWidths(m.width)
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		m.listPane(lw),
		m.dividerPane(),
		m.previewPane(pw),
	)
	return strings.Join([]string{m.tabBar(), body, m.footer()}, "\n")
}

// tabBar names where you are and what the mirror is worth. The count on the right is the
// same number the bands above the line add up to - what is actually asking something of
// you.
func (m Model) tabBar() string {
	t := m.theme
	active := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	idle := lipgloss.NewStyle().Foreground(t.Muted)

	labels := make([]string, 0, 4)
	for _, v := range workdesk.Views() {
		label := v.Key() + " " + v.String()
		if v == m.view {
			labels = append(labels, active.Render(label))
			continue
		}
		labels = append(labels, idle.Render(label))
	}
	left := strings.Join(labels, idle.Render(sep))

	right := idle.Render(m.staleness())
	if n := m.attentionCount(); n > 0 {
		right = lipgloss.NewStyle().Foreground(t.Asking).Render(fmt.Sprintf("⚑ %d", n)) +
			idle.Render(sep) + right
	}
	rule := lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", m.width))
	return m.spread(left, right) + "\n" + rule
}

// spread puts left at the margin and right against the far edge, so nothing reflows as
// the counts change - the same reason the status bar pins its clock.
func (m Model) spread(left, right string) string {
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

// previewWidth is the viewport's own width, one narrower than its pane to leave the
// gutter beside the divider.
func previewWidth(pane int) int {
	if pane <= 1 {
		return 0
	}
	return pane - 1
}

// staleness is how old the snapshot is. Shown always: every row here is only as true as
// the last sync, and a view that hides that is a view that quietly lies.
func (m Model) staleness() string {
	synced := workdesk.ParseTime(m.deps.Mirror.Meta.Synced)
	if synced.IsZero() {
		// meta.synced is written in local time without a zone, so fall back to it raw
		// rather than claiming an age we cannot compute.
		if s := m.deps.Mirror.Meta.Synced; s != "" {
			return "synced " + s
		}
		return "never synced"
	}
	return "synced " + workdesk.AgeAgo(synced, m.deps.Now())
}

func (m Model) attentionCount() int {
	n := 0
	for _, r := range m.rows {
		if r.Flag == "a" {
			n++
		}
	}
	return n
}

// listPane draws the rows with their band headers derived, and the active/inactive line
// where the flag flips. Headers are not items, so the cursor never has to skip anything.
func (m Model) listPane(w int) string {
	t := m.theme
	height := bodyHeight(m.height)
	if len(m.rows) == 0 {
		empty := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
		return lipgloss.NewStyle().Width(w).Height(height).
			Render(empty.Render("  nothing here"))
	}

	var lines []string
	prevLabel, prevFlag := "", "a"
	for i, r := range m.rows {
		if r.Flag == "i" && prevFlag == "a" {
			lines = append(lines, m.dividerLine(w))
			prevFlag = "i"
		}
		if r.Label != prevLabel {
			lines = append(lines, m.bandHeader(r, w))
			prevLabel = r.Label
		}
		lines = append(lines, m.rowLine(r, w, i == m.cursor))
	}
	// Keep the cursor on screen without a scrollbar: the list is short and a window that
	// follows the cursor is less to look at than a bar that tracks it.
	lines = window(lines, m.cursorLine(), height)
	return lipgloss.NewStyle().Width(w).Height(height).Render(strings.Join(lines, "\n"))
}

// cursorLine is where the cursor lands once headers are counted in.
func (m Model) cursorLine() int {
	n, prevLabel, prevFlag := 0, "", "a"
	for i, r := range m.rows {
		if r.Flag == "i" && prevFlag == "a" {
			n++
			prevFlag = "i"
		}
		if r.Label != prevLabel {
			n++
			prevLabel = r.Label
		}
		if i == m.cursor {
			return n
		}
		n++
	}
	return n
}

// window scrolls a rendered list so line stays visible, keeping a little context either
// side rather than snapping the cursor to an edge.
func window(lines []string, line, height int) []string {
	if len(lines) <= height {
		return lines
	}
	const margin = 2
	start := line - height + margin + 1
	if start < 0 {
		start = 0
	}
	if line-margin < start {
		start = line - margin
	}
	if start < 0 {
		start = 0
	}
	if start+height > len(lines) {
		start = len(lines) - height
	}
	return lines[start : start+height]
}

// bandHeader carries its own count, taken from the rows under it, so the two can never
// disagree. Amber while the band is asking something of you, muted once it is not - the
// same two roles the sidebar uses for the same distinction.
func (m Model) bandHeader(r workdesk.Row, w int) string {
	t := m.theme
	colour := t.Asking
	if r.Flag != "a" {
		colour = t.Muted
	}
	n := 0
	for _, c := range m.rows {
		if c.Label == r.Label {
			n++
		}
	}
	mark := lipgloss.NewStyle().Foreground(colour).Render(bandMark)
	label := lipgloss.NewStyle().Foreground(colour).Bold(r.Flag == "a").Render(r.Label)
	count := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf(" ·%d", n))
	// The todo band is filtered, so it says how much it left out. A view that caps
	// silently reads as complete when it is not.
	if r.Label == workdesk.TodoBand && m.idx.TodosDropped > 0 {
		count += lipgloss.NewStyle().Foreground(t.Muted).
			Render(fmt.Sprintf("   +%d the bands already cover", m.idx.TodosDropped))
	}
	return truncate(mark+" "+label+count, w)
}

func (m Model) dividerLine(w int) string {
	t := m.theme
	text := " nothing below asks anything of you "
	rule := w - lipgloss.Width(text)
	if rule < 4 {
		return lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", max(w, 0)))
	}
	left := 2
	return lipgloss.NewStyle().Foreground(t.Muted).
		Render(strings.Repeat("─", left) + text + strings.Repeat("─", rule-left))
}

// rowLine is reference, title, age and - where it earns the room - the note.
func (m Model) rowLine(r workdesk.Row, w int, selected bool) string {
	t := m.theme
	refW, ageW := 6, 6
	if m.view == workdesk.ViewAgents {
		refW = 4
	}
	titleW := w - refW - ageW - 5
	if titleW < 10 {
		titleW = 10
	}

	note := ""
	// The agents view shows its note because whether the work reached GitLab at all is
	// the reason that view exists; elsewhere the note is a list of gates and the preview
	// is right there.
	if m.view == workdesk.ViewAgents {
		noteW := 18
		if titleW > 24+noteW {
			titleW -= noteW + 2
			note = "  " + workdesk.Pad(r.Note, noteW)
		}
	}

	ref := lipgloss.NewStyle().Foreground(t.Accent).Render(workdesk.Pad(r.Ref, refW))
	titleColour := t.Fg
	if selected {
		titleColour = t.Emphasis
	}
	title := lipgloss.NewStyle().Foreground(titleColour).Render(workdesk.Pad(r.Title, titleW))
	age := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%*s", ageW, r.Age))
	noteStyled := lipgloss.NewStyle().Foreground(t.Muted).Render(note)

	line := "  " + ref + " " + title + age + noteStyled
	if selected {
		// A full-width band, so the selection reads as a row rather than as coloured
		// text - the sidebar and the pickers all mark selection the same way.
		return lipgloss.NewStyle().Width(w).Background(t.SelBg).Render(
			"▸ " + ref + " " + title + age + noteStyled)
	}
	return truncate(line, w)
}

func (m Model) dividerPane() string {
	return lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Render(strings.Repeat("│\n", bodyHeight(m.height)-1) + "│")
}

func (m Model) previewPane(w int) string {
	if w <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Width(w).Height(bodyHeight(m.height)).
		PaddingLeft(1).Render(m.preview.View())
}

// footer is the hint strip, or whatever just happened. Generated from the keymap, so a
// binding cannot exist without being documented.
func (m Model) footer() string {
	t := m.theme
	if m.filtering || m.filter.Value() != "" {
		left := m.filter.View()
		right := lipgloss.NewStyle().Foreground(t.Muted).
			Render(fmt.Sprintf("%d of %d", len(m.rows), m.totalRows()))
		return m.spread(left, right)
	}
	if m.notice != "" {
		return lipgloss.NewStyle().Foreground(t.Asking).Render("  " + m.notice)
	}
	return m.help.View(m.keys)
}

// totalRows is the unfiltered count, for the "N of M" the filter shows.
func (m Model) totalRows() int {
	if m.view == workdesk.ViewAgents {
		return len(m.rows)
	}
	return len(m.idx.Rows(m.view, m.deps.Now()))
}

func (m Model) helpScreen() string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Emphasis).Bold(true).Render("workdesk")
	sub := lipgloss.NewStyle().Foreground(t.Muted).Render(
		"the GitLab work you own, banded by who owns the next move")
	m.help.ShowAll = true
	return strings.Join([]string{
		title, sub, "",
		m.help.View(m.keys), "",
		lipgloss.NewStyle().Foreground(t.Muted).Render("? closes this"),
	}, "\n")
}

func truncate(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}
