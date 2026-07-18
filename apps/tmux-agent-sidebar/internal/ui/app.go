package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/abhishekrana/tmux-agent-sidebar/internal/model"
	"github.com/abhishekrana/tmux-agent-sidebar/internal/tmux"
)

const (
	spinnerInterval  = 200 * time.Millisecond
	snapshotInterval = time.Second

	// The terminal sends no "pointer left" event, so hover expires after a
	// few idle spinner frames (no motion): ~600ms after the pointer stops.
	hoverIdleFrames = 3

	// wait-for channel signalled by jumps; sidebars adopt the shared
	// selection immediately instead of on the next tick.
	refreshChannel = "tmux-agent-sidebar-refresh"
)

type tickMsg time.Time

type snapMsg struct {
	snap   model.Snapshot
	sel    string // global @sidebar_selected at snapshot time
	notify bool   // global @agent_notify at snapshot time
	signal bool   // woken by the wait-for channel, not the 1s tick
}

// App is the Bubble Tea model for the sidebar. In mockup mode the
// snapshot is static fake data and Enter just flashes what it would do.
type App struct {
	theme      Theme
	snap       model.Snapshot
	blocks     []block
	cursor     int // index into blocks; kept on a selectable block when possible
	hover      int // block index under the mouse pointer, -1 when none
	hoverFrame int // frame at the last motion event; hover expires after some idle
	frame      int
	width      int
	height     int
	flash      string
	mockup     bool
	notify     bool // desktop-notification toggle (@agent_notify), mirrored for the footer

	// live-mode plumbing (nil in mockup mode)
	runner   tmux.Runner
	branches *tmux.BranchCache
	current  string // session the sidebar pane lives in
	debug    string // log file path (@agent-sidebar-debug), "" = off
	lastSel  string // last @sidebar_selected value we adopted
	attached string // attachedKey of the last snapshot
}

// NewLive builds the sidebar against the real tmux server.
func NewLive(theme Theme) App {
	runner := tmux.Exec{}
	debug, _ := runner.Run("show-option", "-gqv", "@agent-sidebar-debug")
	app := App{
		theme:    theme,
		hover:    -1,
		runner:   runner,
		branches: tmux.NewBranchCache(),
		current:  tmux.CurrentSession(runner),
		debug:    strings.TrimSpace(debug),
	}
	app.setSnapshot(tmux.Snapshot(runner, app.branches, app.current))
	app.attached = attachedKey(app.snap)
	// Selection is shared across sidebars via the global @sidebar_selected.
	if sel, err := runner.Run("show-option", "-gqv", "@sidebar_selected"); err == nil {
		app.lastSel = strings.TrimSpace(sel)
		app.adoptSelection(app.lastSel)
	}
	if v, err := runner.Run("show-option", "-gqv", "@agent_notify"); err == nil {
		app.notify = strings.TrimSpace(v) == "on"
	}
	app.register()
	return app
}

// attachedKey fingerprints the sessions that have a client attached AND
// at least one agent. Ignoring agent-less sessions keeps the transition
// pending: switching to a session before its agent starts changes nothing,
// so the agent still gets the highlight when it appears.
func attachedKey(snap model.Snapshot) string {
	var names []string
	for _, s := range snap.Sessions {
		if s.Attached && len(s.Agents) > 0 {
			names = append(names, s.Name)
		}
	}
	return strings.Join(names, ",")
}

// scriptPath locates one of the plugin's shell scripts relative to the
// running binary (bin/tmux-agent-sidebar -> scripts/<name>).
func scriptPath(name string) string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(filepath.Dir(exe)), "scripts", name)
}

// register stamps this sidebar's session options and follow hook, so a
// sidebar started outside open.sh (tmux-resurrect restore) works fully.
func (a App) register() {
	pane := os.Getenv("TMUX_PANE")
	follow := scriptPath("follow.sh")
	if pane == "" || a.current == "" || follow == "" {
		return
	}
	_, _ = a.runner.Run(
		"set-option", "-t", a.current, "-q", "@sidebar_pane", pane, ";",
		"set-option", "-t", a.current, "-q", "@sidebar_on", "1", ";",
		"set-hook", "-t", a.current, "session-window-changed",
		"run-shell '"+follow+" #{session_name}'", ";",
		// Session switches wake every sidebar so the highlight follows.
		"set-hook", "-g", "client-session-changed",
		"run-shell 'tmux wait-for -S "+refreshChannel+"'",
	)
}

