// Package deskui is the workdesk terminal UI.
//
// Bubble Tea rather than fzf, and the difference is structural rather than cosmetic.
// fzf re-invoked a process for every cursor movement, so a preview had to be something
// cheap enough to rebuild per keystroke, and band headers had to be smuggled into the
// row list as fake items the cursor was taught to step over. Here the model is loaded
// once and held: headers are derived at render time so the cursor is always on a real
// row, previews are built from the snapshot in memory, and the preview scrolls.
//
// The palette comes from internal/ui, generated from design/palette.toml, so this looks
// like the rest of the terminal rather than approximately like it.
package deskui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/abhishekrana/agentbar/internal/ui"
	"github.com/abhishekrana/agentbar/internal/workdesk"
)

// Action is something the model wants done outside itself - opening a browser, adding a
// worktree, writing to GitLab. Returned rather than performed: the model stays pure, and
// the caller owns every side effect and every confirm.
type Action struct {
	Key string // the binding that asked for it: o, y, c, d, a, e, M, s, i, P, r
	Ref string // kind:id, the same handle `workdesk act` takes
}

// Deps is what the model cannot do for itself.
type Deps struct {
	// Mirror is the snapshot every view is derived from.
	Mirror *workdesk.Mirror
	// Agents is re-read on demand, because agent state changes while the UI is open.
	Agents func() []workdesk.Agent
	// Now is injected so a render is a pure function of its inputs.
	Now func() time.Time
}

// Model is the whole UI.
type Model struct {
	deps  Deps
	theme ui.Theme
	keys  keyMap
	help  help.Model

	// idx is built once. Classification is the same work for every view, and reload
	// runs on each filter keystroke - rebuilding it there would cost a millisecond per
	// character typed for no reason.
	idx     *workdesk.Index
	view    workdesk.View
	rows    []workdesk.Row
	cursor  int
	preview viewport.Model
	filter  textinput.Model

	width, height int
	showHelp      bool
	filtering     bool
	notice        string

	// Pending is the action the caller should carry out once Run returns. Nil when the
	// person simply closed the UI.
	Pending *Action
}

// New builds the model. The mirror is decoded by the caller and handed over whole: at a
// few milliseconds once, that buys previews with no further I/O.
func New(deps Deps, theme ui.Theme, view workdesk.View) Model {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	fi := textinput.New()
	fi.Prompt = "/"
	fi.CharLimit = 60

	// The separator is a string on the model, not text carried by the style: setting it
	// on the style appends instead of replacing, which renders both.
	h := help.New()
	h.ShortSeparator = sep
	h.FullSeparator = sep
	h.Styles.ShortSeparator = h.Styles.ShortSeparator.Foreground(theme.Muted)
	h.Styles.FullSeparator = h.Styles.FullSeparator.Foreground(theme.Muted)
	h.Styles.ShortKey = h.Styles.ShortKey.Foreground(theme.Accent)
	h.Styles.FullKey = h.Styles.FullKey.Foreground(theme.Accent)
	h.Styles.ShortDesc = h.Styles.ShortDesc.Foreground(theme.Muted)
	h.Styles.FullDesc = h.Styles.FullDesc.Foreground(theme.Muted)

	m := Model{
		deps:    deps,
		idx:     workdesk.BuildIndex(deps.Mirror),
		theme:   theme,
		keys:    defaultKeys(),
		help:    h,
		view:    view,
		preview: viewport.New(0, 0),
		filter:  fi,
	}
	m.reload()
	return m
}

func (m Model) Init() tea.Cmd { return nil }

// CurrentView is which view was showing when the UI quit, so reopening after an action
// lands you back where you were. Not named View: that is Bubble Tea's renderer.
func (m Model) CurrentView() workdesk.View { return m.view }

// reload rebuilds the current view's rows. Called on a view switch, a filter change and
// after a sync - never during a render, so View stays free of side effects.
func (m *Model) reload() {
	now := m.deps.Now()
	if m.view == workdesk.ViewAgents {
		var agents []workdesk.Agent
		if m.deps.Agents != nil {
			agents = m.deps.Agents()
		}
		m.rows = workdesk.AgentRows(agents, m.idx)
	} else {
		m.rows = m.idx.Rows(m.view, now)
	}
	if q := strings.TrimSpace(m.filter.Value()); q != "" {
		m.rows = matching(m.rows, q)
	}
	m.clampCursor()
	m.syncPreview()
}

