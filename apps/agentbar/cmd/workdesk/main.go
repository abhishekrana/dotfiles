// Command workdesk is a local, agent-readable view of the GitLab work you own.
//
// GitLab stays the source of truth. Everything here reads a full snapshot written by
// `workdesk sync`, so a view opens at file-read speed and never blocks on the network;
// only assign, auto and merge change anything on the server, and each asks first.
//
// The project comes from the git remote and the identity from glab's own token, so this
// program holds no host, group, project or username.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abhishekrana/agentbar/internal/gitlab"
	"github.com/abhishekrana/agentbar/internal/trace"
	"github.com/abhishekrana/agentbar/internal/workdesk"
)

const usage = `usage: workdesk [command]

With no command, opens the UI.

commands:
  open [view]        the UI: inbox, issues, mrs or agents (default inbox)
  sync               fetch and rewrite the mirror
  render             rebuild the derived views from the snapshot, no network
  list <view>        tab-separated rows, for a reader
  board              the whole queue, bucketed by what is blocking it
  mr <iid>           one merge request end to end
  issue <iid>        one issue
  matrix             one row per merge request, one column per gate, and totals
  preview <ref>      the preview for one row (mrs:412, issues:128, agents:%3)
  act <key> <ref>    run one action without a terminal
  ready              the rows asking something of you, one per line, for agents
  fixture <dir>      write the invented mirror the mockup and the tests share
  schema-check       validate the query against the live GitLab schema
  path               print the mirror directory

environment:
  WORKDESK_MIRROR    where the mirror lives
  WORKDESK_AGENTS    read agents from a file instead of tmux
  WORKDESK_DRY       print what a write would run, and stop
`

// glabTimeout bounds every forge call. A sync walks pages, so it is generous; without it
// a hung network would hang a popup.
const glabTimeout = 3 * time.Minute

func main() {
	// Bare `workdesk` opens the UI rather than printing usage: it is what you want
	// nine times in ten, and `--help` is there for the tenth.
	if len(os.Args) < 2 {
		if err := runOpen(nil); err != nil {
			fmt.Fprintln(os.Stderr, "workdesk:", err)
			os.Exit(1)
		}
		return
	}
	args := os.Args[2:]
	var err error
	switch os.Args[1] {
	case "open":
		err = runOpen(args)
	case "sync":
		err = runSync()
	case "render":
		err = runRender()
	case "list":
		err = runList(args)
	case "board":
		err = catFile("board.md")
	case "mr":
		err = runDoc("mr", args)
	case "issue":
		err = runDoc("issue", args)
	case "matrix":
		err = runMatrix()
	case "preview":
		err = runPreview(args)
	case "act":
		err = runAct(args)
	case "ready":
		err = runReady()
	case "fixture":
		err = runFixture(args)
	case "schema-check":
		err = runSchemaCheck()
	case "path":
		fmt.Println(mirrorDir())
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		if errors.Is(err, workdesk.ErrNoMirror) {
			fmt.Fprintln(os.Stderr, "workdesk: nothing synced yet - run 'workdesk sync'")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "workdesk:", err)
		os.Exit(1)
	}
}

// mirrorDir is overridable so the mockup and the tests can point at a fixture without
// touching the real mirror. It lives outside any repository because merge request bodies
// and comments can carry credentials.
func mirrorDir() string {
	if d := os.Getenv("WORKDESK_MIRROR"); d != "" {
		return d
	}
	state := os.Getenv("XDG_STATE_HOME")
	if state == "" {
		state = filepath.Join(os.Getenv("HOME"), ".local", "state")
	}
	return filepath.Join(state, "dotfiles", "workdesk")
}

func runSync() error {
	dir := mirrorDir()
	repo, err := os.Getwd()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), glabTimeout)
	defer cancel()

	client := gitlab.New()
	project, via, err := syncProject(ctx, client, dir, repo)
	if err != nil {
		return err
	}
	line := newProgressLine(project)
	res, err := workdesk.SyncWithProgress(ctx, client, dir, project, time.Now(), line.leg)
	line.close()
	if err != nil {
		trace.Log("workdesk", "sync", "project", project, "rc", 1, "err", trace.Err(err))
		return err
	}
	trace.Log("workdesk", "sync", "project", res.Project, "via", via, "user", res.User,
		"mrs", res.MRs, "issues", res.Issues, "todos", res.Todos,
		"fetched", res.MRsFetched+res.IssuesFetched,
		"ms", res.Took.Milliseconds(), "rc", 0)
	// The fetched count is the interesting number now: the rest of the queue was already
	// on disk and GitLab said it had not changed.
	fmt.Printf("synced %d mrs, %d issues, %d todos in %s (%d refreshed) -> %s\n",
		res.MRs, res.Issues, res.Todos, res.Took.Round(time.Millisecond),
		res.MRsFetched+res.IssuesFetched, dir)
	return nil
}

