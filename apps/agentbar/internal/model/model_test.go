package model

import (
	"slices"
	"testing"
	"time"
)

func names(sessions []Session) []string {
	out := make([]string, len(sessions))
	for i, s := range sessions {
		out[i] = s.Name
	}
	return out
}

// Arrange groups sessions into pinned / active / dormant bands, alphabetical
// within each, and stamps Pinned - without mutating the input.
func TestArrangeGroupsSortsAndStamps(t *testing.T) {
	// A bare Agent{} has no Since, so it is not fresh: give the active ones a
	// recent state change, which is what puts them in the active band.
	live := Agent{Since: time.Now().Add(-time.Minute)}
	in := []Session{
		{Name: "zeta", Agents: []Agent{live}}, // active
		{Name: "alpha"},                       // dormant (no agents)
		{Name: "mid", Agents: []Agent{live}},  // active, will be pinned
		{Name: "beta"},                        // dormant
		{Name: "yak", Agents: []Agent{live}},  // active
	}
	out := Arrange(in, Grouping{Pinned: map[string]bool{"mid": true}, Now: time.Now(), ActiveFor: time.Hour})

	want := []string{"mid", "yak", "zeta", "alpha", "beta"}
	if got := names(out); !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
	if !out[0].Pinned {
		t.Error("pinned session must have Pinned set")
	}
	for _, s := range out[1:] {
		if s.Pinned {
			t.Errorf("%s must not be pinned", s.Name)
		}
	}
	// Input slice is untouched (order and flags).
	if got := names(in); !slices.Equal(got, []string{"zeta", "alpha", "mid", "beta", "yak"}) {
		t.Errorf("Arrange mutated input order: %v", got)
	}
	if in[2].Pinned {
		t.Error("Arrange mutated the input's Pinned flag")
	}
}

func TestBand(t *testing.T) {
	cases := []struct {
		s    Session
		want int
	}{
		{Session{Pinned: true}, 0},
		{Session{Pinned: true, Agents: []Agent{{}}}, 0}, // pinned wins over agent count
		{Session{Agents: []Agent{{}}}, 1},
		{Session{}, 2}, // no agents
	}
	for _, c := range cases {
		if got := c.s.Band(); got != c.want {
			t.Errorf("Band(%+v) = %d, want %d", c.s, got, c.want)
		}
	}
}

func TestBandLabel(t *testing.T) {
	cases := []struct {
		s    Session
		want string
	}{
		{Session{Pinned: true}, "pinned"},
		{Session{Agents: []Agent{{}}}, "active"},
		{Session{}, "dormant"},
	}
	for _, c := range cases {
		if got := c.s.BandLabel(); got != c.want {
			t.Errorf("BandLabel(%+v) = %q, want %q", c.s, got, c.want)
		}
	}
}

// Step is what the session keys walk: the rendered order top to bottom and
// bottom to top, wrapping at both ends so no session is a dead end.
func TestStepWalksTheOrderAndWraps(t *testing.T) {
	names := []string{"blog", "dotfiles", "api", "payments"}
	cases := []struct {
		cur   string
		delta int
		want  string
	}{
		{"dotfiles", 1, "api"},   // down the bar, not alphabetically
		{"dotfiles", -1, "blog"}, // up
		{"payments", 1, "blog"},  // wrap past the bottom
		{"blog", -1, "payments"}, // wrap past the top
		{"gone", 1, "blog"},      // unknown session: enter from the top
		{"gone", -1, "payments"}, // ... and from the bottom going up
	}
	for _, c := range cases {
		if got := Step(names, c.cur, c.delta); got != c.want {
			t.Errorf("Step(%q, %d) = %q, want %q", c.cur, c.delta, got, c.want)
		}
	}
	if got := Step(nil, "dotfiles", 1); got != "" {
		t.Errorf("Step on an empty list = %q, want empty", got)
	}
	if got := Step([]string{"only"}, "only", 1); got != "only" {
		t.Errorf("Step on a single session = %q, want it to stay put", got)
	}
}

// Names must come off Arrange in band order - the keys walk this list, so an
// alphabetical slip here is exactly the jarring jump the bands removed.
func TestNamesFollowsTheBands(t *testing.T) {
	sessions := []Session{
		{Name: "payments"},                      // dormant
		{Name: "api", Agents: []Agent{{}}},      // active
		{Name: "blog", Agents: []Agent{{}}},     // pinned below
		{Name: "dotfiles", Agents: []Agent{{}}}, // pinned below
	}
	pinned := map[string]bool{"blog": true, "dotfiles": true}
	got := Names(Arrange(sessions, Grouping{Pinned: pinned, Now: time.Now(), ActiveFor: time.Hour}))
	want := []string{"blog", "dotfiles", "api", "payments"}
	if len(got) != len(want) {
		t.Fatalf("Names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Names = %v, want %v", got, want)
		}
	}
}