// matching is a plain case-insensitive substring filter over the fields a person can
// see. Deliberately not fuzzy: on a list this size a fuzzy match mostly produces
// surprising ordering, and the reference and the title are what anyone types.
func matching(rows []workdesk.Row, q string) []workdesk.Row {
	q = strings.ToLower(q)
	out := make([]workdesk.Row, 0, len(rows))
	for _, r := range rows {
		hay := strings.ToLower(r.Ref + " " + r.Title + " " + r.Label + " " + r.Note + " " + r.Branch)
		if strings.Contains(hay, q) {
			out = append(out, r)
		}
	}
	return out
}

func (m *Model) clampCursor() {
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// current is the row under the cursor, if there is one. Every band header is derived at
// render time, so unlike the fzf version the cursor can never be sitting on one.
func (m Model) current() (workdesk.Row, bool) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return workdesk.Row{}, false
	}
	return m.rows[m.cursor], true
}

func (m *Model) syncPreview() {
	row, ok := m.current()
	if !ok {
		m.preview.SetContent("")
		return
	}
	m.preview.SetContent(m.renderPreview(row))
	m.preview.GotoTop()
}

func (m *Model) resize(w, h int) {
	m.width, m.height = w, h
	m.help.Width = w
	_, pw := paneWidths(w)
	m.preview.Width = previewWidth(pw)
	m.preview.Height = bodyHeight(h)
	m.syncPreview()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
	}
	return m, nil
}

// wheelLines is how far one notch scrolls the preview.
const wheelLines = 3

// handleMouse is the pointer doing what the keys do: the wheel walks whichever pane it is
// over, a click picks a row, and a click on the row already under the cursor opens it.
//
// Selecting rather than opening on the first click is the one difference from the sidebar,
// which jumps straight away. The preview beside the list is the whole point here, so a
// click is how you look at something and the second click is how you act on it.
//
// Release, not press, as everywhere else in this stack: terminals eat the press of a click
// that also focuses their window, but always deliver the release.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.width == 0 || msg.Action == tea.MouseActionMotion {
		return m, nil
	}
	// A notice reports what just happened and is dismissed by whatever you do next,
	// pointer included.
	m.notice = ""

	if msg.Action == tea.MouseActionPress {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.wheel(msg.X, -1)
		case tea.MouseButtonWheelDown:
			m.wheel(msg.X, 1)
		}
		return m, nil
	}
	if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	// The help overlay owns the whole window, so any click closes it.
	if m.showHelp {
		m.showHelp = false
		m.help.ShowAll = false
		return m, nil
	}
	if msg.Y == 0 {
		if v, ok := m.tabAt(msg.X); ok {
			m.setView(v)
			return m, nil
		}
		// Checked before the staleness beside it: ✕ is the hard-right cell.
		if m.overClose(msg.X) {
			return m, tea.Quit
		}
		if m.overStaleness(msg.X) {
			return m.request("r")
		}
		return m, nil
	}
	if m.overPreview(msg.X) {
		return m, nil
	}
	row := m.rowAt(msg.Y)
	if row < 0 {
		return m, nil
	}
	if row == m.cursor {
		return m.request("enter")
	}
	m.cursor = row
	m.syncPreview()
	return m, nil
}

