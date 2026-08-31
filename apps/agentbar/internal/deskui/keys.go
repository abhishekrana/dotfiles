package deskui

import (
	"github.com/charmbracelet/bubbles/key"

	"github.com/abhishekrana/agentbar/internal/workdesk"
)

// keyMap is every binding, declared once so the footer hints and the help overlay are
// generated from the same source the Update loop dispatches on. The shell version kept
// its key list in a hand-written header string that silently truncated past ~52 columns,
// taking the last keys with it; that failure is not possible here.
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	NextView key.Binding
	PrevView key.Binding
	// Views is one binding per view, generated from the ring, so a reordering there
	// moves the digits and their help text together. These used to be four named
	// fields with their labels typed out, which is how the order came to be encoded in
	// five places at once.
	Views    []key.Binding
	Filter   key.Binding
	Accept   key.Binding
	ScrollUp key.Binding
	ScrollDn key.Binding
	Open     key.Binding
	Copy     key.Binding
	Tree     key.Binding
	Diff     key.Binding
	Matrix   key.Binding
	Assign   key.Binding
	Auto     key.Binding
	Merge    key.Binding
	Status   key.Binding
	Sprint   key.Binding
	MRDiff   key.Binding
	Promote  key.Binding
	Sync     key.Binding
	Help     key.Binding
	Quit     key.Binding
}

// viewBindings turns the ring into one binding per view.
func viewBindings() []key.Binding {
	views := workdesk.Views()
	out := make([]key.Binding, 0, len(views))
	for _, v := range views {
		out = append(out, key.NewBinding(
			key.WithKeys(v.Key()),
			key.WithHelp(v.Key(), v.Title()),
		))
	}
	return out
}

func defaultKeys() keyMap {
	return keyMap{
		Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Top:      key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "first")),
		Bottom:   key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "last")),
		NextView: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next view")),
		PrevView: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("⇧tab", "previous view")),
		Views:    viewBindings(),
		Filter:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Accept:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "jump to that agent")),
		ScrollUp: key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("^u", "scroll preview up")),
		ScrollDn: key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("^d", "scroll preview down")),
		Open:     key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in browser")),
		Copy:     key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy url")),
		Tree:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "add a worktree")),
		Diff:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "diff pane")),
		Matrix:   key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "gate matrix")),
		Assign:   key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "assign a reviewer")),
		Auto:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "set auto-merge")),
		Merge:    key.NewBinding(key.WithKeys("M"), key.WithHelp("M", "merge")),
		Status:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "move to a status")),
		MRDiff:   key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "read the diff")),
		// One key both ways: the row's ◆ already says which way it will go.
		Sprint:  key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "in/out of the sprint")),
		Promote: key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "promote to a pane")),
		Sync:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "re-sync")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		// alt+n is the key tmux opens this with. While the popup is up that key reaches
		// here instead of tmux, so quitting on it is what makes the opener a toggle -
		// the status chip cannot do it, because a popup swallows the click.
		Quit: key.NewBinding(key.WithKeys("q", "esc", "ctrl+c", "alt+n"), key.WithHelp("q/alt+n", "close")),
	}
}

// mouseHints is what the pointer does, listed for the same reason the key hints are
// generated from the keymap: an undocumented gesture is one nobody finds. There is no
// key.Binding for a click, so these are plain pairs the help screen renders itself.
func mouseHints() [][2]string {
	return [][2]string{
		{"click", "select a row"},
		{"click again", "jump to that agent's pane"},
		{"click a link", "open it in the browser - a url, or a #1234 or !1234"},
		{"click a tab", "switch view"},
		{"click ✕", "close"},
		{"click \"synced\"", "re-sync"},
		{"wheel", "walk the list, or scroll the preview under the pointer"},
	}
}

// ShortHelp is the footer strip: moving around, and the actions that apply to the row
// under the cursor. Enter is not among them - it belongs to the agents view alone, and a
// strip that is always on screen should not advertise a key that does nothing on three
// views out of four. The overlay still documents it.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Open, k.Copy, k.NextView, k.Filter, k.Help, k.Quit}
}

// FullHelp is the overlay, grouped the way the work is: navigate, act, write.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom, k.ScrollUp, k.ScrollDn},
		append(k.Views, k.NextView, k.PrevView, k.Filter),
		{k.Open, k.Copy},
		{k.Accept, k.MRDiff, k.Tree, k.Diff, k.Matrix},
		{k.Assign, k.Auto, k.Merge, k.Status, k.Sprint, k.Promote, k.Sync, k.Quit},
	}
}
