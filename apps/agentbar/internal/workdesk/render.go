package workdesk

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Widths the documents truncate titles to. The board has a whole line; the matrix has
// a table cell competing with eight other columns.
const (
	boardTitleWidth  = 52
	matrixTitleWidth = 34
)

// doc accumulates lines and writes them once. Every render here is a list of lines, so
// building them as strings and joining beats interleaving writes with formatting.
type doc struct {
	lines []string
}

func (d *doc) add(lines ...string)          { d.lines = append(d.lines, lines...) }
func (d *doc) addf(format string, a ...any) { d.add(fmt.Sprintf(format, a...)) }
func (d *doc) blank()                       { d.add("") }
func (d *doc) String() string               { return strings.Join(d.lines, "\n") + "\n" }
func (d *doc) writeTo(w io.Writer) error    { _, err := io.WriteString(w, d.String()); return err }

// bits joins the "a · b · c" trail every board row carries, skipping absent fields so
// there is never an orphan separator.
func bits(parts ...string) string { return join("  ·  ", parts...) }

// noteFor is the board's explanation of why a merge request is where it is: how long it
// has sat, what the pipeline says, which gates are refusing, who has it, and whether
// there are threads outstanding.
func noteFor(mr *MergeRequest, now time.Time) string {
	return bits(
		Age(ParseTime(mr.UpdatedAt), now),
		mr.PipelineLabel(),
		strings.Join(mr.Blockers(), ", "),
		strings.Join(mr.ReviewerStates(), " "),
		mr.Threads(),
	)
}

// Board renders the whole queue grouped by band, active bands first.
//
// Bands are emitted generically from the data rather than as a fixed set of sections,
// so a band with nothing in it simply does not appear and adding one costs nothing
// here.
func Board(w io.Writer, m *Mirror, now time.Time) error {
	type group struct {
		band Band
		mrs  []*MergeRequest
	}
	var groups []group
	need := 0
	for i := range m.MRs {
		mr := &m.MRs[i]
		b := mr.Band()
		if b.Active() {
			need++
		}
		if k := slices.IndexFunc(groups, func(g group) bool { return g.band == b }); k >= 0 {
			groups[k].mrs = append(groups[k].mrs, mr)
			continue
		}
		groups = append(groups, group{band: b, mrs: []*MergeRequest{mr}})
	}
	slices.SortStableFunc(groups, func(a, b group) int { return int(a.band) - int(b.band) })
	for i := range groups {
		slices.SortStableFunc(groups[i].mrs, func(a, b *MergeRequest) int {
			return strings.Compare(a.UpdatedAt, b.UpdatedAt)
		})
	}

	d := &doc{}
	d.add("# Work board", "")
	d.addf("_%s · %s · as %s · %d open MRs, %d open issues_",
		m.Meta.Synced, m.Meta.Project, m.Meta.User, len(m.MRs), len(m.Issues))
	d.blank()
	d.addf("%d merge requests are asking something of you. Band names are GitLab's own,", need)
	d.add("from the merge request homepage; `workdesk mr <iid>` is one merge request end to end.")
	d.blank()

	for _, g := range groups {
		inactive := ""
		if !g.band.Active() {
			inactive = "  _inactive_"
		}
		d.addf("## %s (%d)%s", g.band, len(g.mrs), inactive)
		d.blank()
		for _, mr := range g.mrs {
			d.addf("- !%s  %s  ·  %s", mr.IID, Short(mr.Title, boardTitleWidth), noteFor(mr, now))
		}
		d.blank()
	}

	var hot []*Issue
	for i := range m.Issues {
		if m.Issues[i].Priority() == PrioHigh {
			hot = append(hot, &m.Issues[i])
		}
	}
	d.addf("## prio::high issues (%d)", len(hot))
	d.blank()
	for _, is := range hot {
		d.addf("- #%s  %s", is.IID, Short(is.Title, boardTitleWidth))
	}
	return d.writeTo(w)
}

