// Package e2e exercises the plugin against a real, throwaway tmux server.
//
// Every test starts its own server on a private socket (tmux -L), so runs
// never touch the developer's live tmux. Bare `tmux` calls made by the
// scripts and the binary are routed to that server through a PATH shim,
// and agents are simulated with a copy of sleep(1) named "claude" (the
// snapshot filters on #{pane_current_command}) plus real `hook` events.
//
// Skipped with -short and when tmux is not installed.
package e2e

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	repoRoot string
	binPath  string
	shimDir  string // holds the tmux shim and the fake `claude`
)

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Println("e2e: tmux not installed, skipping")
		os.Exit(0)
	}
	var err error
	repoRoot, err = filepath.Abs("..")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(repoRoot, "bin", "agentbar")

	build := exec.Command("go", "build", "-o", binPath, "./cmd/agentbar")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Printf("e2e: build failed: %v\n%s", err, out)
		os.Exit(1)
	}

	shimDir, err = os.MkdirTemp("", "agentbar-e2e-shim")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(shimDir)

	// -f /dev/null keeps tests hermetic: the developer's ~/.tmux.conf
	// (plugins, hooks, resurrect) must never leak into test servers.
	realTmux, _ := exec.LookPath("tmux")
	shim := "#!/bin/sh\nexec " + realTmux + " -L \"$AGENTBAR_TEST_SOCKET\" -f /dev/null \"$@\"\n"
	if err := os.WriteFile(filepath.Join(shimDir, "tmux"), []byte(shim), 0o755); err != nil {
		panic(err)
	}

	// A fake agent: sleep(1) renamed so #{pane_current_command} == "claude".
	realSleep, _ := exec.LookPath("sleep")
	data, err := os.ReadFile(realSleep)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(shimDir, "claude"), data, 0o755); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// server is one isolated tmux server on a private socket.
type server struct {
	t   *testing.T
	env []string
}

var serverSeq int

func start(t *testing.T) *server {
	if testing.Short() {
		t.Skip("e2e skipped with -short")
	}
	serverSeq++ // unique socket even for -count>1 reruns of one test
	sock := fmt.Sprintf("agentbar-e2e-%d-%d-%s", os.Getpid(), serverSeq,
		strings.NewReplacer("/", "-", " ", "-").Replace(t.Name()))
	var env []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		switch k {
		// CI is dropped deliberately: termenv reads any non-empty CI as "not a
		// TTY" and falls back to the Ascii profile, so the sidebar would render
		// without the ANSI attributes the highlight assertions look for.
		case "TMUX", "TMUX_PANE", "PATH", "AGENTBAR_TEST_SOCKET", "TERM", "CI", "XDG_STATE_HOME":
		default:
			env = append(env, kv)
		}
	}
	env = append(env,
		"PATH="+shimDir+":"+os.Getenv("PATH"),
		"AGENTBAR_TEST_SOCKET="+sock,
		"TERM=xterm-256color",
		// Keep the shared state dir out of the developer's: the trace log and
		// the pin mirror both live under it, and a test must never write to
		// either. Tests needing pane-side redirection also set-environment -g.
		"XDG_STATE_HOME="+t.TempDir(),
		// tmux mangles tabs in -F output outside a UTF-8 locale, and the whole
		// pane protocol is tab-separated. CI runners leave LANG unset.
		"LC_ALL=C.UTF-8",
	)
	s := &server{t: t, env: env}
	t.Cleanup(func() { _, _ = s.tmuxErr("kill-server") })
	return s
}

func (s *server) tmuxErr(args ...string) (string, error) {
	cmd := exec.Command(filepath.Join(shimDir, "tmux"), args...)
	cmd.Env = s.env
	out, err := cmd.CombinedOutput()
	return strings.TrimRight(string(out), "\n"), err
}

func (s *server) tmux(args ...string) string {
	s.t.Helper()
	out, err := s.tmuxErr(args...)
	if err != nil {
		s.t.Fatalf("tmux %v: %v\n%s", args, err, out)
	}
	return out
}

// newSession creates a detached session and returns its first pane id.
func (s *server) newSession(name string) string {
	s.t.Helper()
	return s.tmux("new-session", "-d", "-s", name, "-x", "220", "-y", "50",
		"-P", "-F", "#{pane_id}")
}

// agentPane opens a pane running the fake claude and registers it via a
// real hook event, exactly as Claude Code would.
func (s *server) agentPane(session string) string {
	s.t.Helper()
	pane := s.tmux("split-window", "-d", "-t", session, "-P", "-F", "#{pane_id}",
		"claude 600")
	s.hook(pane, `{"hook_event_name":"SessionStart","session_id":"e2e"}`)
	return pane
}

// hook feeds one event to `agentbar hook` for the given pane.
func (s *server) hook(pane, eventJSON string) {
	s.t.Helper()
	cmd := exec.Command(binPath, "hook")
	cmd.Env = append(append([]string{}, s.env...), "TMUX_PANE="+pane)
	cmd.Stdin = strings.NewReader(eventJSON)
	if out, err := cmd.CombinedOutput(); err != nil {
		s.t.Fatalf("hook %s: %v\n%s", eventJSON, err, out)
	}
}

// script runs one of the plugin's shell scripts against this server.
func (s *server) script(name string, args ...string) {
	s.t.Helper()
	cmd := exec.Command("bash", append([]string{filepath.Join(repoRoot, "scripts", name)}, args...)...)
	cmd.Env = s.env
	if out, err := cmd.CombinedOutput(); err != nil {
		s.t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

// scriptFaulty runs a plugin script with a tmux wrapper ahead of the shim on
// PATH that fails every call whose arguments contain marker - standing in for
// the window or pane vanishing mid-command, which no timing-based test can hit
// reliably. Returns the script's error; a fault aborts it under set -e.
func (s *server) scriptFaulty(name, marker string, args ...string) error {
	s.t.Helper()
	dir := s.t.TempDir()
	wrapper := "#!/bin/sh\n" +
		"for a in \"$@\"; do case \"$a\" in *" + marker + "*) exit 1 ;; esac; done\n" +
		"exec " + filepath.Join(shimDir, "tmux") + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "tmux"), []byte(wrapper), 0o755); err != nil {
		s.t.Fatal(err)
	}
	cmd := exec.Command("bash", append([]string{filepath.Join(repoRoot, "scripts", name)}, args...)...)
	// A later PATH wins in exec.Cmd, so this shadows the one start() set.
	cmd.Env = append(append([]string{}, s.env...), "PATH="+dir+":"+shimDir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	s.t.Logf("%s (fault %q) -> %v\n%s", name, marker, err, out)
	return err
}

// plugin sources the .tmux entry point against this server, exactly as TPM
// would at server start (binds the key, wires hooks, runs autostart).
func (s *server) plugin() {
	s.t.Helper()
	cmd := exec.Command("bash", filepath.Join(repoRoot, "agentbar.tmux"))
	cmd.Env = s.env
	if out, err := cmd.CombinedOutput(); err != nil {
		s.t.Fatalf("agentbar.tmux: %v\n%s", err, out)
	}
}

func (s *server) paneOption(pane, name string) string {
	out, _ := s.tmuxErr("show-option", "-pqv", "-t", pane, name)
	return out
}

func (s *server) sidebarPane(session string) string {
	out, _ := s.tmuxErr("show-option", "-t", session, "-qv", "@sidebar_pane")
	return out
}

// sidebarAlive reports whether the session has a live sidebar pane.
func (s *server) sidebarAlive(session string) bool {
	pane := s.sidebarPane(session)
	if pane == "" {
		return false
	}
	panes, err := s.tmuxErr("list-panes", "-s", "-t", session,
		"-F", "#{pane_id} #{pane_current_command}")
	if err != nil {
		return false
	}
	return strings.Contains(panes, pane+" agentbar")
}

// capture returns the sidebar pane content with escape sequences.
func (s *server) capture(pane string) string {
	out, _ := s.tmuxErr("capture-pane", "-p", "-e", "-t", pane)
	return out
}

// captureText returns the pane content without escape sequences.
func (s *server) captureText(pane string) string {
	out, _ := s.tmuxErr("capture-pane", "-p", "-t", pane)
	return out
}

// ptyClient attaches a real client (pty via script(1)) and returns its
// stdin: bytes written there are typed into the client, so keyboard AND
// client-level mouse input (status-line clicks) take the real path.
func (s *server) ptyClient(session string) io.WriteCloser {
	s.t.Helper()
	if _, err := exec.LookPath("script"); err != nil {
		s.t.Skip("script(1) not available for pty client")
	}
	cmd := exec.Command("script", "-qfc", "tmux attach-session -t "+session, "/dev/null")
	cmd.Env = s.env
	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.t.Fatalf("client stdin: %v", err)
	}
	if err := cmd.Start(); err != nil {
		s.t.Fatalf("attach client: %v", err)
	}
	s.t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	waitFor(s.t, "client attached", 5*time.Second, func() bool {
		out, _ := s.tmuxErr("list-clients", "-F", "#{client_session}")
		return strings.Contains(out, session)
	})
	return stdin
}

