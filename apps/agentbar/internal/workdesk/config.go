package workdesk

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Self stands for whoever glab authenticates as, so the config never has to carry a
// project bot's generated username - which is the account you are least likely to be
// able to type from memory.
const Self = "@me"

// Config is whose work the mirror holds, and how far back the inbox reaches.
//
// It lives outside this repository (see ConfigPath) because a username is exactly the
// kind of thing that must not be committed here, and because it is per-machine: the same
// dotfiles serve accounts that have nothing to do with each other.
type Config struct {
	Accounts []string
	// InboxSince is the window the inbox opens at, nil when the file says nothing. A
	// pointer rather than a zero value, because "all" is a window someone can ask for
	// and Window(0) is what it is.
	InboxSince *Window
}

// Window is how far back the inbox opens: what the config asks for, else a week.
func (c *Config) Window() Window {
	if c == nil || c.InboxSince == nil {
		return DefaultWindow
	}
	return *c.InboxSince
}

// LoadConfig reads the config file. A missing file is not an error - with no config the
// mirror holds the work of whoever glab authenticates as, which is what it did before
// there was a config at all.
func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer f.Close()
	cfg, err := ParseConfig(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// ParseConfig reads the slice of TOML this file needs: # comments, key = "string" and
// key = ["a", "b"], one key per line.
//
// Hand-read rather than pulling in a TOML library for two keys, and strict rather than
// lenient: a line it does not understand is refused by name. A config that silently
// ignored a table header or a misspelled key would leave you syncing one account and
// looking at a board that seems complete.
func ParseConfig(r io.Reader) (*Config, error) {
	cfg := &Config{}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("line %d: expected key = value, got %q", n, line)
		}
		key = strings.TrimSpace(key)
		switch key {
		case "accounts":
			list, err := parseList(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("line %d: accounts: %w", n, err)
			}
			cfg.Accounts = list
		case "inbox_since":
			raw, err := parseString(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("line %d: inbox_since: %w", n, err)
			}
			w, err := ParseWindow(raw)
			if err != nil {
				return nil, fmt.Errorf("line %d: inbox_since: %w", n, err)
			}
			cfg.InboxSince = &w
		default:
			return nil, fmt.Errorf("line %d: unknown setting %q (only accounts and inbox_since)", n, key)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// parseString takes one quoted value.
func parseString(s string) (string, error) {
	one, err := strconv.Unquote(s)
	if err != nil {
		return "", fmt.Errorf("expected a quoted value, got %s", s)
	}
	return one, nil
}

// parseList takes a quoted string or an array of them, so a single account needs no
// brackets.
func parseList(s string) ([]string, error) {
	if !strings.HasPrefix(s, "[") {
		one, err := strconv.Unquote(s)
		if err != nil {
			return nil, fmt.Errorf("expected a quoted username, got %s", s)
		}
		return []string{one}, nil
	}
	if !strings.HasSuffix(s, "]") {
		return nil, fmt.Errorf("unclosed [ in %s (one line per setting)", s)
	}
	var out []string
	for _, part := range strings.Split(strings.TrimSuffix(strings.TrimPrefix(s, "["), "]"), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		one, err := strconv.Unquote(part)
		if err != nil {
			return nil, fmt.Errorf("expected a quoted username, got %s", part)
		}
		out = append(out, one)
	}
	return out, nil
}

// Users resolves the configured accounts against the identity glab authenticates as:
// Self becomes that identity, an empty config means that identity alone. Order is kept
// and duplicates dropped, so the manifest calls are one per account.
func (c *Config) Users(self string) []string {
	names := c.Accounts
	if len(names) == 0 {
		names = []string{Self}
	}
	seen := make(map[string]bool, len(names))
	out := make([]string, 0, len(names))
	for _, name := range names {
		if name == Self {
			name = self
		}
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}
