package workdesk

import (
	"regexp"
	"slices"
	"strings"
	"time"
)

// Index is the small, derived tier of the mirror: everything a row needs
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
	// TodosDropped is how many pending todos were left out because the bands already
	// derive what they report. Kept so the count can be shown rather than the omission
	// being silent.
	TodosDropped int `json:"todosDropped"`
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
	idx.Todos, idx.TodosDropped = buildTodos(m.Todos)
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
	// Band first, then newest within a band.
	//
	// The band is already the priority signal, so within one the useful order is the one
	// you recognise: what you touched most recently. Oldest-first was the first attempt,
	// on the theory that the longest-waiting item is the most forgotten - but it opens a
	// band with something from seven months ago, which reads as a different tool's list,
	// and it disagreed with the issue, todo and agent views, which were newest-first all
	// along.
	slices.SortStableFunc(out, func(a, b MRItem) int {
		if a.Band != b.Band {
			return int(a.Band) - int(b.Band)
		}
		return int(b.Updated - a.Updated)
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
	// Priority first, then newest, matching every other view.
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

// TodoBand is the label todo rows carry, and the one band whose contents GitLab vouches
// for rather than this package inferring them. Exported so the UI can recognise the band
// it has to report a filter on.
// TodoBand is exported so the UI can recognise the one band that is filtered.
const TodoBand = "gitlab says it is you"

// TodoMaxAge is how far back a todo is still worth showing. GitLab never marks them
// done, so the pending feed is an accumulating log rather than an inbox: on a real
// account it runs back years. A todo you have not acted on in a fortnight is a decision
// you already made.
const TodoMaxAge = 14 * 24 * time.Hour

// informativeActions are the todo kinds that tell you something the bands cannot derive.
//
// Measured against a real feed rather than assumed. The overwhelming majority of pending
// todos are review_submitted, build_failed, unmergeable and merge_train_removed - all
// machine notifications about state mergeabilityChecks and the pipeline already report
// for a merge request you own, and pure noise for the far larger number you do not.
// `assigned` is the one that earns its place: the mirror only fetches merge requests you
// authored, so somebody assigning you theirs is genuinely not derivable here. Mentions
// are kept for the same reason, even though this feed had none.
var informativeActions = map[string]bool{
	"assigned":           true,
	"mentioned":          true,
	"directly_addressed": true,
	"marked":             true,
}

// buildTodos returns the todos worth a row, and how many were dropped.
func buildTodos(todos []Todo) ([]Item, int) {
	out := make([]Item, 0, len(todos))
	dropped := 0
	for i := range todos {
		td := &todos[i]
		if td.State != "pending" {
			continue
		}
		if !informativeActions[td.ActionName] {
			dropped++
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
			dropped++
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
			Label:   TodoBand,
			Flag:    "a",
			Note:    strings.ReplaceAll(td.ActionName, "_", " "),
			URL:     td.TargetURL,
			Updated: ParseTime(td.CreatedAt).Unix(),
		})
	}
	// Newest first: a todo feed is read like a feed.
	slices.SortStableFunc(out, func(a, b Item) int { return int(b.Updated - a.Updated) })
	return out, dropped
}
