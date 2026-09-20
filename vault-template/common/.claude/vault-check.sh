#!/usr/bin/env bash
# vault-check - deterministic integrity check for the notes vault. Detects broken and
# untidy states; it does NOT fix them (that is the managing-vault skill's job). Run it
# manually, from the git pre-commit hook, or by managing-vault after a change.
#
# Exit 1 on HARD errors (unclosed frontmatter, empty notes, duplicate note names) so a
# commit is blocked. WARNINGS (unresolved [[links]], filed notes missing `type`) print
# but do not fail - forward-links to not-yet-created notes are a supported convention.
set -uo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root" || exit 0

mapfile -t notes < <(find . -type f -name '*.md' \
    -not -path './.git/*' -not -path './.claude/*' -not -path './templates/*' \
    -not -path './assets/*' 2>/dev/null | sed 's|^\./||')
[ "${#notes[@]}" -eq 0 ] && exit 0

hard=0

# HARD: frontmatter opened but never closed.
for f in "${notes[@]}"; do
    [ "$(head -1 "$f")" = "---" ] || continue
    awk 'NR>1 && $0=="---"{ok=1; exit} END{exit ok?0:1}' "$f" ||
        {
            echo "ERROR unclosed frontmatter: $f" >&2
            hard=1
        }
done

# HARD: frontmatter that does not parse as YAML. Agents write these fields, and a bare
# GitLab reference (mr: !6087) is a YAML tag, not a string - it must be quoted.
if command -v python3 >/dev/null 2>&1; then
    for f in "${notes[@]}"; do
        [ "$(head -1 "$f")" = "---" ] || continue
        python3 - "$f" <<'PYCHECK' || {
import sys

try:
    import yaml
except ImportError:
    sys.exit(0)
lines = open(sys.argv[1], encoding="utf-8").read().split("\n")
if not lines or lines[0].strip() != "---":
    sys.exit(0)
try:
    end = lines.index("---", 1)
except ValueError:
    sys.exit(0)
try:
    yaml.safe_load("\n".join(lines[1:end]))
except Exception:
    sys.exit(1)
PYCHECK
            echo "ERROR frontmatter is not valid YAML: $f" >&2
            hard=1
        }
    done
fi

# HARD: empty notes (no non-whitespace content).
for f in "${notes[@]}"; do
    grep -q '[^[:space:]]' "$f" || {
        echo "ERROR empty note: $f" >&2
        hard=1
    }
done

# HARD: duplicate note filenames (ambiguous [[link]] targets).
while IFS= read -r d; do
    [ -n "$d" ] && {
        echo "ERROR duplicate note name: $d" >&2
        hard=1
    }
done < <(printf '%s\n' "${notes[@]}" | sed 's|.*/||' | sort | uniq -d)

# WARN: unresolved [[wiki-links]] - forward-links are allowed, so warn only.
declare -A have=()
for f in "${notes[@]}"; do have["$(basename "$f" .md)"]=1; done
for f in "${notes[@]}"; do
    while IFS= read -r t; do
        [ -z "$t" ] && continue
        [ -n "${have[$t]:-}" ] || echo "warn unresolved link [[$t]] in $f"
    done < <(sed 's/`[^`]*`//g' "$f" | grep -oE '\[\[[^]]+\]\]' | sed -E 's/^\[\[|\]\]$//g; s/\|.*$//; s/#.*$//')
done

# HARD: a filed note declares a `type:`, and which values are allowed depends on the
# directory holding it. A work note names what sort of work it is; every other
# directory names itself. Validating against the directory also catches a misfiled
# note, which one flat vocabulary cannot.
work_types="ticket adhoc spike review incident"
for f in "${notes[@]}"; do
    case "$f" in
        work/* | archive/work/*) allowed="$work_types" ;;
        knowledge/*) allowed="knowledge" ;;
        log/*) allowed="log" ;;
        design/*) allowed="design" ;;
        archive/*) allowed="$work_types knowledge log design" ;;
        *) continue ;;
    esac
    t=$(head -20 "$f" | sed -n 's/^type:[[:space:]]*//p' | head -1)
    if [ -z "$t" ]; then
        echo "ERROR filed note missing type: $f" >&2
        hard=1
    elif ! printf '%s\n' $allowed | grep -qx "$t"; then
        echo "ERROR type '$t' not allowed in $(dirname "$f")/ (expected: $allowed): $f" >&2
        hard=1
    fi
done

if [ "$hard" -ne 0 ]; then
    echo "vault-check: fix the ERROR lines above." >&2
    exit 1
fi
exit 0
