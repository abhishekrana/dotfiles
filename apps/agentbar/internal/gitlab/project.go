package gitlab

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ProjectFor resolves owner/name from a repository's origin remote, for both ssh and
// https forms.
//
// Never hardcoded: that is what keeps this program free of any employer identifier, and
// what makes it work on any GitLab project unchanged.
//
// The host is checked against glab's own, because a remote on some other forge still
// parses into a plausible owner/name - and GitLab answers about an unknown path with a
// null project rather than an error, which reads as "you have no work" and would
// overwrite a good mirror with an empty board.
func ProjectFor(ctx context.Context, c *Client, dir string) (string, error) {
	url, err := originURL(ctx, dir)
	if err != nil {
		return "", err
	}
	host, path, err := splitRemote(url)
	if err != nil {
		return "", fmt.Errorf("cannot read a project from the origin remote %q", url)
	}
	want, err := c.Host(ctx)
	if err == nil && want != "" && host != want {
		return "", fmt.Errorf("%s is on %s, not %s (glab cannot see it)", dir, host, want)
	}
	return path, nil
}

func originURL(ctx context.Context, dir string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "remote", "get-url", "origin").Output()
	if err != nil {
		return "", fmt.Errorf("no origin remote in %s", dir)
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(string(out)), ".git")), nil
}

// splitRemote pulls the host and the project path out of either remote form:
//
//	https://gitlab.example.com/group/sub/project
//	git@gitlab.example.com:group/sub/project
func splitRemote(url string) (host, path string, err error) {
	if rest, ok := cutScheme(url); ok {
		h, p, found := strings.Cut(rest, "/")
		if !found || p == "" {
			return "", "", fmt.Errorf("no path in %q", url)
		}
		return stripUser(h), p, nil
	}
	h, p, found := strings.Cut(url, ":")
	if !found || p == "" {
		return "", "", fmt.Errorf("unrecognised remote %q", url)
	}
	return stripUser(h), p, nil
}

func cutScheme(url string) (string, bool) {
	if _, rest, found := strings.Cut(url, "://"); found {
		return rest, true
	}
	return url, false
}

func stripUser(host string) string {
	if _, h, found := strings.Cut(host, "@"); found {
		return h
	}
	return host
}