// Sheet renders one merge request end to end: whether it can merge, why not, and every
// rule and reviewer that bears on it.
//
// "Can I merge it?" comes from GitLab's own mergeabilityChecks and is never inferred,
// and the approvals table is the part that explains a stuck merge request - a count can
// read as satisfied while GitLab refuses, because the approver was not eligible for the
// rule that gates it.
func Sheet(w io.Writer, mr *MergeRequest, meta Meta, now time.Time) error {
	d := &doc{}
	d.addf("# !%s  %s", mr.IID, mr.Title)
	d.blank()
	d.addf("_%s · [open in gitlab](%s)_", meta.Synced, mr.WebURL)
	d.blank()
	d.addf("branch    `%s` → `%s`", mr.SourceBranch, mr.TargetBranch)
	d.addf("size      %d commits, +%d/-%d in %d files",
		mr.CommitCount, mr.DiffStats.Additions, mr.DiffStats.Deletions, mr.DiffStats.FileCount)
	d.addf("pipeline  %s", mr.PipelineLabel())
	d.addf("auto      %s", autoMergeLine(mr))
	d.addf("updated   %s", AgeAgo(ParseTime(mr.UpdatedAt), now))
	d.blank()

	blockers := mr.Blockers()
	if len(blockers) == 0 {
		d.add("## Can I merge it?  **Yes**", "", "Every gate GitLab checks is green.")
	} else {
		d.addf("## Can I merge it?  **No - %d blocker(s)**", len(blockers))
		d.blank()
		for _, b := range blockers {
			d.addf("- ✗ %s", b)
		}
		for _, p := range mr.Pending() {
			d.addf("- … %s (still checking)", p)
		}
	}
	d.blank()

	d.add("## Pipeline", "")
	if mr.HeadPipeline == nil {
		d.add("No pipeline has run.")
	} else {
		d.add("| stage | status |", "|---|---|")
		for _, s := range mr.HeadPipeline.Stages.Nodes {
			d.addf("| %s | %s |", s.Name, strings.ToLower(s.Status))
		}
	}
	d.blank()

	d.addf("## Approvals  %d of %d", mr.ApprovalsRequired-mr.ApprovalsLeft, mr.ApprovalsRequired)
	d.blank()
	d.add("A rule met by nobody eligible is why an approval count can read as satisfied",
		"while GitLab still refuses the merge.", "")
	d.add("| rule | required | met | approved by |", "|---|---|---|---|")
	for _, r := range mr.ApprovalState.Rules {
		by := make([]string, 0, len(r.ApprovedBy.Nodes))
		for _, u := range r.ApprovedBy.Nodes {
			by = append(by, u.Username)
		}
		who := strings.Join(by, ", ")
		if who == "" {
			who = "-"
		}
		d.addf("| %s | %d | %s | %s |", r.Name, r.ApprovalsRequired, yesNo(r.Approved), who)
	}
	d.blank()

	d.add("## Reviewers", "")
	if len(mr.Reviewers.Nodes) == 0 {
		d.add("**Nobody is assigned.** This will not move until someone is.")
	} else {
		d.add("| who | state |", "|---|---|")
		for _, r := range mr.Reviewers.Nodes {
			d.addf("| %s | %s |", r.Username, strings.ToLower(r.Interaction.ReviewState))
		}
	}
	d.blank()

	d.add("## Description", "")
	if strings.TrimSpace(mr.Description) == "" {
		d.add("_No description._")
	} else {
		d.add(mr.Description)
	}
	d.blank()

	d.add("## Threads", "")
	if mr.ResolvableDiscussionsCount == 0 {
		d.add("No review threads.")
	} else {
		d.addf("%d of %d resolved.", mr.ResolvedDiscussionsCount, mr.ResolvableDiscussionsCount)
	}
	d.blank()
	// System notes are GitLab talking to itself ("added 3 commits", "assigned to @x")
	// and drown the human argument, so a thread with nothing but system notes is
	// dropped rather than shown empty.
	for _, disc := range mr.Discussions.Nodes {
		var human []Note
		for _, n := range disc.Notes.Nodes {
			if !n.System {
				human = append(human, n)
			}
		}
		if len(human) == 0 {
			continue
		}
		d.add("", "**["+resolvedLabel(disc.Resolved)+"]**")
		for _, n := range human {
			who := n.Author.Username
			if who == "" {
				who = "?"
			}
			d.addf("- **%s**  %s", who, indentBody(n.Body))
		}
	}
	return d.writeTo(w)
}

