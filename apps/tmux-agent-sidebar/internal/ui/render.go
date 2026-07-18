package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/abhishekrana/tmux-agent-sidebar/internal/model"
)

// spinnerFrames animates the working state (braille, like Claude's own UI).
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type blockKind int

const (
	blockSession blockKind = iota // group header, one line, not selectable
	blockAgent                    // agent + branch + subagents: one selectable unit
)

// block is one navigation unit; an agent block renders as 1-3 lines that
// select, highlight, and click together.
type block struct {
	kind    blockKind
	session int // index into snapshot.Sessions
	agent   int // index into session.Agents (blockAgent only)
}

// Both row kinds are selectable: an agent block jumps to its pane, a
// session header switches to that session.
func (block) selectable() bool { return true }

// buildBlocks flattens the snapshot: session headers are pure group
// labels; agents form the flat, selectable list.
func buildBlocks(snap model.Snapshot) []block {
	var blocks []block
	for si, sess := range snap.Sessions {
		blocks = append(blocks, block{kind: blockSession, session: si})
		for ai := range sess.Agents {
			blocks = append(blocks, block{kind: blockAgent, session: si, agent: ai})
		}
	}
	return blocks
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
	nameW int // agent-name column width, derived from the snapshot
}

// labelW fits the longest state label ("permission").
const labelW = 10

// nameColumn returns the width of the agent-name column: the longest name
// in the snapshot, clamped so long names can't crowd out the state columns.
func nameColumn(snap model.Snapshot) int {
	w := 6 // len("claude")
	for _, sess := range snap.Sessions {
		for _, a := range sess.Agents {
			if n := len([]rune(a.Command)); n > w {
				w = n
			}
		}
	}
	return min(w, 12)
}

// padCol pads (or truncates) a plain string to exactly w cells.
func padCol(s string, w int) string {
	r := []rune(s)
	if len(r) > w {
		return string(r[:w-1]) + "…"
	}
	return s + strings.Repeat(" ", w-len(r))
}

func (r renderer) sep() string {
	return lipgloss.NewStyle().Foreground(r.theme.Muted).Render(strings.Repeat("─", r.width))
}

