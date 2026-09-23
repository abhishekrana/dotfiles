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
    "ci view"*)
        printf 'glab %s\n' "\$*" >>"$TMP/tools.log"
        exit 0
        ;;
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
# The stub herdr writes down every call, answers a new tab with a new pane, and lists the tabs in
# tabs.json.
cat >"$TMP/bin/herdr" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >>"$TMP/herdr.log"
case "\$1 \$2" in
    "tab create") echo '{"result":{"type":"tab_created","tab":{"tab_id":"w1:t5"},"root_pane":{"pane_id":"w1:p20"}}}' ;;
    "tab list") cat "$TMP/tabs.json" 2>/dev/null || echo '{"result":{"tabs":[]}}' ;;
    "pane list") cat "$TMP/panes.json" 2>/dev/null || echo '{"result":{"panes":[]}}' ;;
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
# The stub tools write down what they were given; while hold exists, folio stays open.
cat >"$TMP/bin/folio" <<EOF
#!/usr/bin/env bash
printf 'folio %s\n' "\$*" >>"$TMP/tools.log"
cp "\$1" "$TMP/page.seen" 2>/dev/null
while [ -e "$TMP/hold" ]; do sleep 0.05; done
EOF
cat >"$TMP/bin/hunk" <<EOF
#!/usr/bin/env bash
printf 'hunk %s\n' "\$*" >>"$TMP/tools.log"
EOF
chmod +x "$TMP/bin/folio" "$TMP/bin/hunk"
export GIT_SSH_COMMAND=false
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
check "the pipeline says how long it has run" "running · 6m 1"
check "the pipeline runs against its last run" "s of ~11m"
check "the bar counts the jobs that run, children included" "50 of 66 jobs"
check "manual jobs are one count" "manual     47 jobs, not run"
check "a child pipeline has a line" "● images"
order=$(awk '/TICKET/ {t = index($0, "TICKET"); m = index($0, "MERGE REQUEST"); p = index($0, "PIPELINE")
    print (t < m && m < p)}' "$TMP/frame")
[ "$order" = 1 ] && ok "the columns read ticket, MR, pipeline" ||
    no "the columns read ticket, MR, pipeline" "$(grep TICKET "$TMP/frame")"
eq "the toolbar's chips, then every section's two buttons" "r a close t T m M p P" \
    "$(awk '$1 == "region" {print $5}' "$TMP/frame" | awk '!seen[$0]++' | tr '\n' ' ' | sed 's/ $//')"
[ "$(region m | cut -d- -f2)" -le "$(region M | cut -d- -f1)" ] &&
    ok "the buttons read Browser, New tab" ||
    no "the buttons read Browser, New tab" "m $(region m), M $(region M)"
[ "$(awk '$1 == "region" && $5 == "close" {print $2; exit}' "$TMP/frame")" = 1 ] &&
    [ "$(region r | cut -d- -f2)" -le "$(region a | cut -d- -f1)" ] &&
    [ "$(region a | cut -d- -f2)" -le "$(region close | cut -d- -f1)" ] &&
    ok "the top line reads Refresh, All in tabs, Close" ||
    no "the top line reads Refresh, All in tabs, Close" "r $(region r), a $(region a), close $(region close)"
check "the stamp sits in the toolbar, before its chips" "ago    ↻  Refresh"
too_wide=$(grep -v '^region' "$TMP/frame" | python3 -c 'import sys; print(max(len(l.rstrip("\n")) for l in sys.stdin))')
[ "$too_wide" -le 200 ] && ok "no line is wider than the popup" ||
    no "no line is wider than the popup" "$too_wide columns"

# The finish: soft fill in Solarized's own roles - buttons and chips on background highlights (surface), labels in
# primary content (fg), nothing in emphasized content.
mkdir -p "$TMP/config/theme"
printf '_theme_surface="#010203"\n_theme_fg="#040506"\n_theme_emphasis="#070809"\n_theme_muted="#0a0b0c"\n' \
    >"$TMP/config/theme/colors.sh"
