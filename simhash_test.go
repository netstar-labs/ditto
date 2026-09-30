package ditto

import (
	"math"
	"strings"
	"testing"
)

func TestNearDuplicate(t *testing.T) {
	f := Default()
	a := f.Of("Your Apple ID has been locked. Please verify your account at the link below.")
	b := f.Of("Your Apple ID has been locked. Please verify your account at the link below!!")
	c := f.Of("Weekly newsletter: this week's top stories from around the web, curated for you.")

	if d := Distance(a, b); d > 8 {
		t.Errorf("near-dup distance = %d, want small", d)
	}
	if d := Distance(a, c); d < 12 {
		t.Errorf("unrelated distance = %d, want large", d)
	}
	if !a.Near(b, 8) {
		t.Error("a should be near b")
	}
	if a.Near(c, 8) {
		t.Error("a should not be near c")
	}
}

func TestDeterministic(t *testing.T) {
	f := Default()
	s := "The quick brown fox jumps over the lazy dog."
	first, second := f.Of(s), f.Of(s)
	if first != second {
		t.Errorf("fingerprint not deterministic: %s vs %s", first, second)
	}
}

func TestSumOrderIndependent(t *testing.T) {
	feats := []Feature{{Hash: 1, Weight: 1}, {Hash: 2, Weight: 2}, {Hash: 3, Weight: 1}}
	rev := []Feature{{Hash: 3, Weight: 1}, {Hash: 2, Weight: 2}, {Hash: 1, Weight: 1}}
	if Sum(feats) != Sum(rev) {
		t.Error("Sum must be order-independent")
	}
}

// Reproduces the accumulator overflow an adversarial skeptic confirmed before
// satAdd64 existed: two features voting the same bit with Weight: math.MaxInt
// wrapped int64's accumulator through two's complement and flipped bit 0 to 0
// despite two unanimous positive votes.
func TestSumLargeWeightsDoNotFlipBit(t *testing.T) {
	feats := []Feature{
		{Hash: 1, Weight: math.MaxInt},
		{Hash: 1, Weight: math.MaxInt},
	}
	fp := Sum(feats)
	if fp&1 == 0 {
		t.Errorf("Sum(%v) bit0 = 0, want 1 (two positive votes must not flip the sign)", feats)
	}

	negFeats := []Feature{
		{Hash: 0, Weight: math.MaxInt}, // bit 0 unset: votes -weight
		{Hash: 0, Weight: math.MaxInt},
	}
	fp = Sum(negFeats)
	if fp&1 != 0 {
		t.Errorf("Sum(%v) bit0 = 1, want 0 (two negative votes must not flip the sign)", negFeats)
	}
}

func TestSumZeroAndNegativeWeightTreatedAsOne(t *testing.T) {
	// Feature's doc comment: "Weight; <=0 is treated as 1."
	base := Sum([]Feature{{Hash: 1, Weight: 1}})
	zero := Sum([]Feature{{Hash: 1, Weight: 0}})
	neg := Sum([]Feature{{Hash: 1, Weight: -5}})
	if zero != base {
		t.Errorf("Sum with Weight=0 = %s, want same as Weight=1 (%s)", zero, base)
	}
	if neg != base {
		t.Errorf("Sum with Weight=-5 = %s, want same as Weight=1 (%s)", neg, base)
	}
}

func TestSatAdd64(t *testing.T) {
	cases := []struct{ a, b, want int64 }{
		{0, 0, 0},
		{5, 3, 8},
		{-5, -3, -8},
		{math.MaxInt64, 1, math.MaxInt64},
		{math.MaxInt64, math.MaxInt64, math.MaxInt64},
		{math.MinInt64, -1, math.MinInt64},
		{math.MinInt64, math.MinInt64, math.MinInt64},
		{math.MaxInt64, math.MinInt64, -1}, // opposite signs never overflow
	}
	for _, c := range cases {
		if got := satAdd64(c.a, c.b); got != c.want {
			t.Errorf("satAdd64(%d, %d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestNormalizeStripsTagsAndCase(t *testing.T) {
	got := Normalize("  <p>Hello   WORLD</p>\n<a href=x>Link</a>  ")
	want := "hello world link"
	if got != want {
		t.Errorf("Normalize = %q, want %q", got, want)
	}
}

func TestNormalizePreservesStrayAngleBrackets(t *testing.T) {
	// A '<' that does not open a tag must survive rather than swallow the rest of
	// the text (regression: an unbalanced '<' used to drop everything after it,
	// and interior '<'/'>' ate comparison/emoticon text).
	cases := []struct{ in, want string }{
		{"if x < y then swap the list", "if x < y then swap the list"},
		{"5 < 3 and 10 > 2 in the report", "5 < 3 and 10 > 2 in the report"},
		{"i <3 gambling and casino slots", "i <3 gambling and casino slots"},
		{"trailing angle bracket foo <", "trailing angle bracket foo <"},
		{"<p>real tag</p> stays stripped", "real tag stays stripped"},
	}
	for _, c := range cases {
		if got := Normalize(c.in); got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestShortInputStillFingerprints(t *testing.T) {
	f := Default()
	if f.Of("hi") == 0 && f.Of("yo") == 0 {
		t.Error("short inputs should still get non-trivial fingerprints via fallback")
	}
	if f.Of("hi") == f.Of("yo") {
		t.Error("distinct short inputs should differ")
	}
}

func TestWordShingles(t *testing.T) {
	f := Featurizer{WordN: 2}
	a := f.Of("send bitcoin to this wallet now")
	b := f.Of("send bitcoin to this wallet now please")
	if !a.Near(b, 10) {
		t.Errorf("word-shingle near-dup distance = %d, want small", Distance(a, b))
	}
}

func TestFingerprintString(t *testing.T) {
	if got := Fingerprint(0xabc).String(); got != "0000000000000abc" {
		t.Errorf("String = %q", got)
	}
	if len(Fingerprint(^uint64(0)).String()) != 16 {
		t.Error("String must be 16 hex digits")
	}
}

// genDoc builds a deterministic pseudo-document from a seed for benchmarks/tests.
func genDoc(seed, words int) string {
	var b strings.Builder
	x := uint64(seed)*2862933555777941757 + 3037000493
	for i := 0; i < words; i++ {
		x = x*6364136223846793005 + 1442695040888963407
		b.WriteString("w")
		b.WriteString(Fingerprint(x).String()[11:16]) // low hex digits vary per word
		b.WriteByte(' ')
	}
	return b.String()
}
