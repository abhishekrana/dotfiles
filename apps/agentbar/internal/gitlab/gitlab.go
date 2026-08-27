// Package gitlab talks to GitLab through the glab CLI.
//
// Not an HTTP client of our own, deliberately: glab already holds the host and the
// token, so nothing here needs to know either. That is what keeps this repository free
// of any host, group or username - the project comes from the git remote and the
// identity from glab's own credentials.
//
// Everything is read-only except Assign, AutoMerge and Merge, which are the only three
// calls in this package that change anything on the server.
package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNotVisible means GitLab answered for a project this token cannot see.
//
// This is the one failure worth its own error. GitLab returns a null project rather
// than an error for a path that does not exist or is not visible, which reads as "you
// have no work" - and once overwrote a good mirror with an empty board while reporting
// success.
var ErrNotVisible = errors.New("project not visible to this token")

// Client runs glab. Runner is an interface so tests can drive the whole sync without a
// forge, and so a caller can wrap it with tracing.
type Client struct {
	Runner Runner
}

// Runner executes one glab invocation and returns its stdout.
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

// Exec is the real runner.
type Exec struct{}

// Run invokes glab, returning stderr as the error text: glab explains refusals there,
// and that explanation is the only useful thing to show a person.
func (Exec) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "glab", args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			return nil, fmt.Errorf("glab %s: %w", args[0], err)
		}
		return nil, fmt.Errorf("glab %s: %w: %s", args[0], err, truncate(msg, 300))
	}
	return out.Bytes(), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// New returns a client over the real glab.
func New() *Client { return &Client{Runner: Exec{}} }

// Host is the GitLab host glab is configured for. Used to refuse a repository on some
// other forge: a remote elsewhere still parses into a plausible owner/name, and asking
// GitLab about it returns a null project rather than an error.
func (c *Client) Host(ctx context.Context) (string, error) {
	out, err := c.Runner.Run(ctx, "config", "get", "host")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// User is the username glab authenticates as, which is whose work the mirror holds.
func (c *Client) User(ctx context.Context) (string, error) {
	out, err := c.Runner.Run(ctx, "api", "user")
	if err != nil {
		return "", err
	}
	var u struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(out, &u); err != nil {
		return "", fmt.Errorf("decode glab api user: %w", err)
	}
	if u.Username == "" {
		return "", errors.New("glab returned no username (is it authenticated?)")
	}
	return u.Username, nil
}

// graphQL runs one query and decodes data into out. A response carrying an errors array
// aborts rather than being treated as a partial success: half a mirror that looks whole
// is worse than no mirror.
func (c *Client) graphQL(ctx context.Context, query string, out any) error {
	raw, err := c.Runner.Run(ctx, "api", "graphql", "-f", "query="+query)
	if err != nil {
		return err
	}
	var probe struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &probe); err == nil && len(probe.Errors) > 0 {
		msgs := make([]string, 0, len(probe.Errors))
		for _, e := range probe.Errors {
			msgs = append(msgs, e.Message)
		}
		return fmt.Errorf("graphql rejected the query: %s", strings.Join(msgs, "; "))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode graphql response: %w", err)
	}
	return nil
}
