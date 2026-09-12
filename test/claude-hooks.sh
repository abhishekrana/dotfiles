#!/usr/bin/env bash
# Guards the Claude hook entries in the stowed settings.json.
#
# `herdr integration install claude` writes its SessionStart hook with the
# absolute home of whichever machine ran it. That file is stowed and tracked, so
# without normalising it a second machine adds a second entry, both get
# committed, and every machine then runs one hook path that does not exist -
# which Claude reports as a startup hook error on every single session.
#
# install.sh is sourced rather than run, the way bootstrap.sh sources it, so the
# function is exercised without installing anything.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
pass=0 fail=0

command -v jq >/dev/null 2>&1 || {
    echo "jq missing; skipping" >&2
    exit 0
}

# shellcheck source=/dev/null
source "$REPO/install.sh"

# install.sh defines its own ok(), so these come after sourcing it.
ok() {
    pass=$((pass + 1))
    printf '  \033[32m✓\033[0m %s\n' "$1"
}
no() {
    fail=$((fail + 1))
    printf '  \033[31m✗\033[0m %s\n' "$1"
    [ $# -gt 1 ] && printf '      %s\n' "$2"
}
eq() { [ "$2" = "$3" ] && ok "$1" || no "$1" "want [$2] got [$3]"; }

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

HERDR_HOOK='bash "$HOME/.claude/hooks/herdr-agent-state.sh" session'

fixture() {
    cat >"$TMP/settings.json" <<'JSON'
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "agentbar hook",
            "timeout": 10
          }
        ]
      },
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "bash \"$HOME/.claude/hooks/herdr-agent-state.sh\" session",
            "timeout": 10
          }
        ]
      },
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "bash '/home/one/.claude/hooks/herdr-agent-state.sh' session",
            "timeout": 10
          }
        ]
      },
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "bash '/home/two/.claude/hooks/herdr-agent-state.sh' session",
            "timeout": 10
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "agentbar hook",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
JSON
}

herdr_commands() {
    jq -r '.hooks.SessionStart[].hooks[].command | select(test("herdr-agent-state"))' "$TMP/settings.json"
}

echo "claude hooks"

fixture
normalise_claude_hooks "$TMP/settings.json"
eq "three machines collapse to one entry" "1" "$(herdr_commands | wc -l)"
eq "the entry that survives is \$HOME-relative" "$HERDR_HOOK" "$(herdr_commands)"

# Running the installer twice must not grow the file.
before=$(cat "$TMP/settings.json")
normalise_claude_hooks "$TMP/settings.json"
eq "normalising again changes nothing" "$before" "$(cat "$TMP/settings.json")"

eq "other hooks on the same event are kept" "1" \
    "$(jq '[.hooks.SessionStart[].hooks[] | select(.command == "agentbar hook")] | length' "$TMP/settings.json")"
eq "other events are untouched" "1" "$(jq '.hooks.Stop | length' "$TMP/settings.json")"
eq "the timeout is preserved" "10" \
    "$(jq -r '.hooks.SessionStart[].hooks[] | select(.command | test("herdr")) | .timeout' "$TMP/settings.json")"

# A file the installer has never touched must come through unharmed.
printf '{"hooks":{}}\n' >"$TMP/settings.json"
normalise_claude_hooks "$TMP/settings.json"
eq "a file with no hooks survives" '{"hooks":{}}' "$(jq -c . "$TMP/settings.json")"

# Stow points ~/.claude/settings.json at the repo, and replacing that symlink
# would silently detach the file from the repo.
ln -s "$TMP/settings.json" "$TMP/link.json"
printf '{"hooks":{}}\n' >"$TMP/settings.json"
normalise_claude_hooks "$TMP/link.json"
[ -L "$TMP/link.json" ] && ok "the stow symlink is not replaced" || no "the stow symlink is not replaced"

normalise_claude_hooks "$TMP/does-not-exist.json"
eq "a missing file is not an error" "0" "$?"

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