// debugf appends a timestamped line to the debug log when enabled via
// `tmux set -g @agent-sidebar-debug /path/to/log`.
func (a App) debugf(format string, args ...any) {
	if a.debug == "" {
		return
	}
	f, err := os.OpenFile(a.debug, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, time.Now().Format("15:04:05.000 ")+format+"\n", args...)
}

// setSnapshot swaps in fresh data, keeping the selection anchored across
// refreshes by pane (agent rows) or session name (session headers).
func (a *App) setSnapshot(snap model.Snapshot) {
	var anchorPane, anchorSess string
	if a.blockSelectable(a.cursor) {
		b := a.blocks[a.cursor]
		switch b.kind {
		case blockAgent:
			anchorPane = a.snap.Sessions[b.session].Agents[b.agent].PaneID
		case blockSession:
			anchorSess = a.snap.Sessions[b.session].Name
		}
	}
	a.snap = snap
	a.rebuild()
	if a.hover >= len(a.blocks) {
		a.hover = -1 // pointer target no longer exists
	}
	switch {
	case anchorPane != "":
		for i, b := range a.blocks {
			if b.kind == blockAgent && snap.Sessions[b.session].Agents[b.agent].PaneID == anchorPane {
				a.cursor = i
				return
			}
		}
	case anchorSess != "":
		for i, b := range a.blocks {
			if b.kind == blockSession && snap.Sessions[b.session].Name == anchorSess {
				a.cursor = i
				return
			}
		}
	}
}

func (a App) snapshotTick() tea.Cmd {
	return tea.Tick(snapshotInterval, func(time.Time) tea.Msg {
		return a.gather(false)
	})
}

// waitRefresh blocks until a jump or session switch signals the channel.
// The blocked wait-for child dies with the pane.
func (a App) waitRefresh() tea.Cmd {
	return func() tea.Msg {
		if _, err := a.runner.Run("wait-for", refreshChannel); err != nil {
			return nil // degrade to tick-based adoption
		}
		return a.gather(true)
	}
}

// gather takes a fresh snapshot plus the shared selection and notify state.
func (a App) gather(signal bool) snapMsg {
	sel, _ := a.runner.Run("show-option", "-gqv", "@sidebar_selected")
	notify, _ := a.runner.Run("show-option", "-gqv", "@agent_notify")
	return snapMsg{
		snap:   tmux.Snapshot(a.runner, a.branches, a.current),
		sel:    strings.TrimSpace(sel),
		notify: strings.TrimSpace(notify) == "on",
		signal: signal,
	}
}

// focusNewlyAttached selects the agent of a session newly in the
// attached-with-agents set: a session switch made outside the sidebar,
// or an agent starting in the attached session right after one.
func (a *App) focusNewlyAttached() {
	old := map[string]bool{}
	for n := range strings.SplitSeq(a.attached, ",") {
		old[n] = true
	}
	for _, s := range a.snap.Sessions {
		if !s.Attached || old[s.Name] || len(s.Agents) == 0 {
			continue
		}
		pane := s.Agents[0].PaneID
		for _, ag := range s.Agents {
			if ag.Focused {
				pane = ag.PaneID
				break
			}
		}
		a.selectPane(pane)
		return
	}
}

// adoptSelection moves the cursor to the shared selection: a session row
// (token "=name", published by a session click) or an agent's pane.
func (a *App) adoptSelection(sel string) {
	if name, ok := strings.CutPrefix(sel, "="); ok {
		a.selectSession(name)
	} else {
		a.selectPane(sel)
	}
}

// selectSession moves the cursor to the named session's header, if listed.
func (a *App) selectSession(name string) {
	if name == "" {
		return
	}
	for i, b := range a.blocks {
		if b.kind == blockSession && a.snap.Sessions[b.session].Name == name {
			a.cursor = i
			return
		}
	}
}

// selectPane moves the cursor to the block owning pane, if it's listed.
func (a *App) selectPane(pane string) {
	if pane == "" {
		return
	}
	for i, b := range a.blocks {
		if b.kind == blockAgent && a.snap.Sessions[b.session].Agents[b.agent].PaneID == pane {
			a.cursor = i
			return
		}
	}
}

