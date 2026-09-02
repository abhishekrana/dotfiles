package workdesk

import (
	"strings"
	"testing"
	"time"
)

func TestParseWindowRoundTrips(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want Window
		bad  string
	}{
		{in: "7d", want: 7 * day},
		{in: "30d", want: 30 * day},
		{in: " 1d ", want: day},
		{in: "0d", want: WindowAll},
		{in: "all", want: WindowAll},
		{in: "7", bad: "count of days"},
		{in: "7w", bad: "count of days"},
		{in: "-3d", bad: "count of days"},
		{in: "", bad: "count of days"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			t.Parallel()
			got, err := ParseWindow(c.in)
			if c.bad != "" {
				if err == nil {
					t.Fatalf("parsed %q, wanted an error", c.in)
				}
				if !strings.Contains(err.Error(), c.bad) {
					t.Errorf("error %q does not mention %q", err, c.bad)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseWindow(%q): %v", c.in, err)
			}
			if got != c.want {
				t.Errorf("ParseWindow(%q) = %v, want %v", c.in, got, c.want)
			}
			// What the tab bar prints has to be a value the config would take back.
			back, err := ParseWindow(got.String())
			if err != nil || back != got {
				t.Errorf("%q renders as %q, which parses to %v (%v)", c.in, got, back, err)
			}
		})
	}
}

// A window is a lens on the clock, so it has to be read against the clock rather than
// against whenever the mirror happened to be written.
func TestWindowCovers(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	at := func(days int) int64 { return now.Add(-time.Duration(days) * 24 * time.Hour).Unix() }

	week := 7 * day
	if !week.Covers(at(6), now) {
		t.Error("a six-day-old row falls outside a seven-day window")
	}
	if week.Covers(at(8), now) {
		t.Error("an eight-day-old row falls inside a seven-day window")
	}
	if !WindowAll.Covers(at(4000), now) {
		t.Error("all left something out")
	}
	// A date that would not parse is stored as zero, and Age renders it as nothing
	// rather than as old - so the window must not read it as ancient and drop the row.
	if !week.Covers(0, now) {
		t.Error("a row with no timestamp was dropped as old")
	}
}

// The ring only ever widens until it wraps: a stop the configured window already covers
// would narrow the list halfway round, which is not what w means.
func TestRingWidensThenWraps(t *testing.T) {
	t.Parallel()
	cases := []struct {
		start Window
		want  []string
	}{
		{start: DefaultWindow, want: []string{"7d", "30d", "all"}},
		{start: 14 * day, want: []string{"14d", "30d", "all"}},
		{start: 30 * day, want: []string{"30d", "all"}},
		{start: 90 * day, want: []string{"90d", "all"}},
		{start: WindowAll, want: []string{"all"}},
	}
	for _, c := range cases {
		t.Run(c.start.String(), func(t *testing.T) {
			t.Parallel()
			r := NewRing(c.start)
			var got []string
			for _, w := range r {
				got = append(got, w.String())
			}
			if strings.Join(got, ",") != strings.Join(c.want, ",") {
				t.Fatalf("ring from %v is %v, want %v", c.start, got, c.want)
			}
			// Pressing w once per stop comes back to where it started.
			w := c.start
			for range r {
				w = r.Next(w)
			}
			if w != c.start {
				t.Errorf("walking the ring landed on %v, want %v", w, c.start)
			}
		})
	}
}

// A config edited under a running UI leaves the live window off the ring; w has to move
// rather than stick.
func TestRingNextRecoversFromAnUnknownWindow(t *testing.T) {
	t.Parallel()
	r := NewRing(DefaultWindow)
	if got := r.Next(3 * day); got != DefaultWindow {
		t.Errorf("w from a window off the ring went to %v, want %v", got, DefaultWindow)
	}
	if got := (Ring{}).Next(DefaultWindow); got != WindowAll {
		t.Errorf("an empty ring answered %v, want all", got)
	}
}
