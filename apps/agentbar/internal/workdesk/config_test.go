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
		bad  string
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
		})
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
