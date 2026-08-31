package deskui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/abhishekrana/agentbar/internal/workdesk"
)

// The preview is built from the snapshot in memory rather than by reading a
// pre-rendered markdown file.
//
// That is only possible because the model is held: fzf re-ran its preview command on
// every cursor movement, so the shell had to pre-render markdown at sync time and cat it.
// Rendering natively means the gates can carry colour, the approvals can be a real table,
// and the whole thing can scroll. The markdown documents still exist - `workdesk mr
// <iid>` and `board` are for reading and for agents to consume - but nothing interactive
// depends on them now.
func (m Model) renderPreview(r workdesk.Row) string {
	var body string
	switch {
	case strings.HasPrefix(r.Ref, "!"):
		body = m.mrPreview(strings.TrimPrefix(r.Ref, "!"))
	case strings.HasPrefix(r.Ref, "#"):
		body = m.issuePreview(strings.TrimPrefix(r.Ref, "#"))
	default:
		body = m.agentPreview(r)
	}
	// Wrapped once, here, so nothing a preview builds can run off the pane: the
	// viewport truncates the lines it cannot fit, and a truncated line is one whose
	// right-hand half is silently missing. A title, a url and a ticket body are all
	// written to no particular width. Sections that wrapped themselves are already
	// inside it, so this is a no-op for them.
	return m.wrap(body, 0)
}

// styles bundles the roles a preview draws with, so each render reads as layout rather
// than as a pile of lipgloss calls.
type styles struct {
	head, key, val, muted, ok, bad, warn, accent lipgloss.Style
}

func (m Model) styles() styles {
	t := m.theme
	base := lipgloss.NewStyle()
	return styles{
		head:   base.Foreground(t.Emphasis).Bold(true),
		key:    base.Foreground(t.Muted),
		val:    base.Foreground(t.Fg),
		muted:  base.Foreground(t.Muted),
		ok:     base.Foreground(t.Done),
		bad:    base.Foreground(t.Blocked),
		warn:   base.Foreground(t.Asking),
		accent: base.Foreground(t.Accent),
	}
}

// kv is the aligned key/value block every preview opens with.
func (s styles) kv(k, v string) string {
	return s.key.Render(fmt.Sprintf("%-10s", k)) + v
}

