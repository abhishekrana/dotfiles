package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/abhishekrana/agentbar/internal/gitlab"
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

// r used to refuse whenever the working directory was not a GitLab repo - which is most
// of them, since the float inherits the cwd of the pane it was opened from. "dotfiles is
// on github.com, not gitlab.com" is what a resync said, with a good mirror on disk that
// names the project it holds.
func TestSyncProjectFallsBackToTheMirror(t *testing.T) {
	ctx := context.Background()
	notARepo := t.TempDir() // git has no origin here, so ProjectFor fails without a network

	t.Run("no mirror: the working directory's error stands", func(t *testing.T) {
		_, _, err := syncProject(ctx, gitlab.New(), t.TempDir(), notARepo)
		if err == nil {
			t.Fatal("want the resolve error, got nil")
		}
	})

	t.Run("a mirror names the project to refresh", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "index.json"),
			[]byte(`{"project":"acme/platform","mrs":[],"issues":[],"todos":[]}`), 0o644); err != nil {
			t.Fatal(err)
		}
		project, via, err := syncProject(ctx, gitlab.New(), dir, notARepo)
		if err != nil {
			t.Fatal(err)
		}
		if project != "acme/platform" || via != "mirror" {
			t.Fatalf("want acme/platform via mirror, got %q via %q", project, via)
		}
	})
}
