package workdesk

import (
	"strconv"
	"strings"
)

// Band is which group a merge request falls into: who owns the next move on it.
//
// The names are GitLab's own, from the merge request homepage that has shipped by
// default since 18.2, so this view and the web UI say the same words. Three are ours
// - BandStuck, BandCI and BandAutoMerge - for states GitLab folds into the merge
// request page rather than naming on its dashboard.
type Band int

// Declared in the order they are shown. Everything from BandApprovals down is
// inactive: GitLab excludes those sections from its attention count, and the UI draws
// its line where Active() flips.
const (
	BandReturned    Band = iota // a reviewer sent it back
	BandCI                      // the pipeline is red
	BandStuck                   // conflicts, or unresolved threads
	BandUnasked                 // finished, and no reviewer was ever asked
	BandAutoMerge               // green, but auto-merge was never set
	BandApprovals               // genuinely waiting on someone else
	BandAutoMerging             // green, auto-merge set: it will land itself
	BandDraft                   // yours, and deliberate
)

var bandNames = map[Band]string{
	BandReturned:    "returned to you",
	BandCI:          "pipeline failed",
	BandStuck:       "blocked on you",
	BandUnasked:     "no reviewer yet",
	BandAutoMerge:   "auto-merge not set",
	BandApprovals:   "waiting for approvals",
	BandAutoMerging: "auto-merging",
	BandDraft:       "draft",
}

// String is the band's label as a row header shows it.
func (b Band) String() string { return bandNames[b] }

// Active reports whether the band is asking something of you. GitLab's dashboard
// splits its sections this way and excludes the inactive ones from its count; that
// split is the whole idea, so it is modelled rather than inferred at render time.
func (b Band) Active() bool { return b < BandApprovals }

// Flag is the single letter a row carries so a reader can find where the active/inactive
// line falls without knowing what any band means.
func (b Band) Flag() string {
	if b.Active() {
		return "a"
	}
	return "i"
}

// Mergeability check identifiers, as GitLab's own words for what is wrong. This is a
// presentation lookup, deliberately not a closed set: GitLab adds checks over time
// and the list is not documented, so an identifier we have never seen degrades to a
// readable label instead of vanishing from the blockers.
var gateMessages = map[string]string{
	"CONFLICT":                       "conflicts with the target branch",
	"DISCUSSIONS_NOT_RESOLVED":       "unresolved review threads",
	"NOT_APPROVED":                   "not enough approvals",
	"DRAFT_STATUS":                   "still marked draft",
	"CI_MUST_PASS":                   "pipeline must pass",
	"NEED_REBASE":                    "needs a rebase",
	"REQUESTED_CHANGES":              "a reviewer requested changes",
	"MERGE_REQUEST_BLOCKED":          "blocked by another merge request",
	"COMMITS_STATUS":                 "source branch has no mergeable commits",
	"NOT_OPEN":                       "not open",
	"LOCKED_PATHS":                   "touches locked paths",
	"LOCKED_LFS_FILES":               "touches locked LFS files",
	"TITLE_REGEX":                    "title does not match the required pattern",
	"JIRA_ASSOCIATION_MISSING":       "no linked jira issue",
	"STATUS_CHECKS_MUST_PASS":        "external status checks must pass",
	"SECURITY_POLICY_VIOLATIONS":     "security policy violations",
	"SECURITY_POLICY_PIPELINE_CHECK": "security policy pipeline check",
	"MERGE_TIME":                     "merge-after time has not passed",
}

// GateMessage turns a mergeability check identifier into the phrase a row shows.
func GateMessage(id string) string {
	if m, ok := gateMessages[id]; ok {
		return m
	}
	return strings.ToLower(id)
}

// Gate statuses. INACTIVE means the check is not configured on this project, which is
// silence rather than a pass - only FAILED and CHECKING are worth a reader's
// attention, so neither Blockers nor Pending reports it.
const (
	gateFailed   = "FAILED"
	gateChecking = "CHECKING"
)

// Blockers is every gate GitLab currently refuses the merge over, in its own words.
//
// This is GitLab's verdict, never inferred: detailedMergeStatus names only one
// blocker and is computed lazily, so it reads UNCHECKED for much of any real queue,
// while mergeabilityChecks returns every gate with its own state. A merge request
// with three problems says so instead of revealing them one at a time.
func (mr *MergeRequest) Blockers() []string {
	return mr.gates(gateFailed)
}

// Pending is the gates GitLab has not finished evaluating yet.
func (mr *MergeRequest) Pending() []string {
	return mr.gates(gateChecking)
}

