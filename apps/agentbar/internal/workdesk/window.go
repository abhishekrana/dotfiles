package workdesk

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Window is how far back the inbox reaches. Zero is the whole queue.
//
// It is a lens on one view, not a fact about the work: the bands say who owns the next
// move, which stays true of a merge request nobody has touched since spring, and the
// complete lists are still one digit away in views 2 and 3.
type Window time.Duration

const (
	// WindowAll is no bound at all - every row the inbox would otherwise hold.
	WindowAll Window = 0
	// DefaultWindow is what the inbox opens at with nothing configured. A week is the
	// span the work itself runs on, and it is the same judgement TodoMaxAge already
	// makes about a todo: past it, you decided.
	DefaultWindow Window = Window(7 * 24 * time.Hour)
	day                  = Window(24 * time.Hour)
)

// Covers reports whether an item last updated at this epoch falls inside the window.
// An item with no timestamp is covered: a date that would not parse is not evidence of
// age, and Age already renders it as nothing rather than as old.
func (w Window) Covers(updated int64, now time.Time) bool {
	if w == WindowAll || updated == 0 {
		return true
	}
	return updated >= now.Add(-time.Duration(w)).Unix()
}

// String is the window as the tab bar says it: whole days, or "all".
func (w Window) String() string {
	if w == WindowAll {
		return "all"
	}
	if w%day == 0 {
		return strconv.Itoa(int(w/day)) + "d"
	}
	return time.Duration(w).String()
}

// ParseWindow reads a window as the config and the environment write it: a count of days
// ("7d"), or "all" for no bound. Days only, because days are the unit every age in this
// tool is already rendered in.
func ParseWindow(s string) (Window, error) {
	s = strings.TrimSpace(s)
	if s == "all" {
		return WindowAll, nil
	}
	n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
	if !strings.HasSuffix(s, "d") || err != nil || n < 0 {
		return 0, fmt.Errorf("expected a count of days like \"7d\", or \"all\", got %q", s)
	}
	return Window(n) * day, nil
}

// widerStops are the fixed stops the ring passes through after the configured window: a
// month, because that is the other unit anyone thinks in, and then the whole queue.
var widerStops = []Window{30 * day, WindowAll}

// Ring is the sequence w cycles through.
type Ring []Window

// NewRing builds the ring from the configured window: that window, then every fixed stop
// wider than it. w only ever widens until it wraps, so a stop the configured window
// already covers is dropped rather than narrowing the list halfway round - and a config
// of "all" leaves a ring of one, where w is simply inert.
func NewRing(start Window) Ring {
	r := Ring{start}
	if start == WindowAll {
		return r
	}
	for _, s := range widerStops {
		if s != WindowAll && s <= start {
			continue
		}
		r = append(r, s)
	}
	return r
}

// Next is the window after cur, wrapping. A window the ring does not hold - a config
// edited under a running UI - lands back at the start rather than sticking.
func (r Ring) Next(cur Window) Window {
	if len(r) == 0 {
		return WindowAll
	}
	for i, w := range r {
		if w == cur {
			return r[(i+1)%len(r)]
		}
	}
	return r[0]
}
