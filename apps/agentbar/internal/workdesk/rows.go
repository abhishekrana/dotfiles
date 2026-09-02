package workdesk

import (
	"io"
	"strconv"
	"strings"
	"time"
)

// View is which question is being asked.
//
// The const order IS the ring: the order the tab bar reads left to right, the order 1-4
// select, and the order tab walks. It follows the work's own progression - what has not
// been started, then what is in flight, then who is doing it right now - with the inbox
// first because it cuts across all three.
//
// Everything else derives from this block. There used to be five separate lists encoding
// this order, which is how three views ended up sorting one way and three the other.
type View int

const (
	ViewInbox View = iota
	ViewIssues
	ViewMRs
	ViewAgents
	viewCount
)

// Keyed by the constants rather than positional, so this cannot drift out of step with
// the block above.
var viewNames = [...]string{
	ViewInbox:  "inbox",
	ViewIssues: "issues",
	ViewMRs:    "mrs",
	ViewAgents: "agents",
}

// Titles are what a person reads; names are what a command line takes.
var viewTitles = [...]string{
	ViewInbox:  "inbox",
	ViewIssues: "issues",
	ViewMRs:    "merge requests",
	ViewAgents: "agents",
}

// String is the name a command line and a row reference use.
func (v View) String() string {
	if int(v) < len(viewNames) {
		return viewNames[v]
	}
	return viewNames[ViewInbox]
}

// Title is the name a person reads, in the tab bar and the help.
func (v View) Title() string {
	if int(v) < len(viewTitles) {
		return viewTitles[v]
	}
	return viewTitles[ViewInbox]
}

// Key is the digit that selects this view: its position in the ring, one-based.
func (v View) Key() string { return strconv.Itoa(int(v) + 1) }

// ParseView maps a name or a ring position to a view, defaulting to the inbox rather
// than failing: a typo should land you somewhere useful.
func ParseView(s string) View {
	if n, err := strconv.Atoi(s); err == nil {
		if n >= 1 && n <= int(viewCount) {
			return View(n - 1)
		}
		return ViewInbox
	}
	for v, name := range viewNames {
		if s == name {
			return View(v)
		}
	}
	// The singular forms, because both read naturally on a command line.
	switch s {
	case "mr":
		return ViewMRs
	case "issue":
		return ViewIssues
	case "agent":
		return ViewAgents
	}
	return ViewInbox
}

// Views is the ring, in order.
func Views() []View {
	out := make([]View, viewCount)
	for i := range out {
		out[i] = View(i)
	}
	return out
}

// Next returns the view after this one, wrapping.
func (v View) Next() View { return (v + 1) % viewCount }

// Prev returns the view before this one, wrapping.
func (v View) Prev() View { return (v + viewCount - 1) % viewCount }

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
	// Tags is the labels, unjoined: the TSV wants commas and the UI wants its own
	// separator, and joining here would make one of them undo the other.
	Tags []string
	// Sprint marks membership of the current iteration.
	Sprint bool
}

// TSV is the wire format for a reader that is not this program:
//
//	label  flag  ref  title  age  note  branch  tags  sprint
//
// The title is padded here rather than in the model, because a fixed column only makes
// sense for a fixed-width consumer - the UI sizes it to the terminal instead.
// Separators inside a field are escaped, so a merge request titled with a tab cannot
// invent a column. New columns are appended, never inserted, so a reader that knows the
// first seven keeps working.
func (r Row) TSV() string {
	sprint := ""
	if r.Sprint {
		sprint = "sprint"
	}
	return TSV(r.Label, r.Flag, r.Ref, Pad(r.Title, titleWidth), r.Age, r.Note, r.Branch,
		strings.Join(r.Tags, ","), sprint)
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
		Tags:   it.Tags,
		Sprint: it.Sprint,
	}
}

// Listing is a view's rows and how many the window left out.
//
// One value rather than two calls: a view that owns up to an omission has to have it
// counted by the pass that made it, or the two can disagree.
type Listing struct {
	Rows []Row
	// Older is how many rows widening the window would bring back, and nothing else - a
	// todo past TodoMaxAge is not counted, because no window reaches that far.
	//
	// A total rather than a tally per band: at a narrow window whole bands age out, and
	// a band with no rows left has no header to carry a count.
	Older int
}

// Rows renders a view whole. now is passed in so a render is a pure function of the
// index plus the clock, which is what lets the golden tests freeze time.
//
// No window: this is what `list` and `ready` print, and an agent handed a shortened
// queue would be an agent quietly missing work.
func (idx *Index) Rows(v View, now time.Time) []Row {
	return idx.List(v, WindowAll, now).Rows
}

// List renders a view through a window.
//
// The window is the inbox's alone. It is the view that asks what you are working on now,
// where views 2 and 3 are the complete lists you go to for the thing you last touched in
// spring - so widening is a tab away even when the window is narrow.
func (idx *Index) List(v View, w Window, now time.Time) Listing {
	switch v {
	case ViewMRs:
		out := make([]Row, 0, len(idx.MRs))
		for _, it := range idx.MRs {
			out = append(out, it.row(now))
		}
		return Listing{Rows: out}
	case ViewIssues:
		out := make([]Row, 0, len(idx.Issues))
		for _, it := range idx.Issues {
			out = append(out, it.row(now))
		}
		return Listing{Rows: out}
	default:
		return idx.inbox(w, now)
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
//
// The window then decides which of those rows you are looking at now. It is applied last
// and counted, so the foot of the list can say what it held back; membership is settled
// before it, which is why claimed is stamped whether or not a row survives - otherwise a
// merge request ageing out would hand its todo a row and the item would appear to change
// bands as the window widened.
func (idx *Index) inbox(w Window, now time.Time) Listing {
	older := 0

	// Aged out at read time rather than at sync, so the cutoff is always relative to
	// now instead of to whenever the snapshot happened to be taken. This is the todos'
	// own bound, not the window's: GitLab never marks one done, so the feed is an
	// accumulating log however far back the view is asked to reach.
	cutoff := now.Add(-TodoMaxAge).Unix()
	fresh := make([]Item, 0, len(idx.Todos))
	for _, td := range idx.Todos {
		if td.Updated < cutoff {
			continue
		}
		fresh = append(fresh, td)
	}

	reason := make(map[string]string, len(fresh))
	for _, td := range fresh {
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
		if !w.Covers(it.Updated, now) {
			older++
			continue
		}
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
		if !w.Covers(it.Updated, now) {
			older++
			continue
		}
		r := it.row(now)
		r.Label = "not started"
		r.Flag = "i"
		issueRows = append(issueRows, r)
	}

	out := make([]Row, 0, len(fresh)+len(mrRows)+len(issueRows))
	for _, td := range fresh {
		if claimed[td.Ref] {
			continue
		}
		if !w.Covers(td.Updated, now) {
			older++
			continue
		}
		out = append(out, td.row(now))
	}
	out = append(out, mrRows...)
	out = append(out, issueRows...)
	return Listing{Rows: out, Older: older}
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
