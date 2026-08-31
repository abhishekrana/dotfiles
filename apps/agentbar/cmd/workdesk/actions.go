package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/abhishekrana/agentbar/internal/deskui"
	"github.com/abhishekrana/agentbar/internal/gitlab"
	"github.com/abhishekrana/agentbar/internal/tmux"
	"github.com/abhishekrana/agentbar/internal/trace"
	"github.com/abhishekrana/agentbar/internal/ui"
	"github.com/abhishekrana/agentbar/internal/workdesk"
)

// tmuxRunner is the real tmux, for the agent view and for the actions that move you
// around.
type tmuxRunner struct{}

func (tmuxRunner) Run(args ...string) (string, error) { return tmux.Exec{}.Run(args...) }

// runOpen shows the UI, then carries out whatever it asked for and shows it again.
//
// The UI never acts: it records what the person chose and quits, so every action stays a
// plain function that runs with no terminal attached - which is what `workdesk act` uses
// and what the tests exercise.
func runOpen(args []string) error {
	view := workdesk.ParseView(first(args))
	trace.Log("workdesk", "open", "view", view.String())

	// Where the last pass was, so the next one opens there. The UI is rebuilt after every
	// action, and landing back at the top of the list is a poor answer to having clicked
	// a link forty lines into a description.
	at, offset := "", 0
	for {
		mirror, err := workdesk.Load(mirrorDir())
		if err != nil {
			return err
		}
		model := deskui.New(deskui.Deps{
			Mirror: mirror,
			Agents: func() []workdesk.Agent {
				agents, err := agentsNow()
				if err != nil {
					return nil
				}
				return agents
			},
			Now: time.Now,
		}, ui.ThemeByName(themeName()), view)
		model.Restore(at, offset)

		final, err := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
		if err != nil {
			return fmt.Errorf("ui: %w", err)
		}
		done, ok := final.(deskui.Model)
		if !ok || done.Pending == nil {
			return nil
		}
		view, at, offset = done.CurrentView(), done.CurrentRef(), done.PreviewOffset()

		switch done.Pending.Key {
		case "P":
			return promote(view)
		default:
			if err := act(done.Pending.Key, done.Pending.Ref, ""); err != nil {
				say(err.Error())
			}
		}
	}
}

