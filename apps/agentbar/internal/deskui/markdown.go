package deskui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"

	"github.com/abhishekrana/agentbar/internal/ui"
)

// A ticket body is markdown, and it was being shown as the source text: headings as
// `##`, code as backticks, a table as a row of pipes. GitLab renders it, so reading one
// here meant reading around the syntax.
//
// glamour rather than a renderer of our own: it is the same family as the rest of this
// UI, it is what glab itself renders with, and it parses rather than pattern-matches -
// which is what makes nested lists, tables and paragraph reflow come out right. It costs
// about 9MB of binary, most of it chroma's lexers, and 1.3ms a render on a preview that
// only rebuilds when the cursor moves.

// markdownStyle is the palette as glamour's stylesheet.
//
// Built in Go from the theme rather than loaded from one of glamour's own JSON styles,
// for the reason every other colour here is: the flavors come from design/palette.toml,
// and a second palette that only approximates them would show.
func markdownStyle(t ui.Theme) ansi.StyleConfig {
	hex := func(c ui.Theme, pick func(ui.Theme) string) *string { s := pick(c); return &s }
	fg := hex(t, func(t ui.Theme) string { return string(t.Fg) })
	muted := hex(t, func(t ui.Theme) string { return string(t.Muted) })
	accent := hex(t, func(t ui.Theme) string { return string(t.Accent) })
	emphasis := hex(t, func(t ui.Theme) string { return string(t.Emphasis) })
	warn := hex(t, func(t ui.Theme) string { return string(t.Asking) })
	yes := true

	// No margin and no indent anywhere: the preview pane is already narrow and glamour's
	// defaults spend four columns of it on whitespace.
	var none uint
	block := func(colour *string) ansi.StyleBlock {
		return ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: colour}, Margin: &none}
	}
	heading := ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{Color: emphasis, Bold: &yes, BlockSuffix: "\n"},
		Margin:         &none,
	}
	return ansi.StyleConfig{
		Document:      block(fg),
		Paragraph:     block(fg),
		Heading:       heading,
		H1:            heading,
		H2:            heading,
		H3:            heading,
		H4:            heading,
		H5:            heading,
		H6:            heading,
		Text:          ansi.StylePrimitive{},
		Strong:        ansi.StylePrimitive{Color: emphasis, Bold: &yes},
		Emph:          ansi.StylePrimitive{Italic: &yes},
		Strikethrough: ansi.StylePrimitive{CrossedOut: &yes},
		// Inline code in the colour the row's own labels use, so a symbol in a sentence
		// reads as a symbol.
		Code: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: warn}},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: fg}, Margin: &none},
		},
		// A link is its text, then the target dim behind it: the URL is what you would
		// copy, and hiding it would make the text a dead end.
		Link:      ansi.StylePrimitive{Color: accent, Underline: &yes},
		LinkText:  ansi.StylePrimitive{Color: fg},
		Image:     ansi.StylePrimitive{Color: muted, Underline: &yes},
		ImageText: ansi.StylePrimitive{Color: muted, Format: "image: {{.text}}"},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: muted, Italic: &yes},
			Indent:         &none,
			Margin:         &none,
		},
		List: ansi.StyleList{
			StyleBlock:  ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: fg}, Margin: &none},
			LevelIndent: 2,
		},
		Item:           ansi.StylePrimitive{BlockPrefix: "· "},
		Enumeration:    ansi.StylePrimitive{BlockPrefix: ". "},
		Task:           ansi.StyleTask{Ticked: "[x] ", Unticked: "[ ] "},
		HorizontalRule: ansi.StylePrimitive{Color: muted, Format: "\n───\n"},
		Table:          ansi.StyleTable{StyleBlock: ansi.StyleBlock{Margin: &none}},
		HTMLBlock:      ansi.StyleBlock{},
		HTMLSpan:       ansi.StyleBlock{},
	}
}

// newMarkdown builds the renderer for one pane width. Rebuilt on resize rather than per
// render, because the width is what it wraps to.
func newMarkdown(t ui.Theme, width int) *glamour.TermRenderer {
	if width <= 0 {
		return nil
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(markdownStyle(t)),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		// A preview that renders the source text is worse than one that renders
		// markdown, and better than one that renders nothing.
		return nil
	}
	return r
}

// commentIndent is the column a comment's body sits at, under the author line that
// names it. The renderer that draws those bodies wraps that much narrower, so an
// indented line is still inside the pane.
const commentIndent = 2

// markdown renders a body, falling back to the wrapped source when there is no renderer
// - a pane with no width yet, or `workdesk preview` on a command line, which has no pane
// at all.
func (m Model) markdown(body string) string { return m.render(m.md, body, 0) }

// comment renders one comment's body, indented under its author.
func (m Model) comment(body string) string {
	return m.indent(m.render(m.mdIndented, body, commentIndent), strings.Repeat(" ", commentIndent))
}

func (m Model) render(r *glamour.TermRenderer, body string, indent int) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	if r == nil {
		return m.wrap(body, indent)
	}
	out, err := r.Render(body)
	if err != nil {
		return m.wrap(body, indent)
	}
	// glamour opens and closes with a blank line of its own; the preview owns its own
	// spacing between sections.
	return strings.Trim(out, "\n")
}