func (m Model) mrPreview(iid string) string {
	mr := m.findMR(iid)
	if mr == nil {
		return m.styles().muted.Render("not in the mirror")
	}
	s := m.styles()
	now := m.deps.Now()
	var b strings.Builder

	fmt.Fprintln(&b, s.head.Render("!"+mr.IID+"  "+mr.Title))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, s.kv("branch", s.accent.Render(mr.SourceBranch)+s.muted.Render(" → "+mr.TargetBranch)))
	fmt.Fprintln(&b, s.kv("size", s.val.Render(fmt.Sprintf("%d commits, +%d/-%d in %d files",
		mr.CommitCount, mr.DiffStats.Additions, mr.DiffStats.Deletions, mr.DiffStats.FileCount))))
	fmt.Fprintln(&b, s.kv("pipeline", m.ciStyled(mr)))
	fmt.Fprintln(&b, s.kv("auto", m.autoStyled(mr)))
	fmt.Fprintln(&b, s.kv("updated", s.val.Render(workdesk.AgeAgo(workdesk.ParseTime(mr.UpdatedAt), now))))
	fmt.Fprintln(&b)

	// The one question the sheet exists to answer, and GitLab's own answer to it.
	blockers := mr.Blockers()
	if len(blockers) == 0 {
		fmt.Fprintln(&b, s.head.Render("Can I merge it?")+"  "+s.ok.Render("yes"))
		fmt.Fprintln(&b, s.muted.Render("every gate GitLab checks is green"))
	} else {
		fmt.Fprintln(&b, s.head.Render("Can I merge it?")+"  "+
			s.bad.Render(fmt.Sprintf("no · %d blocker(s)", len(blockers))))
		for _, g := range blockers {
			fmt.Fprintln(&b, "  "+s.bad.Render("✗")+" "+s.val.Render(g))
		}
		for _, g := range mr.Pending() {
			fmt.Fprintln(&b, "  "+s.warn.Render("…")+" "+s.muted.Render(g+" (still checking)"))
		}
	}
	fmt.Fprintln(&b)

	// Approvals, because this is the part that explains a stuck merge request: a count
	// can read as satisfied while GitLab refuses, because the approver was not eligible
	// for the rule that gates it.
	fmt.Fprintln(&b, s.head.Render(fmt.Sprintf("Approvals  %d of %d",
		mr.ApprovalsRequired-mr.ApprovalsLeft, mr.ApprovalsRequired)))
	if len(mr.ApprovalState.Rules) > 0 {
		fmt.Fprintln(&b, m.rulesTable(mr))
	} else {
		fmt.Fprintln(&b, s.muted.Render("no approval rules on this project"))
	}
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, s.head.Render("Reviewers"))
	if len(mr.Reviewers.Nodes) == 0 {
		fmt.Fprintln(&b, s.warn.Render("nobody is assigned")+
			s.muted.Render(" — this will not move until someone is"))
	} else {
		for _, rv := range mr.Reviewers.Nodes {
			glyph, style := m.reviewGlyph(rv.Interaction.ReviewState)
			fmt.Fprintln(&b, "  "+style.Render(glyph)+" "+s.val.Render(rv.Username)+
				s.muted.Render("  "+strings.ToLower(rv.Interaction.ReviewState)))
		}
	}
	fmt.Fprintln(&b)

	if t := mr.Threads(); t != "" {
		fmt.Fprintln(&b, s.head.Render("Threads")+"  "+s.muted.Render(t))
		// One line each: on a merge request the threads are one per line of the diff,
		// and what is wanted here is which arguments are still open.
		m.writeThreads(&b, mr.Discussions.Nodes, true)
		fmt.Fprintln(&b)
	}

	m.writeDescription(&b, mr.Description)
	return b.String()
}

// writeThreads renders a discussion list. brief keeps each note to its first line, for
// the merge request preview where the count matters more than the argument.
func (m Model) writeThreads(b *strings.Builder, discussions []workdesk.Discussion, brief bool) {
	s := m.styles()
	for _, disc := range discussions {
		for _, n := range disc.Notes.Nodes {
			// System notes are GitLab talking to itself ("added 3 commits").
			if n.System {
				continue
			}
			state := s.warn.Render("OPEN")
			if disc.Resolved {
				state = s.muted.Render("resolved")
			}
			who := n.Author.Username
			if who == "" {
				who = "?"
			}
			fmt.Fprintf(b, "  [%s] %s\n", state, s.accent.Render(who))
			if brief {
				style := s.val
				if disc.Resolved {
					style = s.muted
				}
				fmt.Fprintln(b, "      "+style.Render(firstLine(n.Body)))
				continue
			}
			fmt.Fprintln(b, m.comment(n.Body))
		}
	}
}

func (m Model) writeDescription(b *strings.Builder, body string) {
	s := m.styles()
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	fmt.Fprintln(b, s.head.Render("Description"))
	fmt.Fprintln(b, m.markdown(body))
}

// wrap reflows a body to the preview's own width, less an indent.
//
// A ticket body is prose written to no particular width, and the viewport truncates the
// lines it cannot fit rather than wrapping them - so without this the right-hand half of
// every long paragraph is simply not on screen. Zero width means the preview has not been
// sized yet, where the text is left alone rather than reflowed to nothing.
func (m Model) wrap(body string, indent int) string {
	w := m.preview.Width - indent
	if w <= 0 {
		return body
	}
	return lipgloss.NewStyle().Width(w).Render(body)
}

func (m Model) indent(body, pad string) string {
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}

