package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/abhishekrana/agentbar/internal/tmux"
	"github.com/abhishekrana/agentbar/internal/trace"
	"github.com/abhishekrana/agentbar/internal/workdesk"
)

// Reading a merge request means reading its diff, and neither a clone nor a worktree is
// needed for that: GitLab hands over the patch and hunk reads one on stdin. Measured on a
// real queue, the patch is about a second and answers for every merge request in it,
// where a clone answers only for the four projects this machine happens to have.
//
// The fetched form is the same diff with the file around it, for when the hunks are not
// enough. It costs a fetch, so it is a second key rather than the default.

// runDiff shows one merge request's diff, in this terminal. `act` puts it in a window of
// its own; a person can also run it directly.
func runDiff(args []string) error {
	iid := first(args)
	if iid == "" {
		return errors.New("usage: workdesk diff <iid> [--patch]")
	}
	idx, err := workdesk.LoadIndex(mirrorDir())
	if err != nil {
		return err
	}
	if idx.Project == "" {
		return errors.New("the mirror does not name a project - run 'workdesk sync'")
	}
	ctx, cancel := context.WithTimeout(context.Background(), glabTimeout)
	defer cancel()
	if len(args) > 1 && args[1] == "--patch" {
		return patchDiff(ctx, idx.Project, iid)
	}
	return fullDiff(ctx, idx.Project, iid)
}

// patchDiff reads what GitLab already has: no clone, no fetch, and the only form that can
// answer for a project this machine has never cloned.
func patchDiff(ctx context.Context, project, iid string) error {
	out, err := exec.CommandContext(ctx, "glab", "mr", "diff", iid,
		"-R", project, "--color=never").Output()
	if err != nil {
		return fmt.Errorf("glab mr diff %s: %w", iid, err)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return fmt.Errorf("!%s has no diff to show", iid)
	}
	return hunk(bytes.NewReader(out), "", "patch", "-")
}

// fullDiff fetches the merge request's head into a clone and shows the range with the
// files around it.
//
// The base is GitLab's own, never a local merge-base: origin/main in a working clone is
// routinely months behind, and computing the base against a stale one reported 8996 files
// changed for a two-file merge request. The head ref is fetched without writing a ref -
// FETCH_HEAD is enough to diff, and nothing is left behind in the clone.
func fullDiff(ctx context.Context, project, iid string) error {
	repo, err := cloneOf(project)
	if err != nil {
		return err
	}
	base, head, err := diffRefs(ctx, project, iid)
	if err != nil {
		return err
	}
	ref := fmt.Sprintf("refs/merge-requests/%s/head", iid)
	// The window is already up and the fetch is seconds; say what it is waiting on
	// rather than showing a blank pane until hunk starts.
	fmt.Fprintf(os.Stderr, "fetching !%s into %s…\n", iid, repo)
	fetch := exec.CommandContext(ctx, "git", "-C", repo, "fetch", "--quiet", "origin", ref)
	fetch.Stderr = os.Stderr
	if err := fetch.Run(); err != nil {
		return fmt.Errorf("fetch %s: %w", ref, err)
	}
	return hunk(nil, repo, "diff", base+"..."+head)
}

// cloneOf is a checkout of the project, taken from the working directory - the same rule
// `c` uses to add a worktree. A float inherits the directory of the pane it was opened
// from, so this is a checkout when you opened it in one and an error naming the reason
// when you did not.
func cloneOf(project string) (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("no clone of %s here - open the float in one, or run: workdesk diff <iid> --patch", project)
	}
	repo := strings.TrimSpace(string(out))
	remote, err := exec.Command("git", "-C", repo, "remote", "get-url", "origin").Output()
	if err != nil || !strings.Contains(string(remote), project) {
		return "", fmt.Errorf("%s is not a clone of %s - run: workdesk diff <iid> --patch", repo, project)
	}
	return repo, nil
}

// diffRefs is GitLab's own answer to what this merge request is measured against.
func diffRefs(ctx context.Context, project, iid string) (base, head string, err error) {
	path := fmt.Sprintf("projects/%s/merge_requests/%s",
		strings.ReplaceAll(project, "/", "%2F"), iid)
	out, err := exec.CommandContext(ctx, "glab", "api", path).Output()
	if err != nil {
		return "", "", fmt.Errorf("read !%s: %w", iid, err)
	}
	var mr struct {
		DiffRefs struct {
			BaseSHA string `json:"base_sha"`
			HeadSHA string `json:"head_sha"`
		} `json:"diff_refs"`
	}
	if err := json.Unmarshal(out, &mr); err != nil {
		return "", "", fmt.Errorf("decode !%s: %w", iid, err)
	}
	if mr.DiffRefs.BaseSHA == "" || mr.DiffRefs.HeadSHA == "" {
		return "", "", fmt.Errorf("!%s carries no diff refs", iid)
	}
	return mr.DiffRefs.BaseSHA, mr.DiffRefs.HeadSHA, nil
}

// hunk runs the viewer, in the flavour the rest of the terminal is wearing and side by
// side.
//
// Split is passed here rather than set in ~/.config/hunk/config.toml, because that file
// is where every other hunk in this environment starts from - the diff pane beside your
// work, the shell wrapper - and those want the full width per line that stack gives them.
// A merge request's diff is the one that gets a window of its own, so it is the one with
// room for two columns.
func hunk(in io.Reader, dir string, args ...string) error {
	bin := hunkBin()
	if bin == "" {
		return errors.New("hunk is not installed")
	}
	args = append(args, "--mode", "split")
	if t := themeName(); t != "" {
		args = append(args, "--theme", t)
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if in != nil {
		cmd.Stdin = in
	}
	return cmd.Run()
}

// hunkBin resolves the viewer the same way the diff pane's helper does.
func hunkBin() string {
	if p, err := exec.LookPath("hunk"); err == nil {
		return p
	}
	p := os.ExpandEnv("$HOME/.local/bin/hunk")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// diffWindow puts the diff in a window of its own: a diff wants the full width, and this
// leaves both the float and the agent's own diff pane where they were. Closing hunk closes
// the window and returns you to the float, still on the row you opened it from.
func diffWindow(iid string) error {
	cmd := selfPath() + " diff " + iid
	if os.Getenv("TMUX") == "" {
		// No server to open a window on - show it here instead of refusing.
		return runDiff(strings.Fields(cmd)[1:])
	}
	// Detached first, so the window can decline the sidebar before it is ever the
	// active one. The sidebar follows the session's active window, and a window it
	// has already moved into does not close when the diff exits - it is left holding
	// a full-width sidebar and the screen never comes back.
	var tm tmux.Exec
	out, err := tm.Run("new-window", "-d", "-P", "-F", "#{window_id}", "-n", "!"+iid, cmd)
	if err != nil {
		trace.Log("workdesk", "mrdiff", "mr", iid, "rc", rc(err))
		return err
	}
	win := strings.TrimSpace(out)
	if _, err = tm.Run("set-option", "-t", win, "-w", "@agentbar-skip", "1"); err != nil {
		return err
	}
	_, err = tm.Run("select-window", "-t", win)
	trace.Log("workdesk", "mrdiff", "mr", iid, "win", win, "rc", rc(err))
	return err
}