// clientClick types a left press+release at 1-based (col, row) into the
// attached client - the same bytes a terminal sends for a single click.
func clientClick(stdin io.Writer, col, row int) {
	fmt.Fprintf(stdin, "\x1b[<0;%d;%dM", col, row)
	fmt.Fprintf(stdin, "\x1b[<0;%d;%dm", col, row)
}

// click injects a left mouse press+release at 1-based (col, row) into the
// pane's input as raw SGR sequences - exactly the bytes a terminal sends,
// so the TUI's real mouse path runs.
func (s *server) click(pane string, col, row int) {
	s.t.Helper()
	s.mouse(pane, col, row, "M") // press
	s.mouse(pane, col, row, "m") // release
}

// releaseClick sends only the release, like a terminal that ate the
// press of a window-focusing click.
func (s *server) releaseClick(pane string, col, row int) {
	s.t.Helper()
	s.mouse(pane, col, row, "m")
}

func (s *server) mouse(pane string, col, row int, suffix string) {
	s.mouseRaw(pane, fmt.Sprintf("\x1b[<0;%d;%d%s", col, row, suffix))
}

// mouseRaw sends an arbitrary escape sequence straight into a pane's input.
func (s *server) mouseRaw(pane, seq string) {
	args := []string{"send-keys", "-H", "-t", pane}
	for _, b := range []byte(seq) {
		args = append(args, fmt.Sprintf("%02x", b))
	}
	s.tmux(args...)
}

func waitFor(t *testing.T, desc string, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", desc)
}

// selBG is the Solarized Light selection background every theme test uses.
const selBG = "48;2;238;232;213"

// highlightedAgentLine returns the text of the line carrying the selection
// background and the word claude, "" if none.
func highlightedAgentLine(capture string) (line string, lineNo int) {
	for i, l := range strings.Split(capture, "\n") {
		if strings.Contains(l, selBG) && strings.Contains(l, "claude") {
			return l, i
		}
	}
	return "", -1
}

// highlightBelowHeader reports whether the selection highlight in capture sits
// on or below the header line for session name - i.e. on one of that session's
// agent rows. False if nothing is highlighted or the header isn't shown.
func highlightBelowHeader(capture, name string) bool {
	_, lineNo := highlightedAgentLine(capture)
	if lineNo < 0 {
		return false
	}
	headerLine := -1
	for i, l := range strings.Split(capture, "\n") {
		if strings.Contains(l, name) {
			headerLine = i
		}
	}
	return headerLine >= 0 && lineNo >= headerLine
}

// --- tests ---

// TestHookStateMachineLive drives the full Claude Code event sequence
// against a real pane and watches @agent_* options transition.
func TestHookStateMachineLive(t *testing.T) {
	s := start(t)
	s.newSession("work")
	pane := s.agentPane("work")

	if got := s.paneOption(pane, "@agent_state"); got != "idle" {
		t.Fatalf("after SessionStart: state=%q, want idle", got)
	}
	if got := s.paneOption(pane, "@agent_present"); got != "1" {
		t.Fatalf("after SessionStart: present=%q, want 1", got)
	}

	s.hook(pane, `{"hook_event_name":"UserPromptSubmit","session_id":"e2e"}`)
	if got := s.paneOption(pane, "@agent_state"); got != "working" {
		t.Errorf("after UserPromptSubmit: state=%q, want working", got)
	}
	since := s.paneOption(pane, "@agent_since")

	// PreToolUse while already working must not reset the clock.
	s.hook(pane, `{"hook_event_name":"PreToolUse","session_id":"e2e"}`)
	if got := s.paneOption(pane, "@agent_since"); got != since {
		t.Errorf("PreToolUse reset @agent_since: %q -> %q", since, got)
	}

	// A real tool-permission request (any tool but AskUserQuestion).
	s.hook(pane, `{"hook_event_name":"PermissionRequest","tool_name":"Bash"}`)
	if got := s.paneOption(pane, "@agent_state"); got != "permission" {
		t.Errorf("after PermissionRequest: state=%q, want permission", got)
	}

	// permission_prompt is a tool-blind echo of PermissionRequest: ignored.
	s.hook(pane, `{"hook_event_name":"Notification","notification_type":"permission_prompt"}`)
	if got := s.paneOption(pane, "@agent_state"); got != "permission" {
		t.Errorf("permission_prompt changed state to %q", got)
	}

	// AskUserQuestion arrives as a PermissionRequest but is Claude asking a
	// question, not a tool approval: it must read as question, not permission.
	s.hook(pane, `{"hook_event_name":"PermissionRequest","tool_name":"AskUserQuestion"}`)
	if got := s.paneOption(pane, "@agent_state"); got != "question" {
		t.Errorf("after AskUserQuestion PermissionRequest: state=%q, want question", got)
	}

	// idle_prompt nudges are deliberately ignored.
	s.hook(pane, `{"hook_event_name":"Notification","notification_type":"idle_prompt"}`)
	if got := s.paneOption(pane, "@agent_state"); got != "question" {
		t.Errorf("idle_prompt changed state to %q", got)
	}

	s.hook(pane, `{"hook_event_name":"SubagentStart"}`)
	s.hook(pane, `{"hook_event_name":"SubagentStart"}`)
	s.hook(pane, `{"hook_event_name":"SubagentStop"}`)
	if got := s.paneOption(pane, "@agent_subagents"); got != "1" {
		t.Errorf("subagents=%q, want 1", got)
	}

	s.hook(pane, `{"hook_event_name":"Stop"}`)
	if got := s.paneOption(pane, "@agent_state"); got != "done" {
		t.Errorf("after Stop: state=%q, want done", got)
	}

	s.hook(pane, `{"hook_event_name":"SessionEnd"}`)
	if got := s.paneOption(pane, "@agent_present"); got != "" {
		t.Errorf("after SessionEnd: present=%q, want unset", got)
	}
	if got := s.paneOption(pane, "@agent_state"); got != "" {
		t.Errorf("after SessionEnd: state=%q, want unset", got)
	}
}

// An attention state must never outlive the agent that entered it. With agent
// view open, Claude fires agent_needs_input for background sessions and later
// agent_completed; the first must be ignored (it never pairs with a clear and
// stranded the pane in "asking" for minutes) and the second read as a finish.
func TestHookAttentionStateNeverSticks(t *testing.T) {
	s := start(t)
	s.newSession("work")
	pane := s.agentPane("work")

	// A background-session "needs input" nudge must not fabricate "asking":
	// it fires while the agent is still working and never clears itself.
	s.hook(pane, `{"hook_event_name":"UserPromptSubmit","session_id":"e2e"}`)
	s.hook(pane, `{"hook_event_name":"Notification","notification_type":"agent_needs_input"}`)
	if got := s.paneOption(pane, "@agent_state"); got != "working" {
		t.Errorf("agent_needs_input fabricated attention state: %q, want working", got)
	}

	// A real question (AskUserQuestion) then agent_completed: the completion
	// clears the "asking" the bug left pinned across whole turns.
	s.hook(pane, `{"hook_event_name":"PermissionRequest","tool_name":"AskUserQuestion"}`)
	if got := s.paneOption(pane, "@agent_state"); got != "question" {
		t.Fatalf("after AskUserQuestion: state=%q, want question", got)
	}
	s.hook(pane, `{"hook_event_name":"Notification","notification_type":"agent_completed"}`)
	if got := s.paneOption(pane, "@agent_state"); got != "done" {
		t.Errorf("agent_completed did not clear asking: state=%q, want done", got)
	}

	// MCP elicitation round-trips: the dialog opens "asking", the response resumes.
	s.hook(pane, `{"hook_event_name":"Notification","notification_type":"elicitation_dialog"}`)
	if got := s.paneOption(pane, "@agent_state"); got != "question" {
		t.Fatalf("after elicitation_dialog: state=%q, want question", got)
	}
	s.hook(pane, `{"hook_event_name":"Notification","notification_type":"elicitation_complete"}`)
	if got := s.paneOption(pane, "@agent_state"); got != "working" {
		t.Errorf("elicitation_complete did not resume: state=%q, want working", got)
	}
}

