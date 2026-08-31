// Package workdesk reads the on-disk mirror of the GitLab work you own and answers
// the four questions the popup asks of it: what is waiting on you, where are your
// merge requests, what have you not started, and which agent produced what.
//
// The mirror is a full snapshot written by `workdesk sync`. Every other command is a
// pure function of it, so a view can never disagree with another view, and no read
// touches the network.
package workdesk

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

// MirrorSchema is the shape of the rows the current selection produces. Bump it whenever
// a field is added to what sync asks GitLab for.
//
// A sync fetches in full only the rows whose updatedAt moved, so without this a mirror
// written before a field existed keeps serving rows that lack it, for as long as they sit
// untouched: adding status to the query left every quiet issue banded as "no status" and
// nothing said why. A mismatch refetches everything once.
const MirrorSchema = 2

// ErrNoMirror means nothing has been synced yet. Callers branch on it to say "run
// workdesk sync" rather than reporting a bare file-not-found.
var ErrNoMirror = errors.New("no mirror yet")

// Mirror is one snapshot, decoded. Todos may be empty: the feed needs a personal
// token, and a project bot has no inbox.
type Mirror struct {
	MRs    []MergeRequest
	Issues []Issue
	Todos  []Todo
	Meta   Meta
}

type Meta struct {
	Project string `json:"project"`
	User    string `json:"user"`
	Synced  string `json:"synced"`
	// Schema is the shape of the rows on disk, so a sync can tell that they were written
	// by an older selection. See MirrorSchema.
	Schema int `json:"schema,omitempty"`
	// Users is every account the mirror was fetched for. User stays the identity glab
	// authenticates as, which is whose todo feed this holds - the two differ the moment
	// a second account is configured.
	Users []string `json:"users,omitempty"`
	// Statuses is the project's status lifecycle, in GitLab's own declared order, which
	// is the order the issue bands are drawn in. Stored rather than derived: the bands
	// have to survive `workdesk render`, which has no network.
	Statuses []Status `json:"statuses,omitempty"`
	// Iteration is the sprint running when the mirror was written, so a row can say
	// whether it is in it.
	Iteration *Iteration `json:"iteration,omitempty"`
}

// Status is one entry of the project's work-item status lifecycle. GitLab owns the name,
// the order and the category; nothing here invents a workflow.
type Status struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

// Iteration is one sprint. Title is empty on a cadence-generated iteration - GitLab
// names those by their dates - so Label falls back to them rather than rendering blank.
type Iteration struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	StartDate string `json:"startDate"`
	DueDate   string `json:"dueDate"`
	Cadence   struct {
		Title string `json:"title"`
	} `json:"iterationCadence"`
}

// Label is what a person reads: the iteration's own title, else its dates.
func (it *Iteration) Label() string {
	if it == nil {
		return ""
	}
	if it.Title != "" {
		return it.Title
	}
	return join(" - ", it.StartDate, it.DueDate)
}

// Only the fields a view reads are declared. An addition upstream is ignored rather
// than breaking the decode, and a removal shows up as a zero value in one place.
type MergeRequest struct {
	IID                        string `json:"iid"`
	Title                      string `json:"title"`
	Draft                      bool   `json:"draft"`
	Conflicts                  bool   `json:"conflicts"`
	DetailedMergeStatus        string `json:"detailedMergeStatus"`
	Description                string `json:"description"`
	AutoMergeEnabled           bool   `json:"autoMergeEnabled"`
	AutoMergeStrategy          string `json:"autoMergeStrategy"`
	Approved                   bool   `json:"approved"`
	ApprovalsRequired          int    `json:"approvalsRequired"`
	ApprovalsLeft              int    `json:"approvalsLeft"`
	ResolvableDiscussionsCount int    `json:"resolvableDiscussionsCount"`
	ResolvedDiscussionsCount   int    `json:"resolvedDiscussionsCount"`
	SourceBranch               string `json:"sourceBranch"`
	TargetBranch               string `json:"targetBranch"`
	CommitCount                int    `json:"commitCount"`
	CreatedAt                  string `json:"createdAt"`
	UpdatedAt                  string `json:"updatedAt"`
	WebURL                     string `json:"webUrl"`

	DiffStats struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
		FileCount int `json:"fileCount"`
	} `json:"diffStatsSummary"`

	MergeabilityChecks []Check `json:"mergeabilityChecks"`

	// A pointer: an MR with no pipeline yet decodes to nil rather than a zero value
	// that would read as a pipeline reporting nothing.
	HeadPipeline *Pipeline `json:"headPipeline"`

	ApprovalState struct {
		Rules []ApprovalRule `json:"rules"`
	} `json:"approvalState"`

	Labels      nodes[Label]      `json:"labels"`
	Reviewers   nodes[Reviewer]   `json:"reviewers"`
	Discussions nodes[Discussion] `json:"discussions"`
}

