package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/abhishekrana/tmux-agent-sidebar/internal/model"
)

// snapFormat: one line per pane across the whole server. The parsing
// loop in Snapshot indexes fields by position in this format.
const snapFormat = "#{session_name}\t#{session_attached}\t#{window_index}\t#{window_active}\t" +
	"#{pane_id}\t#{pane_active}\t#{pane_current_command}\t#{pane_current_path}\t" +
	"#{@agent_present}\t#{@agent_state}\t#{@agent_since}\t#{@agent_seen}\t#{@agent_subagents}"

const snapFields = 13

// agentCommands guards against zombies: a pane whose Claude died without
// SessionEnd keeps its options, but its foreground command changes.
// Claude Code runs as `claude` (native) or `node` (npm install).
var agentCommands = map[string]bool{"claude": true, "node": true}

// CurrentSession names the session the sidebar pane lives in, anchored
// to our own pane: a bare display-message resolves against the attached
// client, which may be looking at a different session.
func CurrentSession(r Runner) string {
	args := []string{"display-message", "-p"}
	if pane := os.Getenv("TMUX_PANE"); pane != "" {
		args = append(args, "-t", pane)
	}
	args = append(args, "#S")
	out, _ := r.Run(args...)
	return out
}

// Snapshot builds the full model: every session on the server (sessions
// without agents included), agents from panes stamped by the hooks.
// Sessions are sorted alphabetically, agents by window then pane.
//
// As a side effect, a done agent whose pane the user is looking at is
// marked seen (@agent_seen=1) so it renders dimmed from then on.
func Snapshot(r Runner, bc *BranchCache, currentSession string) model.Snapshot {
	out, err := r.Run("list-panes", "-a", "-F", snapFormat)
	if err != nil || out == "" {
		return model.Snapshot{}
	}

	bySession := map[string]*model.Session{}
	for ln := range strings.SplitSeq(out, "\n") {
		f := strings.Split(ln, "\t")
		if len(f) < snapFields {
			continue
		}
		name := f[0]
		sess := bySession[name]
		if sess == nil {
			sess = &model.Session{
				Name:     name,
				Current:  name == currentSession,
				Attached: f[1] != "0" && f[1] != "",
			}
			bySession[name] = sess
		}
		if f[8] != "1" || !agentCommands[f[6]] {
			continue // not an agent pane
		}
		windowIdx, _ := strconv.Atoi(f[2])
		since, _ := strconv.ParseInt(f[10], 10, 64)
		subagents, _ := strconv.Atoi(f[12])
		state := model.AgentState(f[9])
		if state == "" {
			state = model.StateIdle
		}
		seen := f[11] == "1"
		paneActive, windowActive := f[5] == "1", f[3] == "1"
		if state == model.StateDone && !seen && sess.Attached && paneActive && windowActive {
			// User is looking at the finished agent right now.
			_, _ = r.Run("set-option", "-pq", "-t", f[4], "@agent_seen", "1")
			seen = true
		}
		sess.Agents = append(sess.Agents, model.Agent{
			PaneID:      f[4],
			WindowIndex: windowIdx,
			Command:     f[6],
			Branch:      bc.Get(f[7]),
			State:       state,
			Seen:        seen,
			Since:       time.Unix(since, 0),
			Subagents:   subagents,
			Focused:     paneActive && windowActive,
		})
	}

	names := make([]string, 0, len(bySession))
	for name := range bySession {
		names = append(names, name)
	}
	sort.Strings(names)
	snap := model.Snapshot{}
	for _, name := range names {
		sess := bySession[name]
		sort.Slice(sess.Agents, func(i, j int) bool {
			a, b := sess.Agents[i], sess.Agents[j]
			if a.WindowIndex != b.WindowIndex {
				return a.WindowIndex < b.WindowIndex
			}
			return a.PaneID < b.PaneID
		})
		snap.Sessions = append(snap.Sessions, *sess)
	}
	return snap
}

// ClientFor picks the client tty a jump should switch: prefer a client
// attached to the given session (whoever is looking at this sidebar),
// else the first attached client. Empty when no clients (detached
// server) — callers fall back to tmux's own "current client" guess.
func ClientFor(r Runner, session string) string {
	out, err := r.Run("list-clients", "-F", "#{client_tty}\t#{client_session}")
	if err != nil || out == "" {
		return ""
	}
	first := ""
	for ln := range strings.SplitSeq(out, "\n") {
		tty, sess, ok := strings.Cut(ln, "\t")
		if !ok {
			continue
		}
		if first == "" {
			first = tty
		}
		if sess == session {
			return tty
		}
	}
	return first
}

// StatusSegment renders a compact status-line summary with tmux colour
// markup: attention count (red) and working count (yellow). Empty when
// no agents are running, so the segment vanishes rather than showing 0s.
func StatusSegment(r Runner) string {
	snap := Snapshot(r, nil, "") // nil cache: skip git lookups, counts only
	att, work := snap.Attention(), snap.Working()
	parts := []string{}
	if att > 0 {
		parts = append(parts, fmt.Sprintf("#[fg=#dc322f,bold]⚠%d#[default]", att))
	}
	if work > 0 {
		parts = append(parts, fmt.Sprintf("#[fg=#b58900]●%d#[default]", work))
	}
	return strings.Join(parts, " ")
}

// BranchCache memoizes git branch lookups per directory so the 1s
// snapshot tick doesn't fork git for every agent every time.
type BranchCache struct {
	entries map[string]branchEntry
}

type branchEntry struct {
	branch string
	at     time.Time
}

const branchTTL = 5 * time.Second

func NewBranchCache() *BranchCache {
	return &BranchCache{entries: map[string]branchEntry{}}
}

func (c *BranchCache) Get(dir string) string {
	if c == nil || dir == "" {
		return ""
	}
	if e, ok := c.entries[dir]; ok && time.Since(e.at) < branchTTL {
		return e.branch
	}
	out, _ := exec.Command("git", "-C", dir, "branch", "--show-current").Output()
	branch := strings.TrimSpace(string(out))
	c.entries[dir] = branchEntry{branch: branch, at: time.Now()}
	return branch
}