// TestGlobalToggleLifecycle: one toggle opens a sidebar in every session,
// sessions born while on get one automatically, next toggle closes all.
func TestGlobalToggleLifecycle(t *testing.T) {
	s := start(t)
	for _, name := range []string{"aaa", "bbb", "ccc"} {
		s.newSession(name)
	}

	s.script("toggle.sh")
	for _, name := range []string{"aaa", "bbb", "ccc"} {
		waitFor(t, "sidebar in "+name, 5*time.Second, func() bool {
			return s.sidebarAlive(name)
		})
	}
	if hook := s.tmux("show-hooks", "-g"); !strings.Contains(hook, "session-created") {
		t.Error("global session-created hook not installed")
	}

	// A session created while globally on gets a sidebar automatically.
	s.newSession("ddd")
	waitFor(t, "auto sidebar in new session", 5*time.Second, func() bool {
		return s.sidebarAlive("ddd")
	})

	s.script("toggle.sh")
	for _, name := range []string{"aaa", "bbb", "ccc", "ddd"} {
		waitFor(t, "sidebar gone in "+name, 5*time.Second, func() bool {
			return !s.sidebarAlive(name)
		})
		if got := s.sidebarPane(name); got != "" {
			t.Errorf("%s: @sidebar_pane=%q after toggle off, want unset", name, got)
		}
	}
	// tmux 3.7 keeps an empty "session-created" entry after set-hook -gu,
	// so assert on the command being gone, then behaviorally: a session
	// born after toggle-off must NOT get a sidebar.
	if hook := s.tmux("show-hooks", "-g"); strings.Contains(hook, "open.sh") {
		t.Errorf("global session-created hook survived toggle off:\n%s", hook)
	}
	s.newSession("eee")
	time.Sleep(1 * time.Second)
	if s.sidebarAlive("eee") {
		t.Error("session created after toggle-off got a sidebar")
	}
}

// TestAutostartLifecycle: sourcing the plugin at server start opens a sidebar
// in the existing session, installs the session-created hook so later sessions
// get one too, and wires resurrect's restore-coordination hooks.
func TestAutostartLifecycle(t *testing.T) {
	s := start(t)
	s.newSession("aaa")
	s.plugin() // @agentbar-autostart defaults to on

	waitFor(t, "autostart sidebar in aaa", 5*time.Second, func() bool {
		return s.sidebarAlive("aaa")
	})
	if hook := s.tmux("show-hooks", "-g"); !strings.Contains(hook, "session-created") {
		t.Error("session-created hook not installed at server start")
	}
	// Auto-open is suspended during a restore and re-run (adopting) after.
	if got := s.tmux("show-option", "-gqv", "@resurrect-hook-pre-restore-all"); !strings.Contains(got, "session-created") {
		t.Errorf("pre-restore hook does not suspend auto-open: %q", got)
	}
	if got := s.tmux("show-option", "-gqv", "@resurrect-hook-post-restore-all"); !strings.Contains(got, "on.sh") {
		t.Errorf("post-restore hook does not re-run on.sh: %q", got)
	}

	// A session born after start gets a sidebar automatically.
	s.newSession("bbb")
	waitFor(t, "auto sidebar in new session", 5*time.Second, func() bool {
		return s.sidebarAlive("bbb")
	})
}

// TestAutostartDisabled: @agentbar-autostart 'off' starts closed - no sidebar
// and no session-created hook.
func TestAutostartDisabled(t *testing.T) {
	s := start(t)
	s.newSession("aaa")
	s.tmux("set-option", "-g", "@agentbar-autostart", "off")
	s.plugin()

	time.Sleep(1 * time.Second)
	if s.sidebarAlive("aaa") {
		t.Error("autostart off still opened a sidebar")
	}
	if hook := s.tmux("show-hooks", "-g"); strings.Contains(hook, "open.sh") {
		t.Error("autostart off still installed the session-created hook")
	}
}

// TestOnAdoptsExistingSidebar: on.sh is idempotent - a second run adopts the
// live sidebar rather than opening a second one. This is the post-restore
// re-run path (a restored sidebar must be adopted, not duplicated).
func TestOnAdoptsExistingSidebar(t *testing.T) {
	s := start(t)
	s.newSession("aaa")

	s.script("on.sh")
	waitFor(t, "sidebar in aaa", 5*time.Second, func() bool {
		return s.sidebarAlive("aaa")
	})
	first := s.sidebarPane("aaa")

	s.script("on.sh")
	time.Sleep(500 * time.Millisecond)
	if got := s.sidebarPane("aaa"); got != first {
		t.Errorf("second on.sh replaced the sidebar pane: %q -> %q", first, got)
	}
	panes := s.tmux("list-panes", "-s", "-t", "aaa", "-F", "#{pane_current_command}")
	if n := strings.Count(panes, "agentbar"); n != 1 {
		t.Errorf("want exactly one sidebar pane, got %d:\n%s", n, panes)
	}
}

// TestPinHoldsSidebarWidthOnResize: tmux takes a window shrink evenly from every
// pane in the row, which collapses the narrow sidebar; the plugin's
// window-resized hook puts the width back. tmux has no fixed-size pane.
func TestPinHoldsSidebarWidthOnResize(t *testing.T) {
	s := start(t)
	s.newSession("aaa")
	s.tmux("split-window", "-d", "-t", "aaa") // something for the row to take from
	s.plugin()                                // binds the key, installs the hook, autostarts
	waitFor(t, "sidebar in aaa", 5*time.Second, func() bool {
		return s.sidebarAlive("aaa")
	})
	pane := s.sidebarPane("aaa")
	if got := s.tmux("display-message", "-p", "-t", pane, "#{pane_width}"); got != "30" {
		t.Fatalf("sidebar width = %s before resize, want 30", got)
	}

	// manual: a test server has no client to resize.
	s.tmux("set", "-g", "window-size", "manual")
	s.tmux("resize-window", "-t", "aaa", "-x", "120", "-y", "40")
	waitFor(t, "sidebar width restored after shrink", 5*time.Second, func() bool {
		w, err := s.tmuxErr("display-message", "-p", "-t", pane, "#{pane_width}")
		return err == nil && strings.TrimSpace(w) == "30"
	})
	s.tmux("resize-window", "-t", "aaa", "-x", "220", "-y", "50")
	waitFor(t, "sidebar width held after grow", 5*time.Second, func() bool {
		w, err := s.tmuxErr("display-message", "-p", "-t", pane, "#{pane_width}")
		return err == nil && strings.TrimSpace(w) == "30"
	})

	// Hooks are arrays and the plugin is re-sourced on every reload, so a second
	// run must replace ours rather than stack another copy.
	s.plugin()
	hooks := s.tmux("show-hooks", "-gw")
	if n := strings.Count(hooks, "pin.sh"); n != 1 {
		t.Errorf("window-resized hook installed %d times, want 1:\n%s", n, hooks)
	}
}

// TestRestartRefreshesOneSidebar: restart.sh opens a sidebar where there is
// none and otherwise replaces the process in place, keeping the pane id (and so
// @sidebar_pane and the layout) - the reset path relies on both.
func TestRestartRefreshesOneSidebar(t *testing.T) {
	s := start(t)
	s.newSession("aaa")
	s.newSession("bbb")

	s.script("restart.sh", "aaa")
	waitFor(t, "sidebar in aaa", 5*time.Second, func() bool {
		return s.sidebarAlive("aaa")
	})
	pane := s.sidebarPane("aaa")
	pid := s.tmux("display-message", "-p", "-t", pane, "#{pane_pid}")

	s.script("restart.sh", "aaa")
	waitFor(t, "sidebar process replaced", 5*time.Second, func() bool {
		now, err := s.tmuxErr("display-message", "-p", "-t", pane, "#{pane_pid}")
		return err == nil && strings.TrimSpace(now) != pid && s.sidebarAlive("aaa")
	})
	if got := s.sidebarPane("aaa"); got != pane {
		t.Errorf("restart moved the sidebar pane: %q -> %q", pane, got)
	}
	// Per-session on purpose: a bulk restart storms this tmux+Ghostty client.
	if s.sidebarAlive("bbb") {
		t.Error("restart.sh touched another session's sidebar")
	}
}

