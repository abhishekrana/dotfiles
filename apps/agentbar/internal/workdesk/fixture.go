package workdesk

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// The invented mirror, embedded so one copy of it serves both the tests and the mockup.
//
// Every merge request, issue, branch, agent and username in here is fabricated, and that
// is load-bearing rather than incidental: the mockup gets published in screenshots and
// the tests run in CI, so real data in either would leak an employer's queue somewhere it
// cannot be taken back from.
//
// It covers one merge request per band, so the banding is exercised end to end
// rather than sampled.
//
//go:embed fixture
var fixtureFS embed.FS

// FixtureMirror decodes the invented snapshot.
func FixtureMirror() (*Mirror, error) {
	m, err := LoadFS(fixtureFS, "fixture")
	if err != nil {
		return nil, fmt.Errorf("fixture: %w", err)
	}
	return m, nil
}

// FixtureAgents is the agent half of the fixture, in the shape LoadAgents reads.
func FixtureAgents() ([]byte, error) {
	b, err := fs.ReadFile(fixtureFS, "fixture/agents.tsv")
	if err != nil {
		return nil, fmt.Errorf("fixture agents: %w", err)
	}
	return b, nil
}

// WriteFixture puts the invented mirror on disk, rendered by exactly the code a real
// sync uses. The mockup calls this: only the data is fake, never the rendering, so the
// mock shows what production shows.
func WriteFixture(dir string, now time.Time) error {
	m, err := FixtureMirror()
	if err != nil {
		return err
	}
	if err := WriteMirror(dir, m, now); err != nil {
		return err
	}
	agents, err := FixtureAgents()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "agents.tsv"), agents, 0o644); err != nil {
		return fmt.Errorf("write fixture agents: %w", err)
	}
	return nil
}
