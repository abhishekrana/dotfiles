package deskui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// A preview is full of things worth opening - the row's own url, the merge request in
// flight, and every #1234 and !1234 a description mentions - and tmux owns the mouse
// while this is up, so the terminal never gets the chance to make them clickable itself.
// It is handed the click instead, which is what lets a bare reference be a link at all:
// !5408 is not a URL, and only the mirror knows it is a merge request.

// link is one clickable span: where it is on screen, and where it goes.
type link struct {
	line       int // index into the preview's own content, before scrolling
	start, end int // display columns, end exclusive
	url        string
}

// A URL as written, and a GitLab reference as GitLab writes one. The reference is
// bounded either side so that #12 is not found inside #123, and the URL stops at
// whitespace - trailing sentence punctuation is trimmed after the match, because a link
// at the end of a sentence would otherwise swallow the full stop.
var (
	urlRE = regexp.MustCompile(`https?://[^\s)>\]]+`)
	refRE = regexp.MustCompile(`(^|[^\w/])([#!])(\d+)([^\w]|$)`)
)

// findLinks indexes everything clickable in a rendered preview.
//
// It reads the rendered text rather than being built up as the preview is written,
// because by then everything is in one place: a markdown link glamour has already
// expanded, a reference inside a comment, and the kv lines all look the same here. resolve
// turns a reference into a URL and returns empty for one that cannot be placed.
func findLinks(content string, resolve func(sigil, iid string) string) []link {
	var out []link
	for n, styled := range strings.Split(content, "\n") {
		plain, cols := columns(styled)
		add := func(from, to int, url string) {
			if url == "" || from >= len(cols) {
				return
			}
			if to >= len(cols) {
				to = len(cols) - 1
			}
			out = append(out, link{line: n, start: cols[from], end: cols[to], url: url})
		}
		for _, m := range urlRE.FindAllStringIndex(plain, -1) {
			from, to := m[0], m[1]
			for to > from && strings.ContainsRune(".,;:!?", rune(plain[to-1])) {
				to--
			}
			add(from, to, plain[from:to])
		}
		for _, m := range refRE.FindAllStringSubmatchIndex(plain, -1) {
			// Groups: 2 is the boundary before, 4 the sigil, 6 the digits.
			sigil, iid := plain[m[4]:m[5]], plain[m[6]:m[7]]
			add(m[4], m[7], resolve(sigil, iid))
		}
	}
	return out
}

// columns maps a styled line to its plain text and, for each byte of that text, the
// display column it lands on. Escape sequences occupy no columns and wide runes occupy
// more than one, so a click cannot be placed by counting bytes.
func columns(styled string) (plain string, cols []int) {
	var b strings.Builder
	col := 0
	for _, r := range ansi.Strip(styled) {
		w := ansi.StringWidth(string(r))
		for range len(string(r)) {
			cols = append(cols, col)
		}
		b.WriteRune(r)
		col += w
	}
	// One past the end, so a match that runs to the end of the line has a column to
	// close on.
	cols = append(cols, col)
	return b.String(), cols
}

// linkAt is the link under a point in the preview's content, if there is one.
func linkAt(links []link, line, col int) (string, bool) {
	for _, l := range links {
		if l.line == line && col >= l.start && col < l.end {
			return l.url, true
		}
	}
	return "", false
}
