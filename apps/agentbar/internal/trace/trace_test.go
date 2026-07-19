package trace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestField(t *testing.T) {
	cases := map[string]string{
		"dictate":          "dictate",          // bare word, unquoted
		"":                 `""`,               // empty -> quoted
		"a b c":            `"a b c"`,          // space -> quoted
		"k=v":              `"k=v"`,            // '=' -> quoted
		`say "hi"`:         `"say \"hi\""`,     // quotes escaped
		`back\slash`:       `"back\\slash"`,    // backslash escaped, forces quoting
		"line\nbreak\ttab": `"line break tab"`, // control chars -> spaces, then quoted for the spaces
	}
	for in, want := range cases {
		if got := field(in); got != want {
			t.Errorf("field(%q) = %q, want %q", in, got, want)
		}
	}
	// 200-rune cap.
	if got := field(strings.Repeat("x", 500)); len(got) != 200 {
		t.Errorf("cap: len = %d, want 200", len(got))
	}
}

func TestLogWritesLogfmt(t *testing.T) {
	logPath = filepath.Join(t.TempDir(), "dotfiles", "trace.log")
	Log("agentbar", "jump", "pane", "%5", "ms", 41, "err", "")
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	line := strings.TrimRight(string(b), "\n")
	for _, want := range []string{"src=agentbar", "evt=jump", "pane=%5", "ms=41", `err=""`, "pid="} {
		if !strings.Contains(line, want) {
			t.Errorf("line %q missing %q", line, want)
		}
	}
	if !strings.HasPrefix(line, "ts=") {
		t.Errorf("line %q must start with ts=", line)
	}
}

func TestLogvGated(t *testing.T) {
	logPath = filepath.Join(t.TempDir(), "trace.log")
	SetVerbose(false)
	Logv("agentbar", "mouse", "x", 1)
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatal("Logv wrote while verbose off")
	}
	SetVerbose(true)
	Logv("agentbar", "mouse", "x", 1)
	b, _ := os.ReadFile(logPath)
	if !strings.Contains(string(b), "evt=mouse") {
		t.Fatal("Logv did not write while verbose on")
	}
	SetVerbose(false)
}

func TestRotation(t *testing.T) {
	logPath = filepath.Join(t.TempDir(), "trace.log")
	// Cross the 1 MiB cap; each line is ~60 bytes, so ~20k lines suffice.
	for i := range 20000 {
		Log("t", "x", "n", i, "pad", "0123456789012345678901234567890123456789")
	}
	if _, err := os.Stat(logPath + ".1"); err != nil {
		t.Fatalf("expected rotated %s.1: %v", logPath, err)
	}
	fi, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("current log missing: %v", err)
	}
	if fi.Size() >= capBytes {
		t.Errorf("current log %d bytes, should be under cap %d after rotation", fi.Size(), capBytes)
	}
}