func (mr *MergeRequest) gates(status string) []string {
	var out []string
	for _, c := range mr.MergeabilityChecks {
		if c.Status == status {
			out = append(out, GateMessage(c.Identifier))
		}
	}
	return out
}

// hasGate reports whether a check with this identifier is failing. Matching on the
// identifier rather than the rendered message keeps the band logic independent of
// how a gate happens to be worded.
func (mr *MergeRequest) hasGate(id string) bool {
	for _, c := range mr.MergeabilityChecks {
		if c.Identifier == id && c.Status == gateFailed {
			return true
		}
	}
	return false
}

// CIStatus is the head pipeline's status, or NONE when nothing has run. A merge
// request with no pipeline decodes to a nil pointer, so this is the one place that
// has to know the difference between "no pipeline" and "a pipeline reporting nothing".
func (mr *MergeRequest) CIStatus() string {
	if mr.HeadPipeline == nil {
		return "NONE"
	}
	if mr.HeadPipeline.Status == "" {
		return "NONE"
	}
	return mr.HeadPipeline.Status
}

// PipelineLabel is GitLab's own human phrase for the pipeline state.
func (mr *MergeRequest) PipelineLabel() string {
	if mr.HeadPipeline == nil || mr.HeadPipeline.DetailedStatus.Label == "" {
		return "no pipeline"
	}
	return mr.HeadPipeline.DetailedStatus.Label
}

// Returned reports whether a reviewer has asked for changes - the one reviewer state
// that puts the next move back on the author.
func (mr *MergeRequest) Returned() bool {
	for _, r := range mr.Reviewers.Nodes {
		if r.Interaction.ReviewState == "REQUESTED_CHANGES" {
			return true
		}
	}
	return false
}

// Threads is the resolved-of-total review threads, or empty when there are none to
// resolve.
func (mr *MergeRequest) Threads() string {
	total := mr.ResolvableDiscussionsCount
	if total == 0 {
		return ""
	}
	return "threads " + strconv.Itoa(mr.ResolvedDiscussionsCount) + "/" + strconv.Itoa(total)
}

// ReviewerStates renders each reviewer and where they got to, as "name glyph" pairs.
func (mr *MergeRequest) ReviewerStates() []string {
	out := make([]string, 0, len(mr.Reviewers.Nodes))
	for _, r := range mr.Reviewers.Nodes {
		out = append(out, r.Username+" "+reviewGlyph(r.Interaction.ReviewState))
	}
	return out
}

func reviewGlyph(state string) string {
	switch state {
	case "APPROVED":
		return "✓"
	case "REQUESTED_CHANGES":
		return "✗"
	case "REVIEWED":
		return "•"
	default:
		return "·"
	}
}

// Band decides who owns the next move, in priority order: your own fixes first, then
// work no human was ever asked to look at, then work genuinely waiting on someone.
//
// The order matters more than any single arm. A merge request that is both red and
// unreviewed is yours to fix, so BandCI wins; one that is green and unassigned is
// nobody's, so BandUnasked beats BandApprovals. That ordering is why "finished work in
// no court at all" gets a band of its own instead of being buried by date.
func (mr *MergeRequest) Band() Band {
	switch {
	case mr.Draft:
		return BandDraft
	case mr.Returned():
		return BandReturned
	case mr.CIStatus() == "FAILED":
		return BandCI
	case mr.hasGate("CONFLICT"), mr.hasGate("DISCUSSIONS_NOT_RESOLVED"):
		return BandStuck
	case len(mr.Reviewers.Nodes) == 0:
		return BandUnasked
	case len(mr.Blockers()) == 0:
		if mr.AutoMergeEnabled {
			return BandAutoMerging
		}
		return BandAutoMerge
	default:
		return BandApprovals
	}
}

// Priority is the prio:: label an issue carries, as its band. Unlabelled sorts last
// and stays visible: an issue nobody triaged is a decision nobody made.
type Priority int

const (
	PrioHigh Priority = iota
	PrioMid
	PrioLow
	PrioNone
)

var prioNames = map[Priority]string{
	PrioHigh: "high", PrioMid: "mid", PrioLow: "low", PrioNone: "unlabelled",
}

func (p Priority) String() string { return prioNames[p] }

// Priority reads the first prio:: label. GitLab allows several; the highest wins
// rather than whichever the API happened to return first.
func (i *Issue) Priority() Priority {
	best := PrioNone
	for _, l := range i.Labels.Nodes {
		var p Priority
		switch l.Title {
		case "prio::high":
			p = PrioHigh
		case "prio::mid":
			p = PrioMid
		case "prio::low":
			p = PrioLow
		default:
			continue
		}
		if p < best {
			best = p
		}
	}
	return best
}
