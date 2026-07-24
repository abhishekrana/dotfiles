// Package model holds the agent/session tree the sidebar renders.
package model

import (
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
	State       AgentState
	Seen        bool      // done + visited since finishing: render dimmed
	Since       time.Time // last state transition
	Subagents   int
	Focused     bool // active pane of the session's active window
}

// Session groups the agents of one tmux session.
type Session struct {
	Name     string
	Current  bool // the session the sidebar pane lives in
	Attached bool // a client is attached to this session
	Pinned   bool // user-pinned: floats to the top band (see Arrange)
	Agents   []Agent
}

// Band orders sessions into the three sidebar groups: pinned (0),
// active with agents (1), dormant/no-agents (2).
func (s Session) Band() int {
	switch {
	case s.Pinned:
		return 0
	case len(s.Agents) == 0:
		return 2
	default:
		return 1
	}
}

// Arrange returns a copy of sessions grouped into bands (pinned, active,
// dormant) and alphabetical within each, stamping Pinned from the given set.
// Positions only move when the pin set changes - never on agent state - so
// the list stays predictable.
func Arrange(sessions []Session, pinned map[string]bool) []Session {
	out := make([]Session, len(sessions))
	copy(out, sessions)
	for i := range out {
		out[i].Pinned = pinned[out[i].Name]
	}
	sort.SliceStable(out, func(i, j int) bool {
		if bi, bj := out[i].Band(), out[j].Band(); bi != bj {
			return bi < bj
		}
		return out[i].Name < out[j].Name
	})
	return out
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
