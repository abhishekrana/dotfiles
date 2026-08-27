package workdesk

import (
	"io"
	"strings"
	"time"
)

// View is which question is being asked. The order is the ring 1-4 and tab walk,
// defined once here so no caller can disagree about it.
type View int

const (
	ViewInbox View = iota
	ViewMRs
	ViewIssues
	ViewAgents
)

var viewNames = [...]string{"inbox", "mrs", "issues", "agents"}

func (v View) String() string {
	if int(v) < len(viewNames) {
		return viewNames[v]
	}
	return "inbox"
}

// ParseView maps a name or a 1-4 key to a view, defaulting to the inbox rather than
// failing: a typo should land you somewhere useful.
func ParseView(s string) View {
	switch s {
	case "mrs", "mr", "2":
		return ViewMRs
	case "issues", "issue", "3":
		return ViewIssues
	case "agents", "agent", "4":
		return ViewAgents
	default:
		return ViewInbox
	}
}

// Views is the ring, in order.
func Views() []View { return []View{ViewInbox, ViewMRs, ViewIssues, ViewAgents} }

// Next returns the view after this one, wrapping.
func (v View) Next() View { return View((int(v) + 1) % len(viewNames)) }

// Row is one line: the band it sits under, whether that band is asking anything
// of you, and the fields the eye reads.
type Row struct {
	Label  string
	Flag   string
	Ref    string
	Title  string
	Age    string
	Note   string
	Branch string
}

// TSV is the wire format for a reader that is not this program:
//
//	label  flag  ref  title  age  note  branch
//
// The title is padded here rather than in the model, because a fixed column only makes
// sense for a fixed-width consumer - the UI sizes it to the terminal instead.
// Separators inside a field are escaped, so a merge request titled with a tab cannot
// invent a column.
func (r Row) TSV() string {
	return TSV(r.Label, r.Flag, r.Ref, Pad(r.Title, titleWidth), r.Age, r.Note, r.Branch)
}

// WriteRows emits one TSV line per row.
func WriteRows(w io.Writer, rows []Row) error {
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(r.TSV())
		b.WriteByte('\n')
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func (it Item) row(now time.Time) Row {
	return Row{
		Label:  it.Label,
		Flag:   it.Flag,
		Ref:    it.Ref,
		Title:  it.Title,
		Age:    Age(time.Unix(it.Updated, 0), now),
		Note:   it.Note,
		Branch: it.Branch,
	}
}

// Rows renders a view. now is passed in so a render is a pure function of the index
// plus the clock, which is what lets the golden tests freeze time.
func (idx *Index) Rows(v View, now time.Time) []Row {
	switch v {
	case ViewMRs:
		out := make([]Row, 0, len(idx.MRs))
		for _, it := range idx.MRs {
			out = append(out, it.row(now))
		}
		return out
	case ViewIssues:
		out := make([]Row, 0, len(idx.Issues))
		for _, it := range idx.Issues {
			out = append(out, it.row(now))
		}
		return out
	default:
		return idx.inbox(now)
	}
}

// inbox is the only view that mixes sources, so it is the only one with rules of its
// own:
//
//   - Todos come first, because GitLab vouches for them rather than this package
//     inferring them. But a todo for something that already has a row of its own would
//     be the same item twice, so its reason is folded into that row instead and only
//     todos with nowhere else to appear get a line.
//   - Merge requests appear when their band is asking something of you, plus the one
//     inactive band worth seeing.
//   - Issues appear only when they are top priority and nothing is in flight - the
//     unstarted work most worth picking up.
func (idx *Index) inbox(now time.Time) []Row {
	reason := make(map[string]string, len(idx.Todos))
	for _, td := range idx.Todos {
		if _, seen := reason[td.Ref]; !seen {
			reason[td.Ref] = td.Note
		}
	}

	mrRows := make([]Row, 0, len(idx.MRs))
	claimed := make(map[string]bool, len(idx.MRs))
	for _, it := range idx.MRs {
		if !it.Band.Active() && it.Band != BandApprovals {
			continue
		}
		claimed[it.Ref] = true
		r := it.row(now)
		if why, ok := reason[it.Ref]; ok {
			r.Note = why + " · " + r.Note
		}
		mrRows = append(mrRows, r)
	}

	issueRows := make([]Row, 0, 4)
	for _, it := range idx.Issues {
		if it.Prio != PrioHigh || it.Note != "not started" {
			continue
		}
		claimed[it.Ref] = true
		r := it.row(now)
		r.Label = "not started"
		r.Flag = "i"
		issueRows = append(issueRows, r)
	}

	out := make([]Row, 0, len(idx.Todos)+len(mrRows)+len(issueRows))
	for _, td := range idx.Todos {
		if claimed[td.Ref] {
			continue
		}
		out = append(out, td.row(now))
	}
	out = append(out, mrRows...)
	return append(out, issueRows...)
}

// RefFor is the handle an action acts on: kind and identifier, so nothing downstream
// needs to know which view produced the row.
func RefFor(r Row) string {
	switch {
	case strings.HasPrefix(r.Ref, "%"):
		return "agents:" + r.Ref
	case strings.HasPrefix(r.Ref, "#"):
		return "issues:" + strings.TrimPrefix(r.Ref, "#")
	case strings.HasPrefix(r.Ref, "!"):
		return "mrs:" + strings.TrimPrefix(r.Ref, "!")
	default:
		return r.Ref
	}
}
