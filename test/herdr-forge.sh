#!/usr/bin/env bash
# Guards herdr-forge: the tab bar line for each merge request state, and when it says nothing.
#
# glab is a stub that prints a canned GraphQL answer, so no network is touched.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FORGE=$REPO/herdr/.local/bin/herdr-forge
pass=0 fail=0

command -v jq >/dev/null 2>&1 || {
    echo "jq missing; skipping" >&2
    exit 0
}

ok() {
    pass=$((pass + 1))
    printf '  \033[32m✓\033[0m %s\n' "$1"
}
no() {
    fail=$((fail + 1))
    printf '  \033[31m✗\033[0m %s\n' "$1"
    printf '      %s\n' "$2"
}
eq() { [ "$2" = "$3" ] && ok "$1" || no "$1" "want [$2] got [$3]"; }

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
export XDG_CACHE_HOME=$TMP/cache

# The stub answers with $TMP/answer.json, or fails when it is absent.
mkdir -p "$TMP/bin"
cat >"$TMP/bin/glab" <<EOF
#!/usr/bin/env bash
[ -r "$TMP/answer.json" ] || exit 1
cat "$TMP/answer.json"
EOF
chmod +x "$TMP/bin/glab"
# The stub browser writes down what it was asked to open.
cat >"$TMP/bin/xdg-open" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$1" >>"$TMP/opened"
EOF
chmod +x "$TMP/bin/xdg-open"
export PATH=$TMP/bin:$PATH

CO=$TMP/co
git init -q -b main "$CO"
git -C "$CO" config user.email t@t
git -C "$CO" config user.name t
git -C "$CO" remote add origin git@gitlab.example:group/project.git
: >"$CO/f"
git -C "$CO" add f
git -C "$CO" commit -qm init

# answer <branch> <node json|""> writes the stub's answer.
answer() {
    if [ -n "$2" ]; then
        jq -nc --arg b "$1" --argjson n "$2" '{data: {project: {webUrl: "https://gitlab.example/group/project",
            mergeRequests: {nodes: [$n + {sourceBranch: $b}]}}}}'
    else
        echo '{"data":{"project":{"webUrl":"https://gitlab.example/group/project","mergeRequests":{"nodes":[]}}}}'
    fi >"$TMP/answer.json"
}

# line <branch>: refresh synchronously, then print the line without starting another refresh.
line() {
    git -C "$CO" checkout -q -B "$1"
    "$FORGE" refresh "$CO" "$1"
    HERDR_ACTIVE_PANE_CWD=$CO HERDR_FORGE_TTL=99999 "$FORGE" line
}

node() {
    printf '{"iid":"%s","state":"%s","draft":%s,"detailedMergeStatus":"%s","approvalsLeft":%s,"headPipeline":%s}' \
        "$1" "$2" "$3" "$4" "$5" "${6:-null}"
}

echo "herdr-forge"

answer 123-feature "$(node 45 opened false NOT_APPROVED 1 '{"status":"SUCCESS"}')"
eq "an open MR names what it waits on, and its pipeline" \
    "#123 · !45 · needs 1 approval · CI ✓" "$(line 123-feature)"

answer 123-feature "$(node 45 opened false NOT_APPROVED 2 '{"status":"FAILED"}')"
eq "approvals are plural past one" "#123 · !45 · needs 2 approvals · CI ✗" "$(line 123-feature)"

answer 123-feature "$(node 45 merged false NOT_OPEN 0 '{"status":"SUCCESS"}')"
eq "a merged MR drops its pipeline" "#123 · !45 · merged" "$(line 123-feature)"

answer 12-y "$(node 7 opened true DRAFT_STATUS 1 '{"status":"RUNNING"}')"
eq "a draft says draft, not what it waits on" "#12 · !7 · draft · CI ●" "$(line 12-y)"

answer slim "$(node 8 opened false MERGEABLE 0 '{"status":"SUCCESS"}')"
eq "a branch with no ticket number still shows its MR" "!8 · ready · CI ✓" "$(line slim)"

answer 124-idea ""
eq "a ticket branch with no MR says so" "#124 · no MR" "$(line 124-idea)"

answer plain ""
eq "no ticket and no MR print nothing" "" "$(line plain)"

jq -nc '{data: {project: {mergeRequests: {nodes: [{iid: "9", state: "opened", sourceBranch: "someone-else"}]}}}}' \
    >"$TMP/answer.json"
eq "an MR from another branch is not this branch's" "#77 · no MR" "$(line 77-mine)"

