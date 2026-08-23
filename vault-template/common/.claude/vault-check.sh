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

# WARN: filed notes (not inbox/dailies) missing a `type:` frontmatter field.
for f in "${notes[@]}"; do
    case "$f" in
        projects/* | areas/* | resources/* | archive/*)
            head -20 "$f" | grep -qE '^type:[[:space:]]' || echo "warn filed note missing type: $f"
            ;;
    esac
done

if [ "$hard" -ne 0 ]; then
    echo "vault-check: fix the ERROR lines above." >&2
    exit 1
fi
exit 0