// themeName is the flavor the rest of the terminal is wearing.
//
// Read from the file the theme switcher writes, which is the same one bash, hunk and
// leaf read - not from a tmux option, which nothing sets. An unreadable file falls back
// to the default flavor rather than failing: a wrong palette is a bad look, not a
// broken tool.
func themeName() string {
	if t := os.Getenv("WORKDESK_THEME"); t != "" {
		return t
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	b, err := os.ReadFile(filepath.Join(dir, "theme", "current"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func selfPath() string {
	if p, err := os.Executable(); err == nil {
		return p
	}
	return "workdesk"
}

// promote puts the current view in an ordinary pane. The float is always fresh but
// always transient; this is for a long triage sitting beside your code.
//
// tmux refuses to split a floating pane ("size or position can't split a floating
// pane"), so from inside one the split targets {last} - the pane you were in when you
// opened it, and the one whose directory you want.
func promote(v workdesk.View) error {
	args := []string{"split-window", "-h", "-c", "#{pane_current_path}"}
	if floating() {
		args = append(args, "-t", "{last}")
	}
	args = append(args, selfPath()+" open "+v.String())
	_, err := tmux.Exec{}.Run(args...)
	trace.Log("workdesk", "promote", "view", v.String(), "rc", rc(err))
	return err
}

// floating reports whether this program is running in a floating pane.
func floating() bool {
	out, err := tmux.Exec{}.Run("display-message", "-p", "#{pane_floating_flag}")
	return err == nil && strings.TrimSpace(out) == "1"
}

// say reports something with nowhere better to go. In the float there is no status line,
// so it prints and waits - the pane is the terminal here.
func say(msg string) {
	fmt.Println(msg)
	if isTTY() {
		fmt.Print("any key…")
		var b [1]byte
		_, _ = os.Stdin.Read(b[:])
		fmt.Println()
	}
}

func isTTY() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func rc(err error) int {
	if err != nil {
		return 1
	}
	return 0
}

// refKind splits a row reference into its kind and identifier, so an action needs no
// other context.
func refKind(ref string) (kind, id string) {
	k, i, found := strings.Cut(ref, ":")
	if !found {
		return "", ref
	}
	return k, i
}

func runPreview(args []string) error {
	ref := first(args)
	if ref == "" {
		return nil
	}
	kind, id := refKind(ref)
	switch kind {
	case "mrs":
		return catFile("mr/" + id + ".md")
	case "issues":
		return catFile("issue/" + id + ".md")
	case "agents":
		return agentPreview(id)
	default:
		return nil
	}
}

// agentPreview spells the join out: what the agent is doing, where it is writing, and
// which merge request came out of it. A missing merge request is the finding, not an
// error.
func agentPreview(pane string) error {
	agents, err := agentsNow()
	if err != nil {
		return err
	}
	idx, err := workdesk.LoadIndex(mirrorDir())
	if err != nil {
		return err
	}
	for _, a := range agents {
		if a.Pane != pane {
			continue
		}
		mr := ""
		for _, it := range idx.MRs {
			if it.Branch != "" && it.Branch == a.Branch {
				mr = it.Ref
			}
		}
		w := bufio.NewWriter(os.Stdout)
		defer w.Flush()
		if err := workdesk.AgentSheet(w, a, mr); err != nil {
			return err
		}
		if mr != "" {
			fmt.Fprint(w, "\n---\n\n")
			w.Flush()
			return catFile("mr/" + strings.TrimPrefix(mr, "!") + ".md")
		}
		return nil
	}
	fmt.Println("that pane is gone")
	return nil
}

// page shows markdown in whatever this box has. leaf already carries the flavour; bat and
// less are the fallbacks.
func page(text string) error {
	for _, try := range [][]string{
		{"leaf", "-"},
		{"bat", "--language=markdown", "--style=plain", "--paging=always"},
		{"less", "-R"},
	} {
		if _, err := exec.LookPath(try[0]); err != nil {
			continue
		}
		cmd := exec.Command(try[0], try[1:]...)
		cmd.Stdin = strings.NewReader(text)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	fmt.Print(text)
	return nil
}

func jump(pane string) error {
	r := tmux.Exec{}
	if _, err := r.Run("switch-client", "-t", pane); err == nil {
		trace.Log("workdesk", "jump", "pane", pane, "rc", 0)
		return nil
	}
	if _, err := r.Run("select-pane", "-t", pane); err != nil {
		return fmt.Errorf("cannot reach pane %s", pane)
	}
	trace.Log("workdesk", "jump", "pane", pane, "rc", 0)
	return nil
}

func runAct(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: workdesk act <key> <ref> [status]")
	}
	// The status a move goes to, named rather than picked, so `s` is one command for an
	// agent as well as a keypress for a person.
	return act(args[0], args[1], first(args[2:]))
}

// act is one keypress. Every action is an ordinary function reachable without a terminal,
// which is what lets the suite exercise them all.
func act(key, ref, choice string) error {
	if ref == "" {
		return nil
	}
	kind, id := refKind(ref)
	// A link clicked in the preview carries its target rather than a row: the thing it
	// points at is routinely not in the mirror at all.
	if kind == "url" {
		return open(id)
	}
	if kind == "agents" {
		if key == "d" {
			return diffFor(agentBranch(id))
		}
		return jump(id)
	}

	switch key {
	case "r":
		return runSync()
	case "m":
		var buf strings.Builder
		m, err := workdesk.Load(mirrorDir())
		if err != nil {
			return err
		}
		if err := workdesk.Matrix(&buf, m); err != nil {
			return err
		}
		return page(buf.String())
	}

	it, ok := findItem(kind, id)
	if !ok {
		return fmt.Errorf("%s is not in the mirror", ref)
	}
	switch key {
	case "o":
		return open(it.URL)
	case "y":
		return copyToClipboard(it.URL)
	case "c":
		return worktreeFor(it)
	case "d":
		return diffFor(it.Branch)
	case "a", "e", "M":
		if kind != "mrs" {
			return errors.New("that only applies to a merge request")
		}
		return write(key, id, strings.TrimSpace(it.Title))
	case "s", "i":
		if kind != "issues" {
			return errors.New("that only applies to an issue")
		}
		return move(key, id, choice)
	case "D":
		if kind != "mrs" {
			return errors.New("that only applies to a merge request")
		}
		return diffWindow(id)
	}
	return nil
}

// move is the status change and the sprint toggle: the two writes that act on an issue.
//
// The full snapshot rather than the index, because both need what a row does not carry -
// the issue's global ID, the lifecycle to choose from, and the sprint to move it to or
// from. It is one decode on a keypress that is about to make a network call.
func move(key, iid, choice string) error {
	m, err := workdesk.Load(mirrorDir())
	if err != nil {
		return err
	}
	var is *workdesk.Issue
	for i := range m.Issues {
		if m.Issues[i].IID == iid {
			is = &m.Issues[i]
		}
	}
	if is == nil {
		return fmt.Errorf("#%s is not in the mirror", iid)
	}
	title := strings.TrimSpace(is.Title)

	if key == "i" {
		sprint := m.Meta.Iteration
		if sprint == nil {
			return errors.New("this project has no current sprint")
		}
		// The row already says which way this goes, so the key is one key: in the
		// sprint means out of it, and out means in.
		id := sprint.ID
		if is.InSprint(sprint) {
			id = ""
		}
		return confirm(key, iid, gitlab.SetIteration(iid, title, is.ID, id, sprint.Label()))
	}

	to, ok := pickStatus(m.Meta.Statuses, is.StatusName(), choice)
	if !ok {
		return nil
	}
	return confirm(key, iid, gitlab.SetStatus(iid, title, is.ID, to.ID, to.Name))
}

// pickStatus asks which column to move to, listing the lifecycle in GitLab's own order
// and marking where the issue is now. The UI never acts, so this is the terminal it quit
// to - and the same list an agent gets by number from `workdesk act`.
func pickStatus(statuses []workdesk.Status, now, choice string) (workdesk.Status, bool) {
	if len(statuses) == 0 {
		fmt.Println("no statuses in the mirror - run 'workdesk sync'")
		return workdesk.Status{}, false
	}
	if choice != "" {
		for _, st := range statuses {
			if strings.EqualFold(st.Name, choice) {
				return st, true
			}
		}
		fmt.Printf("no status named %q in this project\n", choice)
		return workdesk.Status{}, false
	}
	for i, st := range statuses {
		here := ""
		if st.Name == now {
			here = "  ← now"
		}
		fmt.Printf("  %d  %s%s\n", i+1, st.Name, here)
	}
	n, err := strconv.Atoi(prompt("move to: "))
	if err != nil || n < 1 || n > len(statuses) {
		say("left alone")
		return workdesk.Status{}, false
	}
	return statuses[n-1], true
}

func findItem(kind, id string) (workdesk.Item, bool) {
	idx, err := workdesk.LoadIndex(mirrorDir())
	if err != nil {
		return workdesk.Item{}, false
	}
	want := "!" + id
	if kind == "issues" {
		want = "#" + id
	}
	for _, it := range idx.MRs {
		if it.Ref == want {
			return it.Item, true
		}
	}
	for _, it := range idx.Issues {
		if it.Ref == want {
			return it.Item, true
		}
	}
	return workdesk.Item{}, false
}

func agentBranch(pane string) string {
	agents, err := agentsNow()
	if err != nil {
		return ""
	}
	for _, a := range agents {
		if a.Pane == pane {
			return a.Branch
		}
	}
	return ""
}

// write is the three merge request calls that change GitLab.
func write(key, iid, title string) error {
	var w gitlab.Write
	switch key {
	case "a":
		reviewer := prompt("reviewer username: ")
		if reviewer == "" {
			return errors.New("no reviewer, nothing done")
		}
		w = gitlab.Assign(iid, title, reviewer)
	case "e":
		w = gitlab.AutoMerge(iid, title)
	case "M":
		w = gitlab.Merge(iid, title)
	}
	return confirm(key, iid, w)
}

// confirm is the gate in front of every call that changes GitLab: what it will do, the
// command that will do it, and a yes. WORKDESK_DRY stops before running, which is how the
// mockup stays harmless.
func confirm(key, iid string, w gitlab.Write) error {
	fmt.Printf("%s\n\n  %s\n\n", w.Label, w.Command())
	if os.Getenv("WORKDESK_DRY") != "" {
		say("WORKDESK_DRY is set - not run.")
		return nil
	}
	if prompt("run it? [y/N] ") != "y" {
		say("left alone")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), glabTimeout)
	defer cancel()
	out, err := gitlab.New().Do(ctx, w)
	trace.Log("workdesk", "write", "key", key, "ref", iid, "rc", rc(err))
	if err != nil {
		return err
	}
	say(out)
	return nil
}

func prompt(q string) string {
	if !isTTY() {
		return ""
	}
	fmt.Print(q)
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(sc.Text()))
}

// open reuses the tmux GitLab helper's own path, so the browser-versus-copy-over-ssh
// decision is made in exactly one place instead of twice.
func open(url string) error {
	if url == "" {
		return errors.New("no url for that row")
	}
	// via= is the point of this line: the helper used to answer 0 for a subcommand it
	// did not have, so `o` reported success and opened nothing.
	helper := os.ExpandEnv("$HOME/.local/bin/tmux-gitlab.sh")
	if _, err := os.Stat(helper); err == nil {
		if err := exec.Command(helper, "open-url", url).Run(); err == nil {
			trace.Log("workdesk", "open", "via", "gitlab-helper", "rc", 0)
			return nil
		}
	}
	if os.Getenv("SSH_CONNECTION") != "" {
		return copyToClipboard(url)
	}
	if _, err := exec.LookPath("xdg-open"); err == nil {
		if err := exec.Command("xdg-open", url).Start(); err == nil {
			trace.Log("workdesk", "open", "via", "xdg-open", "rc", 0)
			return nil
		}
	}
	return copyToClipboard(url)
}

// copyToClipboard goes through clip, so the wl-copy/xclip/pbcopy decision stays in one
// place for every copy path in this environment.
func copyToClipboard(url string) error {
	if url == "" {
		return nil
	}
	cmd := exec.Command(os.ExpandEnv("$HOME/.local/bin/clip"))
	cmd.Stdin = strings.NewReader(url)
	err := cmd.Run()
	trace.Log("workdesk", "copy", "rc", rc(err))
	if err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}
	say("copied " + url)
	return nil
}

// worktreeFor adds a worktree, never `glab mr checkout`: that switches the current
// worktree's branch, and with agents writing in these trees it can pull a branch out from
// under a running one.
func worktreeFor(it workdesk.Item) error {
	if it.Branch == "" {
		return errors.New("that row has no branch")
	}
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return errors.New("not in a git repo")
	}
	top := strings.TrimSpace(string(root))
	dest := strings.TrimSuffix(top, "/") + "-" + strings.TrimLeft(it.Ref, "!#")
	if _, err := os.Stat(dest); err == nil {
		say("already there: " + dest)
		return nil
	}
	cmd := exec.Command("git", "-C", top, "worktree", "add", dest, it.Branch)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not add it - try: gwta %s %s",
			strings.TrimLeft(it.Ref, "!#"), it.Branch)
	}
	trace.Log("workdesk", "worktree", "branch", it.Branch, "rc", 0)
	say("worktree ready: " + dest)
	return nil
}