func (m Model) rulesTable(mr *workdesk.MergeRequest) string {
	s := m.styles()
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(m.theme.Muted)).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return s.key.Padding(0, 1)
			}
			return s.val.Padding(0, 1)
		}).
		Headers("rule", "need", "met", "by")
	for _, r := range mr.ApprovalState.Rules {
		by := make([]string, 0, len(r.ApprovedBy.Nodes))
		for _, u := range r.ApprovedBy.Nodes {
			by = append(by, u.Username)
		}
		who := strings.Join(by, ", ")
		met := s.bad.Render("no")
		if r.Approved {
			met = s.ok.Render("yes")
		}
		if who == "" {
			who = s.warn.Render("nobody")
		}
		t.Row(workdesk.Short(r.Name, 24), fmt.Sprint(r.ApprovalsRequired), met, who)
	}
	return t.String()
}

func (m Model) issuePreview(iid string) string {
	is := m.findIssue(iid)
	if is == nil {
		return m.styles().muted.Render("not in the mirror")
	}
	s := m.styles()
	var b strings.Builder
	fmt.Fprintln(&b, s.head.Render("#"+is.IID+"  "+is.Title))
	fmt.Fprintln(&b)

	labels := make([]string, 0, len(is.Labels.Nodes))
	for _, l := range is.Labels.Nodes {
		labels = append(labels, l.Title)
	}
	joined := strings.Join(labels, sep)
	if joined == "" {
		joined = s.muted.Render("none")
	}
	who := make([]string, 0, len(is.Assignees.Nodes))
	for _, a := range is.Assignees.Nodes {
		who = append(who, a.Username)
	}
	assignees := strings.Join(who, sep)
	if assignees == "" {
		assignees = s.warn.Render("nobody")
	}

	fmt.Fprintln(&b, s.kv("status", s.accent.Render(is.StatusName())))
	fmt.Fprintln(&b, s.kv("priority", s.warn.Render(is.Priority().String())))
	fmt.Fprintln(&b, s.kv("labels", s.val.Render(joined)))
	fmt.Fprintln(&b, s.kv("assignees", s.val.Render(assignees)))
	fmt.Fprintln(&b, s.kv("sprint", s.val.Render(m.sprintLine(is))))
	fmt.Fprintln(&b, s.kv("updated", s.val.Render(
		workdesk.AgeAgo(workdesk.ParseTime(is.UpdatedAt), m.deps.Now()))))

	// The column that turns an issue list into a pick-next list.
	if mr := workdesk.InFlightFor(is.IID, m.deps.Mirror.MRs); mr != nil {
		fmt.Fprintln(&b, s.kv("in flight", s.accent.Render("!"+mr.IID)+
			s.muted.Render("  "+mr.SourceBranch)))
	} else {
		fmt.Fprintln(&b, s.kv("in flight", s.muted.Render("nothing yet")))
	}
	fmt.Fprintln(&b, s.kv("url", s.muted.Render(is.WebURL)))
	fmt.Fprintln(&b)

	// The ticket itself, in the order GitLab writes it: what was asked, then what was
	// said about it. Whole comments rather than first lines - on an issue the argument
	// is the content, where on a merge request it annotates a diff you can go and read.
	m.writeDescription(&b, is.Description)
	if n := comments(is.Discussions.Nodes); n > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, s.head.Render("Comments")+"  "+s.muted.Render(fmt.Sprint(n)))
		m.writeThreads(&b, is.Discussions.Nodes, false)
	}
	return b.String()
}

// comments counts what a person wrote, which is every note GitLab did not.
func comments(discussions []workdesk.Discussion) int {
	n := 0
	for _, d := range discussions {
		for _, note := range d.Notes.Nodes {
			if !note.System {
				n++
			}
		}
	}
	return n
}

// sprintLine says where the issue stands against the sprint the sync recorded, because
// that is what `i` will move it into or out of. An issue in some other iteration is named
// rather than reported as out: "not in it" would be false.
func (m Model) sprintLine(is *workdesk.Issue) string {
	current := m.deps.Mirror.Meta.Iteration
	switch {
	case is.InSprint(current):
		return sprintMark + " " + current.Label() + "  " + is.Iteration.Cadence.Title
	case is.Iteration != nil:
		return is.Iteration.Label() + "  " + is.Iteration.Cadence.Title
	case current != nil:
		return "not in " + current.Label()
	default:
		return "none"
	}
}

