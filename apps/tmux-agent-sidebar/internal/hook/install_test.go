package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestInstallCreatesAndIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.local.json")

	changed, err := Install(path, "/opt/bin/tmux-agent-sidebar")
	if err != nil || !changed {
		t.Fatalf("first install: changed=%v err=%v", changed, err)
	}
	root := readJSON(t, path)
	hooks := root["hooks"].(map[string]any)
	for _, event := range events {
		if _, ok := hooks[event]; !ok {
			t.Errorf("event %s not installed", event)
		}
	}

	changed, err = Install(path, "/opt/bin/tmux-agent-sidebar")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("second install must be a no-op")
	}
}

func TestHookCommandGuardedAndPortable(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	got := hookCommand(filepath.Join(home, ".tmux", "plugins", "x", "bin", "sb"))
	want := `[ -x "$HOME/.tmux/plugins/x/bin/sb" ] && "$HOME/.tmux/plugins/x/bin/sb" hook || true`
	if got != want {
		t.Errorf("hookCommand = %q, want %q", got, want)
	}
	if got := hookCommand("/opt/bin/sb"); got != `[ -x "/opt/bin/sb" ] && "/opt/bin/sb" hook || true` {
		t.Errorf("non-home path must stay absolute, got %q", got)
	}
}

func TestInstallMigratesOldStyleEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	old := `{
	  "hooks": {
	    "Stop": [
	      {"matcher": "", "hooks": [{"type": "command", "command": "bash ~/.claude/hooks/on-stop.sh"}]},
	      {"hooks": [{"type": "command", "command": "$HOME/.tmux/plugins/tmux-agent-sidebar/bin/tmux-agent-sidebar hook"}]}
	    ]
	  }
	}`
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(path, "/opt/bin/tmux-agent-sidebar"); err != nil {
		t.Fatal(err)
	}
	stop := readJSON(t, path)["hooks"].(map[string]any)["Stop"].([]any)
	if len(stop) != 2 {
		t.Fatalf("old entry must be replaced, not stacked: %d entries", len(stop))
	}
	last := stop[1].(map[string]any)["hooks"].([]any)[0].(map[string]any)["command"].(string)
	if !strings.Contains(last, `[ -x "/opt/bin/tmux-agent-sidebar" ]`) {
		t.Errorf("migrated entry must be guarded: %q", last)
	}
}

func TestInstallWritesThroughSymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real-settings.json")
	link := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(real, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(link, "/opt/bin/sb"); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("install replaced the symlink with a regular file")
	}
	if root := readJSON(t, real); root["hooks"] == nil {
		t.Error("hooks not written through to the symlink target")
	}
}

func TestInstallPreservesExistingEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.local.json")
	existing := `{
	  "model": "claude-fable-5",
	  "hooks": {
	    "Stop": [
	      {"matcher": "", "hooks": [{"type": "command", "command": "bash ~/.claude/hooks/on-stop.sh"}]}
	    ]
	  }
	}`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(path, "/opt/bin/tmux-agent-sidebar"); err != nil {
		t.Fatal(err)
	}
	root := readJSON(t, path)
	if root["model"] != "claude-fable-5" {
		t.Error("unrelated settings must be preserved")
	}
	stop := root["hooks"].(map[string]any)["Stop"].([]any)
	if len(stop) != 2 {
		t.Fatalf("Stop must keep the user's entry and add ours, got %d entries", len(stop))
	}
	first := stop[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)["command"]
	if first != "bash ~/.claude/hooks/on-stop.sh" {
		t.Errorf("user's entry must stay first, got %v", first)
	}
}
