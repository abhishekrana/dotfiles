package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/abhishekrana/agentbar/internal/model"
)

// spinnerFrames animates the working state (braille, like Claude's own UI).
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type blockKind int

const (
	blockSession blockKind = iota // group header, one line
	blockAgent                    // agent + branch + subagents: one selectable unit
	blockSection                  // band divider (pinned/active/dormant): label only, not selectable
)

// block is one navigation unit; an agent block renders as 1-3 lines that
// select, highlight, and click together.
type block struct {
	kind     blockKind
	session  int    // index into snapshot.Sessions
	agent    int    // index into session.Agents (blockAgent only)
	label    string // blockSection only; "" means a bare rule
	pad      bool   // blockSection only: render a blank line above the divider
	gapAfter bool   // blockSection only: render a blank line below (dormant packs tight)
}

// buildBlocks flattens the snapshot: a band divider heads each group of
// sessions (only when more than one band is present), session headers are
// pure group labels, and agents form the flat, selectable list.
func buildBlocks(snap model.Snapshot) []block {
	var nP, nA, nD int
	for _, s := range snap.Sessions {
		switch s.Band() {
		case 0:
			nP++
		case 1:
			nA++
		default:
			nD++
		}
	}
	var blocks []block
	prev := -1
	for si, sess := range snap.Sessions {
		band := sess.Band()
		if band != prev {
			if label, ok := sectionHeader(band, nP, nA, nD); ok {
				// Every divider below the top one gets a blank line above it so
				// the band boundaries breathe; the top divider (pinned) stays
				// tight under the header rule. Non-dormant bands get their blank
				// below from the next session's leading spacer; dormant sessions
				// pack tight, so its divider adds the gap below itself.
				blocks = append(blocks, block{
					kind:     blockSection,
					label:    label,
					pad:      len(blocks) > 0,
					gapAfter: band == 2,
				})
			}
			prev = band
		}
		blocks = append(blocks, block{kind: blockSession, session: si})
		for ai := range sess.Agents {
			blocks = append(blocks, block{kind: blockAgent, session: si, agent: ai})
		}
	}
	return blocks
}

// sectionHeader returns the divider that heads band, and whether to draw one.
// A divider only appears when it actually separates two non-empty bands, so a
// single-band list (the common case today) shows no headers at all.
//
// All three bands are named, on the same rule: an unlabelled middle band left
// you counting rows to work out where "the rest" ended. The labels are
// model.BandLabel, so what you read here is what `agentbar order` prints.
func sectionHeader(band, nP, nA, nD int) (string, bool) {
	label := func(name string, n int) string { return name + " ·" + strconv.Itoa(n) }
	switch band {
	case 0: // pinned
		if nP > 0 && nA+nD > 0 {
			return label("pinned", nP), true
		}
	case 1: // active
		if nA > 0 && nP+nD > 0 {
			return label("active", nA), true
		}
	case 2: // dormant
		if nD > 0 && nP+nA > 0 {
			return label("dormant", nD), true
		}
	}
	return "", false
}

func stateIcon(s model.AgentState, frame int) string {
	switch s {
	case model.StateWorking:
		return spinnerFrames[frame%len(spinnerFrames)]
	case model.StatePermission:
		return "◔"
	case model.StateQuestion:
		return "?"
	case model.StateDone:
		return "✓"
	default:
		return "·"
	}
}