func (m Model) agentPreview(r workdesk.Row) string {
	s := m.styles()
	var agent workdesk.Agent
	if m.deps.Agents != nil {
		for _, a := range m.deps.Agents() {
			if a.Pane == r.Ref {
				agent = a
			}
		}
	}
	if agent.Pane == "" {
		return s.muted.Render("that pane is gone")
	}

	var b strings.Builder
	title := agent.Title
	if title == "" {
		title = "untitled"
	}
	fmt.Fprintln(&b, s.head.Render(title))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, s.kv("state", m.agentStateStyled(agent.State)))
	fmt.Fprintln(&b, s.kv("pane", s.val.Render(agent.Pane)))
	fmt.Fprintln(&b, s.kv("worktree", s.val.Render(orDash(agent.Worktree))))
	fmt.Fprintln(&b, s.kv("branch", s.accent.Render(orDash(agent.Branch))))
	fmt.Fprintln(&b, s.kv("age", s.val.Render(r.Age)))
	fmt.Fprintln(&b)

	if strings.HasPrefix(r.Note, "!") {
		fmt.Fprintln(&b, s.kv("merge req", s.accent.Render(r.Note)))
		fmt.Fprintln(&b)
		fmt.Fprint(&b, m.mrPreview(strings.TrimPrefix(r.Note, "!")))
		return b.String()
	}

	// The finding this whole view exists for.
	fmt.Fprintln(&b, s.warn.Render("no merge request for this branch"))
	fmt.Fprintln(&b, s.muted.Render("Nothing in GitLab knows this work exists, so it"))
	fmt.Fprintln(&b, s.muted.Render("appears in no other view."))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, s.muted.Render("↵ switches to this pane."))
	return b.String()
}

func (m Model) ciStyled(mr *workdesk.MergeRequest) string {
	s := m.styles()
	switch mr.CIStatus() {
	case "FAILED":
		return s.bad.Render(mr.PipelineLabel())
	case "SUCCESS":
		return s.ok.Render(mr.PipelineLabel())
	case "NONE":
		return s.muted.Render(mr.PipelineLabel())
	default:
		return s.warn.Render(mr.PipelineLabel())
	}
}

func (m Model) autoStyled(mr *workdesk.MergeRequest) string {
	s := m.styles()
	if mr.AutoMergeEnabled {
		strategy := mr.AutoMergeStrategy
		if strategy == "" {
			return s.ok.Render("on")
		}
		return s.ok.Render("on") + s.muted.Render("  "+strings.ToLower(strategy))
	}
	// Not an error, but worth a colour: a green merge request with auto-merge off is a
	// setting nobody turned on rather than work anybody has to do.
	return s.warn.Render("not set")
}

func (m Model) reviewGlyph(state string) (string, lipgloss.Style) {
	s := m.styles()
	switch state {
	case "APPROVED":
		return "✓", s.ok
	case "REQUESTED_CHANGES":
		return "✗", s.bad
	case "REVIEWED":
		return "•", s.warn
	default:
		return "·", s.muted
	}
}

func (m Model) agentStateStyled(state string) string {
	s := m.styles()
	switch state {
	case "permission":
		return s.bad.Render(state)
	case "question":
		return s.warn.Render(state)
	case "working":
		return lipgloss.NewStyle().Foreground(m.theme.Working).Render(state)
	case "done":
		return s.ok.Render(state)
	default:
		return s.muted.Render(state)
	}
}

func (m Model) findMR(iid string) *workdesk.MergeRequest {
	for i := range m.deps.Mirror.MRs {
		if m.deps.Mirror.MRs[i].IID == iid {
			return &m.deps.Mirror.MRs[i]
		}
	}
	return nil
}

func (m Model) findIssue(iid string) *workdesk.Issue {
	for i := range m.deps.Mirror.Issues {
		if m.deps.Mirror.Issues[i].IID == iid {
			return &m.deps.Mirror.Issues[i]
		}
	}
	return nil
}

func firstLine(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return workdesk.Short(s[:i], 60)
	}
	return workdesk.Short(s, 60)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
