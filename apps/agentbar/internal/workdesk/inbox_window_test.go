package workdesk

import (
	"testing"
	"time"
)

// windowFixture is a queue with two rows either side of a week, in the bands the inbox
// actually shows, plus a todo whose merge request is one of them.
func windowFixture(now time.Time) *Mirror {
	at := func(days int) string {
		return now.Add(-time.Duration(days) * 24 * time.Hour).Format(time.RFC3339)
	}
	// No reviewer puts a merge request in BandUnasked, which is a band the inbox shows.
	mr := func(iid string, days int) MergeRequest {
		return MergeRequest{IID: iid, Title: "mr " + iid, UpdatedAt: at(days)}
	}
	issue := func(iid, title string, days int) Issue {
		return Issue{IID: iid, Title: title, UpdatedAt: at(days),
			Labels: nodes[Label]{Nodes: []Label{{Title: "prio::high"}}}}
	}
	m := &Mirror{
		MRs:    []MergeRequest{mr("500", 1), mr("501", 20), mr("502", 40)},
		Issues: []Issue{issue("10", "fresh issue", 2), issue("11", "old issue", 45)},
		Todos: []Todo{
			{ID: 1, State: "pending", ActionName: "assigned", TargetType: "MergeRequest",
				CreatedAt: at(1)},
			{ID: 2, State: "pending", ActionName: "assigned", TargetType: "MergeRequest",
				CreatedAt: at(10)},
		},
	}
	// One todo about a merge request the inbox already shows, one about a merge request
	// it does not hold at all.
	m.Todos[0].Target.IID = "500"
	m.Todos[0].Target.Title = "already has a row"
	m.Todos[1].Target.IID = "900"
	m.Todos[1].Target.Title = "somebody else's"
	return m
}

func refsIn(rows []Row) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Ref)
	}
	return out
}

func has(rows []Row, ref string) bool {
	for _, r := range rows {
		if r.Ref == ref {
			return true
		}
	}
	return false
}

// The window is the whole point of w: what it holds back has to depend on the window and
// on nothing else.
func TestInboxWindowKeepsWhatItCovers(t *testing.T) {
	t.Parallel()
	now := frozen(t)
	idx := BuildIndex(windowFixture(now))

	week := idx.List(ViewInbox, 7*day, now)
	if has(week.Rows, "!501") || has(week.Rows, "!502") || has(week.Rows, "#11") {
		t.Errorf("a seven-day window kept something older: %v", refsIn(week.Rows))
	}
	for _, ref := range []string{"!500", "#10"} {
		if !has(week.Rows, ref) {
			t.Errorf("a seven-day window dropped %s: %v", ref, refsIn(week.Rows))
		}
	}

	month := idx.List(ViewInbox, 30*day, now)
	if !has(month.Rows, "!501") {
		t.Errorf("widening to a month did not bring !501 back: %v", refsIn(month.Rows))
	}
	if has(month.Rows, "!502") {
		t.Errorf("a month window kept the forty-day-old row: %v", refsIn(month.Rows))
	}

	all := idx.List(ViewInbox, WindowAll, now)
	for _, ref := range []string{"!500", "!501", "!502", "#10", "#11"} {
		if !has(all.Rows, ref) {
			t.Errorf("all dropped %s: %v", ref, refsIn(all.Rows))
		}
	}
	if all.Older != 0 {
		t.Errorf("all reported %d rows held back, want none", all.Older)
	}
}

// A list that caps silently reads as complete when it is not - the rule the todo band
// already follows. The count has to be rows widening would actually bring back, so it is
// checked against widening rather than against a second tally.
func TestInboxWindowCountsWhatItHeldBack(t *testing.T) {
	t.Parallel()
	now := frozen(t)
	idx := BuildIndex(windowFixture(now))

	for _, w := range []Window{7 * day, 30 * day, WindowAll} {
		narrowed := idx.List(ViewInbox, w, now)
		revealed := len(idx.List(ViewInbox, WindowAll, now).Rows) - len(narrowed.Rows)
		if narrowed.Older != revealed {
			t.Errorf("at %v the inbox offers %d older rows, but widening reveals %d",
				w, narrowed.Older, revealed)
		}
	}
	if got := idx.List(ViewInbox, 7*day, now).Older; got != 4 {
		t.Errorf("a seven-day window held back %d rows, want 4"+
			" (two merge requests, an issue and a todo)", got)
	}
}

