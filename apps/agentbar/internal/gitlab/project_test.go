package gitlab

import "testing"

// Remote parsing decides which project the whole mirror is about, and getting it subtly
// wrong is how a sync ends up asking GitLab about a project that does not exist - which
// GitLab answers with a null rather than an error.
func TestSplitRemote(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, url, host, path string
		wantErr               bool
	}{
		{name: "https", url: "https://gitlab.example.com/group/project",
			host: "gitlab.example.com", path: "group/project"},
		{name: "https with a subgroup", url: "https://gitlab.example.com/group/sub/project",
			host: "gitlab.example.com", path: "group/sub/project"},
		{name: "https with a user", url: "https://me@gitlab.example.com/group/project",
			host: "gitlab.example.com", path: "group/project"},
		{name: "https with a port", url: "https://gitlab.example.com:8443/group/project",
			host: "gitlab.example.com:8443", path: "group/project"},
		{name: "ssh", url: "git@gitlab.example.com:group/project",
			host: "gitlab.example.com", path: "group/project"},
		{name: "ssh with a subgroup", url: "git@gitlab.example.com:group/sub/project",
			host: "gitlab.example.com", path: "group/sub/project"},
		{name: "ssh scheme", url: "ssh://git@gitlab.example.com/group/project",
			host: "gitlab.example.com", path: "group/project"},
		{name: "no path at all", url: "https://gitlab.example.com", wantErr: true},
		{name: "not a remote", url: "gitlab.example.com", wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			host, path, err := splitRemote(c.url)
			if c.wantErr {
				if err == nil {
					t.Errorf("splitRemote(%q) = %q, %q; want an error", c.url, host, path)
				}
				return
			}
			if err != nil {
				t.Fatalf("splitRemote(%q): %v", c.url, err)
			}
			if host != c.host || path != c.path {
				t.Errorf("splitRemote(%q) = %q, %q; want %q, %q", c.url, host, path, c.host, c.path)
			}
		})
	}
}