// BranchOf names the worktree a session's agents share; when they do not share
// one, the first wins and "+N" says how many others are in there.
func TestBranchOf(t *testing.T) {
	one := []Agent{{Branch: "main"}, {Branch: "main"}}
	if got := BranchOf(one); got != "main" {
		t.Errorf("agreed branch = %q, want main", got)
	}
	if got := BranchOf(nil); got != "" {
		t.Errorf("no agents = %q, want empty", got)
	}
	// An agent with no branch (a pane outside any repo) must not win the slot.
	if got := BranchOf([]Agent{{Branch: ""}, {Branch: "main"}}); got != "main" {
		t.Errorf("blank branch should not win, got %q", got)
	}
	two := []Agent{{Branch: "main"}, {Branch: "feat/x"}, {Branch: "feat/x"}}
	if got := BranchOf(two); got != "main +1" {
		t.Errorf("divergent branches = %q, want \"main +1\"", got)
	}
	three := []Agent{{Branch: "main"}, {Branch: "a"}, {Branch: "b"}}
	if got := BranchOf(three); got != "main +2" {
		t.Errorf("two others = %q, want \"main +2\"", got)
	}
}

// Fresh is what decides the active band: a live agent counts however long it has
// been at it, because @agent_since is the time of the last state *change* - a
// ninety-minute turn would otherwise age out mid-work.
func TestFresh(t *testing.T) {
	now := time.Now()
	old := now.Add(-3 * time.Hour)
	cases := map[string]struct {
		agents []Agent
		want   bool
	}{
		"working, long past its last change": {[]Agent{{State: StateWorking, Since: old}}, true},
		"blocked on you, however old":        {[]Agent{{State: StatePermission, Since: old}}, true},
		"asking, however old":                {[]Agent{{State: StateQuestion, Since: old}}, true},
		"done a moment ago":                  {[]Agent{{State: StateDone, Since: now.Add(-time.Minute)}}, true},
		"done three hours ago":               {[]Agent{{State: StateDone, Since: old}}, false},
		"idle three hours ago":               {[]Agent{{State: StateIdle, Since: old}}, false},
		"no timestamp at all":                {[]Agent{{State: StateDone}}, false},
		"one cold, one warm":                 {[]Agent{{State: StateDone, Since: old}, {State: StateDone, Since: now}}, true},
		"no agents":                          {nil, false},
	}
	for name, c := range cases {
		if got := Fresh(c.agents, now, time.Hour); got != c.want {
			t.Errorf("%s: Fresh = %v, want %v", name, got, c.want)
		}
	}
}

// A session whose agents have all gone quiet sinks to dormant on the clock
// alone - but a pinned one never moves, because pins are the user's.
func TestArrangeSinksQuietSessionsButNeverPinned(t *testing.T) {
	now := time.Now()
	cold := Agent{State: StateDone, Since: now.Add(-3 * time.Hour)}
	warm := Agent{State: StateDone, Since: now.Add(-time.Minute)}
	in := []Session{
		{Name: "cold", Agents: []Agent{cold}},
		{Name: "warm", Agents: []Agent{warm}},
		{Name: "pinned-cold", Agents: []Agent{cold}},
	}
	out := Arrange(in, Grouping{Pinned: map[string]bool{"pinned-cold": true}, Now: now, ActiveFor: time.Hour})

	byName := map[string]Session{}
	for _, s := range out {
		byName[s.Name] = s
	}
	if got := byName["cold"].Band(); got != 2 {
		t.Errorf("a quiet session should sink to dormant, got band %d", got)
	}
	if !byName["cold"].Quiet {
		t.Error("a quiet session should be stamped Quiet")
	}
	if got := byName["warm"].Band(); got != 1 {
		t.Errorf("a session with recent activity stays active, got band %d", got)
	}
	if got := byName["pinned-cold"].Band(); got != 0 {
		t.Errorf("a pinned session must not move, however quiet: got band %d", got)
	}
	// The window is what decides it: widen it and the cold one comes back.
	wide := Arrange(in, Grouping{Now: now, ActiveFor: 4 * time.Hour})
	for _, s := range wide {
		if s.Name == "cold" && s.Band() != 1 {
			t.Errorf("with a 4h window the cold session is active, got band %d", s.Band())
		}
	}
	// An unset (zero) window must not flatten the bar to dormant.
	for _, s := range Arrange(in, Grouping{Now: now}) {
		if s.Name == "warm" && s.Band() != 1 {
			t.Error("a zero window should fall back to the default, not sink everything")
		}
	}
}

