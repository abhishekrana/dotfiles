#!/usr/bin/env bash
# PreToolUse guard: deny file access that escapes this vault's root.
#
# The hard wall between the personal and work vaults. An agent running in one vault
# must never read or write the other (or anywhere else on disk). Claude Code passes the
# tool input as JSON on stdin; exit 2 blocks the call and returns stderr to the agent.
set -euo pipefail

input="$(cat)"
root="${CLAUDE_PROJECT_DIR:-$PWD}"
path="$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty')"
[ -z "$path" ] && exit 0

case "$path" in
  /*) abs="$path" ;;
  *)  abs="$root/$path" ;;
esac

# realpath -m resolves .. and symlinks without requiring the path to exist yet.
abs="$(realpath -m "$abs")"
rootabs="$(realpath -m "$root")"

case "$abs" in
  "$rootabs" | "$rootabs"/*) exit 0 ;; # inside the vault - allow
  *)
    echo "Blocked: '$abs' is outside this vault ($rootabs). Refusing cross-vault file access." >&2
    exit 2
    ;;
esac