// wheel scrolls the pane the pointer is over: the preview by lines, the list by one row.
//
// Unlike j/k it stops at the ends. Wrapping is right for a key you press to walk a list
// and wrong for a wheel you spin to look down one - a notch past the last row should stop,
// not teleport.
func (m *Model) wheel(x, d int) {
	if m.overPreview(x) {
		if d < 0 {
			m.preview.LineUp(wheelLines)
		} else {
			m.preview.LineDown(wheelLines)
		}
		return
	}
	next := m.cursor + d
	if next < 0 || next >= len(m.rows) {
		return
	}
	m.cursor = next
	m.syncPreview()
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// A notice is dismissed by the next keypress, whatever it is: it reports what just
	// happened and should never need its own key.
	m.notice = ""

	if m.filtering {
		return m.handleFilterKey(msg)
	}
	k := m.keys
	switch {
	case key.Matches(msg, k.Quit):
		return m, tea.Quit
	case key.Matches(msg, k.Help):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return m, nil
	case key.Matches(msg, k.Down):
		m.move(1)
	case key.Matches(msg, k.Up):
		m.move(-1)
	case key.Matches(msg, k.Top):
		m.cursor = 0
		m.syncPreview()
	case key.Matches(msg, k.Bottom):
		m.cursor = len(m.rows) - 1
		m.clampCursor()
		m.syncPreview()
	case key.Matches(msg, k.ScrollDn):
		m.preview.HalfPageDown()
	case key.Matches(msg, k.ScrollUp):
		m.preview.HalfPageUp()
	case key.Matches(msg, k.NextView):
		m.setView(m.view.Next())
	case key.Matches(msg, k.PrevView):
		m.setView(m.view.Prev())
	case m.matchesView(msg):
		m.setView(workdesk.ParseView(msg.String()))
	case key.Matches(msg, k.Filter):
		m.filtering = true
		m.filter.Focus()
		return m, textinput.Blink
	case key.Matches(msg, k.Accept):
		return m.request("enter")
	case key.Matches(msg, k.Open):
		return m.request("o")
	case key.Matches(msg, k.Copy):
		return m.request("y")
	case key.Matches(msg, k.Tree):
		return m.request("c")
	case key.Matches(msg, k.Diff):
		return m.request("d")
	case key.Matches(msg, k.Matrix):
		return m.request("m")
	case key.Matches(msg, k.Assign):
		return m.request("a")
	case key.Matches(msg, k.Auto):
		return m.request("e")
	case key.Matches(msg, k.Merge):
		return m.request("M")
	case key.Matches(msg, k.Status):
		return m.request("s")
	case key.Matches(msg, k.Sprint):
		return m.request("i")
	case key.Matches(msg, k.Promote):
		return m.request("P")
	case key.Matches(msg, k.Sync):
		return m.request("r")
	}
	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter.Blur()
		m.filter.SetValue("")
		m.reload()
		return m, nil
	case "enter":
		m.filtering = false
		m.filter.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.cursor = 0
	m.reload()
	return m, cmd
}

// request records what the caller should do and stops the UI.
//
// The UI does not act. Opening a browser, adding a worktree and writing to GitLab all
// need a terminal the popup is occupying, or a confirm this model has no business
// owning - so the decision is handed back and every action stays a plain function that
// runs without a UI at all.
func (m Model) request(k string) (tea.Model, tea.Cmd) {
	row, ok := m.current()
	if !ok {
		return m, nil
	}
	// Views without a write to make say so rather than silently doing nothing.
	if (k == "a" || k == "e" || k == "M") && !strings.HasPrefix(row.Ref, "!") {
		m.notice = "that only applies to a merge request"
		return m, nil
	}
	if (k == "s" || k == "i") && !strings.HasPrefix(row.Ref, "#") {
		m.notice = "that only applies to an issue"
		return m, nil
	}
	m.Pending = &Action{Key: k, Ref: workdesk.RefFor(row)}
	return m, tea.Quit
}

func (m *Model) move(d int) {
	if len(m.rows) == 0 {
		return
	}
	m.cursor += d
	// Wraps, like the session picker's Alt-h/Alt-l: a list you can walk off the end of
	// is a list you have to look at to use.
	if m.cursor < 0 {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor >= len(m.rows) {
		m.cursor = 0
	}
	m.syncPreview()
}

func (m *Model) setView(v workdesk.View) {
	if v == m.view {
		return
	}
	m.view = v
	m.cursor = 0
	m.reload()
}

// matchesView reports whether the key selects a view by its digit. One check over the
// ring rather than one case per view, so adding a fifth view needs no edit here.
func (m Model) matchesView(msg tea.KeyMsg) bool {
	for _, b := range m.keys.Views {
		if key.Matches(msg, b) {
			return true
		}
	}
	return false
}