// Discussion is one review thread. Taken with `last:` rather than `first:`: GitLab
// returns them oldest-first, and on a busy merge request the oldest are all system
// notes ("added 3 commits") - the human argument is at the tail.
type Discussion struct {
	Resolved bool        `json:"resolved"`
	Notes    nodes[Note] `json:"notes"`
}

// Note is one comment. System notes are GitLab talking to itself and are dropped from
// every render, so the field is decoded only to filter on it.
type Note struct {
	Body   string `json:"body"`
	System bool   `json:"system"`
	Author struct {
		Username string `json:"username"`
	} `json:"author"`
}

type Check struct {
	Identifier string `json:"identifier"`
	Status     string `json:"status"`
}

type Pipeline struct {
	Status         string `json:"status"`
	DetailedStatus struct {
		Label string `json:"label"`
	} `json:"detailedStatus"`
	Stages nodes[Stage] `json:"stages"`
}

type Stage struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ApprovalRule struct {
	Name              string          `json:"name"`
	ApprovalsRequired int             `json:"approvalsRequired"`
	Approved          bool            `json:"approved"`
	ApprovedBy        nodes[Username] `json:"approvedBy"`
}

type Username struct {
	Username string `json:"username"`
}

type Label struct {
	Title string `json:"title"`
}

type Reviewer struct {
	Username    string `json:"username"`
	Interaction struct {
		ReviewState string `json:"reviewState"`
	} `json:"mergeRequestInteraction"`
}

// GraphQL wraps every collection in `{ nodes: [...] }`; this unwraps it once instead
// of in every struct that has one.
type nodes[T any] struct {
	Nodes []T `json:"nodes"`
}

type Issue struct {
	// ID is the global ID. Kept because a status or iteration move addresses the work
	// item, not the iid: gid://gitlab/WorkItem/<n> resolves for the same n.
	ID        string       `json:"id"`
	IID       string       `json:"iid"`
	Title     string       `json:"title"`
	UpdatedAt string       `json:"updatedAt"`
	WebURL    string       `json:"webUrl"`
	Labels    nodes[Label] `json:"labels"`
	// Pointers: an issue with no status or no iteration decodes to nil rather than a
	// zero value that would read as a status named "".
	Status    *Status    `json:"status"`
	Iteration *Iteration `json:"iteration"`
}

// Todo is REST, not GraphQL - the only part of the mirror that is. Field names and
// the numeric iid come from /api/v4/todos.
type Todo struct {
	ID         int    `json:"id"`
	ActionName string `json:"action_name"`
	State      string `json:"state"`
	CreatedAt  string `json:"created_at"`
	TargetType string `json:"target_type"`
	TargetURL  string `json:"target_url"`
	Body       string `json:"body"`
	Target     struct {
		IID   flexID `json:"iid"`
		Title string `json:"title"`
	} `json:"target"`
	Project struct {
		Path string `json:"path_with_namespace"`
	} `json:"project"`
}

// GraphQL gives iids as strings and REST gives them as numbers; both land here.
type flexID string

func (f *flexID) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		*f = ""
		return nil
	}
	if len(s) > 0 && s[0] == '"' {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*f = flexID(v)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexID(n.String())
	return nil
}

func (f flexID) String() string { return string(f) }

// Load decodes a mirror directory.
func Load(dir string) (*Mirror, error) { return LoadFS(os.DirFS(dir), ".") }

// LoadFS decodes a mirror from any file system, which is what lets the embedded fixture
// and a real mirror on disk go through exactly one decode path.
//
// mrs.json is required - without it there is no view to render. The rest are optional: a
// mirror written before the todo feed existed has no todos.json, and treating that as an
// error would make every read fail.
func LoadFS(fsys fs.FS, dir string) (*Mirror, error) {
	m := &Mirror{}
	for _, f := range []struct {
		name     string
		into     any
		required bool
	}{
		{"mrs.json", &m.MRs, true},
		{"issues.json", &m.Issues, false},
		{"todos.json", &m.Todos, false},
		{"meta.json", &m.Meta, false},
	} {
		if err := readJSON(fsys, path.Join(dir, f.name), f.into, f.required); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func readJSON(fsys fs.FS, name string, into any, required bool) error {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if !required {
				return nil
			}
			return fmt.Errorf("%s: %w", filepath.Base(name), ErrNoMirror)
		}
		return fmt.Errorf("read %s: %w", filepath.Base(name), err)
	}
	if err := json.Unmarshal(b, into); err != nil {
		return fmt.Errorf("decode %s: %w", filepath.Base(name), err)
	}
	return nil
}