// TestSidebarRendersAgentState: the TUI picks up hook-driven state changes.
func TestSidebarRendersAgentState(t *testing.T) {
	s := start(t)
	s.newSession("work")
	agent := s.agentPane("work")
	s.hook(agent, `{"hook_event_name":"UserPromptSubmit","session_id":"e2e"}`)

	s.script("open.sh", "work")
	side := s.sidebarPane("work")
	if active, _ := s.tmuxErr("display-message", "-t", "work", "-p", "#{pane_id}"); active == side {
		t.Error("opening the sidebar stole focus")
	}
	waitFor(t, "sidebar shows working agent", 5*time.Second, func() bool {
		c := s.capture(side)
		return strings.Contains(c, "claude") && strings.Contains(c, "working")
	})

	s.hook(agent, `{"hook_event_name":"Stop"}`)
	waitFor(t, "sidebar shows done", 5*time.Second, func() bool {
		return strings.Contains(s.capture(side), "done")
	})

	// Killing the agent pane removes its entry within a tick.
	s.tmux("kill-pane", "-t", agent)
	waitFor(t, "dead agent dropped", 5*time.Second, func() bool {
		return !strings.Contains(s.capture(side), "claude ")
	})
}

// TestSelectionSyncAcrossSidebars is the regression test for the click
// bug: each session's sidebar is its own process, so a jump published in
// one must move the highlight in all of them - immediately (wait-for
// signal), not on the next 1s tick.
func TestSelectionSyncAcrossSidebars(t *testing.T) {
	s := start(t)
	s.newSession("aaa")
	s.newSession("bbb")
	s.agentPane("aaa")
	agentB := s.agentPane("bbb")

	s.script("toggle.sh")
	sideA, sideB := s.sidebarPane("aaa"), s.sidebarPane("bbb")
	waitFor(t, "both sidebars list both agents", 5*time.Second, func() bool {
		return strings.Contains(s.capture(sideA), "bbb") &&
			strings.Contains(s.capture(sideB), "bbb")
	})

	// Publish a selection the way activate() does: option + signal.
	s.tmux("set-option", "-g", "@sidebar_selected", agentB, ";",
		"wait-for", "-S", "agentbar-refresh")

	// Both sidebars must adopt it well under the 1s snapshot tick.
	waitFor(t, "highlight on bbb's agent in both sidebars", 700*time.Millisecond, func() bool {
		for _, side := range []string{sideA, sideB} {
			if !highlightBelowHeader(s.capture(side), "bbb") {
				return false
			}
		}
		return true
	})
}

// TestJumpViaEnter walks the full user story with a real attached client:
// select the other session's agent in the sidebar, press Enter, and land
// there with that sidebar already highlighting the agent.
func TestJumpViaEnter(t *testing.T) {
	if _, err := exec.LookPath("script"); err != nil {
		t.Skip("script(1) not available for pty client")
	}
	s := start(t)
	s.newSession("aaa")
	s.newSession("bbb")
	s.agentPane("aaa")
	s.agentPane("bbb")
	// Keep windows at their detached size when the small pty attaches.
	s.tmux("set-option", "-g", "window-size", "manual")

	s.script("toggle.sh")
	sideA, sideB := s.sidebarPane("aaa"), s.sidebarPane("bbb")
	waitFor(t, "sidebars ready", 5*time.Second, func() bool {
		return strings.Contains(s.capture(sideA), "bbb") &&
			strings.Contains(s.capture(sideB), "bbb")
	})

	// Attach a real client (pty via script) to aaa.
	s.ptyClient("aaa")

	// In aaa's sidebar: G selects the last agent (bbb's), Enter jumps.
	s.tmux("send-keys", "-t", sideA, "G", "")
	s.tmux("send-keys", "-t", sideA, "Enter", "")

	waitFor(t, "client switched to bbb", 5*time.Second, func() bool {
		out, _ := s.tmuxErr("list-clients", "-F", "#{client_session}")
		return strings.Contains(out, "bbb")
	})

	// The bug: bbb's own sidebar (a different process) must show the
	// highlight on the jumped-to agent without any further clicks.
	waitFor(t, "bbb sidebar highlights jumped-to agent", 700*time.Millisecond, func() bool {
		_, lineNo := highlightedAgentLine(s.capture(sideB))
		return lineNo >= 0
	})
}

// TestTabJumpsToAttention: Tab is the work queue - it steps straight to an
// agent waiting on the user (permission/asking) in another session and
// switches the client there, skipping idle/working agents.
func TestTabJumpsToAttention(t *testing.T) {
	if _, err := exec.LookPath("script"); err != nil {
		t.Skip("script(1) not available for pty client")
	}
	s := start(t)
	s.newSession("aaa")
	s.newSession("bbb")
	s.agentPane("aaa") // idle agent in the current session
	bpane := s.agentPane("bbb")
	s.tmux("set-option", "-g", "window-size", "manual")

	// bbb's agent blocks on the user (a real tool-permission request).
	s.hook(bpane, `{"hook_event_name":"PermissionRequest","tool_name":"Bash"}`)

	s.script("toggle.sh")
	sideA := s.sidebarPane("aaa")
	waitFor(t, "sidebar shows bbb's waiting agent", 5*time.Second, func() bool {
		return s.paneOption(bpane, "@agent_state") == "permission" &&
			strings.Contains(s.capture(sideA), "permission")
	})

	s.ptyClient("aaa")

	// From aaa's sidebar (cursor on aaa's idle agent), Tab jumps past it to
	// the only agent waiting on the user - bbb's.
	s.tmux("send-keys", "-t", sideA, "Tab", "")

	waitFor(t, "Tab switched client to the waiting agent in bbb", 5*time.Second, func() bool {
		out, _ := s.tmuxErr("list-clients", "-F", "#{client_session}")
		return strings.Contains(out, "bbb")
	})
}

// TestClickJump: a single mouse click on another session's agent must
// switch there and highlight it in that session's own sidebar.
func TestClickJump(t *testing.T) {
	if _, err := exec.LookPath("script"); err != nil {
		t.Skip("script(1) not available for pty client")
	}
	s := start(t)
	s.newSession("aaa")
	s.newSession("bbb")
	s.agentPane("aaa")
	s.agentPane("bbb")
	s.tmux("set-option", "-g", "window-size", "manual")

	s.script("toggle.sh")
	sideA, sideB := s.sidebarPane("aaa"), s.sidebarPane("bbb")
	waitFor(t, "sidebars ready", 5*time.Second, func() bool {
		return strings.Contains(s.capture(sideA), "bbb") &&
			strings.Contains(s.capture(sideB), "bbb")
	})

	s.ptyClient("aaa")

	// Find bbb's agent row in aaa's sidebar: the first claude line after
	// the bbb session header (rows are 0-based, SGR is 1-based).
	lines := strings.Split(s.captureText(sideA), "\n")
	row := -1
	for i, l := range lines {
		if strings.Contains(l, "bbb") {
			for j := i + 1; j < len(lines); j++ {
				if strings.Contains(lines[j], "claude") {
					row = j + 1
					break
				}
			}
			break
		}
	}
	if row < 0 {
		t.Fatalf("bbb's agent row not found in sidebar:\n%s", strings.Join(lines, "\n"))
	}

	s.click(sideA, 5, row)

	waitFor(t, "client switched to bbb after single click", 5*time.Second, func() bool {
		out, _ := s.tmuxErr("list-clients", "-F", "#{client_session}")
		return strings.Contains(out, "bbb")
	})
	// Both sidebars - including bbb's, a separate process that never saw
	// the click - must highlight the clicked agent, faster than the tick.
	waitFor(t, "clicked agent highlighted in both sidebars", 700*time.Millisecond, func() bool {
		for _, side := range []string{sideA, sideB} {
			if _, lineNo := highlightedAgentLine(s.capture(side)); lineNo != row-1 {
				return false
			}
		}
		return true
	})

	// Release-only click (terminal ate the focusing press) back on aaa's
	// agent must jump too.
	lines = strings.Split(s.captureText(sideB), "\n")
	backRow := -1
	for i, l := range lines {
		if strings.Contains(l, "claude") {
			backRow = i + 1 // first agent listed is aaa's
			break
		}
	}
	if backRow < 0 {
		t.Fatal("aaa's agent row not found in bbb's sidebar")
	}
	s.releaseClick(sideB, 5, backRow)
	waitFor(t, "release-only click switched back to aaa", 5*time.Second, func() bool {
		out, _ := s.tmuxErr("list-clients", "-F", "#{client_session}")
		return strings.Contains(out, "aaa")
	})
}

