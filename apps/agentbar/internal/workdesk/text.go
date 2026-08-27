package workdesk

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// Column widths the picker's rows are built from. The title is padded here rather
// than by the caller so every render agrees, and so the byte-versus-codepoint bug
// that dogged the shell version cannot come back: Pad counts runes, once.
const titleWidth = 40

// Short truncates to n display cells, marking the cut with a single ellipsis. The
// ellipsis replaces the last kept rune rather than being appended, so the result is
// exactly n cells wide and columns after it stay aligned.
func Short(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// Pad truncates to n cells and then right-pads to exactly n. Rune-based: the shell
// version padded with awk's %-Ns, which counts bytes, so every row whose title was
// truncated came out three columns short - "…" is one cell but three bytes.
func Pad(s string, n int) string {
	s = Short(s, n)
	if w := n - len([]rune(s)); w > 0 {
		return s + strings.Repeat(" ", w)
	}
	return s
}

// Age renders how long ago t was, in the picker's compact form: whole days, or
// "today" for anything under one. Callers pass now explicitly so every render is a
// pure function of its inputs and the tests need no clock.
func Age(t time.Time, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	// A timestamp in the future is clock skew between here and the forge, not
	// information, so any count below one day reads as "today".
	days := int(math.Floor(now.Sub(t).Seconds() / 86400))
	if days >= 1 {
		return strconv.Itoa(days) + "d"
	}
	return "today"
}

// AgeAgo is Age with the preposition a sentence needs. "today ago" is not a thing,
// so only the day counts take the suffix.
func AgeAgo(t time.Time, now time.Time) string {
	a := Age(t, now)
	if a == "" || a == "today" {
		return a
	}
	return a + " ago"
}

// SyncedLayout is how sync stamps meta.synced: local time, no zone, because it is read
// by people as often as by code.
const SyncedLayout = "2006-01-02 15:04:05"

// ParseTime decodes the timestamps this tool handles: ISO-8601 as GitLab returns it, and
// the local-time stamp sync writes into meta.json. A field we cannot parse yields the
// zero time, which Age renders as empty - a missing date is not worth failing a view
// over.
func ParseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.ParseInLocation(SyncedLayout, s, time.Local); err == nil {
		return t
	}
	return time.Time{}
}

// TSV joins fields the way the picker's awk reader expects, escaping the separators
// so a title containing a tab or a newline cannot invent a column or a row.
func TSV(fields ...string) string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = tsvEscape(f)
	}
	return strings.Join(out, "\t")
}

func tsvEscape(s string) string {
	if !strings.ContainsAny(s, "\\\t\n\r") {
		return s
	}
	r := strings.NewReplacer("\\", `\\`, "\t", `\t`, "\n", `\n`, "\r", `\r`)
	return r.Replace(s)
}

// join is strings.Join over only the non-empty parts, which is what every "a · b · c"
// line in these renders wants: an absent field leaves no orphan separator.
func join(sep string, parts ...string) string {
	var kept []string
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}