// syncProject is the project a resync refreshes.
//
// The working directory wins when it names a GitLab project, which is what lets a cd
// point the board at another one. It usually does not: the float inherits the cwd of the
// pane it was opened from and a shell is wherever you were, so r used to refuse from any
// repo that is not on GitLab. The mirror knows which project it holds, and refreshing
// what is on screen is what r means.
func syncProject(ctx context.Context, c *gitlab.Client, dir, repo string) (project, via string, err error) {
	project, err = gitlab.ProjectFor(ctx, c, repo)
	if err == nil {
		return project, "cwd", nil
	}
	if held := mirrorProject(dir); held != "" {
		return held, "mirror", nil
	}
	return "", "", err
}

// mirrorProject is the project the mirror on disk already holds, or "" if there is no
// mirror to read.
func mirrorProject(dir string) string {
	idx, err := workdesk.LoadIndex(dir)
	if err != nil {
		return ""
	}
	return idx.Project
}

func runRender() error {
	m, err := workdesk.Render(mirrorDir(), time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("rendered %d mrs, %d issues -> %s\n", len(m.MRs), len(m.Issues), mirrorDir())
	return nil
}

// rowsFor is the one place a view turns into rows, so `list` and the UI can never
// disagree about what a view contains.
func rowsFor(v workdesk.View) ([]workdesk.Row, error) {
	idx, err := workdesk.LoadIndex(mirrorDir())
	if err != nil {
		return nil, err
	}
	if v == workdesk.ViewAgents {
		agents, err := agentsNow()
		if err != nil {
			return nil, err
		}
		return workdesk.AgentRows(agents, idx), nil
	}
	return idx.Rows(v, time.Now()), nil
}

// agentsNow reads the live agent panes, or a file when one is named - which is how the
// mockup renders this view with no agent running.
func agentsNow() ([]workdesk.Agent, error) {
	if path := os.Getenv("WORKDESK_AGENTS"); path != "" {
		return workdesk.LoadAgents(path)
	}
	return workdesk.AgentsFromTmux(tmuxRunner{}, time.Now().Unix()), nil
}

func runList(args []string) error {
	view := workdesk.ParseView(first(args))
	rows, err := rowsFor(view)
	if err != nil {
		return err
	}
	return workdesk.WriteRows(os.Stdout, rows)
}

func runMatrix() error {
	m, err := workdesk.Load(mirrorDir())
	if err != nil {
		return err
	}
	return workdesk.Matrix(os.Stdout, m)
}

func runReady() error {
	idx, err := workdesk.LoadIndex(mirrorDir())
	if err != nil {
		return err
	}
	for _, it := range idx.MRs {
		if !it.Band.Active() {
			continue
		}
		note := it.Note
		if note == "" {
			note = "-"
		}
		fmt.Printf("%s\t%s\t%s\t%s\n", it.Label, it.Ref, note, strings.TrimSpace(it.Title))
	}
	return nil
}

func runDoc(kind string, args []string) error {
	id := strings.TrimLeft(first(args), "!#")
	if id == "" {
		return fmt.Errorf("usage: workdesk %s <iid>", kind)
	}
	return catFile(filepath.Join(kind, id+".md"))
}

func catFile(rel string) error {
	b, err := os.ReadFile(filepath.Join(mirrorDir(), rel))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s is not in the mirror (merged, closed, or not yours)", rel)
		}
		return err
	}
	_, err = os.Stdout.Write(b)
	return err
}

func runFixture(args []string) error {
	dir := first(args)
	if dir == "" {
		return errors.New("usage: workdesk fixture <dir>")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := workdesk.WriteFixture(dir, time.Now()); err != nil {
		return err
	}
	fmt.Printf("fixture mirror: %s\n", dir)
	return nil
}

func runSchemaCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), glabTimeout)
	defer cancel()
	if err := gitlab.New().SchemaCheck(ctx); err != nil {
		return fmt.Errorf("the query no longer matches the live schema: %w", err)
	}
	fmt.Println("every field validates against the live GitLab schema")
	return nil
}

func first(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return strings.TrimSpace(args[0])
}