// TestClickSessionSwitches: clicking a session name (not an agent) must
// switch-client to that session - including an agent-less one, which has
// no row to click today and is otherwise unreachable from the sidebar.
func TestClickSessionSwitches(t *testing.T) {
	if _, err := exec.LookPath("script"); err != nil {
		t.Skip("script(1) not available for pty client")
	}
	s := start(t)
	s.newSession("aaa")
	// Route the shared trace log to a temp dir so we can assert the switch was
	// recorded (always-on; no verbose needed). After the server exists.
	state := t.TempDir()
	s.tmux("set-environment", "-g", "XDG_STATE_HOME", state)
	traceLog := filepath.Join(state, "dotfiles", "trace.log")
	s.newSession("bbb") // deliberately no agent
	s.agentPane("aaa")
	s.tmux("set-option", "-g", "window-size", "manual")

	s.script("toggle.sh")
	sideA := s.sidebarPane("aaa")
	waitFor(t, "sidebar lists the agent-less bbb", 5*time.Second, func() bool {
		return strings.Contains(s.capture(sideA), "bbb")
	})

	s.ptyClient("aaa")

	// bbb has no agent, so its only row is the session header.
	lines := strings.Split(s.captureText(sideA), "\n")
	row := -1
	for i, l := range lines {
		if strings.Contains(l, "bbb") {
			row = i + 1 // rows are 0-based, SGR is 1-based
			break
		}
	}
	if row < 0 {
		t.Fatalf("bbb's session row not found in sidebar:\n%s", strings.Join(lines, "\n"))
	}

	s.click(sideA, 2, row) // column 2 is inside the "bbb" name

	sideB := s.sidebarPane("bbb")
	waitFor(t, "click on the session name switched the client to bbb", 5*time.Second, func() bool {
		out, _ := s.tmuxErr("list-clients", "-F", "#{client_session}")
		return strings.Contains(out, "bbb")
	})
	// The clicked session's row must be highlighted in its own sidebar -
	// even with no agent to fall back to (the old bug left the highlight
	// stuck on the previously-selected row).
	waitFor(t, "bbb's session row is highlighted after the switch", 3*time.Second, func() bool {
		for l := range strings.SplitSeq(s.capture(sideB), "\n") {
			if strings.Contains(l, "bbb") && strings.Contains(l, selBG) {
				return true
			}
		}
		return false
	})
	// The click-driven session switch must land in the shared trace log.
	waitFor(t, "the session switch was traced", 3*time.Second, func() bool {
		b, _ := os.ReadFile(traceLog)
		return strings.Contains(string(b), "evt=switch") && strings.Contains(string(b), "session=bbb")
	})
}

// TestHoverMotionReachesUnfocusedSidebar is the feasibility gate for a
// hover highlight: tmux must forward bare pointer-motion (any-motion
// tracking) to the sidebar pane while a different pane is focused. A
// motion sequence written to the client's stdin goes through tmux's real
// mouse routing (unlike send-keys into the pane).
func TestHoverMotionReachesUnfocusedSidebar(t *testing.T) {
	if _, err := exec.LookPath("script"); err != nil {
		t.Skip("script(1) not available for pty client")
	}
	s := start(t)
	// Build the window at the pty client's native 80x24 so client mouse
	// coords map 1:1 onto the window (no resize, which redistributes panes).
	s.tmux("new-session", "-d", "-s", "work", "-x", "80", "-y", "24")
	// Route the shared trace log to a temp dir and enable verbose tracing, so
	// the sidebar records mouse motion. Both are read from the tmux env at
	// sidebar startup (set-environment -g reaches the spawned pane); it must run
	// after the server exists (new-session above created it).
	state := t.TempDir()
	s.tmux("set-environment", "-g", "XDG_STATE_HOME", state)
	s.tmux("set-environment", "-g", "DOTFILES_TRACE_VERBOSE", "1")
	log := filepath.Join(state, "dotfiles", "trace.log")
	s.agentPane("work")
	s.tmux("set-option", "-g", "mouse", "on")

	s.script("open.sh", "work")
	side := s.sidebarPane("work")
	waitFor(t, "sidebar shows the agent", 5*time.Second, func() bool {
		return strings.Contains(s.capture(side), "claude")
	})
	if active, _ := s.tmuxErr("display-message", "-t", "work", "-p", "#{pane_id}"); active == side {
		t.Fatal("sidebar is focused; test needs it unfocused")
	}

	// Inject a bare-motion sequence through a real client's stdin (routed by
	// tmux, unlike send-keys) at the sidebar's screen coords, with the work
	// pane focused. The sidebar must log a motion event (action=2).
	stdin := s.ptyClient("work")
	time.Sleep(400 * time.Millisecond)
	fmt.Fprintf(stdin, "\x1b[<35;4;5M")
	waitFor(t, "unfocused sidebar received the routed motion", 5*time.Second, func() bool {
		b, _ := os.ReadFile(log)
		return strings.Contains(string(b), "evt=mouse") && strings.Contains(string(b), "action=2")
	})
}

// TestFollowKeepsColumnWidths: moving the sidebar in and out of a window
// must not redistribute the other columns - tmux takes the inserted
// width proportionally from all panes but returns it to the leftmost
// only, which drained the right column a bit per window switch.
func TestFollowKeepsColumnWidths(t *testing.T) {
	s := start(t)
	s.newSession("work")
	// Three columns: the drain needs panes beyond the leftmost.
	right := s.tmux("split-window", "-h", "-d", "-t", "work:0.0", "-l", "60",
		"-P", "-F", "#{pane_id}")
	mid := s.tmux("split-window", "-h", "-d", "-t", "work:0.0", "-l", "60",
		"-P", "-F", "#{pane_id}")

	s.script("open.sh", "work")
	side := s.sidebarPane("work")
	widths := func() string {
		m, _ := s.tmuxErr("display-message", "-p", "-t", mid, "#{pane_width}")
		r, _ := s.tmuxErr("display-message", "-p", "-t", right, "#{pane_width}")
		return strings.TrimSpace(m) + "," + strings.TrimSpace(r)
	}
	if got := widths(); got != "60,60" {
		t.Fatalf("columns = %s after sidebar open, want 60,60", got)
	}

	sidebarIn := func(win string) func() bool {
		return func() bool {
			cur, _ := s.tmuxErr("display-message", "-t", win, "-p", "#{window_id}")
			sidewin, _ := s.tmuxErr("display-message", "-t", side, "-p", "#{window_id}")
			return cur != "" && cur == sidewin
		}
	}
	s.tmux("new-window", "-t", "work") // window the sidebar will bounce via
	for i := range 5 {
		s.tmux("select-window", "-t", "work:1")
		waitFor(t, "sidebar in window 1", 5*time.Second, sidebarIn("work:1"))
		s.tmux("select-window", "-t", "work:0")
		waitFor(t, "sidebar in window 0", 5*time.Second, sidebarIn("work:0"))
		// join-pane moves the sidebar, then insert_keeping_widths restores the
		// columns - so sample until they settle rather than in that gap. A real
		// drain never settles and still fails here.
		waitFor(t, fmt.Sprintf("columns restored after switch %d (last: %s)", i+1, widths()),
			5*time.Second, func() bool { return widths() == "60,60" })
	}
}

// TestFollowWindowAndSelfHeal: the sidebar pane follows the active window,
// and stale state self-heals when the sidebar process is gone.
func TestFollowWindowAndSelfHeal(t *testing.T) {
	s := start(t)
	s.newSession("work")
	s.script("open.sh", "work")
	side := s.sidebarPane("work")
	waitFor(t, "sidebar open", 5*time.Second, func() bool { return s.sidebarAlive("work") })

	s.tmux("new-window", "-t", "work")
	waitFor(t, "sidebar followed to new window", 5*time.Second, func() bool {
		cur, _ := s.tmuxErr("display-message", "-t", "work", "-p", "#{window_id}")
		sidewin, _ := s.tmuxErr("display-message", "-t", side, "-p", "#{window_id}")
		return cur != "" && cur == sidewin
	})
	// The move must not steal focus or the window name (join-pane -d).
	if active, _ := s.tmuxErr("display-message", "-t", "work", "-p", "#{pane_id}"); active == side {
		t.Error("sidebar stole focus after following the window")
	}
	if name, _ := s.tmuxErr("display-message", "-t", "work", "-p", "#{window_name}"); name == "agentbar" {
		t.Error("window auto-renamed to the sidebar after follow")
	}

	// Kill the sidebar process behind tmux's back; the next window change
	// must clean up the stale options and hook.
	s.tmux("kill-pane", "-t", side)
	s.tmux("new-window", "-t", "work")
	waitFor(t, "stale sidebar state cleaned", 5*time.Second, func() bool {
		return s.sidebarPane("work") == ""
	})
	if on, _ := s.tmuxErr("show-option", "-t", "work", "-qv", "@sidebar_on"); on != "" {
		t.Errorf("@sidebar_on=%q after self-heal, want unset", on)
	}
}