raw=$(cd "$CO" && XDG_CONFIG_HOME=$TMP/config HERDR_ACTIVE_PANE_CWD=$CO "$POPUP" --dump 200)
buttons=$(printf '%s\n' "$raw" | grep -F 'Browser')
case $buttons in *$'\e[34m'* | *$'\e[35m'*)
    no "buttons carry no colour of their own" "blue or magenta in the button row"
    ;;
*) ok "buttons carry no colour of their own" ;; esac
case $buttons in *┌* | *│*) no "buttons draw no outlines" "box lines in the button row" ;;
*) ok "buttons draw no outlines" ;;
esac
case $buttons in *$'\e[48;2;1;2;3m\e[38;2;4;5;6m'*)
    ok "buttons are background highlights with primary content"
    ;;
*) no "buttons are background highlights with primary content" "$(printf '%q' "$buttons" | head -c 300)" ;;
esac
case $raw in *$'\e[38;2;7;8;9m'*) no "no chrome in emphasized content" "base01 used for chrome" ;;
*) ok "no chrome in emphasized content" ;;
esac

# Nothing in the toolbar moves when the answer arrives: Refresh and All in tabs hold their places while it asks.
toolbar() {
    (
        cd "$CO" && HERDR_ACTIVE_PANE_CWD=$CO python3 - "$POPUP" <<'PY'
import importlib.machinery, importlib.util, sys, time
loader = importlib.machinery.SourceFileLoader("popup", sys.argv[1])
spec = importlib.util.spec_from_loader("popup", loader)
m = importlib.util.module_from_spec(spec)
loader.exec_module(m)
d = m.fetch(*m.place())
asking = m.frame({"branch": d["branch"], "data": None, "fetching": True}, 200, time.time())[1]
known = m.frame({"branch": d["branch"], "data": d, "fetching": False}, 200, time.time())[1]
print(" ".join(f"{a}:{x0}" for r, x0, x1, a in asking if r == 1), "|",
      " ".join(f"{a}:{x0}" for r, x0, x1, a in known if r == 1))
PY
    )
}
places=$(toolbar)
case $places in "r:"*"a:"*"close:"*" | r:"*) ok "the toolbar test compares Refresh, All in tabs and Close" ;;
*) no "the toolbar test compares Refresh, All in tabs and Close" "[$places]" ;;
esac
eq "the toolbar's chips keep their places when the answer arrives" "${places% |*}" "${places#*| }"

frame 140
[ "$(grep -c -E '^ +(TICKET|MERGE REQUEST|PIPELINE)$' "$TMP/frame")" = 3 ] && ok "a narrow popup stacks the three" ||
    no "a narrow popup stacks the three" "$(grep -E 'TICKET|MERGE|PIPELINE' "$TMP/frame")"
eq "stacked, the toolbar and every section's two buttons" 9 \
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
grep -qF "tab create --workspace w1 --cwd $CO --label diff !45 --focus" "$TMP/herdr.log" &&
    ok "M opens a tab named for what it shows" ||
    no "M opens a tab named for what it shows" "$(cat "$TMP/herdr.log")"
grep -F "pane run w1:p20 " "$TMP/herdr.log" | grep -qF "run-tab m" && ! grep -qF "pane split" "$TMP/herdr.log" &&
    ok "the tab runs the diff's supervisor, and nothing splits" ||
    no "the tab runs the diff's supervisor, and nothing splits" "$(cat "$TMP/herdr.log")"
echo '{"result":{"tabs":[{"tab_id":"w1:t3","label":"diff !45"}]}}' >"$TMP/tabs.json"
act M
grep -qF "tab focus w1:t3" "$TMP/herdr.log" && ! grep -qF "tab create" "$TMP/herdr.log" &&
    ok "New tab again focuses that tab, and opens no second one" ||
    no "New tab again focuses that tab, and opens no second one" "$(cat "$TMP/herdr.log")"
rm -f "$TMP/tabs.json"
act T
grep -qF -- "--label ticket #123" "$TMP/herdr.log" && ok "the ticket's tab is named for it" ||
    no "the ticket's tab is named for it" "$(cat "$TMP/herdr.log")"
