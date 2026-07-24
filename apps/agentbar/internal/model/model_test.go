package model

import (
	"slices"
	"testing"
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
	in := []Session{
		{Name: "zeta", Agents: []Agent{{}}}, // active
		{Name: "alpha"},                     // dormant (no agents)
		{Name: "mid", Agents: []Agent{{}}},  // active, will be pinned
		{Name: "beta"},                      // dormant
		{Name: "yak", Agents: []Agent{{}}},  // active
	}
	out := Arrange(in, map[string]bool{"mid": true})

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
