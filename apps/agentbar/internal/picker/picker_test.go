package picker

import (
	"strings"
	"testing"
)

// The fzf argument list is where quoting bugs used to live, so its shape is asserted
// rather than trusted.
func TestArgs(t *testing.T) {
	t.Parallel()
	args := Options{
		Header:  "inbox",
		Preview: "workdesk preview {1}",
		Keys:    []string{"tab", "q"},
		Colors:  "light,fg:#657b83",
	}.Args()

	joined := strings.Join(args, " ")
	for _, want := range []string{
		"--expect=tab,q",
		"--header=inbox",
		"--preview workdesk preview {1}",
		"--color=light,fg:#657b83",
		"--with-nth=2",
		// Explicit, to override the FZF_DEFAULT_OPTS inherited from the server.
		"--height=100%", "--border=none",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("args are missing %q\n  got: %s", want, joined)
		}
	}

	// The parenthesised transform form is required: `transform:` without parentheses
	// swallows the rest of the --bind string and silently eats every binding after it.
	for _, a := range args {
		if strings.Contains(a, "transform:") {
			t.Errorf("bare transform: in %q - it eats the bindings that follow", a)
		}
	}
	if !strings.Contains(joined, BandMark) {
		t.Error("the band-skip bindings do not mention the band mark")
	}
}

func TestArgsWithoutColorsOrKeys(t *testing.T) {
	t.Parallel()
	// Before the first theme run there is no palette, and an empty --color would make
	// fzf reject the whole invocation.
	joined := strings.Join(Options{}.Args(), " ")
	if strings.Contains(joined, "--color=") {
		t.Error("an empty palette still produced --color=")
	}
	if strings.Contains(joined, "--expect=") {
		t.Error("no keys still produced --expect=")
	}
}

// fzf writes the expected key on the first line and the selection on the second. A band
// header is not a selection.
func TestParse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, out, key, ref string
	}{
		{"a key and a row", "tab\nmrs:412\t  !412  a title\n", "tab", "mrs:412"},
		{"enter leaves the key line empty", "\nmrs:412\t  !412\n", "", "mrs:412"},
		{"a band header is not selectable", "\n" + BandMark + "\tready ·1\n", "", ""},
		{"nothing at all", "", "", ""},
		{"a key with no row", "q\n", "q", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := parse(c.out)
			if got.Key != c.key || got.Ref != c.ref {
				t.Errorf("parse(%q) = %+v; want key %q ref %q", c.out, got, c.key, c.ref)
			}
		})
	}
}
