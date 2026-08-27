package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The flavor has one home: the file the theme switcher writes, which bash, hunk and leaf
// all read. This shipped once reading a tmux option that nothing sets, and the fallback
// hid it - the UI simply came up in the default flavor, which happened to be right.
func TestThemeNameReadsTheSwitchersFile(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		env     string
		want    string
		noWrite bool
	}{
		{name: "the switcher's file", file: "catppuccin-mocha", want: "catppuccin-mocha"},
		{name: "trailing newline", file: "solarized-dark\n", want: "solarized-dark"},
		{name: "the env override wins", file: "solarized-light", env: "catppuccin-latte",
			want: "catppuccin-latte"},
		{name: "no file: fall back to the default flavor", noWrite: true, want: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", dir)
			t.Setenv("WORKDESK_THEME", c.env)
			if !c.noWrite {
				if err := os.MkdirAll(filepath.Join(dir, "theme"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "theme", "current"),
					[]byte(c.file), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if got := themeName(); got != c.want {
				t.Errorf("themeName() = %q, want %q", got, c.want)
			}
		})
	}
}

// The mirror lives outside any repository, because merge request bodies and comments can
// carry credentials and a notes repo would commit them.
func TestMirrorDirIsOverridableAndOutsideTheRepo(t *testing.T) {
	t.Run("the override wins", func(t *testing.T) {
		t.Setenv("WORKDESK_MIRROR", "/tmp/somewhere")
		if got := mirrorDir(); got != "/tmp/somewhere" {
			t.Errorf("mirrorDir() = %q, want the override", got)
		}
	})
	t.Run("otherwise the state dir", func(t *testing.T) {
		t.Setenv("WORKDESK_MIRROR", "")
		t.Setenv("XDG_STATE_HOME", "/tmp/state")
		want := filepath.Join("/tmp/state", "dotfiles", "workdesk")
		if got := mirrorDir(); got != want {
			t.Errorf("mirrorDir() = %q, want %q", got, want)
		}
	})
}

// refKind is what every action dispatches on, so a malformed reference must not be
// mistaken for a merge request.
func TestRefKind(t *testing.T) {
	t.Parallel()
	cases := []struct{ ref, kind, id string }{
		{"mrs:412", "mrs", "412"},
		{"issues:128", "issues", "128"},
		{"agents:%3", "agents", "%3"},
		{"412", "", "412"},
		{"", "", ""},
	}
	for _, c := range cases {
		kind, id := refKind(c.ref)
		if kind != c.kind || id != c.id {
			t.Errorf("refKind(%q) = %q, %q; want %q, %q", c.ref, kind, id, c.kind, c.id)
		}
	}
}