// NewMockup builds the sidebar with representative fake data so the
// layout and palette can be previewed in any pane.
func NewMockup(theme Theme) App {
	now := time.Now()
	// Sessions in alphabetical order, as the real snapshot delivers them.
	snap := model.Snapshot{Sessions: []model.Session{
		// One workspace = one checked-out branch, so a session's Claudes all
		// share it (here a single Claude, working, with two subagents).
		{Name: "api-server", Current: true, Agents: []model.Agent{
			{PaneID: "%1", WindowIndex: 1, Command: "claude", Branch: "feat/rate-limit-middleware-rollout",
				State: model.StateWorking, Since: now.Add(-2 * time.Minute), Subagents: 2},
		}},
		{Name: "blog", Agents: []model.Agent{
			{PaneID: "%7", WindowIndex: 1, Command: "claude", Branch: "draft/tmux-agents-post",
				State: model.StateDone, Since: now.Add(-12 * time.Minute)},
		}},
		{Name: "dotfiles", Agents: []model.Agent{
			{PaneID: "%5", WindowIndex: 1, Command: "claude", Branch: "main",
				State: model.StateQuestion, Since: now.Add(-4 * time.Minute)},
		}},
		// Three Claudes on one branch: the branch shows once, colored by the
		// most-urgent of them (here the one waiting on a permission).
		{Name: "payments", Agents: []model.Agent{
			{PaneID: "%9", WindowIndex: 1, Command: "claude", Branch: "2091-refund-idempotency-keys",
				State: model.StateWorking, Since: now.Add(-6 * time.Minute)},
			{PaneID: "%10", WindowIndex: 2, Command: "claude", Branch: "2091-refund-idempotency-keys",
				State: model.StatePermission, Since: now.Add(-30 * time.Second)},
			{PaneID: "%11", WindowIndex: 3, Command: "claude", Branch: "2091-refund-idempotency-keys",
				State: model.StateDone, Since: now.Add(-11 * time.Minute)},
		}},
		{Name: "scratch"},
		{Name: "www", Agents: []model.Agent{
			{PaneID: "%8", WindowIndex: 1, Command: "claude", Branch: "main",
				State: model.StateDone, Seen: true, Since: now.Add(-33 * time.Minute)},
		}},
	}}
	app := App{
		theme:  theme,
		hover:  -1,
		snap:   snap,
		mockup: true,
	}
	app.rebuild()
	return app
}

func (a *App) rebuild() {
	a.blocks = buildBlocks(a.snap)
	if a.cursor >= 0 && a.cursor < len(a.blocks) && a.blocks[a.cursor].kind == blockAgent {
		return // keep an explicit agent selection; setSnapshot re-anchors headers
	}
	// Default to the first agent; a session-only view lands on the first row.
	for i, b := range a.blocks {
		if b.kind == blockAgent {
			a.cursor = i
			return
		}
	}
	a.cursor = 0
}

func (a App) blockSelectable(i int) bool {
	return i >= 0 && i < len(a.blocks) && a.blocks[i].selectable()
}

// moveCursor advances to the next selectable block in direction dir,
// starting from the current position (or the top when nothing selected).
func (a *App) moveCursor(dir int) {
	if len(a.blocks) == 0 {
		a.cursor = 0
		return
	}
	i := a.cursor
	for step := 0; step < len(a.blocks); step++ {
		i += dir
		if i < 0 || i >= len(a.blocks) {
			return // stay put at the edge
		}
		if a.blocks[i].selectable() {
			a.cursor = i
			return
		}
	}
}

// nextAttention returns the block index of the next agent blocked on the
// user (permission/asking), scanning forward from `from` and wrapping. It
// is the sidebar's work queue: one key steps through everyone waiting on
// you, across sessions. Returns -1 when nobody needs attention.
func (a App) nextAttention(from int) int {
	n := len(a.blocks)
	for step := 1; step <= n; step++ {
		i := (from + step) % n
		b := a.blocks[i]
		if b.kind == blockAgent && a.snap.Sessions[b.session].Agents[b.agent].State.NeedsAttention() {
			return i
		}
	}
	return -1
}

func (a App) Init() tea.Cmd {
	if a.mockup {
		return tick()
	}
	return tea.Batch(tick(), a.snapshotTick(), a.waitRefresh())
}

func tick() tea.Cmd {
	return tea.Tick(spinnerInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		a.frame++
		if a.hover >= 0 && a.frame-a.hoverFrame >= hoverIdleFrames {
			a.hover = -1 // pointer stopped moving (left the pane or came to rest)
		}
		return a, tick()
	case snapMsg:
		a.setSnapshot(msg.snap)
		a.notify = msg.notify
		key := attachedKey(a.snap)
		switch {
		case msg.sel != a.lastSel: // explicit jump wins
			a.lastSel = msg.sel
			a.adoptSelection(msg.sel)
		case key != a.attached: // session switch: follow to its agent
			a.focusNewlyAttached()
		}
		a.attached = key
		if msg.signal {
			return a, a.waitRefresh()
		}
		return a, a.snapshotTick()
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil
	case tea.KeyMsg:
		return a.handleKey(msg)
	case tea.MouseMsg:
		return a.handleMouse(msg)
	}
	return a, nil
}

