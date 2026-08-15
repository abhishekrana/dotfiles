#!/usr/bin/env bash
# Guards gwtm, the one alias that rewrites history. All it has to get right is
# WHICH tool it picks: rebase when the branch is unpushed, merge when it is on
# origin. The wrong one either buries the branch in merge commits or rewrites
# published history, and neither is visible until later. Run against a throwaway
# origin.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
pass=0 fail=0

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
has() { case "$2" in *"$3"*) ok "$1" ;; *) no "$1" "[$2] lacks [$3]" ;; esac }

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
# Own git identity and config: the suite must not read the machine's, and
# pull.rebase=false is the setting that produces the merge commits case B is about.
export GIT_CONFIG_GLOBAL="$TMP/gitconfig" GIT_CONFIG_SYSTEM=/dev/null
git config --global user.email t@t
git config --global user.name t
git config --global init.defaultBranch main
git config --global pull.rebase false

# bind(1) has no line editing to attach to in a script - the only noise sourcing
# an interactive rc file makes here.
# shellcheck source=bash/.bashrc.d/aliases.bash
. "$REPO/bash/.bashrc.d/aliases.bash" 2>/dev/null

# --- fixture: a bare origin, a seed clone that moves main, and the worktree ----
git init -q --bare "$TMP/origin.git"
git clone -q "$TMP/origin.git" "$TMP/seed" 2>/dev/null
cd "$TMP/seed" || exit 1
echo base >f.txt
git add -A
git commit -qm base
git push -q origin main

WT="$TMP/wt-9" # the branch gwtm acts on is named after this directory

advance() { # one new commit on origin/main
    git -C "$TMP/seed" pull -q origin main 2>/dev/null
    echo "$1" >>"$TMP/seed/main.txt"
    git -C "$TMP/seed" add -A
    git -C "$TMP/seed" commit -qm "main: $1"
    git -C "$TMP/seed" push -q origin main
}
fresh() { # a clean wt-9 checkout, and no wt-9 on the remote
    cd "$TMP" || exit 1
    rm -rf "$WT"
    git -C "$TMP/seed" push -q origin --delete wt-9 2>/dev/null
    git clone -q "$TMP/origin.git" "$WT" 2>/dev/null
    cd "$WT" || exit 1
    git checkout -q -B wt-9 origin/main
    git branch -q -D main 2>/dev/null
}
state() { # ahead/behind origin/main, as the local refs see it
    echo "$(git rev-list --count origin/main..HEAD)/$(git rev-list --count HEAD..origin/main)"
}
CLASH=0
# A commit here and a conflicting one on main, same file. Counted, so each round
# differs from the last - re-writing the same content leaves main nothing to commit.
clash() {
    CLASH=$((CLASH + 1))
    echo "ours $CLASH" >clash.txt
    git add -A
    git commit -qm "ours on clash"
    git -C "$TMP/seed" pull -q origin main
    echo "theirs $CLASH" >"$TMP/seed/clash.txt"
    git -C "$TMP/seed" add -A
    git -C "$TMP/seed" commit -qm "theirs on clash"
    git -C "$TMP/seed" push -q origin main
}

printf '\nclean branch, main moved\n'
fresh
advance one
gwtm >/dev/null 2>&1
eq "fast-forwards to main" "0/0" "$(state)"

printf '\nown commits, never pushed\n'
fresh
echo mine >mine.txt
git add -A
git commit -qm "my work"
advance two
git pull -q --no-edit origin main # the merge commit a plain `git pull` leaves
advance three
git fetch -q origin
eq "diverged: 2 of ours (one a merge), 1 behind" "2/1" "$(state)"
gwtm >/dev/null 2>&1
eq "succeeds" "0" "$?"
eq "our commit on top of main, nothing behind" "1/0" "$(state)"
eq "the pull merge is gone" "0" "$(git log --oneline --merges origin/main..HEAD | grep -c .)"
eq "our work survived" "mine" "$(cat mine.txt)"

printf '\nown commits, branch is on origin\n'
fresh
echo mine >mine.txt
git add -A
git commit -qm "my work"
git push -q -u origin wt-9
pushed=$(git rev-parse HEAD)
advance four
gwtm >/dev/null 2>&1
eq "up to date with main" "0" "$(git rev-list --count HEAD..origin/main)"
eq "the pushed commit was not rewritten" "1" "$(git rev-list HEAD | grep -c "$pushed")"

# A conflict is git's to report and yours to resolve - gwtm adds nothing. What it
# must not do is lose the branch or pick the tool that rewrites a pushed one.
printf '\nconflict, never pushed\n'
fresh
clash
gwtm >/dev/null 2>&1
eq "fails" "1" "$?"
eq "left in the rebase, to resolve or abort" "1" "$(git rev-parse -q --verify REBASE_HEAD | grep -c .)"
git rebase --abort

printf '\nconflict, branch is on origin\n'
fresh
echo seed >seeded.txt
git add -A
git commit -qm "published work"
git push -q -u origin wt-9
clash
before=$(git rev-parse HEAD)
gwtm >/dev/null 2>&1
eq "fails" "1" "$?"
eq "a merge, not a rewrite" "1" "$(git rev-parse -q --verify MERGE_HEAD | grep -c .)"
eq "the pushed tip is still there" "$before" "$(git rev-parse HEAD)"
git merge --abort

printf '\ndirty tree\n'
fresh
advance five
echo scratch >dirty.txt
git add dirty.txt
gwtm >/dev/null 2>&1
eq "autostash lets it fast-forward" "0/0" "$(state)"
eq "the uncommitted edit came back" "scratch" "$(cat dirty.txt 2>/dev/null)"

printf '\nno branch for this worktree yet\n'
cd "$TMP" || exit 1
rm -rf "$WT"
git clone -q "$TMP/origin.git" "$WT" 2>/dev/null
cd "$WT" || exit 1
git branch -q -D wt-9 2>/dev/null
gwtm >/dev/null 2>&1
eq "creates one named after the dir" "wt-9" "$(git branch --show-current)"
eq "off the latest main" "0/0" "$(state)"

printf '\nnot a git repo\n'
cd "$TMP" || exit 1
mkdir -p "$TMP/plain"
cd "$TMP/plain" || exit 1
out=$(gwtm 2>&1)
eq "refuses" "1" "$?"
has "says why" "$out" "not in a git repo"

printf '\n%s passed, %s failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