func (r renderer) header(snap model.Snapshot, frame int) string {
	title := lipgloss.NewStyle().Foreground(r.theme.Emphasis).Bold(true).Render(" tmux agents")
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
// an empty session, nothing otherwise. (The current session isn't marked —
// the selection highlight, which follows every click, shows where you are.)
func (r renderer) sessionMarker(sess model.Session) string {
	if len(sess.Agents) == 0 {
		return "no agents "
	}
	return ""
}

// sessionRow is the session's single name line, indented one column so the
// selection's accent edge has a place to sit. When lit (hovered or
// selected) it fills; the selected row also shows the edge.
func (r renderer) sessionRow(sess model.Session, lit, bar bool) string {
	marker := r.sessionMarker(sess)
	if lit {
		contentW := max(r.width-1, 0) // column 0 is the edge
		gap := max(contentW-lipgloss.Width(sess.Name)-lipgloss.Width(marker), 0)
		plain := sess.Name + strings.Repeat(" ", gap) + marker
		return r.leftEdge(bar) + lipgloss.NewStyle().Foreground(r.theme.Emphasis).
			Background(r.theme.SelBg).Render(padCol(plain, contentW))
	}
	name := lipgloss.NewStyle().Foreground(r.theme.Emphasis).Render(sess.Name)
	right := ""
	if marker != "" {
		right = lipgloss.NewStyle().Foreground(r.theme.Muted).Render(marker)
	}
	return line(" "+name, right, r.width)
}

// sessionBlock is a blank spacer (groups the sessions) above the name line.
// Both lines select the session; only the name lights.
func (r renderer) sessionBlock(sess model.Session, lit, bar bool) []string {
	return []string{"", r.sessionRow(sess, lit, bar)}
}

// stateColor is an agent's state color, muted once a finished agent has
// been seen so acknowledged work stops shouting.
func (r renderer) stateColor(a model.Agent) lipgloss.Color {
	if a.State == model.StateDone && a.Seen {
		return r.theme.Muted
	}
	return r.theme.StateColor(a.State)
}

// attentionRank orders agents by how much they want the user, so a branch
// shared by several Claudes takes the color of its most-urgent one.
func attentionRank(a model.Agent) int {
	switch {
	case a.State.NeedsAttention():
		return 4
	case a.State == model.StateWorking:
		return 3
	case a.State == model.StateDone && !a.Seen:
		return 2
	default:
		return 1
	}
}

// agentShowsBranch reports whether this agent draws the branch headline:
// true unless it repeats the branch of the previous agent in the session,
// so several Claudes on one branch show that branch once.
func agentShowsBranch(sess model.Session, idx int) bool {
	if sess.Agents[idx].Branch == "" {
		return false
	}
	return idx == 0 || sess.Agents[idx-1].Branch != sess.Agents[idx].Branch
}

// groupColor is the state color of the most-urgent agent in the run of
// consecutive same-branch agents starting at idx.
func (r renderer) groupColor(sess model.Session, idx int) lipgloss.Color {
	lead := sess.Agents[idx]
	for j := idx + 1; j < len(sess.Agents) && sess.Agents[j].Branch == lead.Branch; j++ {
		if attentionRank(sess.Agents[j]) > attentionRank(lead) {
			lead = sess.Agents[j]
		}
	}
	return r.stateColor(lead)
}

// branchRow is the agent block's headline: the branch, colored by state so
// scanning the list reads as attention at a glance.
func (r renderer) branchRow(branch string, col lipgloss.Color, lit, bar bool) string {
	s := lipgloss.NewStyle().Foreground(col).Bold(true)
	if lit {
		// The edge occupies column 0 in place of the leading space, so the
		// branch stays at column 1 whether or not the row is lit.
		s = s.Background(r.theme.SelBg)
		return r.leftEdge(bar) + s.Render(padCol(branch, max(r.width-1, 0)))
	}
	return s.Render(padCol(" "+branch, r.width))
}

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
	// When lit, column 0 is the edge; the icon stays at column 3 either way.
	var row string
	if lit {
		row = r.leftEdge(bar) + frag(col).Render("  "+stateIcon(a.State, frame)+" ")
	} else {
		row = frag(col).Render("   " + stateIcon(a.State, frame) + " ")
	}
	row += frag(r.theme.Fg).Render(padCol(a.Command, r.nameW)) +
		frag(col).Render("  "+padCol(a.State.Label(), labelW)) +
		frag(r.theme.Muted).Render(fmt.Sprintf("%5s", elapsed(a.Since, now)))
	if pad := r.width - lipgloss.Width(row); pad > 0 && lit {
		row += frag(r.theme.Fg).Render(strings.Repeat(" ", pad))
	}
	return line(row, "", r.width)
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

// agentBlock renders an agent's full block: the branch as a state-colored
// headline (shown once per same-branch run), the status line beneath, and a
// subagent count when any.
func (r renderer) agentBlock(sess model.Session, idx int, lit, bar bool, frame int, now time.Time) []string {
	a := sess.Agents[idx]
	var lines []string
	if agentShowsBranch(sess, idx) {
		lines = append(lines, r.branchRow(a.Branch, r.groupColor(sess, idx), lit, bar))
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

// blockLineCount mirrors agentBlock's line count without rendering.
func blockLineCount(b block, snap model.Snapshot) int {
	if b.kind == blockSession {
		return 2 // name + summary line
	}
	sess := snap.Sessions[b.session]
	a := sess.Agents[b.agent]
	n := 1
	if agentShowsBranch(sess, b.agent) {
		n++
	}
	if a.Subagents > 0 {
		n++
	}
	return n
}

func (r renderer) footer(snap model.Snapshot, notify bool) string {
	var status string
	if att := snap.Attention(); att > 0 {
		status = lipgloss.NewStyle().Foreground(r.theme.Asking).Bold(true).
			Render(fmt.Sprintf(" ⚠ %d need attention", att))
	} else {
		status = lipgloss.NewStyle().Foreground(r.theme.Muted).Render(" all quiet")
	}
	help := " j/k · ⏎ · n notify · q hide"
	if snap.Attention() > 0 {
		help = " j/k · tab ⚠ · ⏎ jump · q" // tab steps through agents waiting on you
	}
	hint := lipgloss.NewStyle().Foreground(r.theme.Muted).Render(help)
	return r.sep() + "\n" + line(status, r.notifyChip(notify), r.width) + "\n" + hint
}

// notifyChip is the desktop-notification toggle's state, shown at the right
// of the status line: green "notify on", muted "notify off".
func (r renderer) notifyChip(on bool) string {
	if on {
		return lipgloss.NewStyle().Foreground(r.theme.Done).Render("notify on")
	}
	return lipgloss.NewStyle().Foreground(r.theme.Muted).Render("notify off")
}