// layout describes the body area in display lines: which block owns each
// line and the scroll window that keeps the cursor block fully visible.
type layout struct {
	owners []int // line index -> block index
	start  int   // first visible line
	avail  int   // visible body lines
}

func (a App) layout() layout {
	l := layout{avail: a.height - 5} // 2 header + 3 footer lines are fixed
	if a.flash != "" {
		l.avail--
	}
	if l.avail < 1 {
		l.avail = 1
	}
	firsts := make([]int, len(a.blocks))
	for i, b := range a.blocks {
		firsts[i] = len(l.owners)
		for n := blockLineCount(b, a.snap); n > 0; n-- {
			l.owners = append(l.owners, i)
		}
	}
	if a.blockSelectable(a.cursor) {
		first := firsts[a.cursor]
		last := first + blockLineCount(a.blocks[a.cursor], a.snap) - 1
		if last >= l.start+l.avail {
			l.start = last - l.avail + 1
		}
		if first < l.start {
			l.start = first
		}
	}
	if l.start+l.avail > len(l.owners) {
		l.start = max(0, len(l.owners)-l.avail)
	}
	return l
}

func (a App) handleMouse(m tea.MouseMsg) (tea.Model, tea.Cmd) {
	a.debugf("mouse action=%v button=%v x=%d y=%d cursor=%d", m.Action, m.Button, m.X, m.Y, a.cursor)
	switch {
	// Track the pointer so the row under it lights (any-motion tracking).
	case m.Action == tea.MouseActionMotion:
		a.hover = a.blockAt(m.Y)
		a.hoverFrame = a.frame
	case m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonWheelUp:
		a.moveCursor(-1)
	case m.Action == tea.MouseActionPress && m.Button == tea.MouseButtonWheelDown:
		a.moveCursor(1)
	// Jump on release, not press: terminals eat the press of a click
	// that also focuses their window, but always deliver the release.
	case m.Action == tea.MouseActionRelease && m.Button == tea.MouseButtonLeft:
		if a.onNotifyChip(m.X, m.Y) {
			return a.toggleNotify()
		}
		if b := a.blockAt(m.Y); b >= 0 {
			a.cursor = b
			return a.activate()
		}
	}
	return a, nil
}

// blockAt maps a screen row (0-based, incl. the 2 header lines) to the
// selectable block under it, or -1 if it isn't over one.
func (a App) blockAt(y int) int {
	l := a.layout()
	idx := l.start + y - 2 // 2 header lines above the body
	if y >= 2 && y < 2+l.avail && idx >= 0 && idx < len(l.owners) && a.blockSelectable(l.owners[idx]) {
		return l.owners[idx]
	}
	return -1
}

// activate acts on the row under the cursor (Enter or click): a session
// header switches sessions, an agent block jumps to its pane.
func (a App) activate() (tea.Model, tea.Cmd) {
	if !a.blockSelectable(a.cursor) {
		return a, nil
	}
	b := a.blocks[a.cursor]
	sess := a.snap.Sessions[b.session]
	if b.kind == blockSession {
		return a.activateSession(sess)
	}
	ag := sess.Agents[b.agent]
	if a.mockup {
		a.flash = "would jump to " + ag.PaneID
		return a, nil
	}
	// Address the client explicitly: with several clients attached,
	// tmux's "current client" guess can switch the wrong one.
	args := []string{"switch-client"}
	if tty := tmux.ClientFor(a.runner, a.current); tty != "" {
		args = append(args, "-c", tty)
	}
	args = append(args,
		"-t", sess.Name, ";",
		"select-window", "-t", fmt.Sprintf("%s:%d", sess.Name, ag.WindowIndex), ";",
		"select-pane", "-t", ag.PaneID, ";",
		// Publish + signal so every sidebar highlights it immediately.
		"set-option", "-g", "@sidebar_selected", ag.PaneID, ";",
		"wait-for", "-S", refreshChannel,
	)
	a.lastSel = ag.PaneID
	_, err := a.runner.Run(args...)
	a.debugf("jump session=%s window=%d pane=%s args=%v err=%v", sess.Name, ag.WindowIndex, ag.PaneID, args, err)
	if err != nil {
		a.flash = "jump failed: " + err.Error()
	}
	return a, nil
}