// TestFollowClearsMovingGuardOnFailure: a failed move must not strand the
// @sidebar_moving re-entrancy guard. The window or pane disappearing mid-move
// is the very race the guard exists for, and a stranded guard makes every
// later hook bail at the check - the sidebar then silently stops following
// windows for the rest of the session, with nothing to self-heal it.
func TestFollowClearsMovingGuardOnFailure(t *testing.T) {
	s := start(t)
	s.newSession("work")
	s.script("open.sh", "work")
	waitFor(t, "sidebar open", 5*time.Second, func() bool { return s.sidebarAlive("work") })
	side := s.sidebarPane("work")

	// Drive follow.sh by hand: drop the hook so tmux can't race us to the
	// move, then leave the sidebar in a window other than the active one.
	s.tmux("set-hook", "-u", "-t", "work", "session-window-changed")
	s.tmux("new-window", "-t", "work")
	sidebarInActiveWindow := func() bool {
		cur, _ := s.tmuxErr("display-message", "-t", "work", "-p", "#{window_id}")
		sidewin, _ := s.tmuxErr("display-message", "-t", side, "-p", "#{window_id}")
		return cur != "" && cur == sidewin
	}
	if sidebarInActiveWindow() {
		t.Fatal("sidebar already in the active window - nothing for follow.sh to move")
	}

	// insert_keeping_widths opens with a list-panes carrying #{pane_left}, the
	// first thing to run after the guard goes up.
	if err := s.scriptFaulty("follow.sh", "pane_left", "work"); err == nil {
		t.Fatal("fault did not abort follow.sh - test no longer covers the failed move")
	}
	if g, _ := s.tmuxErr("show-option", "-t", "work", "-qv", "@sidebar_moving"); g != "" {
		t.Errorf("@sidebar_moving = %q after a failed move, want unset", g)
	}

	// The symptom that matters: following still works afterwards.
	s.script("follow.sh", "work")
	waitFor(t, "sidebar follows after a failed move", 5*time.Second, sidebarInActiveWindow)
}

// TestStatusBarTabClick: with Second/TripleClick1Status bound (stock
// tmux drops chained rapid clicks - README tip), every tab click must
// switch, even while the follow hook is moving the sidebar.
func TestStatusBarTabClick(t *testing.T) {
	s := start(t)
	s.newSession("work")
	s.agentPane("work")
	s.tmux("new-window", "-t", "work") // window 1; window 0 is the first
	// Deterministic tab geometry: window list only, ' #I ' per tab.
	s.tmux("set-option", "-g", "mouse", "on", ";",
		"set-option", "-g", "status-left", "", ";",
		"set-option", "-g", "status-right", "", ";",
		"set-option", "-g", "window-status-format", " #I ", ";",
		"set-option", "-g", "window-status-current-format", " #I ")
	s.tmux("bind-key", "-n", "SecondClick1Status", "switch-client -t =")
	s.tmux("bind-key", "-n", "TripleClick1Status", "switch-client -t =")

	s.script("toggle.sh")
	waitFor(t, "sidebar open", 5*time.Second, func() bool { return s.sidebarAlive("work") })

	stdin := s.ptyClient("work")
	var height int
	waitFor(t, "client size known", 5*time.Second, func() bool {
		out, _ := s.tmuxErr("list-clients", "-F", "#{client_height}")
		_, err := fmt.Sscanf(strings.TrimSpace(out), "%d", &height)
		return err == nil && height > 0
	})
	statusRow := height // status line is the bottom row
	// Rendered window list: ` 0 ` ` 1 ` (separator between) -> tab
	// centers at columns 2 and 6.
	tabCol := map[int]int{0: 2, 1: 6}

	activeWindow := func() string {
		out, _ := s.tmuxErr("display-message", "-t", "work", "-p", "#{window_index}")
		return strings.TrimSpace(out)
	}

	for i := range 12 {
		target := i % 2
		clientClick(stdin, tabCol[target], statusRow)
		ok := false
		deadline := time.Now().Add(1500 * time.Millisecond)
		for time.Now().Before(deadline) {
			if activeWindow() == fmt.Sprint(target) {
				ok = true
				break
			}
			time.Sleep(25 * time.Millisecond)
		}
		if !ok {
			t.Fatalf("click %d: single status-bar click on window %d did not switch (active=%s)",
				i, target, activeWindow())
		}
	}
}

// TestResurrectOrphanAdoption: open.sh must adopt an untracked sidebar
// pane (resurrect restores no options/hooks) instead of opening a second
// one, and toggle-off must kill it.
func TestResurrectOrphanAdoption(t *testing.T) {
	s := start(t)
	s.newSession("work")
	s.agentPane("work")

	// Simulate the restore: a pane running the sidebar with no state.
	orphan := s.tmux("split-window", "-d", "-t", "work", "-P", "-F", "#{pane_id}",
		binPath+" run")
	waitFor(t, "orphan sidebar running", 5*time.Second, func() bool {
		out, _ := s.tmuxErr("list-panes", "-s", "-t", "work",
			"-F", "#{pane_id} #{pane_current_command}")
		return strings.Contains(out, orphan+" agentbar")
	})

	s.script("open.sh", "work")
	if got := s.sidebarPane("work"); got != orphan {
		t.Errorf("@sidebar_pane = %q, want adopted orphan %s", got, orphan)
	}
	out := s.tmux("list-panes", "-s", "-t", "work", "-F", "#{pane_current_command}")
	if n := strings.Count(out, "agentbar"); n != 1 {
		t.Errorf("%d sidebar panes after open over orphan, want 1", n)
	}
	if hooks, _ := s.tmuxErr("show-hooks", "-t", "work"); !strings.Contains(hooks, "follow.sh") {
		t.Error("adoption did not install the follow hook")
	}

	// Toggle sees a live sidebar -> closes everywhere, orphan included.
	s.script("toggle.sh")
	waitFor(t, "orphan killed by toggle-off", 5*time.Second, func() bool {
		out := s.tmux("list-panes", "-s", "-t", "work", "-F", "#{pane_current_command}")
		return !strings.Contains(out, "agentbar")
	})
}

// TestSessionSwitchMovesHighlight: switching sessions outside the
// sidebar (keys, session buttons) must move the highlight to the newly
// attached session's agent - instantly via the client-session-changed
// signal, not on the next tick.
func TestSessionSwitchMovesHighlight(t *testing.T) {
	s := start(t)
	s.newSession("aaa")
	s.newSession("bbb")
	s.agentPane("aaa")
	s.agentPane("bbb")
	s.tmux("set-option", "-g", "window-size", "manual")

	s.script("toggle.sh")
	sideA, sideB := s.sidebarPane("aaa"), s.sidebarPane("bbb")
	waitFor(t, "sidebars ready", 5*time.Second, func() bool {
		return strings.Contains(s.capture(sideA), "bbb") &&
			strings.Contains(s.capture(sideB), "bbb")
	})

	s.ptyClient("aaa")
	tty := strings.TrimSpace(s.tmux("list-clients", "-F", "#{client_tty}"))
	waitFor(t, "highlight on aaa's agent (above the bbb header)", 2*time.Second, func() bool {
		capture := s.capture(sideA)
		_, lineNo := highlightedAgentLine(capture)
		for i, l := range strings.Split(capture, "\n") {
			if strings.Contains(l, "bbb") {
				return lineNo >= 0 && lineNo < i
			}
		}
		return false
	})

	// External switch, no sidebar involved.
	s.tmux("switch-client", "-c", tty, "-t", "bbb")
	waitFor(t, "both sidebars highlight bbb's agent", 700*time.Millisecond, func() bool {
		for _, side := range []string{sideA, sideB} {
			if !highlightBelowHeader(s.capture(side), "bbb") {
				return false
			}
		}
		return true
	})
}

