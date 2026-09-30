package ditto

import "testing"

func BenchmarkFingerprint(b *testing.B) {
	f := Default()
	doc := genDoc(1, 120)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Of(doc)
	}
}

func BenchmarkNear(b *testing.B) {
	f := Default()
	ix := NewIndex(3)
	for i := 0; i < 20000; i++ {
		ix.Add(genDoc(i, 4), f.Of(genDoc(i, 40)))
	}
	probe := f.Of(genDoc(7, 40))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ix.Near(probe, 3)
	}
}

// BenchmarkClustersAllDuplicate documents the degenerate case Index's doc
// comment now calls out: every entry shares one fingerprint, so every band
// bucket holds all n entries and Clusters degrades toward O(n²). An
// adversarial skeptic measured ~4x time per doubling (2000->16000) confirming
// quadratic scaling on this shape; this benchmark keeps that regression
// visible (run with -bench, not part of the default `go test`).
func BenchmarkClustersAllDuplicate(b *testing.B) {
	const n = 4000
	ix := NewIndex(3)
	for i := 0; i < n; i++ {
		ix.Add(genDoc(i, 4), Fingerprint(0xDEADBEEFCAFEBABE))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ix.Clusters(3, 1)
	}
}