act P
grep -qF -- "--label jobs !45" "$TMP/herdr.log" && grep -qF "run-tab p" "$TMP/herdr.log" &&
    ok "the pipeline's tab is named for it" ||
    no "the pipeline's tab is named for it" "$(cat "$TMP/herdr.log")"

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

echo "herdr-forge: the tab supervisor and Alt+Shift+U"

TABS=$XDG_CACHE_HOME/herdr-forge/tabs
tab() { (cd "$CO" && HERDR_PANE_ID=$1 "$POPUP" run-tab "${@:2}" </dev/null >/dev/null 2>&1); }
: >"$TMP/tools.log"
tab w1:p31 m
grep -qF "hunk diff origin/main --watch --sidebar" "$TMP/tools.log" &&
    ok "the diff is live, from the target when there is no merge base" ||
    no "the diff is live, from the target when there is no merge base" "$(cat "$TMP/tools.log")"
tab w1:p32 p
grep -qF "glab ci view -p 900" "$TMP/tools.log" && ok "the jobs tab opens the MR's current pipeline" ||
    no "the jobs tab opens the MR's current pipeline" "$(cat "$TMP/tools.log")"
tab w1:p33 t
grep -qF "folio $TABS/w1_p33.md" "$TMP/tools.log" && grep -qF "# #123 Save-as-failed" "$TMP/page.seen" &&
    ok "the ticket tab reads a page written from GitLab" ||
    no "the ticket tab reads a page written from GitLab" "$(cat "$TMP/tools.log")"
[ ! -e "$TABS/w1_p33.md" ] && [ ! -e "$TABS/w1_p33.pid" ] && ok "a supervisor that ends leaves nothing behind" ||
    no "a supervisor that ends leaves nothing behind" "$(ls "$TABS")"

# SIGUSR1 restarts the tool with fresh data; quitting the tool ends the supervisor.
: >"$TMP/tools.log"
touch "$TMP/hold"
(cd "$CO" && HERDR_PANE_ID=w1:p34 exec "$POPUP" run-tab t </dev/null >/dev/null 2>&1) &
for _ in $(seq 100); do
    grep -q folio "$TMP/tools.log" && break
    sleep 0.05
done
kill -USR1 "$(cat "$TABS/w1_p34.pid")"
for _ in $(seq 100); do
    [ "$(grep -c folio "$TMP/tools.log")" -ge 2 ] && break
    sleep 0.05
done
eq "a refresh restarts the tool" 2 "$(grep -c folio "$TMP/tools.log")"
rm -f "$TMP/hold"
wait
left=$(find "$TABS" -name 'w1_p34*' 2>/dev/null)
eq "quitting the tool after a refresh ends the tab" "" "$left"

# Alt+Shift+U in a workspace with none of the three: all three open, none focused.
tabs() { : >"$TMP/herdr.log" &&
    HERDR_ACTIVE_PANE_CWD=$CO HERDR_ACTIVE_WORKSPACE_ID=w1 HERDR_BIN_PATH=$TMP/bin/herdr "$POPUP" --tabs; }
tabs
eq "all three open, in reading order" "ticket #123|diff !45|jobs !45" \
    "$(grep -o -- '--label [^-]*' "$TMP/herdr.log" | sed 's/--label //; s/ $//' | paste -sd'|')"
eq "none of them takes focus" 3 "$(grep -c -- '--no-focus' "$TMP/herdr.log")"
grep -qE -- '--focus( |$)' "$TMP/herdr.log" && no "no tab is focused" "a tab was created with --focus" ||
    ok "no tab is focused"
grep -qF "notification show forge --body opened ticket #123, diff !45, jobs !45" "$TMP/herdr.log" &&
    ok "a toast says what opened" || no "a toast says what opened" "$(grep notification "$TMP/herdr.log")"

