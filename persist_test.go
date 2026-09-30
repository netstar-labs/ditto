package ditto

import (
	"bytes"
	"encoding/gob"
	"testing"
)

func TestIndexSaveLoad(t *testing.T) {
	f := Default()
	docs := sampleDocs // defined in index_test.go
	ix := NewIndex(6)
	for _, d := range docs {
		ix.Add(d.id, f.Of(d.text))
	}

	var buf bytes.Buffer
	if err := ix.Save(&buf); err != nil {
		t.Fatal(err)
	}
	ix2, err := LoadIndex(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if ix2.K() != ix.K() || ix2.Len() != ix.Len() {
		t.Fatalf("reloaded K/Len = %d/%d, want %d/%d", ix2.K(), ix2.Len(), ix.K(), ix.Len())
	}
	// Near and Clusters must be identical on the reloaded index.
	q := f.Of(docs[0].text)
	before, after := ix.Near(q, 6), ix2.Near(q, 6)
	if len(before) != len(after) {
		t.Fatalf("Near lengths differ: %d vs %d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("Near[%d] differs: %+v vs %+v", i, before[i], after[i])
		}
	}
	cb, ca := ix.Clusters(6, 2), ix2.Clusters(6, 2)
	if len(cb) != len(ca) {
		t.Fatalf("cluster counts differ: %d vs %d", len(cb), len(ca))
	}
}

func TestLoadIndexHighFingerprint(t *testing.T) {
	// A fingerprint above 2^53 must round-trip exactly (the reason for gob, not JSON).
	ix := NewIndex(3)
	fp := Fingerprint(0xFFFFFFFFFFFFFFFF)
	ix.Add("x", fp)
	var buf bytes.Buffer
	if err := ix.Save(&buf); err != nil {
		t.Fatal(err)
	}
	ix2, err := LoadIndex(&buf)
	if err != nil {
		t.Fatal(err)
	}
	got := ix2.Near(fp, 0)
	if len(got) != 1 || got[0].Fingerprint != fp {
		t.Fatalf("high fingerprint did not round-trip: %+v", got)
	}
}

func TestLoadIndexCorrupt(t *testing.T) {
	if _, err := LoadIndex(bytes.NewReader([]byte("not gob"))); err == nil {
		t.Error("LoadIndex should error on corrupt input")
	}
}

func TestLoadIndexMismatchedLengths(t *testing.T) {
	var buf bytes.Buffer
	// A well-formed gob stream, but with more ids than fingerprints — not
	// producible via Index.Save (ix.ids and ix.fps always grow together in
	// Add), so this is exercised directly against the snapshot wire type.
	err := gob.NewEncoder(&buf).Encode(snapshot{
		K:   3,
		IDs: []string{"a", "b"},
		FPs: []Fingerprint{1},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadIndex(&buf)
	if err == nil {
		t.Fatal("LoadIndex should error on mismatched ids/fingerprints lengths")
	}
	const want = "ditto: corrupt index snapshot: 2 ids, 1 fingerprints"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
