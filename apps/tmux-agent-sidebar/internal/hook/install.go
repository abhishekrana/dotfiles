package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Events the sidebar needs to observe. Claude Code runs every configured
// entry for an event, so our entries coexist with any the user already
// has (e.g. dotfiles hooks writing their own state files).
var events = []string{
	"SessionStart",
	"UserPromptSubmit",
	"PreToolUse",
	"PermissionRequest",
	"Notification",
	"Stop",
	"SubagentStart",
	"SubagentStop",
	"SessionEnd",
}

// DefaultSettingsPath is where install writes by default. It must be the
// user-level settings.json: Claude Code does not load hooks from a
// user-level settings.local.json (only the project-level one exists).
func DefaultSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

// Install merges the sidebar's hook entries into the Claude Code settings
// file at path, creating it if missing. Idempotent: entries whose command
// already invokes this binary's `hook` subcommand are left alone. All
// unrelated settings content is preserved.
func Install(path, binPath string) (changed bool, err error) {
	command := hookCommand(binPath)

	// Settings files are often symlinks (stow-managed dotfiles); write
	// through to the real file so the atomic rename below doesn't
	// replace the link with a regular file.
	if resolved, rerr := filepath.EvalSymlinks(path); rerr == nil {
		path = resolved
	}

	root := map[string]any{}
	if data, rerr := os.ReadFile(path); rerr == nil {
		if err := json.Unmarshal(data, &root); err != nil {
			return false, fmt.Errorf("parse %s: %w", path, err)
		}
	} else if !os.IsNotExist(rerr) {
		return false, rerr
	}

	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		root["hooks"] = hooks
	}

	for _, event := range events {
		entries, _ := hooks[event].([]any)
		if hasCommand(entries, command) {
			continue
		}
		// Drop any older entry of ours (e.g. unguarded or dev-path
		// variants) so re-running install migrates instead of stacking
		// duplicate hooks that would double-fire.
		entries = withoutOurs(entries)
		entries = append(entries, map[string]any{
			"hooks": []any{
				map[string]any{"type": "command", "command": command},
			},
		})
		hooks[event] = entries
		changed = true
	}
	if !changed {
		return false, nil
	}

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	// Atomic replace so a crash can't corrupt the settings file.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(out, '\n'), 0o644); err != nil {
		return false, err
	}
	return true, os.Rename(tmp, path)
}

// hookCommand builds the hook entry's command string. The home prefix is
// replaced with $HOME (hook commands run through a shell) so the entry is
// portable and keeps machine paths out of git-tracked settings files.
// The command is self-guarding: on a machine where the plugin is not
// installed (settings files often sync via dotfiles) it silently no-ops
// instead of spamming hook-failure warnings on every Claude event.
func hookCommand(binPath string) string {
	binPath = portablePath(binPath)
	return fmt.Sprintf(`[ -x "%s" ] && "%s" hook || true`, binPath, binPath)
}

func portablePath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if rel, ok := strings.CutPrefix(p, home+string(filepath.Separator)); ok {
			return "$HOME/" + rel
		}
	}
	return p
}

// ours matches any command invoking this plugin's hook subcommand,
// whatever the path style of the install that wrote it.
func ours(cmd string) bool {
	return strings.Contains(cmd, "tmux-agent-sidebar") && strings.Contains(cmd, " hook")
}

// withoutOurs filters out entries whose every command is ours.
func withoutOurs(entries []any) []any {
	var out []any
	for _, e := range entries {
		entry, _ := e.(map[string]any)
		inner, _ := entry["hooks"].([]any)
		keep := false
		for _, h := range inner {
			hm, _ := h.(map[string]any)
			cmd, _ := hm["command"].(string)
			if !ours(cmd) {
				keep = true
			}
		}
		if keep || len(inner) == 0 {
			out = append(out, e)
		}
	}
	return out
}

// hasCommand reports whether any entry already runs exactly this hook
// command. A stale variant (moved binary, older format) won't match; it
// is dropped by withoutOurs and replaced.
func hasCommand(entries []any, command string) bool {
	for _, e := range entries {
		entry, _ := e.(map[string]any)
		inner, _ := entry["hooks"].([]any)
		for _, h := range inner {
			hm, _ := h.(map[string]any)
			cmd, _ := hm["command"].(string)
			if strings.TrimSpace(cmd) == command {
				return true
			}
		}
	}
	return false
}