# Again, with the diff's supervisor running and a jobs tab from before supervisors.
# A stand-in supervisor: its command line names run-tab, and SIGUSR1 ends it cleanly.
python3 -c 'import signal, sys, time; signal.signal(signal.SIGUSR1, lambda *_: sys.exit(0)); time.sleep(30)' \
    run-tab m &
fake=$!
mkdir -p "$TABS" && echo "$fake" >"$TABS/w1_p50.pid"
echo '{"result":{"tabs":[{"tab_id":"w1:t8","label":"diff !45"},{"tab_id":"w1:t9","label":"jobs !45"}]}}' \
    >"$TMP/tabs.json"
echo '{"result":{"panes":[{"pane_id":"w1:p50","tab_id":"w1:t8"},{"pane_id":"w1:p60","tab_id":"w1:t9"}]}}' \
    >"$TMP/panes.json"
tabs
wait "$fake" 2>/dev/null
eq "an open tab is refreshed in place, not reopened" "" "$(grep -F -- '--label diff !45' "$TMP/herdr.log")"
kill -0 "$fake" 2>/dev/null && no "its supervisor was signalled" "still running" || ok "its supervisor was signalled"
grep -qF "tab close w1:t9" "$TMP/herdr.log" && grep -qF -- "--label jobs !45" "$TMP/herdr.log" &&
    ok "a tab with no supervisor is replaced" || no "a tab with no supervisor is replaced" "$(cat "$TMP/herdr.log")"
grep -qF "refreshed diff !45" "$TMP/herdr.log" && ok "the toast says what was refreshed" ||
    no "the toast says what was refreshed" "$(grep notification "$TMP/herdr.log")"
rm -f "$TMP/tabs.json" "$TMP/panes.json" "$TABS/w1_p50.pid"

# All in tabs from the popup is the same thing, with the popup's own answer.
act a
eq "All in tabs opens all three" 3 "$(grep -c -- '--no-focus' "$TMP/herdr.log")"

# Not all three always exist: only what does is opened, and the toast names what does not.
mr '{headPipeline: null}'
tabs
eq "no pipeline: the ticket and the diff open" "ticket #123|diff !45" \
    "$(grep -o -- '--label [^-]*' "$TMP/herdr.log" | sed 's/--label //; s/ $//' | paste -sd'|')"
grep -qF "no pipeline yet" "$TMP/herdr.log" && ok "the toast says there is no pipeline" ||
    no "the toast says there is no pipeline" "$(grep notification "$TMP/herdr.log")"
jq '.data.project.mergeRequests.nodes = []' "$TMP/answer.json" >"$TMP/a" && mv "$TMP/a" "$TMP/answer.json"
tabs
eq "no MR: only the ticket opens" "ticket #123" \
    "$(grep -o -- '--label [^-]*' "$TMP/herdr.log" | sed 's/--label //; s/ $//')"
grep -qF "no merge request yet" "$TMP/herdr.log" && ! grep -qF "no pipeline yet" "$TMP/herdr.log" &&
    ok "the toast says there is no MR, and no more" || no "the toast says there is no MR, and no more" \
    "$(grep notification "$TMP/herdr.log")"
jq '.data.project.workItems.nodes = []' "$TMP/answer.json" >"$TMP/a" && mv "$TMP/a" "$TMP/answer.json"
on plain
jq '.data.project.workItems.nodes = []' "$TMP/answer.json" >"$TMP/a" && mv "$TMP/a" "$TMP/answer.json"
tabs
eq "nothing at all: no tab opens" "" "$(grep -- '--label' "$TMP/herdr.log")"
grep -qF "no linked ticket · no merge request yet" "$TMP/herdr.log" && ok "the toast says what does not exist" ||
    no "the toast says what does not exist" "$(grep notification "$TMP/herdr.log")"
frame
eq "nothing to open: no All in tabs chip" "" "$(region a)"
[ -n "$(region r)" ] && [ -n "$(region close)" ] && ok "Refresh and Close stay" ||
    no "Refresh and Close stay" "r $(region r), close $(region close)"
on 123-feature
mr
jobs

printf '\n%d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
