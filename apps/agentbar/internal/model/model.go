// Package model holds the agent/session tree the sidebar renders.
package model

import (
	"fmt"
	"sort"
	"time"
)

// AgentState is the lifecycle state of one Claude Code agent, as stamped
// onto its tmux pane by the `hook` subcommand.
type AgentState string

const (
	// StateIdle: session registered (SessionStart) but no prompt yet.
	StateIdle AgentState = "idle"
	// StateWorking: processing a prompt (UserPromptSubmit / PreToolUse).
	StateWorking AgentState = "working"
	// StatePermission: blocked on a permission dialog.
	StatePermission AgentState = "permission"
	// StateQuestion: waiting for user input (question / elicitation).
	StateQuestion AgentState = "question"
	// StateDone: finished its turn (Stop).
	StateDone AgentState = "done"
)

// NeedsAttention reports whether the agent is blocked on the user.
func (s AgentState) NeedsAttention() bool {
	return s == StatePermission || s == StateQuestion
}

// Label is the short human-readable state text shown in the sidebar.
func (s AgentState) Label() string {
	switch s {
	case StateWorking:
		return "working"
	case StatePermission:
		return "permission"
	case StateQuestion:
		return "asking"
	case StateDone:
		return "done"
	default:
		return "idle"
	}
}

// Agent is one Claude Code instance in a tmux pane.
type Agent struct {
	PaneID      string
	WindowIndex int
	Command     string // pane's current command (claude/node); the row label
	Branch      string // git branch of the pane's cwd
	Title       string // Claude's own name for the session; "" before its first prompt
	State       AgentState
	Seen        bool      // done + visited since finishing: render dimmed
	Since       time.Time // last state transition
	Subagents   int
	Focused     bool // active pane of the session's active window
}

// Session groups the agents of one tmux session.
type Session struct {
	Name     string
	Branch   string // git branch its agents work in; "<branch> +N" when they differ
	Current  bool   // the session the sidebar pane lives in
	Attached bool   // a client is attached to this session
	Pinned   bool   // user-pinned: floats to the top band (see Arrange)
	Forced   string // band the user put it in by hand: "active", "dormant", or ""
	Quiet    bool   // has agents, but none live or recent: sinks to dormant (see Fresh)
	Agents   []Agent
}

// DefaultActiveFor is how long a session stays active after its last agent
// activity. An hour is long enough to cover reading a diff or a phone call,
// short enough that yesterday's worktrees are gone by morning.
const DefaultActiveFor = time.Hour

// Fresh reports whether a session's agents make it active: one working or
// blocked on you counts however long it has been at it - @agent_since is the
// time of the last state *change*, so a long turn would otherwise age out
// mid-work - and otherwise the newest state change must be inside activeFor.
func Fresh(agents []Agent, now time.Time, activeFor time.Duration) bool {
	for _, a := range agents {
		if a.State == StateWorking || a.State.NeedsAttention() {
			return true
		}
		if !a.Since.IsZero() && now.Sub(a.Since) < activeFor {
			return true
		}
	}
	return false
}

// BranchOf is the branch a session's agents work in. They almost always share
// one worktree; when they do not, the first wins and "+N" counts the rest,
// since one line cannot name several branches.
func BranchOf(agents []Agent) string {
	first, others := "", map[string]bool{}
	for _, a := range agents {
		switch {
		case a.Branch == "":
		case first == "":
			first = a.Branch
		case a.Branch != first:
			others[a.Branch] = true
		}
	}
	if len(others) > 0 {
		return fmt.Sprintf("%s +%d", first, len(others))
	}
	return first
}

// Band orders sessions into the three sidebar groups: pinned (0), active (1),
// dormant (2). Arrange stamps Pinned, Forced and Quiet, so every reader here
// needs no clock and no options of its own.
//
// A hand-placed band wins over the clock - that is the whole point of `a` and
// `d` - with one exception: an agent that needs you pulls its session back up
// out of a forced dormant, because nothing should be able to hide a permission
// prompt indefinitely.
func (s Session) Band() int {
	switch {
	case s.Pinned:
		return 0
	case s.Forced == BandActive:
		return 1
	case s.Forced == BandDormant:
		if s.NeedsAttention() {
			return 1
		}
		return 2
	case len(s.Agents) == 0 || s.Quiet:
		return 2
	default:
		return 1
	}
}

// The bands a session can be put in by hand, one key each: `p`, `a`, `d`.
const (
	BandPinned  = "pinned"
	BandActive  = "active"
	BandDormant = "dormant"
)

// Placement is where a session sits by hand, or "" when the clock decides.
func Placement(pinned map[string]bool, forced map[string]string, name string) string {
	if pinned[name] {
		return BandPinned
	}
	return forced[name]
}

