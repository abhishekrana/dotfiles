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
	// The close button, in the corner every window puts one. A popup swallows a click
	// on the tmux chip that opened it, so this is the only pointer that can close it.
	closeMark = "✕"
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

// tabSpan is a view's label and the columns it occupies, so the bar and the mouse hit
// test read one geometry rather than each measuring the labels for itself.
type tabSpan struct {
	view       workdesk.View
	text       string
	start, end int // columns, end exclusive
}

func tabSpans() []tabSpan {
	out := make([]tabSpan, 0, 4)
	col := 0
	for i, v := range workdesk.Views() {
		if i > 0 {
			col += lipgloss.Width(sep)
		}
		text := v.Key() + " " + v.String()
		out = append(out, tabSpan{view: v, text: text, start: col, end: col + lipgloss.Width(text)})
		col = out[len(out)-1].end
	}
	return out
}

// tabBar names where you are and what the mirror is worth. The count on the right is the
// same number the bands above the line add up to - what is actually asking something of
// you.
func (m Model) tabBar() string {
	t := m.theme
	active := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	idle := lipgloss.NewStyle().Foreground(t.Muted)

	spans := tabSpans()
	labels := make([]string, 0, len(spans))
	for _, s := range spans {
		if s.view == m.view {
			labels = append(labels, active.Render(s.text))
			continue
		}
		labels = append(labels, idle.Render(s.text))
	}
	left := strings.Join(labels, idle.Render(sep))
	right, _ := m.tabBarRight()
	rule := lipgloss.NewStyle().Foreground(t.Muted).Render(strings.Repeat("─", m.width))
	return m.spread(left, right) + "\n" + rule
}

// tabBarRight is the right-hand group - what is asking for you, how stale the mirror is,
// then the close button hard against the edge. Returned styled and plain, because the hit
// test needs its width and escape sequences do not have one.
func (m Model) tabBarRight() (styled, plain string) {
	idle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	plain = m.staleness()
	styled = idle.Render(plain)
	if n := m.attentionCount(); n > 0 {
		flag := fmt.Sprintf("⚑ %d", n)
		styled = lipgloss.NewStyle().Foreground(m.theme.Asking).Render(flag) + idle.Render(sep) + styled
		plain = flag + sep + plain
	}
	// Muted like every other control with no state to report.
	plain += "  " + closeMark
	styled += idle.Render("  " + closeMark)
	return styled, plain
}