// TestAgentStartedAfterSwitchGetsHighlight: switching to a session that
// has no agent yet gives the highlight nothing to move to; an agent
// started there right after must still receive it.
func TestAgentStartedAfterSwitchGetsHighlight(t *testing.T) {
	s := start(t)
	s.newSession("aaa")
	s.newSession("bbb")
	s.agentPane("aaa")
	s.tmux("set-option", "-g", "window-size", "manual")

	s.script("toggle.sh")
	sideA, sideB := s.sidebarPane("aaa"), s.sidebarPane("bbb")
	waitFor(t, "sidebars ready", 5*time.Second, func() bool {
		return strings.Contains(s.capture(sideA), "bbb") &&
			strings.Contains(s.capture(sideB), "bbb")
	})

	s.ptyClient("aaa")
	tty := strings.TrimSpace(s.tmux("list-clients", "-F", "#{client_tty}"))
	s.tmux("switch-client", "-c", tty, "-t", "bbb")
	// Let a snapshot tick observe "bbb attached, no agent" so the agent's
	// arrival, not the switch itself, is what must move the highlight.
	time.Sleep(1500 * time.Millisecond)

	s.agentPane("bbb")
	waitFor(t, "late-started agent highlighted in both sidebars", 3*time.Second, func() bool {
		for _, side := range []string{sideA, sideB} {
			if !highlightBelowHeader(s.capture(side), "bbb") {
				return false
			}
		}
		return true
	})
}

// TestSidebarSelfRegisters: a sidebar started outside open.sh (as a
// resurrect restore does) must stamp its own options and follow hook.
func TestSidebarSelfRegisters(t *testing.T) {
	s := start(t)
	s.newSession("work")
	pane := s.tmux("split-window", "-d", "-t", "work", "-P", "-F", "#{pane_id}",
		binPath+" run")

	waitFor(t, "self-registered options", 5*time.Second, func() bool {
		return s.sidebarPane("work") == pane
	})
	if hooks, _ := s.tmuxErr("show-hooks", "-t", "work"); !strings.Contains(hooks, "follow.sh") {
		t.Error("self-registration did not install the follow hook")
	}
	// And the follow hook actually works: sidebar moves with the window.
	s.tmux("new-window", "-t", "work")
	waitFor(t, "sidebar followed", 5*time.Second, func() bool {
		cur, _ := s.tmuxErr("display-message", "-t", "work", "-p", "#{window_id}")
		sidewin, _ := s.tmuxErr("display-message", "-t", pane, "-p", "#{window_id}")
		return cur != "" && cur == sidewin
	})
}

// TestOpenLeavesNoPhantomSidebar: when the split genuinely fails (a window
// with no room for another pane), open.sh must not mark the session as having
// a sidebar. @sidebar_on=1 with an empty @sidebar_pane wedges the session:
// follow.sh bails on the empty pane before its self-heal can clear the flags.
func TestOpenLeavesNoPhantomSidebar(t *testing.T) {
	s := start(t)
	s.tmux("new-session", "-d", "-s", "tiny", "-x", "2", "-y", "20")
	s.tmux("set-option", "-g", "window-size", "manual")
	s.tmux("resize-window", "-t", "tiny", "-x", "2", "-y", "20")
	// A second session proves one unopenable session doesn't abort the sweep.
	s.newSession("roomy")

	s.script("on.sh")

	if on, _ := s.tmuxErr("show-option", "-t", "tiny", "-qv", "@sidebar_on"); on != "" {
		t.Errorf("@sidebar_on = %q on a session whose split failed, want unset", on)
	}
	if pane := s.sidebarPane("tiny"); pane != "" {
		t.Errorf("@sidebar_pane = %q on a session whose split failed, want unset", pane)
	}
	waitFor(t, "sidebar in the roomy session", 5*time.Second, func() bool {
		return s.sidebarAlive("roomy")
	})
}

// TestSidebarSurvivesOneColumnPane: a sidebar squeezed to a single column
// must keep running. tmux CLAMPS the -l 30 split instead of refusing it, so a
// narrow window hands the sidebar a 1-column pane on the ordinary open path,
// and a render that panics there kills a long-lived process for good (its
// stderr goes nowhere, so it just vanishes).
func TestSidebarSurvivesOneColumnPane(t *testing.T) {
	s := start(t)

	// Route 1: open into a window too narrow for the 30-column split.
	s.tmux("new-session", "-d", "-s", "narrow", "-x", "12", "-y", "20")
	s.tmux("set-option", "-g", "window-size", "manual")
	s.tmux("resize-window", "-t", "narrow", "-x", "12", "-y", "20")
	s.script("open.sh", "narrow")

	side := s.sidebarPane("narrow")
	if side == "" {
		t.Fatal("no sidebar pane recorded for the narrow session")
	}
	if w, _ := s.tmuxErr("display-message", "-p", "-t", side, "#{pane_width}"); strings.TrimSpace(w) != "1" {
		t.Fatalf("narrow open gave the sidebar width %q, want 1 - test no longer exercises the squeeze", w)
	}
	// A crash shows up as the pane dying: give it renders to crash in first.
	time.Sleep(time.Second)
	if !s.sidebarAlive("narrow") {
		t.Error("sidebar died in a 1-column pane on open")
	}

	// Route 2: squeeze a healthy 30-column sidebar down to one column.
	s.newSession("work")
	s.script("open.sh", "work")
	waitFor(t, "sidebar in work", 5*time.Second, func() bool { return s.sidebarAlive("work") })
	wide := s.sidebarPane("work")
	s.tmux("resize-pane", "-t", wide, "-x", "1")
	if w, _ := s.tmuxErr("display-message", "-p", "-t", wide, "#{pane_width}"); strings.TrimSpace(w) != "1" {
		t.Fatalf("resize left the sidebar at width %q, want 1", w)
	}
	time.Sleep(time.Second)
	if !s.sidebarAlive("work") {
		t.Fatal("sidebar died when squeezed to 1 column")
	}
	// And it comes back to life when given room again.
	s.tmux("resize-pane", "-t", wide, "-x", "30")
	waitFor(t, "sidebar redraws after widening", 5*time.Second, func() bool {
		return strings.Contains(s.captureText(wide), "agentbar")
	})
}

// TestResurrectSaveHook: the post-save hook stamps a restore command on
// sidebar pane lines (resurrect saves them with an empty command).
func TestResurrectSaveHook(t *testing.T) {
	s := start(t)
	s.newSession("work")

	// Sidebar lines carry an empty command or the blocked wait-for child.
	state := filepath.Join(t.TempDir(), "state.txt")
	lines := "pane\twork\t1\t1\t:*\t1\thost\t:/tmp\t0\tagentbar\t:\n" +
		"pane\twork\t2\t1\t:*\t1\thost\t:/tmp\t0\tagentbar\t:/usr/bin/tmux wait-for agentbar-refresh\n" +
		"pane\twork\t1\t1\t:*\t2\thost\t:/tmp\t1\tclaude\t:claude\n"
	if err := os.WriteFile(state, []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}

	s.script("resurrect-save.sh", state)
	out, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	stamped := "agentbar\t:" + repoRoot + "/bin/agentbar run --theme"
	if n := strings.Count(string(out), stamped); n != 2 {
		t.Errorf("stamped %d sidebar lines, want 2:\n%s", n, out)
	}
	if strings.Contains(string(out), "wait-for") {
		t.Errorf("wait-for child command survived:\n%s", out)
	}
	if !strings.Contains(string(out), "claude\t:claude") {
		t.Errorf("non-sidebar line modified:\n%s", out)
	}
}

