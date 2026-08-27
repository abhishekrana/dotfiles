package workdesk

import (
	"regexp"
	"slices"
	"strings"
)

// Index is the small, derived tier of the mirror: everything the picker's rows need
// and nothing else.
//
// It exists for speed. On a real queue mrs.json runs to megabytes because descriptions
// and discussion threads live in it, and decoding that costs around 7ms against 0.2ms
// for the index. That matters for the one-shot commands - `list`, `ready` - where a
// process start plus a decode is the whole runtime. The interactive UI holds the full
// snapshot instead, because its previews need the descriptions and threads anyway, and
// it pays that decode once rather than per keystroke.
//
// One classification either way: both paths go through BuildIndex, so a view can never
// disagree with another view about which band something is in.
//
// It is derived, never a second source of truth: `workdesk render` rebuilds it from
// the snapshot with no network, so deleting the mirror costs nothing.
type Index struct {
	// Generated is when sync wrote this, so a view can say how stale it is instead of
	// quietly presenting old rows as current.
	Generated int64       `json:"generated"`
	Project   string      `json:"project"`
	MRs       []MRItem    `json:"mrs"`
	Issues    []IssueItem `json:"issues"`
	Todos     []Item      `json:"todos"`
}

// Item is one row's payload, free of presentation: titles are stored at their natural
// length and ages not at all. Updated is an epoch because a baked-in "3d" is wrong by
// morning, and the title is unpadded because how wide a column is depends on the
// terminal, not on the snapshot.
type Item struct {
	Ref     string `json:"ref"`
	Title   string `json:"title"`
	Label   string `json:"label"`
	Flag    string `json:"flag"`
	Note    string `json:"note"`
	Branch  string `json:"branch,omitempty"`
	URL     string `json:"url,omitempty"`
	Updated int64  `json:"updated"`
}

// MRItem keeps the band as a value rather than only its label, so the inbox can select
// on it without string-matching a header.
type MRItem struct {
	Item
	Band Band `json:"band"`
}

// IssueItem keeps the priority for the same reason.
type IssueItem struct {
	Item
	Prio Priority `json:"prio"`
}

// BuildIndex does every classification and join once, at sync time.
func BuildIndex(m *Mirror) *Index {
	idx := &Index{Project: m.Meta.Project}
	if t := ParseTime(m.Meta.Synced); !t.IsZero() {
		idx.Generated = t.Unix()
	}
	idx.MRs = buildMRs(m.MRs)
	idx.Issues = buildIssues(m.Issues, m.MRs)
	idx.Todos = buildTodos(m.Todos)
	return idx
}

func buildMRs(mrs []MergeRequest) []MRItem {
	out := make([]MRItem, 0, len(mrs))
	for i := range mrs {
		mr := &mrs[i]
		band := mr.Band()
		// The note answers "why is it in this band": the gates GitLab is refusing
		// over, or - when nothing is refused - who has it and where they got to.
		note := strings.Join(mr.Blockers(), ", ")
		if note == "" {
			note = strings.Join(mr.ReviewerStates(), " ")
		}
		out = append(out, MRItem{
			Item: Item{
				Ref:     "!" + mr.IID,
				Title:   mr.Title,
				Label:   band.String(),
				Flag:    band.Flag(),
				Note:    note,
				Branch:  mr.SourceBranch,
				URL:     mr.WebURL,
				Updated: ParseTime(mr.UpdatedAt).Unix(),
			},
			Band: band,
		})
	}
	// Band first, then oldest within a band: the thing that has been waiting longest
	// is the one most likely to have been forgotten.
	slices.SortStableFunc(out, func(a, b MRItem) int {
		if a.Band != b.Band {
			return int(a.Band) - int(b.Band)
		}
		return int(a.Updated - b.Updated)
	})
	return out
}

func buildIssues(issues []Issue, mrs []MergeRequest) []IssueItem {
	out := make([]IssueItem, 0, len(issues))
	for i := range issues {
		is := &issues[i]
		prio := is.Priority()
		note := "not started"
		if mr := InFlightFor(is.IID, mrs); mr != nil {
			note = "!" + mr.IID + " in flight"
		}
		out = append(out, IssueItem{
			Item: Item{
				Ref:     "#" + is.IID,
				Title:   is.Title,
				Label:   prio.String(),
				Flag:    "a",
				Note:    note,
				URL:     is.WebURL,
				Updated: ParseTime(is.UpdatedAt).Unix(),
			},
			Prio: prio,
		})
	}
	// Priority first, then newest: an issue queue is read to pick the next thing up,
	// and a stale low-priority issue is not it.
	slices.SortStableFunc(out, func(a, b IssueItem) int {
		if a.Prio != b.Prio {
			return int(a.Prio) - int(b.Prio)
		}
		return int(b.Updated - a.Updated)
	})
	return out
}

// InFlightFor finds a merge request already working on an issue.
//
// Derived backwards from the merge requests we already hold, rather than asked of
// GitLab: closesIssues is served one issue at a time, so asking costs a round trip
// per issue. A branch named after the issue or a description mentioning it is how
// these branches are actually named, so the link falls out for free.
func InFlightFor(iid string, mrs []MergeRequest) *MergeRequest {
	if iid == "" {
		return nil
	}
	// Bounded either side so issue #12 is not matched by a branch mentioning 123.
	branchRE := regexp.MustCompile(`(^|[^0-9])` + regexp.QuoteMeta(iid) + `([^0-9]|$)`)
	descRE := regexp.MustCompile(`#` + regexp.QuoteMeta(iid) + `([^0-9]|$)`)
	for i := range mrs {
		mr := &mrs[i]
		if branchRE.MatchString(mr.SourceBranch) || descRE.MatchString(mr.Description) {
			return mr
		}
	}
	return nil
}

// todoBand is the label todo rows carry. Named because it is the one band whose
// contents GitLab vouches for rather than this package inferring them.
const todoBand = "gitlab says it is you"

func buildTodos(todos []Todo) []Item {
	out := make([]Item, 0, len(todos))
	for i := range todos {
		td := &todos[i]
		if td.State != "pending" {
			continue
		}
		// Only merge requests and issues can be shown as rows: this tool has no view
		// for a commit or a wiki page, and the shell rendered those as "!0" because it
		// assumed anything not an Issue was a merge request.
		var ref string
		switch td.TargetType {
		case "Issue":
			ref = "#" + td.Target.IID.String()
		case "MergeRequest":
			ref = "!" + td.Target.IID.String()
		default:
			continue
		}
		title := td.Target.Title
		if title == "" {
			title = td.Body
		}
		if title == "" {
			title = "?"
		}
		out = append(out, Item{
			Ref:     ref,
			Title:   title,
			Label:   todoBand,
			Flag:    "a",
			Note:    strings.ReplaceAll(td.ActionName, "_", " "),
			URL:     td.TargetURL,
			Updated: ParseTime(td.CreatedAt).Unix(),
		})
	}
	// Newest first: a todo feed is read like a feed.
	slices.SortStableFunc(out, func(a, b Item) int { return int(b.Updated - a.Updated) })
	return out
}