// activateSession switches the client to a session (Enter or a click on
// the session name). Unlike an agent jump it leaves the target's window
// and pane selection alone, so it also reaches agent-less sessions; the
// client-session-changed hook then moves every sidebar's highlight.
func (a App) activateSession(sess model.Session) (tea.Model, tea.Cmd) {
	if a.mockup {
		a.flash = "would switch to " + sess.Name
		return a, nil
	}
	if sess.Current {
		return a, nil // already here
	}
	args := []string{"switch-client"}
	if tty := tmux.ClientFor(a.runner, a.current); tty != "" {
		args = append(args, "-c", tty)
	}
	args = append(args,
		"-t", sess.Name, ";",
		// Publish the session (token "=name") + signal so every sidebar
		// highlights this session's row immediately, not next tick.
		"set-option", "-g", "@sidebar_selected", "="+sess.Name, ";",
		"wait-for", "-S", refreshChannel,
	)
	a.lastSel = "=" + sess.Name
	_, err := a.runner.Run(args...)
	a.debugf("switch session=%s args=%v err=%v", sess.Name, args, err)
	if err != nil {
		a.flash = "switch failed: " + err.Error()
	}
	return a, nil
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if !a.mockup {
			// The toggle is global: q hides the sidebar everywhere,
			// same as prefix+e. The script also kills this pane.
			if script := scriptPath("toggle.sh"); script != "" {
				_ = exec.Command("bash", script).Start()
			}
		}
		return a, tea.Quit
	case "j", "down":
		a.moveCursor(1)
	case "k", "up":
		a.moveCursor(-1)
	case "g", "home":
		a.cursor = -1
		a.moveCursor(1)
	case "G", "end":
		a.cursor = len(a.blocks)
		a.moveCursor(-1)
	case "enter", " ":
		return a.activate()
	case "tab":
		// Work queue: jump straight to the next agent waiting on you,
		// cycling through them across sessions. No-op when all quiet.
		if i := a.nextAttention(a.cursor); i >= 0 {
			a.cursor = i
			return a.activate()
		}
	case "n":
		return a.toggleNotify()
	}
	return a, nil
}

// toggleNotify flips the global desktop-notification switch (@agent_notify),
// which the hook reads. The `n` key and a click on the footer chip both route
// here; in mockup mode it just flips the local preview.
func (a App) toggleNotify() (tea.Model, tea.Cmd) {
	a.notify = !a.notify
	if !a.mockup {
		val := "off"
		if a.notify {
			val = "on"
		}
		_, _ = a.runner.Run("set-option", "-g", "@agent_notify", val)
	}
	return a, nil
}

// onNotifyChip reports whether (x,y) landed on the footer's notify chip: the
// status line is the second-from-last row, and the chip sits on its right.
func (a App) onNotifyChip(x, y int) bool {
	return a.height > 1 && y == a.height-2 && x >= a.width/2
}

func (a App) View() string {
	if a.width == 0 {
		return ""
	}
	now := time.Now()
	// Clamp the name column so icon+name+label+time always fit the width.
	nameW := max(6, min(nameColumn(a.snap), a.width-23))
	r := renderer{theme: a.theme, width: a.width, nameW: nameW}

	var b strings.Builder
	b.WriteString(r.header(a.snap, a.frame) + "\n")
	b.WriteString(r.sep() + "\n")

	var body []string
	for i, blk := range a.blocks {
		sess := a.snap.Sessions[blk.session]
		// lit fills the row (hover or selection); bar is the selection's edge.
		lit, bar := i == a.cursor || i == a.hover, i == a.cursor
		switch blk.kind {
		case blockSession:
			body = append(body, r.sessionBlock(sess, lit, bar)...)
		case blockAgent:
			body = append(body, r.agentBlock(sess, blk.agent, lit, bar, a.frame, now)...)
		}
	}

	l := a.layout()
	for i := l.start; i < len(body) && i < l.start+l.avail; i++ {
		b.WriteString(body[i] + "\n")
	}
	for i := len(body); i < l.avail; i++ {
		b.WriteString("\n")
	}

	if a.flash != "" {
		b.WriteString(" " + a.flash + "\n")
	}
	b.WriteString(r.footer(a.snap, a.notify))
	return b.String()
}