// TestResurrectSaveHookClaudeResume: the post-save hook rewrites each claude
// pane line into `claude --resume <session-id>`, keyed by the pane's live
// @agent_session_id, so a restore resumes the conversation. Covers the edge
// cases resurrect's saved command throws at it: a normal `claude`, an empty
// command (claude was the pane's root process, so ps() sees no child), extra
// user flags to preserve, and a stale `--resume` from an earlier restore that
// must be replaced -- not stacked into a double flag.
func TestResurrectSaveHookClaudeResume(t *testing.T) {
	s := start(t)
	s.newSession("work")

	// One live agent pane per case, each with a distinct stamped id. Each gets
	// its own window so indices stay stable and distinct (splitting the active
	// pane repeatedly would renumber earlier panes). The state line's
	// session/window/pane index must match the live pane.
	agent := func(sid string) (win, idx string) {
		p := s.tmux("new-window", "-d", "-t", "work", "-P", "-F", "#{pane_id}", "claude 600")
		s.hook(p, fmt.Sprintf(`{"hook_event_name":"SessionStart","session_id":%q}`, sid))
		return s.tmux("display-message", "-p", "-t", p, "#{window_index}"),
			s.tmux("display-message", "-p", "-t", p, "#{pane_index}")
	}
	line := func(sid, saved string) (string, string) {
		win, idx := agent(sid)
		l := fmt.Sprintf("pane\twork\t%s\t1\t:*\t%s\thost\t:/tmp\t1\tclaude\t%s\n", win, idx, saved)
		want := ":claude --resume " + sid
		if sid == "sid-flags" {
			want = ":claude --dangerously-skip-permissions --resume " + sid
		}
		return l, want
	}

	cases := map[string]string{
		"sid-plain": ":claude",
		"sid-empty": ":",
		"sid-flags": ":claude --dangerously-skip-permissions --resume STALE-ID",
	}
	var body strings.Builder
	wants := map[string]string{}
	for sid, saved := range cases {
		l, want := line(sid, saved)
		body.WriteString(l)
		wants[sid] = want
	}

	state := filepath.Join(t.TempDir(), "state.txt")
	if err := os.WriteFile(state, []byte(body.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	s.script("resurrect-save.sh", state)
	out, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for sid, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("%s: want line ending %q, got:\n%s", sid, want, got)
		}
	}
	// Idempotency: the stale id is gone and resume appears exactly once.
	if strings.Contains(got, "STALE-ID") {
		t.Errorf("stale --resume survived:\n%s", got)
	}
	if n := strings.Count(got, "--resume sid-flags --resume"); n != 0 {
		t.Errorf("double --resume flag:\n%s", got)
	}
}

// TestNotifyToggle: pressing `n` in the sidebar flips the global
// @agent_notify option (which the hook reads) and the footer chip.
func TestNotifyToggle(t *testing.T) {
	s := start(t)
	s.newSession("work")
	s.agentPane("work")
	s.script("open.sh", "work")
	side := s.sidebarPane("work")
	if side == "" {
		t.Fatal("no sidebar pane")
	}
	waitFor(t, "footer shows notify off", 5*time.Second, func() bool {
		return strings.Contains(s.capture(side), "notify off")
	})

	s.tmux("send-keys", "-t", side, "n")
	waitFor(t, "footer shows notify on", 5*time.Second, func() bool {
		return strings.Contains(s.capture(side), "notify on")
	})
	if got := s.tmux("show-option", "-gqv", "@agent_notify"); got != "on" {
		t.Errorf("@agent_notify = %q, want on", got)
	}
}

// TestStatusSegment: the status subcommand counts attention + working.
func TestStatusSegment(t *testing.T) {
	s := start(t)
	s.newSession("work")
	a := s.agentPane("work")
	b := s.agentPane("work")
	s.hook(a, `{"hook_event_name":"UserPromptSubmit","session_id":"e2e"}`)
	s.hook(b, `{"hook_event_name":"PermissionRequest","tool_name":"Bash"}`)

	cmd := exec.Command(binPath, "status")
	cmd.Env = s.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "⚠1") || !strings.Contains(string(out), "●1") {
		t.Errorf("status segment = %q, want ⚠1 and ●1", out)
	}
}

// agentbar runs the binary against this server and returns its stdout.
func (s *server) agentbar(args ...string) string {
	s.t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = s.env
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		s.t.Fatalf("agentbar %v: %v\n%s", args, err, stderr.String())
	}
	return string(out)
}

// agentbarEnv runs the binary with extra environment - a stale TMUX_PANE, as a
// tmux server that inherited one hands to every run-shell child.
func (s *server) agentbarEnv(env []string, args ...string) string {
	s.t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(append([]string{}, s.env...), env...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		s.t.Fatalf("agentbar %v: %v\n%s", args, err, stderr.String())
	}
	return string(out)
}

// clientSession names the session the attached client is looking at.
func (s *server) clientSession() string {
	return strings.TrimSpace(s.tmux("list-clients", "-F", "#{client_session}"))
}

// `agentbar order` is the single source of the session order: bands first
// (pinned, active, dormant), alphabetical inside each. The session keys and the
// picker popup both walk it, so a drift back to plain alphabetical - the whole
// reason the keys felt jarring - has to fail here.
func TestOrderFollowsSidebarBands(t *testing.T) {
	s := start(t)
	for _, name := range []string{"api", "blog", "dotfiles"} {
		s.newSession(name)
		s.agentPane(name)
	}
	s.newSession("payments") // no agent: dormant
	s.ptyClient("dotfiles")
	s.agentbar("pin", "blog")
	s.agentbar("pin", "dotfiles")

	// Alphabetically this would be api, blog, dotfiles, payments.
	want := "pinned\tblog\npinned\tdotfiles\nactive\tapi\ndormant\tpayments\n"
	if got := s.agentbar("order"); got != want {
		t.Errorf("order =\n%q\nwant\n%q", got, want)
	}
}

// next/prev walk that order top to bottom and back, wrapping at both ends -
// never tmux's own alphabetical session list.
func TestNextPrevWalkSidebarOrder(t *testing.T) {
	s := start(t)
	for _, name := range []string{"api", "blog", "dotfiles"} {
		s.newSession(name)
		s.agentPane(name)
	}
	s.newSession("payments")
	s.ptyClient("dotfiles")
	s.agentbar("pin", "blog")
	s.agentbar("pin", "dotfiles")
	// Order is now: blog, dotfiles, api, payments.

	// The bindings pass tmux's own #{client_session}: tmux never re-stamps
	// TMUX_PANE for a key binding's run-shell child, so a guessed current
	// session walks from the wrong row. Step from an explicit name here, and
	// from a stale TMUX_PANE in the environment, which used to blank it.
	steps := []struct {
		cmd  string
		want string
	}{
		{"next", "api"},  // down a band boundary; alphabetically: payments
		{"next", "payments"}, // bottom row
		{"next", "blog"},  // wraps to the top; alphabetically: api
		{"prev", "payments"}, // wraps back past the top
		{"prev", "api"},
	}
	tty := strings.TrimSpace(s.tmux("list-clients", "-F", "#{client_tty}"))
	for _, step := range steps {
		from := s.clientSession()
		// Exactly what the tmux binding runs: #{client_session} #{client_tty}.
		s.agentbarEnv([]string{"TMUX_PANE=%999"}, step.cmd, from, tty)
		waitFor(t, step.cmd+" from "+from+" -> "+step.want, 2*time.Second, func() bool {
			return s.clientSession() == step.want
		})
	}
}

// Pins are the only thing that reorders the bar now, and tmux drops user
// options when its server exits - so the disk mirror has to bring them back.
func TestPinsSurviveServerRestart(t *testing.T) {
	s := start(t)
	s.newSession("api")
	s.agentPane("api")
	s.newSession("blog")
	s.agentPane("blog")
	s.ptyClient("api")
	s.agentbar("pin", "blog")

	s.tmux("kill-server")
	s.newSession("api") // fresh server: @agentbar-pins is gone
	s.agentPane("api")
	s.newSession("blog")
	s.agentPane("blog")

	want := "pinned\tblog\nactive\tapi\n"
	if got := s.agentbar("order"); got != want {
		t.Errorf("order after restart =\n%q\nwant\n%q", got, want)
	}
	if got := s.tmux("show-option", "-gqv", "@agentbar-pins"); got != "blog" {
		t.Errorf("@agentbar-pins after restart = %q, want it stamped back as blog", got)
	}
}

// `agentbar pin` is the picker popup's pin key: it must toggle the same set
// the sidebar's own p key writes, both ways.
func TestPinCommandToggles(t *testing.T) {
	s := start(t)
	s.newSession("my repo") // a space: tmux allows it, so the storage must
	s.agentPane("my repo")

	s.agentbar("pin", "my repo")
	if got := s.agentbar("order"); got != "pinned\tmy repo\n" {
		t.Errorf("order after pin = %q, want the session pinned", got)
	}
	s.agentbar("pin", "my repo")
	if got := s.agentbar("order"); got != "active\tmy repo\n" {
		t.Errorf("order after unpin = %q, want the session unpinned", got)
	}
}
