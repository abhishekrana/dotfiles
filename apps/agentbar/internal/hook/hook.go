// Package hook turns Claude Code hook events into tmux pane options.
//
// Claude Code runs `agentbar hook` for each lifecycle event;
// the event JSON arrives on stdin and $TMUX_PANE identifies the pane the
// agent lives in (inherited from the pane's environment). Resumed and
// `claude daemon run` sessions fire hooks without TMUX_PANE, so ResolvePane
// falls back to matching the event's cwd to a Claude pane. State is
// stamped as pane-scoped user options, which die with the pane:
//
//	@agent_present    "1" while a Claude session is registered
//	@agent_state      idle|working|permission|question|done
//	@agent_since      unix seconds of the last state *change*
//	@agent_seen       "1" once the user visited the pane after done
//	@agent_session_id Claude session id
//	@agent_subagents  count of live subagents
//	@agent_workdir    worktree root the agent last wrote in (see workdir.go)
package hook

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/abhishekrana/agentbar/internal/model"
	"github.com/abhishekrana/agentbar/internal/tmux"
)

// Event is the subset of Claude Code's hook JSON the sidebar needs.
type Event struct {
	Name             string `json:"hook_event_name"`
	SessionID        string `json:"session_id"`
	NotificationType string `json:"notification_type"`
	ToolName         string `json:"tool_name"`
	Cwd              string `json:"cwd"`    // session working dir; a pane-resolution fallback when TMUX_PANE is absent
	Source           string `json:"source"` // SessionStart only: startup|resume|clear|compact|fork
	// Edit/Write tool arguments: which file the agent is about to change, which
	// names the worktree it is really working in (see workdir.go).
	ToolInput struct {
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
	} `json:"tool_input"`
}

// Effect is what an event should do to the pane options.
type Effect struct {
	State         model.AgentState // "" = leave state alone
	Register      bool             // stamp presence + session id
	ClearAll      bool             // session ended: drop everything
	SubagentDelta int
}

// Decide maps a hook event to its effect. Pure; covered by unit tests.
func Decide(ev Event) Effect {
	switch ev.Name {
	case "SessionStart":
		return Effect{Register: true, State: model.StateIdle}
	case "UserPromptSubmit", "PreToolUse":
		// PreToolUse also registers: it repairs presence for sessions
		// that started before the hooks were installed.
		return Effect{Register: true, State: model.StateWorking}
	case "PermissionRequest":
		// AskUserQuestion arrives as a permission request but is Claude
		// asking the user a question, not a tool approval.
		if ev.ToolName == "AskUserQuestion" {
			return Effect{State: model.StateQuestion}
		}
		return Effect{State: model.StatePermission}
	case "Notification":
		switch ev.NotificationType {
		case "elicitation_dialog":
			// MCP server is prompting the user; genuinely blocked on them.
			return Effect{State: model.StateQuestion}
		case "elicitation_complete", "elicitation_response":
			// The elicitation was answered - the agent is proceeding again.
			return Effect{State: model.StateWorking}
		case "agent_completed":
			// A background (agent-view) session finished its work.
			return Effect{State: model.StateDone}
		}
		// Every other notification type is a false attention signal we ignore:
		//   permission_prompt  - a tool-blind echo of PermissionRequest (which
		//     carries tool_name); acting on it would relabel an AskUserQuestion
		//     "asking" as "permission".
		//   idle_prompt        - Claude's periodic "waiting for input" nudge.
		//   agent_needs_input  - fires for background sessions while agent view
		//     is open and never pairs with a clear, so it strands the pane in
		//     "asking" long after the agent moved on. The reliable "Claude asked
		//     you" signal is PermissionRequest{AskUserQuestion}, kept above.
		return Effect{}
	case "Stop":
		return Effect{State: model.StateDone}
	case "SubagentStart":
		return Effect{SubagentDelta: 1}
	case "SubagentStop":
		return Effect{SubagentDelta: -1}
	case "SessionEnd":
		return Effect{ClearAll: true}
	}
	return Effect{}
}

// isAgentCommand mirrors tmux's agent-pane test: Claude runs as claude
// (native) or node (npm install).
func isAgentCommand(cmd string) bool { return cmd == "claude" || cmd == "node" }