func autoMergeLine(mr *MergeRequest) string {
	if mr.AutoMergeEnabled {
		return "auto-merge on"
	}
	return "auto-merge not set"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func resolvedLabel(b bool) string {
	if b {
		return "resolved"
	}
	return "OPEN"
}

// indentBody keeps a multi-line comment inside its bullet: carriage returns dropped,
// continuation lines indented so markdown does not end the list item.
func indentBody(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	return strings.ReplaceAll(s, "\n", "\n  ")
}

// IssueSheet is everything known about an issue without another API call: an issue
// carries no gates, so its row plus its labels is the whole of it.
func IssueSheet(w io.Writer, is *Issue) error {
	d := &doc{}
	d.addf("# #%s  %s", is.IID, is.Title)
	d.blank()
	d.add(is.WebURL)
	d.blank()
	labels := make([]string, 0, len(is.Labels.Nodes))
	for _, l := range is.Labels.Nodes {
		labels = append(labels, l.Title)
	}
	joined := strings.Join(labels, ", ")
	if joined == "" {
		joined = "none"
	}
	d.addf("labels: %s", joined)
	updated := is.UpdatedAt
	if len(updated) >= 10 {
		updated = updated[:10]
	}
	d.addf("updated: %s", updated)
	return d.writeTo(w)
}

// Matrix renders one row per merge request and one column per gate, with a totals row.
//
// The totals are the point: across a queue they say whether everything is stuck on the
// same thing, which is one setting to change rather than fifty merge requests to chase.
// No list of merge requests can show that.
func Matrix(w io.Writer, m *Mirror) error {
	ordered := make([]*MergeRequest, 0, len(m.MRs))
	for i := range m.MRs {
		ordered = append(ordered, &m.MRs[i])
	}
	slices.SortStableFunc(ordered, func(a, b *MergeRequest) int {
		if ab, bb := a.Band(), b.Band(); ab != bb {
			return int(ab) - int(bb)
		}
		return strings.Compare(a.UpdatedAt, b.UpdatedAt)
	})

	var stuck struct{ draft, pipeline, approvals, threads, conflicts, autoOff int }
	d := &doc{}
	d.add("# Gate matrix", "")
	d.add("| mr | title | draft | ci | appr | thr | conf | auto | band |", "|---|---|---|---|---|---|---|---|---|")
	for _, mr := range ordered {
		if mr.Draft {
			stuck.draft++
		}
		if mr.CIStatus() == "FAILED" {
			stuck.pipeline++
		}
		if mr.ApprovalsLeft > 0 {
			stuck.approvals++
		}
		if mr.ResolvableDiscussionsCount > mr.ResolvedDiscussionsCount {
			stuck.threads++
		}
		if mr.Conflicts {
			stuck.conflicts++
		}
		if !mr.AutoMergeEnabled {
			stuck.autoOff++
		}
		d.addf("| !%s | %s | %s | %s | %s | %s | %s | %s | %s |",
			mr.IID, Short(mr.Title, matrixTitleWidth), mark(mr.Draft), ciCell(mr),
			strconv.Itoa(mr.ApprovalsRequired-mr.ApprovalsLeft)+"/"+strconv.Itoa(mr.ApprovalsRequired),
			threadCell(mr), mark(mr.Conflicts), autoCell(mr), mr.Band())
	}
	d.blank()
	d.add("## Stuck on", "")
	d.add("| gate | count |", "|---|---|")
	for _, row := range []struct {
		name string
		n    int
	}{
		{"draft", stuck.draft},
		{"pipeline", stuck.pipeline},
		{"approvals", stuck.approvals},
		{"threads", stuck.threads},
		{"conflicts", stuck.conflicts},
		{"auto-merge off", stuck.autoOff},
	} {
		d.addf("| %s | %d |", row.name, row.n)
	}
	d.blank()
	d.add("_A gate that is high here is one setting to change, not one merge request at a time._")
	return d.writeTo(w)
}

func mark(b bool) string {
	if b {
		return "x"
	}
	return "·"
}

func ciCell(mr *MergeRequest) string {
	switch mr.CIStatus() {
	case "FAILED":
		return "x"
	case "SUCCESS":
		return "ok"
	default:
		return "·"
	}
}

func threadCell(mr *MergeRequest) string {
	if mr.ResolvableDiscussionsCount == 0 {
		return "·"
	}
	return strconv.Itoa(mr.ResolvedDiscussionsCount) + "/" + strconv.Itoa(mr.ResolvableDiscussionsCount)
}

func autoCell(mr *MergeRequest) string {
	if mr.AutoMergeEnabled {
		return "on"
	}
	return "off"
}
