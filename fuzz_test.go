package ditto

import "testing"

// FuzzFingerprint checks the normalize→shingle→SimHash path never panics on
// arbitrary input (attacker-controlled bodies), across char and word shingling
// and short/empty inputs.
func FuzzFingerprint(f *testing.F) {
	for _, s := range []string{
		"", "a", "hi", "<b>Win BIG</b> now!!", "the quick brown fox",
		"\x00\xff\xfe invalid utf8 \xc0", "😀 emoji 日本語 mixed", "   ",
	} {
		f.Add(s)
	}
	chars := Default()
	words := Featurizer{WordN: 2}
	both := Featurizer{CharN: 3, WordN: 1}
	f.Fuzz(func(t *testing.T, s string) {
		_ = Normalize(s)
		_ = chars.Of(s)
		_ = words.Of(s)
		_ = both.Of(s)
	})
}

// FuzzIndex checks that adding and querying arbitrary content never panics and
// that a document is always its own nearest neighbour at distance 0.
func FuzzIndex(f *testing.F) {
	f.Add("alpha", "beta")
	f.Add("", "")
	feat := Default()
	f.Fuzz(func(t *testing.T, a, b string) {
		ix := NewIndex(3)
		fa := feat.Of(a)
		ix.Add("a", fa)
		ix.Add("b", feat.Of(b))
		found := false
		for _, m := range ix.Near(fa, 3) {
			if m.ID == "a" && m.Distance == 0 {
				found = true
			}
		}
		if !found {
			t.Fatalf("a not found at distance 0 in its own index")
		}
		_ = ix.Clusters(3, 1)
	})
}
