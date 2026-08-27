package workdesk

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/abhishekrana/agentbar/internal/model"
	"github.com/abhishekrana/agentbar/internal/tmux"
)

// Agent is one Claude session, flattened to what a row needs.
//
// The fourth view exists for one case the forge cannot show: an agent that finished and
// opened no merge request. As far as GitLab is concerned nothing happened, so that work
// appears in no other list.
type Agent struct {
	State    string
	Title    string
	Worktree string
	Branch   string
	Pane     string
	AgeSecs  int64
}

// AgentsFromTmux reads the live agent panes.
//
// It wraps the runner so option writes are dropped: tmux.Snapshot marks a finished agent
// seen when you are looking at its pane, and opening an unrelated popup must not change
// what the sidebar shows.
func AgentsFromTmux(r tmux.Runner, nowUnix int64) []Agent {
	snap := tmux.Snapshot(readOnly{r}, tmux.NewBranchCache(), "")
	var out []Agent
	for _, sess := range snap.Sessions {
		for _, a := range sess.Agents {
			age := int64(0)
			if !a.Since.IsZero() {
				age = nowUnix - a.Since.Unix()
			}
			out = append(out, Agent{
				State:   string(a.State),
				Title:   a.Title,
				Branch:  a.Branch,
				Pane:    a.PaneID,
				AgeSecs: age,
			})
		}
	}
	return out
}

// readOnly passes tmux queries through and swallows writes, so a read cannot have a
// side effect on the sidebar's state.
type readOnly struct{ r tmux.Runner }

func (ro readOnly) Run(args ...string) (string, error) {
	if len(args) > 0 && strings.HasPrefix(args[0], "set-") {
		return "", nil
	}
	return ro.r.Run(args...)
}

// LoadAgents reads agents from a tab-separated file instead of tmux:
//
//	state  title  worktree  branch  pane  age-seconds
//
// The mockup and the tests both point at one, so neither needs a live agent to render
// this view.
func LoadAgents(path string) ([]Agent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read agents: %w", err)
	}
	defer f.Close()

	var out []Agent
	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		text := strings.TrimRight(sc.Text(), "\n")
		if text == "" {
			continue
		}
		f := strings.Split(text, "\t")
		if len(f) < 6 {
			return nil, fmt.Errorf("agents line %d has %d fields, want 6", line, len(f))
		}
		age, err := strconv.ParseInt(f[5], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("agents line %d: age %q: %w", line, f[5], err)
		}
		out = append(out, Agent{
			State: f[0], Title: f[1], Worktree: f[2], Branch: f[3], Pane: f[4], AgeSecs: age,
		})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read agents: %w", err)
	}
	return out, nil
}

// agentBand groups agents the way the forge bands group merge requests: by whether they
// want something from you.
type agentBand struct {
	key    int
	label  string
	active bool
}

// agentBandFor classifies an agent. The second band is the one worth having this view
// for at all - work that finished without reaching GitLab.
func agentBandFor(a Agent, hasMR bool) agentBand {
	switch {
	case model.AgentState(a.State).NeedsAttention():
		return agentBand{0, "waiting on you", true}
	case a.State == string(model.StateDone) && !hasMR:
		return agentBand{1, "finished · no merge request", true}
	case a.State == string(model.StateWorking):
		return agentBand{2, "working", false}
	case a.State == string(model.StateDone):
		return agentBand{3, "finished", false}
	default:
		return agentBand{4, "idle", false}
	}
}

// agentAge is coarser than the forge's: an agent's interesting timescale is minutes,
// not days.
func agentAge(secs int64) string {
	switch {
	case secs < 3600:
		return strconv.FormatInt(secs/60, 10) + "m"
	case secs < 86400:
		return strconv.FormatInt(secs/3600, 10) + "h"
	default:
		return strconv.FormatInt(secs/86400, 10) + "d"
	}
}

// AgentRows joins the agents to the work they produced.
//
// The join key is the worktree's branch, which the hook already stamps on every pane, so
// an agent row can name its merge request and a merge request row can name its agent at
// no extra cost.
func AgentRows(agents []Agent, idx *Index) []Row {
	byBranch := make(map[string]string, len(idx.MRs))
	for _, it := range idx.MRs {
		if it.Branch != "" {
			byBranch[it.Branch] = it.Ref
		}
	}

	type entry struct {
		band agentBand
		secs int64
		row  Row
	}
	entries := make([]entry, 0, len(agents))
	for _, a := range agents {
		ref, hasMR := byBranch[a.Branch]
		note := ref
		if !hasMR {
			note = "no merge request"
		}
		band := agentBandFor(a, hasMR)
		flag := "i"
		if band.active {
			flag = "a"
		}
		entries = append(entries, entry{
			band: band,
			secs: a.AgeSecs,
			row: Row{
				Label:  band.label,
				Flag:   flag,
				Ref:    a.Pane,
				Title:  a.Title,
				Age:    agentAge(a.AgeSecs),
				Note:   note,
				Branch: a.Branch,
			},
		})
	}
	// Band first, then the most recent within a band: an agent that stopped a minute
	// ago is the one you were just talking to.
	slices.SortStableFunc(entries, func(x, y entry) int {
		if x.band.key != y.band.key {
			return x.band.key - y.band.key
		}
		return int(x.secs - y.secs)
	})

	out := make([]Row, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.row)
	}
	return out
}

// AgentSheet is the preview for one agent: what it is doing, where it is writing, and
// whether that work reached GitLab at all.
func AgentSheet(w io.Writer, a Agent, mr string) error {
	d := &doc{}
	title := a.Title
	if title == "" {
		title = "untitled"
	}
	d.addf("# %s", title)
	d.blank()
	d.addf("state     %s", a.State)
	d.addf("pane      %s", a.Pane)
	d.addf("worktree  %s", dash(a.Worktree))
	d.addf("branch    %s", dash(a.Branch))
	d.addf("age       %s", agentAge(a.AgeSecs))
	if mr == "" {
		d.add("merge req none opened")
		d.blank()
		d.add("No merge request for this branch. Nothing in GitLab knows this work",
			"exists, so it appears in no other view.")
	} else {
		d.addf("merge req %s", mr)
	}
	d.blank()
	d.add("Enter switches to this pane.")
	return d.writeTo(w)
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
