package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/abhishekrana/agentbar/internal/tmux"
)

// Where the agent is actually writing, which is not where its pane sits: a
// Claude session started in one worktree edits files in another all the time
// (the Bash tool's `cd` never moves the pane, so #{pane_current_path} keeps
// pointing at the session's original checkout). The edited file's repo root is
// stamped as:
//
//	@agent_workdir   pane, window and session - the worktree it writes in now
//	@agent_workdirs  pane and window - the last few, most recent first
//	@agent_elsewhere pane only - set when that worktree is not the pane's own
//
// Several agents in one window share the window-scoped copy, last write wins;
// the per-pane option stays exact.

// EditedPath returns the file an event says the agent wrote, or "".
// Pure; the tool names are the ones that change a file on disk.
//
// A cwd is not a write, and CwdChanged is emphatically not one: Claude restores
// its shell cwd to the session's own directory after every Bash call, so taking
// it would drag the workdir home again seconds after an edit in a sibling
// worktree put it right - which is the one case this whole thing exists for.
func EditedPath(ev Event) string {
	switch ev.Name {
	case "PreToolUse", "PostToolUse":
		switch ev.ToolName {
		case "Edit", "Write", "MultiEdit", "NotebookEdit":
		default:
			return ""
		}
		p := ev.ToolInput.FilePath
		if p == "" {
			p = ev.ToolInput.NotebookPath
		}
		if !filepath.IsAbs(p) {
			return "" // a relative path is not resolvable from here
		}
		return p
	}
	return ""
}

// Within reports whether path is inside dir (or is dir). The short-circuit that
// keeps the common case fork-free: edit after edit in the same worktree needs no
// git call at all.
func Within(dir, path string) bool {
	if dir == "" {
		return false
	}
	return path == dir || strings.HasPrefix(path, strings.TrimSuffix(dir, "/")+"/")
}

// repoRoot resolves the worktree root holding path. A linked worktree reports
// its own root, which is exactly the granularity a diff pane wants.
func repoRoot(path string) string {
	dir := path
	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		dir = filepath.Dir(path)
	}
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// workdirsMax is how many recent worktrees a pane remembers. One agent often
// spans several in a turn; the picker offers all of them, and "the diff pane is
// showing something no agent is touching" is a membership test rather than a
// comparison with only the latest.
const workdirsMax = 5

// PushWorkdir returns list with dir most-recent-first, deduped and capped.
// Stored pipe-delimited AND pipe-wrapped ("|a|b|") so a tmux format can test
// membership with #{m:*|dir|*,…} - no fork, no partial-name false positives.
// Pure; covered by unit tests.
func PushWorkdir(list, dir string) string {
	if dir == "" {
		return list
	}
	out := []string{dir}
	for _, d := range strings.Split(strings.Trim(list, "|"), "|") {
		if d == "" || d == dir {
			continue
		}
		out = append(out, d)
		if len(out) == workdirsMax {
			break
		}
	}
	return "|" + strings.Join(out, "|") + "|"
}

// ApplyWorkdir stamps where the agent is writing when an event moved it to a
// different worktree. Returns the previous and new values; both empty means
// nothing happened. Never fatal: a file outside any repo leaves the last known
// workdir alone rather than blanking it.
//
// Scopes are deliberate. @agent_workdir goes on the pane (this agent), its
// window and its session, so one format reference resolves by tmux's own
// hierarchy wherever it is read - the focused agent pane, a sibling shell, the
// diff pane, or another window of the session. @agent_workdirs (the recent list)
// follows the same path. @agent_elsewhere is pane-scoped ONLY: it says "this
// agent writes outside its own pane's worktree", which is false for a shell pane
// and must not be inherited by one.
func ApplyWorkdir(r tmux.Runner, pane string, ev Event) (before, after string) {
	path := EditedPath(ev)
	if path == "" {
		return "", ""
	}
	cur := tmux.PaneOption(r, pane, "@agent_workdir")
	if Within(cur, path) {
		return "", "" // same worktree as last time - no git, no write
	}
	root := repoRoot(path)
	if root == "" || root == cur {
		return "", ""
	}
	list := PushWorkdir(tmux.PaneOption(r, pane, "@agent_workdirs"), root)
	// The pane's session (a session-scope set-option cannot take a pane id - it
	// errors, and tmux abandons the rest of the command chain) and the pane's own
	// worktree, for the rail's "writing elsewhere" flag. One call for both.
	info, _ := r.Run("display-message", "-p", "-t", pane, "#{session_name}\t#{pane_current_path}")
	sess, own, _ := strings.Cut(info, "\t")
	args := []string{
		"set-option", "-pq", "-t", pane, "@agent_workdir", root,
		";", "set-option", "-wq", "-t", pane, "@agent_workdir", root,
		";", "set-option", "-pq", "-t", pane, "@agent_workdirs", list,
		";", "set-option", "-wq", "-t", pane, "@agent_workdirs", list,
	}
	if sess != "" {
		// Session scope, so another window of the same session can still answer
		// "where is this session's agent working?".
		args = append(args, ";", "set-option", "-q", "-t", sess, "@agent_workdir", root,
			";", "set-option", "-q", "-t", sess, "@agent_workdirs", list)
	}
	if own != "" && repoRoot(own) == root {
		args = append(args, ";", "set-option", "-pqu", "-t", pane, "@agent_elsewhere")
	} else {
		args = append(args, ";", "set-option", "-pq", "-t", pane, "@agent_elsewhere", "1")
	}
	if _, err := r.Run(args...); err != nil {
		return "", ""
	}
	return cur, root
}

// RunWorkdirCmd fires the command in @agentbar-workdir-cmd after the workdir
// changed, with AGENTBAR_PANE and AGENTBAR_WORKDIR in its environment. The one
// seam this package has to anything outside it: unset by default, and started
// detached so the hook never waits on whatever it drives (the dotfiles point it
// at the diff pane's auto-follow).
func RunWorkdirCmd(r tmux.Runner, pane, dir string) {
	cmd, _ := r.Run("show-options", "-gqv", "@agentbar-workdir-cmd")
	if cmd == "" {
		return
	}
	c := exec.Command("sh", "-c", cmd)
	c.Env = append(os.Environ(), "AGENTBAR_PANE="+pane, "AGENTBAR_WORKDIR="+dir)
	_ = c.Start()
}
