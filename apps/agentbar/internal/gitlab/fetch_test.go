package gitlab

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// recorder answers an empty but present project, and keeps every query it was asked.
type recorder struct{ queries []string }

func (r *recorder) Run(_ context.Context, args ...string) ([]byte, error) {
	for _, a := range args {
		if q, ok := strings.CutPrefix(a, "query="); ok {
			r.queries = append(r.queries, q)
		}
	}
	return []byte(`{"data":{"project":{"mergeRequests":{"pageInfo":{"hasNextPage":false},` +
		`"nodes":[]},"issues":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}}`), nil
}

var firstArg = regexp.MustCompile(`first: (\d+)`)

// A detail fetch names its rows, so the page it asks for is that many. GitLab prices a
// selection by the page size asked for, and the merge request selection at first: 100
// costs 535 against a cap of 250 - a query refused outright, leaving the mirror with no
// merge requests at all.
func TestDetailFetchAsksForTheRowsItNames(t *testing.T) {
	t.Parallel()
	var iids []string
	for i := 1; i <= detailChunk*2+1; i++ {
		iids = append(iids, fmt.Sprint(i))
	}
	rec := &recorder{}
	if _, err := (&Client{Runner: rec}).MergeRequestsByIID(context.Background(), "g/p", iids); err != nil {
		t.Fatalf("MergeRequestsByIID: %v", err)
	}
	if len(rec.queries) != 3 {
		t.Fatalf("%d queries for %d iids in chunks of %d, want 3", len(rec.queries), len(iids), detailChunk)
	}
	for _, q := range rec.queries {
		named := strings.Count(q[strings.Index(q, "iids: ["):], `"`) / 2
		m := firstArg.FindStringSubmatch(q)
		if m == nil {
			t.Fatalf("query asks for no page size: %s", q)
		}
		if want := fmt.Sprint(named); m[1] != want {
			t.Errorf("a batch of %d rows asked for first: %s, want %s", named, m[1], want)
		}
	}
}

// The manifest is the other half of the rule: two scalars a row, so it takes the biggest
// page GitLab allows.
func TestManifestAsksForAFullPage(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	if _, err := (&Client{Runner: rec}).MergeRequestStamps(context.Background(), "g/p", []string{"you"}); err != nil {
		t.Fatalf("MergeRequestStamps: %v", err)
	}
	for _, q := range rec.queries {
		if m := firstArg.FindStringSubmatch(q); m == nil || m[1] != fmt.Sprint(manifestPage) {
			t.Errorf("manifest query asked for %v, want first: %d", m, manifestPage)
		}
	}
}

// GitLab caps an authenticated query at complexity 250, and a merge request detail page
// costs about 89 plus 4.5 per row asked for. Measured against the live schema: 36 rows
// pass and 40 are refused.
func TestDetailChunkFitsTheComplexityCap(t *testing.T) {
	t.Parallel()
	const ceiling = 36
	if detailChunk > ceiling {
		t.Errorf("detailChunk is %d; GitLab refuses a detail page above %d rows", detailChunk, ceiling)
	}
}
