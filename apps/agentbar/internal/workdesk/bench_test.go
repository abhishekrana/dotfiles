package workdesk

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"
)

// bigMirror synthesises a queue the size of a real one: descriptions and discussion
// threads are what make the snapshot large, and they are exactly what the index leaves
// behind.
func bigMirror(mrs int) *Mirror {
	body := strings.Repeat("a line of review discussion that nobody will read again\n", 40)
	m := &Mirror{Meta: Meta{Project: "acme/platform", Synced: "2026-08-27T21:27:32Z"}}
	for i := range mrs {
		id := strconv.Itoa(400 + i)
		mr := MergeRequest{
			IID: id, Title: "a change to the registry client " + id,
			SourceBranch: "feat/change-" + id, TargetBranch: "main",
			UpdatedAt:   "2026-08-2" + strconv.Itoa(i%7) + "T09:00:00Z",
			Description: body, WebURL: "https://gitlab.example.com/acme/platform/-/merge_requests/" + id,
			HeadPipeline: &Pipeline{Status: "SUCCESS"},
		}
		mr.MergeabilityChecks = []Check{{Identifier: "NOT_APPROVED", Status: "FAILED"}}
		mr.Reviewers.Nodes = []Reviewer{{Username: "dana"}}
		for d := 0; d < 8; d++ {
			var disc Discussion
			disc.Notes.Nodes = []Note{{Body: body}}
			mr.Discussions.Nodes = append(mr.Discussions.Nodes, disc)
		}
		m.MRs = append(m.MRs, mr)
	}
	for i := range 40 {
		id := strconv.Itoa(100 + i)
		is := Issue{IID: id, Title: "an issue " + id, UpdatedAt: "2026-08-20T09:00:00Z"}
		is.Labels.Nodes = []Label{{Title: "prio::mid"}}
		m.Issues = append(m.Issues, is)
	}
	return m
}

// The two numbers that matter, and the reason the index exists: fzf re-runs the
// preview command on every cursor movement, so whatever the interactive path decodes
// is paid per keystroke.
func BenchmarkDecodeFullSnapshot(b *testing.B) {
	raw, err := json.Marshal(bigMirror(60).MRs)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	for b.Loop() {
		var mrs []MergeRequest
		if err := json.Unmarshal(raw, &mrs); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeIndex(b *testing.B) {
	raw, err := json.Marshal(BuildIndex(bigMirror(60)))
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(raw)))
	b.ReportAllocs()
	for b.Loop() {
		var idx Index
		if err := json.Unmarshal(raw, &idx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildIndex(b *testing.B) {
	m := bigMirror(60)
	b.ReportAllocs()
	for b.Loop() {
		BuildIndex(m)
	}
}

func BenchmarkRowsInbox(b *testing.B) {
	idx := BuildIndex(bigMirror(60))
	now := time.Now()
	b.ReportAllocs()
	for b.Loop() {
		if err := WriteRows(io.Discard, idx.Rows(ViewInbox, now)); err != nil {
			b.Fatal(err)
		}
	}
}
