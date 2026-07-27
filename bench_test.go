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