// Place puts one session in a band by hand, returning fresh copies of both
// sets. One key, one destination: pressing it again lands the session where it
// already is, so nothing happens.
//
// A pin and a forced band are one decision, never two: whichever key you press
// clears the other store. That is what makes `a` on a pinned session move it
// rather than be swallowed by the pin - Band() reads Pinned first.
//
// A session nobody has placed is left to the clock, which is still the normal
// case; these are the exceptions you named.
func Place(pinned map[string]bool, forced map[string]string, name, band string) (map[string]bool, map[string]string) {
	pins := map[string]bool{}
	for k, v := range pinned {
		if k != name {
			pins[k] = v
		}
	}
	bands := map[string]string{}
	for k, v := range forced {
		if k != name {
			bands[k] = v
		}
	}
	switch band {
	case BandPinned:
		pins[name] = true
	case BandActive, BandDormant:
		bands[name] = band
	}
	return pins, bands
}

// NeedsAttention reports whether any agent here is blocked on the user.
func (s Session) NeedsAttention() bool {
	for _, a := range s.Agents {
		if a.State.NeedsAttention() {
			return true
		}
	}
	return false
}

// BandLabel names the band. The sidebar draws its own header text from this
// group; the token is what `agentbar order` publishes, so the picker popup can
// group its rows into the same bands without reimplementing them.
func (s Session) BandLabel() string {
	switch s.Band() {
	case 0:
		return "pinned"
	case 1:
		return "active"
	default:
		return "dormant"
	}
}

// Grouping is everything Arrange needs to band a fleet: the two persisted
// user choices and the clock.
type Grouping struct {
	Pinned    map[string]bool   // @agentbar-pins, the `p` key
	Forced    map[string]string // @agentbar-bands, the `a` and `d` keys
	Now       time.Time
	ActiveFor time.Duration // zero means DefaultActiveFor
}

// Arrange returns a copy of sessions grouped into bands (pinned, active,
// dormant) and alphabetical within each, stamping Pinned, Forced and Quiet.
//
// Positions move when you press `p`, `a` or `d`, and when a session's last
// agent activity passes ActiveFor - nothing else. That last case is deliberate:
// a worktree you stopped touching an hour ago sinks on its own, so the active
// band is what you are working on now. It is a pure function of timestamps
// evaluated at render, so no process owns the transition and there is no state
// to get stale.
func Arrange(sessions []Session, g Grouping) []Session {
	activeFor := g.ActiveFor
	if activeFor <= 0 {
		activeFor = DefaultActiveFor
	}
	out := make([]Session, len(sessions))
	copy(out, sessions)
	for i := range out {
		out[i].Pinned = g.Pinned[out[i].Name]
		out[i].Forced = g.Forced[out[i].Name]
		out[i].Quiet = len(out[i].Agents) > 0 && !Fresh(out[i].Agents, g.Now, activeFor)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if bi, bj := out[i].Band(), out[j].Band(); bi != bj {
			return bi < bj
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// Names lists session names in the order given - post-Arrange, that is the
// order the sidebar renders top to bottom.
func Names(sessions []Session) []string {
	names := make([]string, 0, len(sessions))
	for _, s := range sessions {
		names = append(names, s.Name)
	}
	return names
}

// Step walks the ordered list delta rows from cur, wrapping at both ends, so
// the session keys traverse the sidebar top to bottom and back around. Empty
// for an empty list; an unknown cur (no client attached) enters from the end
// the walk is heading away from.
func Step(names []string, cur string, delta int) string {
	if len(names) == 0 {
		return ""
	}
	at := -1
	for i, n := range names {
		if n == cur {
			at = i
			break
		}
	}
	if at < 0 {
		if delta < 0 {
			return names[len(names)-1]
		}
		return names[0]
	}
	n := len(names)
	return names[((at+delta)%n+n)%n] // Go's % keeps the sign; this wraps both ways
}

// Snapshot is everything the sidebar shows. tmux.Snapshot delivers sessions
// alphabetically; the UI runs them through Arrange to group into bands.
type Snapshot struct {
	Sessions []Session
}

// Working and Attention return server-wide counts for header/footer.
func (s Snapshot) Working() int {
	return s.count(func(a Agent) bool { return a.State == StateWorking })
}
func (s Snapshot) Attention() int {
	return s.count(func(a Agent) bool { return a.State.NeedsAttention() })
}
func (s Snapshot) Total() int { return s.count(func(Agent) bool { return true }) }

func (s Snapshot) count(pred func(Agent) bool) int {
	n := 0
	for _, sess := range s.Sessions {
		for _, a := range sess.Agents {
			if pred(a) {
				n++
			}
		}
	}
	return n
}