// diffFor points the diff pane at the worktree that holds a branch.
//
// The pane's helper takes a DIRECTORY. It was being handed the branch, so it answered
// "<branch> is not a git repo" - and answered 0 while doing it, so the fallback below it
// never fired either and `d` on a merge request row did nothing and said nothing. A branch
// with no worktree now says so, and says which key does want it.
func diffFor(branch string) error {
	if branch == "" {
		return nil
	}
	helper := os.ExpandEnv("$HOME/.local/bin/tmux-diff-pane.sh")
	if _, err := os.Stat(helper); err != nil {
		return errors.New("no diff pane helper")
	}
	dir := worktreeOn(branch)
	if dir == "" {
		return fmt.Errorf("no worktree on %s - c adds one, D reads the diff", branch)
	}
	err := exec.Command(helper, "main", dir).Run()
	trace.Log("workdesk", "diff", "branch", branch, "dir", dir, "rc", rc(err))
	return err
}

// worktreeOn finds the checkout a branch is on, or empty when nothing holds it.
func worktreeOn(branch string) string {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return ""
	}
	return worktreeIn(string(out), branch)
}

// worktreeIn is the parse, split out so it can be tested without a repo. Porcelain lists
// a worktree's path first and its branch after, so the path is carried forward until the
// branch that matches names it; a detached worktree has no branch line at all.
func worktreeIn(porcelain, branch string) string {
	dir := ""
	for _, line := range strings.Split(porcelain, "\n") {
		if path, ok := strings.CutPrefix(line, "worktree "); ok {
			dir = path
			continue
		}
		if ref, ok := strings.CutPrefix(line, "branch "); ok {
			if strings.TrimPrefix(ref, "refs/heads/") == branch {
				return dir
			}
		}
	}
	return ""
}
