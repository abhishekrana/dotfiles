#!/usr/bin/env bash
# Guards herdr-forge: the tab bar line for each merge request state and when it says nothing, and the
# popup's columns, buttons and actions.
#
# glab, herdr and xdg-open are stubs, so no network, server or browser is touched.
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

# The stub answers from a file per call - answer.json for the MR query, jobs.json for the pipeline's job
# counts, closes.json, issue.json and notes.json for the REST calls - and fails when that file is absent.
mkdir -p "$TMP/bin"
cat >"$TMP/bin/glab" <<EOF
#!/usr/bin/env bash
case "\$*" in
    *"fragment jobs"*) f=jobs.json ;;
    *graphql*) f=answer.json ;;
    *closes_issues*) f=closes.json ;;
    */notes*) f=notes.json ;;
    *issues/*) f=issue.json ;;
esac
[ -r "$TMP/\$f" ] || exit 1
cat "$TMP/\$f"
EOF
chmod +x "$TMP/bin/glab"
# The stub herdr writes down every call, answers a split or a new tab with a new pane, and lists the tabs in
# tabs.json.
cat >"$TMP/bin/herdr" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >>"$TMP/herdr.log"
case "\$1 \$2" in
    "pane split") echo '{"result":{"pane":{"pane_id":"w1:p9"}}}' ;;
    "tab create") echo '{"result":{"type":"tab_created","tab":{"tab_id":"w1:t5"},"root_pane":{"pane_id":"w1:p20"}}}' ;;
    "tab list") cat "$TMP/tabs.json" 2>/dev/null || echo '{"result":{"tabs":[]}}' ;;
esac
exit 0
EOF
chmod +x "$TMP/bin/herdr"
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

echo "herdr-forge: the popup"

POPUP=$REPO/herdr/.local/bin/herdr-forge-popup
WEB=https://gitlab.example/group/project
iso() { date -u -d "@$(($(date +%s) - $1))" +%Y-%m-%dT%H:%M:%SZ; }

# mr <jq object merged over a plain open MR> writes the MR query's answer for branch 123-feature, with ticket #123.
mr() {
    local extra=${1:-'{}'}
    jq -nc --arg w "$WEB" --arg started "$(iso 372)" '{data: {
        currentUser: {username: "you"},
        project: {webUrl: $w,
            mergeRequests: {nodes: [({iid: "45", title: "Fix the queue", state: "opened", draft: false,
                webUrl: ($w + "/-/merge_requests/45"), sourceBranch: "123-feature", targetBranch: "main",
                detailedMergeStatus: "NOT_APPROVED", approvalsRequired: 2, approvalsLeft: 1, autoMergeStrategy: null,
                mergeTrainsCount: 0, mergeTrainCar: null, conflicts: false,
                resolvableDiscussionsCount: 5, resolvedDiscussionsCount: 3,
                diffStatsSummary: {additions: 1412, deletions: 37, fileCount: 9},
                assignees: {nodes: [{username: "you"}]}, labels: {nodes: [{title: "area::queue"}, {title: "Backend"}]},
                reviewers: {nodes: [{username: "rev-a", mergeRequestInteraction: {reviewState: "APPROVED"}},
                                    {username: "rev-b", mergeRequestInteraction: {reviewState: "REVIEW_STARTED"}}]},
                pipelines: {nodes: [{iid: "90", duration: null}, {iid: "89", duration: 660}]},
                headPipeline: {iid: "90", status: "RUNNING", path: "/group/project/-/pipelines/900",
                    ref: "refs/merge-requests/45/merge", startedAt: $started, duration: null,
                    stages: {nodes: [{status: "success"}, {status: "running"}, {status: "created"}]}}}
                + ('"$extra"'))]},
            workItems: {nodes: [{iid: "123", title: "Save-as-failed episodes never reach the queue",
                webUrl: ($w + "/-/issues/123"), state: "OPEN",
                widgets: [{status: {name: "In development"}}, {assignees: {nodes: [{username: "you"}]}},
                          {labels: {nodes: [{title: "type::bug"}]}}, {parent: {iid: "98", title: "Reliability"},
                          children: {count: 0}}, {weight: null}, {milestone: null},
                          {iteration: {title: null, dueDate: "2026-10-02"}}, {dueDate: null},
                          {timeEstimate: 0, totalTimeSpent: 0}, {closingMergeRequests: {nodes: []}}]}]}}}}' \
        >"$TMP/answer.json"
}

# jobs <jq object merged over the pipeline's job counts> writes the job query's answer.
jobs() {
    local extra=${1:-'{}'}
    jq -nc '{data: {project: {mergeRequests: {nodes: [{sourceBranch: "123-feature",
        headPipeline: ({ok: {count: 20}, bad: {count: 0}, busy: {count: 3}, wait: {count: 0}, todo: {count: 7},
            manual: {count: 7}, failed: {nodes: []}, queued: {nodes: []},
            downstream: {nodes: [{status: "RUNNING", sourceJob: {name: "images"}, ok: {count: 30},
                bad: {count: 0}, busy: {count: 6}, wait: {count: 0}, todo: {count: 0}, manual: {count: 40},
                failed: {nodes: []}, queued: {nodes: []}}]}} + ('"$extra"'))}]}}}}' >"$TMP/jobs.json"
}

on() { git -C "$CO" checkout -q -B "$1"; }
frame() { HERDR_ACTIVE_PANE_CWD=$CO "$POPUP" --dump "${1:-200}" | sed 's/\x1b\[[0-9;]*m//g' >"$TMP/frame"; }
has() { grep -qF -- "$1" "$TMP/frame"; }
check() { has "$2" && ok "$1" || no "$1" "no [$2] in: $(grep -v '^region' "$TMP/frame" | tr -s ' ' | head -c 900)"; }
# The columns each action's buttons cover on their row, as "x0-x1".
region() { awk -v a="$1" '$1 == "region" && $5 == a {print $3 "-" $4; exit}' "$TMP/frame"; }

on 123-feature
mr
jobs
frame 200
check "the ticket, its title and fields" "#123  Save-as-failed episodes never reach the queue"
check "the ticket's status" "Status     In development"
check "the ticket's parent" "Parent     #98 Reliability"
check "the MR says what it waits on" "!45  needs 1 more approval"
check "approvals are counted" "Approvals  1 of 2 · needs 1 more"
check "each reviewer has a state" "✓ @rev-a approved"
check "threads are counted" "Threads    3 of 5 resolved"
check "a big diff is shown in thousands" "+1.4k −37 · 9 files → main"
check "a scoped label is a chip in two halves" " area  queue "
check "the pipeline runs against its last run" "running · 6m 12s of ~11m"
check "the bar counts the jobs that run, children included" "50 of 66 jobs"
check "manual jobs are one count" "manual     47 jobs, not run"
check "a child pipeline has a line" "● images"
order=$(awk '/TICKET/ {t = index($0, "TICKET"); m = index($0, "MERGE REQUEST"); p = index($0, "PIPELINE")
    print (t < m && m < p)}' "$TMP/frame")
[ "$order" = 1 ] && ok "the columns read ticket, MR, pipeline" ||
    no "the columns read ticket, MR, pipeline" "$(grep TICKET "$TMP/frame")"
eq "every section has Browser, Split and New tab buttons" "t T tab-t m M tab-m p P tab-p" \
    "$(awk '$1 == "region" {print $5}' "$TMP/frame" | awk '!seen[$0]++' | tr '\n' ' ' | sed 's/ $//')"
[ "$(region m | cut -d- -f2)" -le "$(region M | cut -d- -f1)" ] &&
    [ "$(region M | cut -d- -f2)" -le "$(region tab-m | cut -d- -f1)" ] &&
    ok "the buttons read Browser, Split, New tab" ||
    no "the buttons read Browser, Split, New tab" "m $(region m), M $(region M), tab-m $(region tab-m)"
too_wide=$(grep -v '^region' "$TMP/frame" | python3 -c 'import sys; print(max(len(l.rstrip("\n")) for l in sys.stdin))')
[ "$too_wide" -le 200 ] && ok "no line is wider than the popup" ||
    no "no line is wider than the popup" "$too_wide columns"

frame 150
[ "$(grep -c -E '^ +(TICKET|MERGE REQUEST|PIPELINE)$' "$TMP/frame")" = 3 ] && ok "a narrow popup stacks the three" ||
    no "a narrow popup stacks the three" "$(grep -E 'TICKET|MERGE|PIPELINE' "$TMP/frame")"
eq "stacked, every section still has all three buttons" 9 \
    "$(awk '$1 == "region" {print $5}' "$TMP/frame" | sort -u | wc -l)"

mr '{mergeTrainCar: {index: 1}, mergeTrainsCount: 4, autoMergeStrategy: "merge_train"}'
frame
check "a train names its place" "!45  in the merge train · 2nd of 4"
mr '{autoMergeStrategy: "add_to_merge_train_when_checks_pass"}'
frame
check "a queued MR says it joins the train" "joins the merge train when checks pass"

mr '{reviewers: {nodes: [range(6) | {username: "r\(.)", mergeRequestInteraction: {reviewState: "UNREVIEWED"}}]}}'
frame
check "past three reviewers, the rest are counted" "and 3 more"

mr '{conflicts: true}'
jobs '{bad: {count: 2},
       failed: {nodes: [{name: "unit-tests", allowFailure: false}, {name: "lint", allowFailure: false}]},
       queued: {nodes: [{name: "e2e", queuedAt: "'"$(iso 600)"'"}]}}'
frame
check "conflicts are called out" "Conflicts  yes: rebase before merging"
check "failed jobs are named" "✗ 2 failed: unit-tests, lint"
check "a job waiting for a runner is called out" "⚠ 1 job waiting for a runner, longest 10m"
jobs

# A branch with no number finds its ticket through what its MR closes.
on fix-queue
mr '{sourceBranch: "fix-queue"}'
jq '.data.project.mergeRequests.nodes[0].sourceBranch = "fix-queue"' "$TMP/jobs.json" >"$TMP/j" &&
    mv "$TMP/j" "$TMP/jobs.json"
echo '[{"iid": 123}]' >"$TMP/closes.json"
frame
check "a ticket found through the MR says so" "linked by !45, which closes it"
echo '[]' >"$TMP/closes.json"
jq '.data.project.workItems.nodes = []' "$TMP/answer.json" >"$TMP/a" && mv "$TMP/a" "$TMP/answer.json"
frame
check "no number and nothing closed: no ticket" "No linked ticket"
eq "no ticket, no ticket buttons" "" "$(region t)"

on 123-feature
mr
jobs
act() { : >"$TMP/herdr.log" && rm -f "$TMP/opened" &&
    HERDR_ACTIVE_PANE_CWD=$CO HERDR_ACTIVE_PANE_ID=w1:p1 HERDR_ACTIVE_WORKSPACE_ID=w1 HERDR_BIN_PATH=$TMP/bin/herdr \
        "$POPUP" --act "$1"; }
opened() {
    for _ in $(seq 20); do
        [ -s "$TMP/opened" ] && break
        sleep 0.05
    done
    cat "$TMP/opened" 2>/dev/null
}
act m
eq "m opens the MR in the browser" "$WEB/-/merge_requests/45" "$(opened)"
act p
eq "p opens the pipeline, on the project's host" "$WEB/-/pipelines/900" "$(opened)"
act t
eq "t opens the ticket" "$WEB/-/issues/123" "$(opened)"
act M
grep -qF "pane split --pane w1:p1 --direction right --cwd $CO" "$TMP/herdr.log" &&
    ok "M splits the focused pane to the right" || no "M splits the focused pane to the right" "$(cat "$TMP/herdr.log")"
grep -F "git diff --no-color origin/" "$TMP/herdr.log" | grep -qF "main" &&
    grep -qF "...HEAD | hunk patch -" "$TMP/herdr.log" && ! grep -qF -- "--sidebar" "$TMP/herdr.log" &&
    ok "M shows the MR's diff in hunk" || no "M shows the MR's diff in hunk" "$(cat "$TMP/herdr.log")"
act P
grep -qF "glab ci view -p 900" "$TMP/herdr.log" && ok "P opens the pipeline in glab ci view" ||
    no "P opens the pipeline in glab ci view" "$(cat "$TMP/herdr.log")"
act T
grep -F "ticket-md " "$TMP/herdr.log" | grep -qF "group/project" && grep -qF " 123 | folio" "$TMP/herdr.log" &&
    ok "T reads the ticket in folio" ||
    no "T reads the ticket in folio" "$(cat "$TMP/herdr.log")"

act tab-m
grep -qF "tab create --workspace w1 --cwd $CO --label diff !45 --focus" "$TMP/herdr.log" &&
    ok "New tab opens a tab named for what it shows" ||
    no "New tab opens a tab named for what it shows" "$(cat "$TMP/herdr.log")"
grep -F "pane run w1:p20 " "$TMP/herdr.log" | grep -qF "hunk patch --sidebar -" &&
    ok "the diff's tab shows hunk with its file list" ||
    no "the diff's tab shows hunk with its file list" "$(cat "$TMP/herdr.log")"
echo '{"result":{"tabs":[{"tab_id":"w1:t3","label":"diff !45"}]}}' >"$TMP/tabs.json"
act tab-m
grep -qF "tab focus w1:t3" "$TMP/herdr.log" && ! grep -qF "tab create" "$TMP/herdr.log" &&
    ok "New tab again focuses that tab, and opens no second one" ||
    no "New tab again focuses that tab, and opens no second one" "$(cat "$TMP/herdr.log")"
rm -f "$TMP/tabs.json"
act tab-t
grep -qF -- "--label ticket #123" "$TMP/herdr.log" && ok "the ticket's tab is named for it" ||
    no "the ticket's tab is named for it" "$(cat "$TMP/herdr.log")"
act tab-p
grep -qF -- "--label jobs !45" "$TMP/herdr.log" && grep -qF "glab ci view -p 900" "$TMP/herdr.log" &&
    ok "the pipeline's tab runs glab ci view" || no "the pipeline's tab runs glab ci view" "$(cat "$TMP/herdr.log")"

jq -nc '{title: "Save-as-failed", state: "opened", labels: ["type::bug"], assignees: [{username: "you"}],
    description: "Episodes are skipped."}' >"$TMP/issue.json"
jq -nc '[{system: true, body: "changed the status", author: {username: "bot"}, created_at: "2026-09-01T00:00:00Z"},
         {system: false, body: "Seen on two stations.", author: {username: "rev-a"},
          created_at: "2026-09-02T00:00:00Z"}]' \
    >"$TMP/notes.json"
md=$(cd "$CO" && "$POPUP" ticket-md group/project 123)
case $md in *"# #123 Save-as-failed"*"Episodes are skipped."*"**@rev-a** · 2026-09-02"*"Seen on two stations."*)
    ok "the ticket reads as markdown, with its comments"
    ;;
*) no "the ticket reads as markdown, with its comments" "$md" ;;
esac
case $md in *"changed the status"*) no "system notes are left out" "$md" ;; *) ok "system notes are left out" ;; esac

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