// rightSpans locates the two clickable things in the tab bar's right-hand group. ok is
// false when the bar is too narrow to draw that group at all - the same condition spread
// applies, so a click cannot land on text that was never rendered.
func (m Model) rightSpans() (staleStart, staleEnd, closeStart int, ok bool) {
	_, plain := m.tabBarRight()
	spans := tabSpans()
	if m.width-spans[len(spans)-1].end-lipgloss.Width(plain) < 1 {
		return 0, 0, 0, false
	}
	closeStart = m.width - lipgloss.Width(closeMark)
	staleEnd = closeStart - 2
	staleStart = staleEnd - lipgloss.Width(m.staleness())
	return staleStart, staleEnd, closeStart, true
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

// listItem is one display line before anything is rendered: a row, the band header
// derived above it, or the active/inactive divider. A header names the first row beneath
// it, so a click there lands on a real row - the same reason the cursor never sits on one.
//
// One pass, because the renderer, the scroll window and the mouse hit test all need to
// count headers the same way, and three loops that agree today are three that can stop
// agreeing.
type listItem struct {
	row    int // index into m.rows, -1 for the divider
	header bool
}

func (m Model) listItems() []listItem {
	items := make([]listItem, 0, len(m.rows)+8)
	prevLabel, prevFlag := "", "a"
	for i, r := range m.rows {
		if r.Flag == "i" && prevFlag == "a" {
			items = append(items, listItem{row: -1})
			prevFlag = "i"
		}
		if r.Label != prevLabel {
			items = append(items, listItem{row: i, header: true})
			prevLabel = r.Label
		}
		items = append(items, listItem{row: i})
	}
	return items
}

// cursorLine is the display line the cursor row sits on, once headers are counted in.
func cursorLine(items []listItem, cursor int) int {
	for i, it := range items {
		if !it.header && it.row == cursor {
			return i
		}
	}
	return 0
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

	items := m.listItems()
	// Keep the cursor on screen without a scrollbar: the list is short and a window that
	// follows the cursor is less to look at than a bar that tracks it.
	start := windowStart(len(items), cursorLine(items, m.cursor), height)
	lines := make([]string, 0, height)
	for _, it := range items[start:min(start+height, len(items))] {
		switch {
		case it.row < 0:
			lines = append(lines, m.dividerLine(w))
		case it.header:
			lines = append(lines, m.bandHeader(m.rows[it.row], w))
		default:
			lines = append(lines, m.rowLine(m.rows[it.row], w, it.row == m.cursor))
		}
	}
	return lipgloss.NewStyle().Width(w).Height(height).Render(strings.Join(lines, "\n"))
}

// windowStart scrolls a list of n lines so line stays visible, keeping a little context
// either side rather than snapping the cursor to an edge.
func windowStart(n, line, height int) int {
	if n <= height {
		return 0
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
	if start+height > n {
		start = n - height
	}
	return start
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
		m.mouseHelp(), "",
		lipgloss.NewStyle().Foreground(t.Muted).Render("? closes this"),
	}, "\n")
}

// mouseHelp renders the pointer's gestures in the keymap's own two colours, so they read
// as part of the same list rather than as a footnote.
func (m Model) mouseHelp() string {
	k := lipgloss.NewStyle().Foreground(m.theme.Accent)
	d := lipgloss.NewStyle().Foreground(m.theme.Muted)
	hints := mouseHints()
	// Sized from the hints themselves: a fixed column silently ellipsised the longest
	// one, which is exactly the failure the generated keymap exists to rule out.
	col := 0
	for _, h := range hints {
		col = max(col, lipgloss.Width(h[0])+2)
	}
	lines := make([]string, 0, len(hints))
	for _, h := range hints {
		lines = append(lines, k.Render(workdesk.Pad(h[0], col))+d.Render(h[1]))
	}
	return strings.Join(lines, "\n")
}

// bodyTop is the first body line: the tab bar and the rule under it sit above.
const bodyTop = 2

// rowAt maps a screen line to the row under it, or -1 for the divider, the footer and
// anything outside the list.
func (m Model) rowAt(y int) int {
	height := bodyHeight(m.height)
	if len(m.rows) == 0 || y < bodyTop || y >= bodyTop+height {
		return -1
	}
	items := m.listItems()
	i := windowStart(len(items), cursorLine(items, m.cursor), height) + y - bodyTop
	if i < 0 || i >= len(items) {
		return -1
	}
	return items[i].row
}

// overPreview reports whether a column is in the preview pane rather than the list. The
// divider column counts as neither.
func (m Model) overPreview(x int) bool {
	lw, pw := paneWidths(m.width)
	return pw > 0 && x > lw
}

// tabAt maps a column on the tab bar to the view whose label is under it.
func (m Model) tabAt(x int) (workdesk.View, bool) {
	for _, s := range tabSpans() {
		if x >= s.start && x < s.end {
			return s.view, true
		}
	}
	return m.view, false
}

// overStaleness reports whether a column is on the "synced ..." text - the one thing up
// there you would click to refresh.
func (m Model) overStaleness(x int) bool {
	start, end, _, ok := m.rightSpans()
	return ok && x >= start && x < end
}

// overClose reports whether a column is on the ✕.
func (m Model) overClose(x int) bool {
	_, _, start, ok := m.rightSpans()
	return ok && x >= start
}

func truncate(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}
