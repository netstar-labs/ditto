package ditto

import (
	"slices"
	"testing"
)

// sampleDocs is shared with persist_test.go: two near-duplicate phishing
// variants (p1/p2) and one exact duplicate pair (b1/b2), used to exercise both
// Near and Clusters (and, in persist_test.go, that a save/load round-trip
// preserves their results).
var sampleDocs = []struct{ id, text string }{
	{"p1", "Your Apple ID has been locked. Please verify your account now."},
	{"p2", "Your Apple ID has been locked. Please verify your account now!"}, // near p1
	{"b1", "Weekly newsletter with this week's stories from around the web."},
	{"b2", "Weekly newsletter with this week's stories from around the web."}, // exact dup of b1
}

func TestIndexNearAndClusters(t *testing.T) {
	f := Default()
	docs := sampleDocs
	ix := NewIndex(6)
	for _, d := range docs {
		ix.Add(d.id, f.Of(d.text))
	}

	near := ix.Near(f.Of(docs[0].text), 6)
	if len(near) == 0 || near[0].ID != "p1" || near[0].Distance != 0 {
		t.Fatalf("nearest to p1 should be p1@0, got %+v", near)
	}
	found := map[string]bool{}
	for _, m := range near {
		found[m.ID] = true
	}
	if !found["p2"] {
		t.Errorf("p1's neighbours should include p2, got %+v", near)
	}

	cl := ix.Clusters(6, 2)
	if len(cl) != 2 {
		t.Fatalf("want 2 near-dup clusters, got %d: %v", len(cl), cl)
	}
	// b1/b2 are identical -> distance 0 -> always clustered regardless of k.
	var haveBulk bool
	for _, g := range cl {
		if len(g) == 2 && g[0] == "b1" && g[1] == "b2" {
			haveBulk = true
		}
	}
	if !haveBulk {
		t.Errorf("expected {b1,b2} cluster, got %v", cl)
	}
}

func TestClustersDeterministicWithDuplicateIDs(t *testing.T) {
	// Add permits duplicate ids, so two clusters can share a smallest id. The
	// output order must still be deterministic across runs (regression: the
	// tie-break compared only the smallest id, so equal-id clusters came out in
	// map-iteration order).
	build := func() [][]string {
		ix := NewIndex(0)
		ix.Add("dup", Fingerprint(1))
		ix.Add("zzz", Fingerprint(1)) // cluster {dup, zzz}
		ix.Add("dup", Fingerprint(2))
		ix.Add("xxx", Fingerprint(2)) // cluster {dup, xxx}
		return ix.Clusters(0, 2)
	}
	want := [][]string{{"dup", "xxx"}, {"dup", "zzz"}}
	for i := 0; i < 200; i++ {
		got := build()
		if len(got) != len(want) {
			t.Fatalf("run %d: got %d clusters, want %d: %v", i, len(got), len(want), got)
		}
		for j := range want {
			if !slices.Equal(got[j], want[j]) {
				t.Fatalf("run %d: Clusters = %v, want deterministic %v", i, got, want)
			}
		}
	}
}

func TestIndexExactDupAtK0(t *testing.T) {
	f := Default()
	ix := NewIndex(0) // only identical fingerprints cluster
	fp := f.Of("send bitcoin to release your prize")
	ix.Add("a", fp)
	ix.Add("b", fp)
	ix.Add("c", f.Of("something entirely different about the weather"))
	cl := ix.Clusters(0, 2)
	if len(cl) != 1 || len(cl[0]) != 2 {
		t.Fatalf("want one 2-member exact-dup cluster, got %v", cl)
	}
}

func TestIndexClamp(t *testing.T) {
	ix := NewIndex(3)
	if ix.K() != 3 {
		t.Fatalf("K = %d", ix.K())
	}
	ix.Add("x", Fingerprint(0))
	// querying with k above the build-time K is clamped, not an error.
	if got := ix.Near(Fingerprint(0), 99); len(got) != 1 {
		t.Errorf("clamped Near = %+v", got)
	}
}