// elapsed renders a compact duration like 37s, 2m, 1h12m.
func elapsed(since time.Time, now time.Time) string {
	if since.IsZero() {
		return ""
	}
	d := now.Sub(since)
	switch {
	case d < 0:
		return ""
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// line lays out left and right fragments within width, truncating the left
// fragment first. Both fragments may contain ANSI styling.
func line(left, right string, width int) string {
	lw, rw := lipgloss.Width(left), lipgloss.Width(right)
	if rw > 0 && lw+rw+1 > width {
		// Not enough room: drop the right fragment rather than corrupt it.
		if lw > width {
			left = truncate(left, width)
		}
		return left
	}
	if lw > width {
		return truncate(left, width)
	}
	pad := max(width-lw-rw, 0)
	return left + strings.Repeat(" ", pad) + right
}

// truncate cuts a styled string to width cells with an ellipsis.
func truncate(s string, width int) string {
	if width <= 1 {
		return "…"
	}
	return lipgloss.NewStyle().MaxWidth(width-1).Render(s) + "…"
}

// renderer turns a snapshot into the sidebar view.
type renderer struct {
	theme Theme
	width int
}

// labelW fits the longest state label ("permission").
const labelW = 10

// The nesting: a session sits at the margin, its agents' titles one step in,
// their state lines one step further. Two steps plus the state label is what
// 30 columns hold - which is why the row no longer spells out "claude".
const (
	agentIndent = "   "
	stateIndent = "     "
)

// padCol pads (or truncates) a plain string to exactly w cells.
func padCol(s string, w int) string {
	if w <= 0 {
		return "" // a squeezed pane leaves no column to write in
	}
	r := []rune(s)
	if len(r) > w {
		return string(r[:w-1]) + "…"
	}
	return s + strings.Repeat(" ", w-len(r))
}

func (r renderer) sep() string {
	return lipgloss.NewStyle().Foreground(r.theme.Muted).Render(strings.Repeat("─", r.width))
}

// sectionRow renders a band divider: the label then a trailing rule. The
// pinned label reads gold, active and dormant stay muted grey. Never
// selectable, never lit.
func (r renderer) sectionRow(label string) string {
	mut := lipgloss.NewStyle().Foreground(r.theme.Muted)
	used := 1 + len([]rune(label)) + 1 // leading space + label + a space before the rule
	rule := mut.Render(" " + strings.Repeat("─", max(r.width-used, 0)))
	// The pinned label reads warm/gold so your working set pops; dormant stays
	// muted grey so it recedes. Rules are quiet hairlines for both bands.
	labelColor := r.theme.Muted
	if strings.HasPrefix(label, "pinned") {
		labelColor = r.theme.Asking // gold
	}
	return " " + lipgloss.NewStyle().Foreground(labelColor).Render(label) + rule
}

func (r renderer) header(snap model.Snapshot, frame int) string {
	// The plugin's own name, also its pane command and option namespace.
	title := lipgloss.NewStyle().Foreground(r.theme.Emphasis).Bold(true).Render(" agentbar")
	count := fmt.Sprintf("%d/%d", snap.Working(), snap.Total())
	dot := " "
	if snap.Working() > 0 {
		dot = lipgloss.NewStyle().Foreground(r.theme.Working).Render(spinnerFrames[frame%len(spinnerFrames)])
	}
	right := lipgloss.NewStyle().Foreground(r.theme.Muted).Render(count) + " " + dot + " "
	return line(title, right, r.width)
}

// leftEdge is the first column of a lit row: a blue accent bar when the
// block is the selection, otherwise a fill-colored space. Hover fills the
// row; the selected block also carries this edge down its left side.
func (r renderer) leftEdge(bar bool) string {
	s := lipgloss.NewStyle().Background(r.theme.SelBg)
	if bar {
		return s.Foreground(r.theme.Accent).Render("▎")
	}
	return s.Render(" ")
}

// sessionMarker is the right-hand tag of a session header: "no agents" for
// an empty session, nothing otherwise. (The current session isn't marked -
// the selection highlight, which follows every click, shows where you are.)
func (r renderer) sessionMarker(sess model.Session) string {
	if len(sess.Agents) == 0 {
		return "no agents "
	}
	return ""
}

// branchTag is the session's branch as it sits beside the name: "⎇ <branch>",
// truncated to what is left of the line, or "" when there is no room for a
// readable stub. The glyph is the pane rail's, so both surfaces name a branch
// the same way.
func branchTag(branch string, room int) string {
	if branch == "" || room < 6 {
		return ""
	}
	tag := "⎇ " + branch
	if len([]rune(tag)) > room {
		tag = padCol(tag, room)
	}
	return strings.TrimRight(tag, " ")
}

// sessionRow is the session's name line: the name, then its branch, dim,
// because one worktree is one checkout and the branch belongs to the session
// rather than to any agent in it. Indented one column so the selection's accent
// edge has a place to sit. A dormant row is dimmed and drops both its branch
// (no agent means no worktree to read) and its "no agents" tag.
func (r renderer) sessionRow(sess model.Session, dim, lit, bar bool) string {
	marker := r.sessionMarker(sess)
	tag := ""
	if dim {
		marker = ""
	} else {
		// 1 lead + name + 2 gap is what precedes the branch.
		tag = branchTag(sess.Branch, r.width-lipgloss.Width(sess.Name)-3)
	}
	nameColor := r.theme.Emphasis
	if dim && !lit {
		nameColor = r.theme.Muted
	}
	right := marker
	if tag != "" {
		right = "" // the branch takes the room the marker would have used
	}
	if lit {
		contentW := max(r.width-1, 0) // column 0 is the edge
		plain := sess.Name
		if tag != "" {
			plain += "  " + tag
		}
		gap := max(contentW-lipgloss.Width(plain)-lipgloss.Width(right), 0)
		plain += strings.Repeat(" ", gap) + right
		return r.leftEdge(bar) + lipgloss.NewStyle().Foreground(nameColor).
			Background(r.theme.SelBg).Render(padCol(plain, contentW))
	}
	left := " " + lipgloss.NewStyle().Foreground(nameColor).Render(sess.Name)
	if tag != "" {
		left += "  " + lipgloss.NewStyle().Foreground(r.theme.Muted).Render(tag)
	}
	if right != "" {
		right = lipgloss.NewStyle().Foreground(r.theme.Muted).Render(right)
	}
	return line(left, right, r.width)
}

// sessionBlock is a blank spacer (groups the sessions) above the name line.
// Both lines select the session; only the name lights. Dormant sessions pack
// tight - just the dimmed name, no spacer - so the sunk band stays compact.
func (r renderer) sessionBlock(sess model.Session, dim, lit, bar bool) []string {
	if dim {
		return []string{r.sessionRow(sess, true, lit, bar)}
	}
	return []string{"", r.sessionRow(sess, false, lit, bar)}
}

// stateColor is an agent's state color, muted once a finished agent has
// been seen so acknowledged work stops shouting.
func (r renderer) stateColor(a model.Agent) lipgloss.Color {
	if a.State == model.StateDone && a.Seen {
		return r.theme.Muted
	}
	return r.theme.StateColor(a.State)
}

// titleRow is the agent's first line: Claude's own title for the session,
// indented under it and colored by state so scanning the list reads as
// attention at a glance. An agent Claude has not titled yet draws no such line.
func (r renderer) titleRow(text string, col lipgloss.Color, lit, bar bool) string {
	s := lipgloss.NewStyle().Foreground(col).Bold(true)
	if lit {
		// The edge occupies column 0, so the text keeps its indent either way.
		s = s.Background(r.theme.SelBg)
		return r.leftEdge(bar) + s.Render(padCol(agentIndent[1:]+text, max(r.width-1, 0)))
	}
	return s.Render(padCol(agentIndent+text, r.width))
}

// agentRow is the agent's state line, a step deeper than its title: glyph,
// state, and the elapsed time at the right edge. The pane's command is not
// drawn - it is "claude" on every row, and the title above already says whose
// row this is, so the eight columns go to the title instead.
func (r renderer) agentRow(a model.Agent, lit, bar bool, frame int, now time.Time) string {
	col := r.stateColor(a)
	// Each fragment carries its own style: an outer background would break at
	// the inner resets and leave the highlight half-painted.
	frag := func(c lipgloss.Color) lipgloss.Style {
		s := lipgloss.NewStyle().Foreground(c)
		if lit {
			s = s.Background(r.theme.SelBg)
		}
		return s
	}
	indent := stateIndent
	if lit {
		indent = indent[1:] // column 0 is the edge
	}
	left := frag(col).Render(indent+stateIcon(a.State, frame)+" ") +
		frag(col).Render(padCol(a.State.Label(), labelW))
	age := frag(r.theme.Muted).Render(elapsed(a.Since, now))
	if lit {
		gap := max(r.width-1-lipgloss.Width(left)-lipgloss.Width(age), 0)
		return r.leftEdge(bar) + left + frag(r.theme.Muted).Render(strings.Repeat(" ", gap)) + age
	}
	return line(left, age+" ", r.width)
}

// subRow renders a secondary line of an agent block (branch, subagents),
// carrying the block's fill (and selection edge) edge to edge. Text is
// padded before styling so the line sits in one styled run with no bg gaps.
func (r renderer) subRow(text string, italic, lit, bar bool) string {
	s := lipgloss.NewStyle().Foreground(r.theme.Muted).Italic(italic)
	if lit {
		s = s.Background(r.theme.SelBg)
		return r.leftEdge(bar) + s.Render(padCol("    "+text, max(r.width-1, 0)))
	}
	return s.Render(padCol("     "+text, r.width))
}

// agentBlock renders one agent under its session: its title as a state-colored
// line, the state line a step deeper, and a subagent count when any. Every
// agent draws its own title - unlike a branch, no two of them are the same - so
// nothing is collapsed.
func (r renderer) agentBlock(sess model.Session, idx int, lit, bar bool, frame int, now time.Time) []string {
	a := sess.Agents[idx]
	var lines []string
	if a.Title != "" {
		lines = append(lines, r.titleRow(a.Title, r.stateColor(a), lit, bar))
	}
	lines = append(lines, r.agentRow(a, lit, bar, frame, now))
	if a.Subagents > 0 {
		plural := "s"
		if a.Subagents == 1 {
			plural = ""
		}
		lines = append(lines, r.subRow("⤷ "+strconv.Itoa(a.Subagents)+" subagent"+plural, false, lit, bar))
	}
	return lines
}

// blockLineCount mirrors each block's rendered line count without rendering.
func blockLineCount(b block, snap model.Snapshot) int {
	if b.kind == blockSection {
		n := 1 // the divider line
		if b.pad {
			n++ // leading blank
		}
		if b.gapAfter {
			n++ // trailing blank
		}
		return n
	}
	if b.kind == blockSession {
		if snap.Sessions[b.session].Band() == 2 {
			return 1 // dormant: dimmed name only, no spacer
		}
		return 2 // spacer + name
	}
	sess := snap.Sessions[b.session]
	a := sess.Agents[b.agent]
	n := 1
	if a.Title != "" {
		n++
	}
	if a.Subagents > 0 {
		n++
	}
	return n
}

func (r renderer) footer(snap model.Snapshot) string {
	var status string
	if att := snap.Attention(); att > 0 {
		status = lipgloss.NewStyle().Foreground(r.theme.Asking).Bold(true).
			Render(fmt.Sprintf(" ⚠ %d need attention", att))
	} else {
		status = lipgloss.NewStyle().Foreground(r.theme.Muted).Render(" all quiet")
	}
	help := " j/k · ⏎ · p pin · q"
	if snap.Attention() > 0 {
		help = " j/k · tab ⚠ · p pin · q" // tab steps through agents waiting on you
	}
	hint := lipgloss.NewStyle().Foreground(r.theme.Muted).Render(help)
	// Notifications moved to the settings dialogue: one home per setting.
	return r.sep() + "\n" + line(status, "", r.width) + "\n" + hint
}