answer 123-feature "$(node 45 opened false MERGEABLE 0 '{"status":"SUCCESS"}')"
line 123-feature >/dev/null
rm -f "$TMP/answer.json"
eq "a failed ask keeps the last answer" "#123 · !45 · ready · CI ✓" "$(line 123-feature)"

entry=$(find "$XDG_CACHE_HOME/herdr-forge" -name '*123_feature' -type f)
printf '%s\n%s\n%s\n' 1 "$EPOCHSECONDS" "#123 · !45 · ready" >"$entry"
eq "an answer older than STALE is dropped" "" "$(HERDR_ACTIVE_PANE_CWD=$CO HERDR_FORGE_TTL=99999 "$FORGE" line)"

git -C "$CO" checkout -q --detach
eq "a detached HEAD prints nothing" "" "$(HERDR_ACTIVE_PANE_CWD=$CO "$FORGE" line)"
eq "outside a checkout prints nothing" "" "$(HERDR_ACTIVE_PANE_CWD=$TMP "$FORGE" line)"

# A due entry starts a detached refresh, and the answer lands without line waiting on it.
git -C "$CO" checkout -q -B 31-due
answer 31-due "$(node 31 opened false MERGEABLE 0 null)"
HERDR_ACTIVE_PANE_CWD=$CO "$FORGE" line >/dev/null
got=
for _ in $(seq 50); do
    got=$(HERDR_ACTIVE_PANE_CWD=$CO HERDR_FORGE_TTL=99999 "$FORGE" line)
    [ -n "$got" ] && break
    sleep 0.1
done
eq "a due entry refreshes in the background" "#31 · !31 · ready" "$got"

echo "herdr-forge: the open chooser"

WEB=https://gitlab.example/group/project
# opens <branch> <key>: the chooser for that branch with one key pressed, and what the browser was given.
opens() {
    git -C "$CO" checkout -q -B "$1"
    rm -f "$TMP/opened"
    printf '%s' "$2" | HERDR_ACTIVE_PANE_CWD=$CO "$FORGE" open >"$TMP/menu"
    for _ in $(seq 20); do
        [ -s "$TMP/opened" ] && break
        sleep 0.05
    done
    cat "$TMP/opened" 2>/dev/null
}
menu_has() { grep -qF -- "$1" "$TMP/menu"; }

answer 123-feature "$(jq -nc --arg w "$WEB" '{iid: "45", state: "opened", draft: false,
    detailedMergeStatus: "NOT_APPROVED", approvalsLeft: 1, webUrl: ($w + "/-/merge_requests/45"),
    headPipeline: {status: "SUCCESS", path: "/group/project/-/pipelines/900"}}')"
line 123-feature >/dev/null
eq "m opens the merge request" "$WEB/-/merge_requests/45" "$(opens 123-feature m)"
eq "t opens the ticket" "$WEB/-/issues/123" "$(opens 123-feature t)"
eq "p opens the pipeline, on the project's host" "$WEB/-/pipelines/900" "$(opens 123-feature p)"
eq "any other key opens nothing" "" "$(opens 123-feature q)"
menu_has $'\e]8;;'"$WEB/-/merge_requests/45"$'\e\\' && ok "each row is a Ctrl+clickable link" ||
    no "each row is a Ctrl+clickable link" "no OSC 8 link to the MR in the chooser"
menu_has "!45 · needs 1 approval" && ok "the MR row says what it waits on" ||
    no "the MR row says what it waits on" "$(cat "$TMP/menu")"

answer 124-idea ""
line 124-idea >/dev/null
eq "a ticket with no MR still opens" "$WEB/-/issues/124" "$(opens 124-idea t)"
menu_has "merge request" && no "no MR, no MR row" "the chooser offers an MR" || ok "no MR, no MR row"

# An entry written before the links existed is asked again when the chooser opens.
git -C "$CO" checkout -q -B 125-old
answer 125-old "$(jq -nc --arg w "$WEB" '{iid: "46", state: "opened", webUrl: ($w + "/-/merge_requests/46")}')"
entry=$(find "$XDG_CACHE_HOME/herdr-forge" -maxdepth 1 -name '*123_feature' -type f)
[ -n "$entry" ] && printf '%s\n%s\n%s\n' "$EPOCHSECONDS" "$EPOCHSECONDS" "#125 · !46" >"${entry%123_feature}125_old"
eq "an entry with no links is refreshed before choosing" "$WEB/-/merge_requests/46" "$(opens 125-old m)"

answer plain ""
opens plain m >/dev/null
menu_has "Nothing to open" && ok "a branch with nothing says so" ||
    no "a branch with nothing says so" "$(cat "$TMP/menu")"

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
