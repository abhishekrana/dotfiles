package workdesk

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want []string
		// win is the window the file asks for; unset means the file says nothing and
		// the default stands.
		win *Window
		bad string
	}{{
		name: "an array of accounts",
		in:   "accounts = [\"you\", \"you-bot\"]\n",
		want: []string{"you", "you-bot"},
	}, {
		name: "one account needs no brackets",
		in:   "accounts = \"you\"\n",
		want: []string{"you"},
	}, {
		name: "comments and blank lines",
		in:   "# who to sync\n\naccounts = [\"you\"]  \n",
		want: []string{"you"},
	}, {
		// Strict on purpose: a silently ignored line is a board that looks complete
		// while holding one account's work.
		name: "a table header is refused rather than skipped",
		in:   "[workdesk]\naccounts = [\"you\"]\n",
		bad:  "expected key = value",
	}, {
		name: "an unknown setting is named",
		in:   "acounts = [\"you\"]\n",
		bad:  "unknown setting",
	}, {
		name: "an unquoted name is refused",
		in:   "accounts = [you]\n",
		bad:  "quoted username",
	}, {
		name: "a window alongside the accounts",
		in:   "accounts = [\"you\"]\ninbox_since = \"14d\"\n",
		want: []string{"you"},
		win:  ptr(14 * day),
	}, {
		name: "a window of all",
		in:   "inbox_since = \"all\"\n",
		win:  ptr(WindowAll),
	}, {
		name: "a window that is not a count of days is refused",
		in:   "inbox_since = \"a fortnight\"\n",
		bad:  "count of days",
	}, {
		name: "an unquoted window is refused",
		in:   "inbox_since = 7d\n",
		bad:  "quoted value",
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := ParseConfig(strings.NewReader(c.in))
			if c.bad != "" {
				if err == nil {
					t.Fatalf("parsed %q, wanted an error", c.in)
				}
				if !strings.Contains(err.Error(), c.bad) {
					t.Errorf("error %q does not mention %q", err, c.bad)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseConfig: %v", err)
			}
			if got := strings.Join(cfg.Accounts, ","); got != strings.Join(c.want, ",") {
				t.Errorf("accounts = %q, want %q", got, c.want)
			}
			want := DefaultWindow
			if c.win != nil {
				want = *c.win
			}
			if got := cfg.Window(); got != want {
				t.Errorf("inbox window = %v, want %v", got, want)
			}
		})
	}
}

func ptr(w Window) *Window { return &w }

// A file that says nothing about the window gets the default, and so does no file at all
// - the window is a preference, not a thing you have to declare.
func TestConfigWindowDefaults(t *testing.T) {
	t.Parallel()
	var none *Config
	if got := none.Window(); got != DefaultWindow {
		t.Errorf("no config at all gives %v, want %v", got, DefaultWindow)
	}
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "absent.toml"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got := cfg.Window(); got != DefaultWindow {
		t.Errorf("a missing file gives %v, want %v", got, DefaultWindow)
	}
}

func TestUsersResolvesSelfAndDeduplicates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  Config
		want string
	}{
		{"no config is the token's own identity", Config{}, "bot"},
		{"self stands for the token", Config{Accounts: []string{Self, "you"}}, "bot,you"},
		{"naming the token twice fetches it once", Config{Accounts: []string{Self, "bot"}}, "bot"},
		{"order is kept", Config{Accounts: []string{"you", Self}}, "you,bot"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := strings.Join(c.cfg.Users("bot"), ","); got != c.want {
				t.Errorf("Users = %q, want %q", got, c.want)
			}
		})
	}
}

// A machine with no config file syncs exactly what it did before there was one.
func TestLoadConfigTreatsAMissingFileAsNoAccounts(t *testing.T) {
	t.Parallel()
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "nothing.toml"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Accounts) != 0 {
		t.Errorf("accounts = %v, want none", cfg.Accounts)
	}
	if got := strings.Join(cfg.Users("bot"), ","); got != "bot" {
		t.Errorf("Users = %q, want the token's own identity", got)
	}
}