// `a` and `d` place a session by hand, overriding the clock - except that an
// agent needing you pulls its session back out of a forced dormant, since
// nothing should hide a permission prompt.
func TestForcedBandsOverrideTheClock(t *testing.T) {
	now := time.Now()
	cold := Agent{State: StateDone, Since: now.Add(-3 * time.Hour)}
	warm := Agent{State: StateDone, Since: now.Add(-time.Minute)}
	blocked := Agent{State: StatePermission, Since: now.Add(-3 * time.Hour)}
	in := []Session{
		{Name: "held", Agents: []Agent{cold}},      // cold, forced active
		{Name: "sunk", Agents: []Agent{warm}},      // warm, forced dormant
		{Name: "asking", Agents: []Agent{blocked}}, // forced dormant, but needs you
		{Name: "auto", Agents: []Agent{warm}},      // no override
	}
	out := Arrange(in, Grouping{
		Forced: map[string]string{"held": BandActive, "sunk": BandDormant, "asking": BandDormant},
		Now:    now, ActiveFor: time.Hour,
	})
	got := map[string]int{}
	for _, s := range out {
		got[s.Name] = s.Band()
	}
	for name, want := range map[string]int{"held": 1, "sunk": 2, "asking": 1, "auto": 1} {
		if got[name] != want {
			t.Errorf("%s: band = %d, want %d", name, got[name], want)
		}
	}
	// Pinned still beats both the clock and a forced band: pins are the user's.
	pinned := Arrange(in, Grouping{
		Pinned: map[string]bool{"sunk": true},
		Forced: map[string]string{"sunk": BandDormant},
		Now:    now, ActiveFor: time.Hour,
	})
	for _, s := range pinned {
		if s.Name == "sunk" && s.Band() != 0 {
			t.Errorf("a pinned session must stay pinned, got band %d", s.Band())
		}
	}
}

// A forced dormant is a one-shot: work in the session ends it. Band() lifts a
// live session out on the spot, and Expire drops the placement so it obeys the
// clock from there rather than sinking again the moment it goes quiet.
func TestForcedDormantYieldsToWork(t *testing.T) {
	now := time.Now()
	old := 3 * time.Hour
	for _, c := range []struct {
		state   AgentState
		want    int
		expired bool
	}{
		{StateWorking, 1, true},    // being worked in
		{StatePermission, 1, true}, // blocked on you
		{StateQuestion, 1, true},   // blocked on you
		{StateIdle, 2, false},      // quiet: the placement holds
		{StateDone, 2, false},      // quiet: the placement holds
	} {
		in := []Session{{Name: "sunk", Agents: []Agent{{State: c.state, Since: now.Add(-old)}}}}
		bands := map[string]string{"sunk": BandDormant}
		out := Arrange(in, Grouping{Forced: bands, Now: now, ActiveFor: time.Hour})
		if got := out[0].Band(); got != c.want {
			t.Errorf("%s: band = %d, want %d", c.state, got, c.want)
		}
		next, expired := Expire(bands, in, "sunk")
		if expired != c.expired {
			t.Errorf("%s: expired = %v, want %v", c.state, expired, c.expired)
		}
		if _, still := next["sunk"]; still == c.expired {
			t.Errorf("%s: placement survived = %v, want %v", c.state, still, !c.expired)
		}
		if bands["sunk"] != BandDormant {
			t.Errorf("%s: Expire mutated its input", c.state)
		}
	}
}

// Expire touches one name and only a dormant one: `a` holds a quiet session up
// on purpose, and a neighbour's placement is none of this call's business.
func TestExpireLeavesEverythingElseAlone(t *testing.T) {
	working := []Agent{{State: StateWorking}}
	in := []Session{
		{Name: "held", Agents: working},
		{Name: "sunk", Agents: working},
		{Name: "empty"},
	}
	bands := map[string]string{"held": BandActive, "sunk": BandDormant}
	if _, expired := Expire(bands, in, "held"); expired {
		t.Error("a forced active must never expire")
	}
	if _, expired := Expire(bands, in, "gone"); expired {
		t.Error("a name with no placement must not expire")
	}
	if _, expired := Expire(bands, in, "empty"); expired {
		t.Error("a session with no agents is not live")
	}
	next, expired := Expire(bands, in, "sunk")
	if !expired || next["held"] != BandActive || len(next) != 1 {
		t.Errorf("expired = %v, next = %v", expired, next)
	}
}

// Place is one key, one destination: a pin and a forced band are one decision,
// so pressing a key clears the other store rather than stacking on it.
func TestPlace(t *testing.T) {
	pins, bands := Place(nil, nil, "web", BandPinned)
	if !pins["web"] || bands["web"] != "" {
		t.Fatalf("pinned: pins=%v bands=%v", pins, bands)
	}
	// `a` on a pinned session moves it - the case Band() would otherwise swallow.
	pins, bands = Place(pins, bands, "web", BandActive)
	if pins["web"] {
		t.Error("active should clear the pin")
	}
	if bands["web"] != BandActive {
		t.Errorf("bands = %q, want active", bands["web"])
	}
	// Pressing the same key again lands it where it already is.
	again, againBands := Place(pins, bands, "web", BandActive)
	if again["web"] || againBands["web"] != BandActive {
		t.Errorf("a twice changed something: pins=%v bands=%v", again, againBands)
	}
	// Other sessions are untouched, and the input maps are not mutated.
	pins, bands = Place(map[string]bool{"other": true}, map[string]string{"third": BandDormant}, "web", BandDormant)
	if !pins["other"] || bands["third"] != BandDormant || bands["web"] != BandDormant {
		t.Errorf("neighbours disturbed: pins=%v bands=%v", pins, bands)
	}
	if got := Placement(pins, bands, "other"); got != BandPinned {
		t.Errorf("Placement(other) = %q, want pinned", got)
	}
	if got := Placement(pins, bands, "nobody"); got != "" {
		t.Errorf("an unplaced session reads %q, want empty", got)
	}
}
