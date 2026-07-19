// Package trace is agentbar's writer for the shared dotfiles trace log
// (${XDG_STATE_HOME:-~/.local/state}/dotfiles/trace.log). It records action
// EDGES only (clicks, jumps, hook events, drops) so evidence is already there
// when a bug happens. It must match the shell CLI `dotfiles-trace` byte-for-byte
// on timestamp, escaping, and rotation. Best-effort: a trace failure never
// disturbs the caller (the hook contract forbids blocking/erroring).
package trace

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

const capBytes = 1 << 20 // 1 MiB, then rotate to trace.log.1

var (
	disabled   = os.Getenv("DOTFILES_TRACE") == "0"
	verboseEnv = truthy(os.Getenv("DOTFILES_TRACE_VERBOSE"))
	verboseOpt atomic.Bool // live toggle, driven by the sidebar's 1s option poll
	logPath    = resolvePath()
)

func truthy(s string) bool {
	s = strings.TrimSpace(s)
	return s == "1" || s == "on" || s == "true"
}

// resolvePath mirrors the shell idiom: $XDG_STATE_HOME, else ~/.local/state.
func resolvePath() string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".local", "state")
	}
	return filepath.Join(base, "dotfiles", "trace.log")
}

// SetVerbose flips the live verbose gate (motion/tick events).
func SetVerbose(on bool) { verboseOpt.Store(on) }

// Log records an always-on edge. kv is alternating key(string), value(any).
func Log(src, evt string, kv ...any) { write(src, evt, kv) }

// Logv records a verbose-only event; a no-op unless the gate or env is on.
func Logv(src, evt string, kv ...any) {
	if verboseOpt.Load() || verboseEnv {
		write(src, evt, kv)
	}
}

// Err renders an error for the "err" field ("" when nil).
func Err(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func write(src, evt string, kv []any) {
	if disabled {
		return
	}
	defer func() { _ = recover() }() // a trace bug must never crash the hook/TUI
	var b strings.Builder
	b.WriteString("ts=" + time.Now().Format("2006-01-02T15:04:05.000"))
	b.WriteString(" src=" + field(src) + " evt=" + field(evt) + " pid=" + strconv.Itoa(os.Getpid()))
	for i := 0; i+1 < len(kv); i += 2 {
		key, _ := kv[i].(string)
		if key == "" {
			continue
		}
		b.WriteString(" " + key + "=" + field(fmt.Sprint(kv[i+1])))
	}
	b.WriteByte('\n')
	emit(b.String())
}

func emit(line string) {
	_ = os.MkdirAll(filepath.Dir(logPath), 0o755) // idempotent; matches the CLI's mkdir -p
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line) // one write, <4KB => atomic append
	_ = f.Close()
	rotate()
}

func rotate() {
	if fi, err := os.Stat(logPath); err != nil || fi.Size() < capBytes {
		return
	}
	lock, err := os.OpenFile(logPath+".lock", os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer lock.Close()
	if syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return // another writer is rotating; our append already landed
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	if fi, err := os.Stat(logPath); err == nil && fi.Size() >= capBytes {
		_ = os.Rename(logPath, logPath+".1")
	}
}

// field logfmt-escapes a value, matching the shell CLI's esc().
func field(v string) string {
	v = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		return r
	}, v)
	if r := []rune(v); len(r) > 200 {
		v = string(r[:200])
	}
	if v == "" || strings.ContainsAny(v, " =\"\\") {
		v = `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v) + `"`
	}
	return v
}
