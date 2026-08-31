package deskui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// resolve stands in for the mirror: a reference it knows, and the shape it falls back to.
func resolve(sigil, iid string) string {
	if sigil == "#" && iid == "128" {
		return "https://gitlab.example.com/acme/platform/-/issues/128"
	}
	if sigil == "!" {
		return "https://gitlab.example.com/acme/platform/-/merge_requests/" + iid
	}
	return ""
}

func TestFindLinks(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		line  string
		want  string
		click int // the column to click, -1 for "nothing is clickable here"
	}{{
		name:  "a bare url",
		line:  "url       https://gitlab.example.com/acme/platform/-/issues/128",
		want:  "https://gitlab.example.com/acme/platform/-/issues/128",
		click: 12,
	}, {
		// A reference is not a URL, so only the mirror can make it one.
		name:  "a merge request reference",
		line:  "in flight !5408  4825-dexman-numpy-pin",
		want:  "https://gitlab.example.com/acme/platform/-/merge_requests/5408",
		click: 11,
	}, {
		name:  "an issue reference inside a sentence",
		line:  "This is the opposite direction to #128 (station behind cloud).",
		want:  "https://gitlab.example.com/acme/platform/-/issues/128",
		click: 35,
	}, {
		// The full stop is not part of the address.
		name:  "a url ending a sentence",
		line:  "See https://example.com/runbook.",
		want:  "https://example.com/runbook",
		click: 10,
	}, {
		name:  "a reference the mirror cannot place",
		line:  "mentioned in ?99 somewhere",
		click: -1,
	}, {
		// #12 must not be found inside #123, and a version is not a reference.
		name:  "not every hash is a reference",
		line:  "numpy 1.26.4 resolves to 2.5.2",
		click: -1,
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			links := findLinks(c.line, resolve)
			if c.click < 0 {
				if len(links) != 0 {
					t.Fatalf("found %+v in a line with no link", links)
				}
				return
			}
			got, ok := linkAt(links, 0, c.click)
			if !ok {
				t.Fatalf("nothing at column %d of %q; found %+v", c.click, c.line, links)
			}
			if got != c.want {
				t.Errorf("link = %q, want %q", got, c.want)
			}
		})
	}
}

// A click is placed by display column, and styling occupies none of them - so the span
// has to be found through the escape sequences rather than counting bytes.
func TestFindLinksThroughStyling(t *testing.T) {
	t.Parallel()
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#268bd2")).Render("in flight ") +
		lipgloss.NewStyle().Bold(true).Render("!5408")
	links := findLinks(styled, resolve)
	if len(links) != 1 {
		t.Fatalf("found %d links in a styled line, want 1", len(links))
	}
	if links[0].start != 10 {
		t.Errorf("link starts at column %d, want 10 - the escape sequences were counted", links[0].start)
	}
	if _, ok := linkAt(links, 0, 12); !ok {
		t.Error("a click inside the styled reference found nothing")
	}
}

// The line a link is on is its line in the content, so scrolling maps a click correctly.
func TestFindLinksReportsTheLine(t *testing.T) {
	t.Parallel()
	content := strings.Join([]string{"nothing here", "", "see #128 for the detail"}, "\n")
	links := findLinks(content, resolve)
	if len(links) != 1 {
		t.Fatalf("found %d links, want 1", len(links))
	}
	if links[0].line != 2 {
		t.Errorf("link is on line %d, want 2", links[0].line)
	}
	if _, ok := linkAt(links, 0, 5); ok {
		t.Error("a click on line 0 found a link that is on line 2")
	}
}
