package ditto

import (
	"bytes"
	"testing"
)

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

// FuzzLoadIndex checks that LoadIndex never panics on arbitrary bytes. It is
// documented as being for trusted snapshots (persist.go), so a malformed
// snapshot is only required to error cleanly, never to succeed — but
// encoding/gob decoding untrusted bytes still crosses a trust boundary and
// must not crash the process.
func FuzzLoadIndex(f *testing.F) {
	var validSnapshot bytes.Buffer
	ix := NewIndex(3)
	ix.Add("a", 1)
	ix.Add("b", 2)
	if err := ix.Save(&validSnapshot); err != nil {
		f.Fatal(err)
	}
	f.Add(validSnapshot.Bytes())
	f.Add([]byte("not gob"))
	f.Add([]byte(""))
	f.Add([]byte{0x00})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = LoadIndex(bytes.NewReader(data)) // must not panic; error is fine
	})
}