// TodoMaxAge is the todos' own bound, not the window's: no window reaches past it, so a
// header must not offer to bring one back.
func TestInboxWindowDoesNotCountTodosPastTheirOwnCutoff(t *testing.T) {
	t.Parallel()
	now := frozen(t)
	m := windowFixture(now)
	m.Todos[1].CreatedAt = now.Add(-40 * 24 * time.Hour).Format(time.RFC3339)
	idx := BuildIndex(m)

	// The two merge requests and the issue are still held back; the todo is not, because
	// no window reaches past TodoMaxAge to bring it in.
	if n := idx.List(ViewInbox, 7*day, now).Older; n != 3 {
		t.Errorf("the inbox offers %d older rows, want 3 - a todo past TodoMaxAge is not one", n)
	}
	if rows := idx.List(ViewInbox, WindowAll, now).Rows; has(rows, "!900") {
		t.Error("a todo past TodoMaxAge came back at the widest window")
	}
	// One inside TodoMaxAge but outside the window is the case the count is for.
	m.Todos[1].CreatedAt = now.Add(-10 * 24 * time.Hour).Format(time.RFC3339)
	idx = BuildIndex(m)
	if n := idx.List(ViewInbox, 7*day, now).Older; n != 4 {
		t.Errorf("the inbox offers %d older rows, want 4 with the todo inside TodoMaxAge", n)
	}
	if !has(idx.List(ViewInbox, 30*day, now).Rows, "!900") {
		t.Error("widening did not bring the ten-day-old todo back")
	}
}

// Membership is settled before the window, so a merge request ageing out must not hand
// its todo a row: an item that changed bands as the window moved would be worse than one
// that is simply not shown.
func TestInboxWindowDoesNotMoveAnItemBetweenBands(t *testing.T) {
	t.Parallel()
	now := frozen(t)
	m := windowFixture(now)
	// The merge request the first todo points at is now old; the todo itself is fresh.
	m.MRs[0].UpdatedAt = now.Add(-40 * 24 * time.Hour).Format(time.RFC3339)
	idx := BuildIndex(m)

	rows := idx.List(ViewInbox, 7*day, now).Rows
	for _, r := range rows {
		if r.Ref == "!500" && r.Label == TodoBand {
			t.Errorf("!500 surfaced in the todo band once its own row aged out: %v", refsIn(rows))
		}
	}
}

// The window is the UI's lens. A reader that is not this UI - `list`, `ready`, an agent -
// gets the queue whole, or it is an agent quietly missing work.
func TestRowsIgnoresTheWindow(t *testing.T) {
	t.Parallel()
	now := frozen(t)
	idx := BuildIndex(windowFixture(now))
	if a, b := len(idx.Rows(ViewInbox, now)), len(idx.List(ViewInbox, WindowAll, now).Rows); a != b {
		t.Errorf("Rows returned %d rows, List at all returned %d", a, b)
	}
	if !has(idx.Rows(ViewInbox, now), "!502") {
		t.Error("Rows applied a window")
	}
}

// The other two views are the complete lists you widen into, so the window must not
// touch them.
func TestWindowIsTheInboxAlone(t *testing.T) {
	t.Parallel()
	now := frozen(t)
	idx := BuildIndex(windowFixture(now))
	for _, v := range []View{ViewMRs, ViewIssues} {
		narrow := idx.List(v, day, now)
		if len(narrow.Rows) != len(idx.Rows(v, now)) {
			t.Errorf("%s lost rows to a one-day window: %d of %d",
				v, len(narrow.Rows), len(idx.Rows(v, now)))
		}
		if narrow.Older != 0 {
			t.Errorf("%s reported %d rows held back", v, narrow.Older)
		}
	}
}