// ResolvePane finds the pane a paneless hook belongs to. Resumed and
// `claude daemon run` sessions fire hooks with no TMUX_PANE; this matches the
// event's cwd to a Claude pane's current path - the reliable signal, since a
// resume swaps the session id so it no longer matches what's stamped. When
// several Claude panes share that path it prefers the one whose stamped
// session id matches, else the first by pane id. Returns ("","") when nothing
// matches. via names the signal used, for the trace.
func ResolvePane(r tmux.Runner, ev Event) (pane, via string) {
	if ev.Cwd == "" {
		return "", ""
	}
	out, err := r.Run("list-panes", "-a", "-F",
		"#{pane_id}\t#{pane_current_command}\t#{pane_current_path}\t#{@agent_session_id}")
	if err != nil || out == "" {
		return "", ""
	}
	type match struct{ id, sid string }
	var matches []match
	for ln := range strings.SplitSeq(out, "\n") {
		f := strings.Split(ln, "\t")
		if len(f) < 4 || !isAgentCommand(f[1]) || f[2] != ev.Cwd {
			continue
		}
		matches = append(matches, match{f[0], f[3]})
	}
	if len(matches) == 0 {
		return "", ""
	}
	if len(matches) > 1 {
		for _, m := range matches {
			if m.sid == ev.SessionID {
				return m.id, "cwd+sid"
			}
		}
		sort.Slice(matches, func(i, j int) bool { return matches[i].id < matches[j].id })
	}
	return matches[0].id, "cwd"
}

// allOptions is everything Apply may set; ClearAll unsets each.
var allOptions = []string{
	"@agent_present", "@agent_state", "@agent_since",
	"@agent_seen", "@agent_session_id", "@agent_subagents",
}

// Apply writes an effect to the pane's options (empty value = unset).
// now is injected for testability.
func Apply(r tmux.Runner, pane string, ev Event, ef Effect, now time.Time) error {
	set := [][2]string{}
	if ef.ClearAll {
		for _, name := range allOptions {
			set = append(set, [2]string{name, ""})
		}
	}
	if ef.Register {
		set = append(set, [2]string{"@agent_present", "1"})
		if ev.SessionID != "" {
			set = append(set, [2]string{"@agent_session_id", ev.SessionID})
		}
	}
	if ef.SubagentDelta != 0 {
		n, _ := strconv.Atoi(tmux.PaneOption(r, pane, "@agent_subagents"))
		n = max(n+ef.SubagentDelta, 0)
		set = append(set, [2]string{"@agent_subagents", strconv.Itoa(n)})
	}
	if ef.State != "" {
		// Only a state *change* resets the clock and the seen flag:
		// PreToolUse fires constantly while working and must not
		// zero the elapsed time on every tool call.
		cur := tmux.PaneOption(r, pane, "@agent_state")
		if cur != string(ef.State) {
			set = append(set,
				[2]string{"@agent_state", string(ef.State)},
				[2]string{"@agent_since", strconv.FormatInt(now.Unix(), 10)},
				[2]string{"@agent_seen", ""},
			)
		}
	}
	if len(set) == 0 {
		return nil
	}
	args := []string{}
	for i, kv := range set {
		if i > 0 {
			args = append(args, ";")
		}
		if kv[1] == "" {
			args = append(args, "set-option", "-pqu", "-t", pane, kv[0])
		} else {
			args = append(args, "set-option", "-pq", "-t", pane, kv[0], kv[1])
		}
	}
	_, err := r.Run(args...)
	return err
}

// ShouldNotify reports whether a desktop notification should fire for this
// event: the global @agent_notify toggle is on AND the agent is entering an
// attention state (permission or asking) it wasn't already in. prev is the
// pane's @agent_state before Apply ran, so a repeat event in the same state
// (e.g. PreToolUse while working) doesn't re-fire.
func ShouldNotify(prev string, ef Effect, notifyOpt string) bool {
	return notifyOpt == "on" && ef.State.NeedsAttention() && string(ef.State) != prev
}

// Notify fires a desktop notification for an agent that needs the user, via
// notify-send. Fire-and-forget: it never waits on or fails the hook, so a box
// without notify-send (or without a desktop session) just stays silent.
func Notify(r tmux.Runner, pane string, state model.AgentState) {
	where, _ := r.Run("display-message", "-p", "-t", pane, "#{session_name}:#{window_index}")
	if where == "" {
		where = "an agent needs your input"
	}
	_ = exec.Command("notify-send", "-a", "Claude Code", "Claude · "+state.Label(), where).Start()
}
