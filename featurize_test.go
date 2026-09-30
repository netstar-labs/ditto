package ditto

import (
	"strings"
	"testing"
)

// hashWords documents itself as "byte-identical to fnv1a(strings.Join(words,
// " "))" — this is the property fnvStep's extraction depends on being true.
func TestHashWordsMatchesFnv1aOfJoin(t *testing.T) {
	cases := [][]string{
		nil,
		{},
		{"a"},
		{"hello", "world"},
		{"one", "two", "three"},
		{"", "leading empty"},
		{"trailing empty", ""},
		{"", "", ""},
		{"unicode", "héllo", "wörld"},
	}
	for _, words := range cases {
		got := hashWords(words)
		want := fnv1a(strings.Join(words, " "))
		if got != want {
			t.Errorf("hashWords(%q) = %#x, want fnv1a(join) = %#x", words, got, want)
		}
	}
}

func TestFnv1aKnownVectors(t *testing.T) {
	// Standard 64-bit FNV-1a test vectors (empty string hashes to the offset
	// basis by definition; "a" is a well-known published vector).
	cases := []struct {
		in   string
		want uint64
	}{
		{"", fnvOffset},
		{"a", 0xaf63dc4c8601ec8c},
	}
	for _, c := range cases {
		if got := fnv1a(c.in); got != c.want {
			t.Errorf("fnv1a(%q) = %#x, want %#x", c.in, got, c.want)
		}
	}
}

func TestShingleCharsCountsOverlappingNGrams(t *testing.T) {
	counts := make(map[uint64]int)
	shingleChars("abcd", 2, counts)
	// "ab","bc","cd": 3 distinct 2-grams, each appearing once.
	if len(counts) != 3 {
		t.Fatalf("got %d distinct shingles, want 3: %v", len(counts), counts)
	}
	for h, c := range counts {
		if c != 1 {
			t.Errorf("shingle %#x count = %d, want 1", h, c)
		}
	}
}

func TestShingleCharsRepeatedSubstringIncrementsCount(t *testing.T) {
	counts := make(map[uint64]int)
	shingleChars("aaaa", 2, counts)
	// Only one distinct 2-gram ("aa"), appearing 3 times.
	if len(counts) != 1 {
		t.Fatalf("got %d distinct shingles, want 1: %v", len(counts), counts)
	}
	for _, c := range counts {
		if c != 3 {
			t.Errorf("shingle count = %d, want 3", c)
		}
	}
}

func TestShingleCharsShorterThanNProducesNothing(t *testing.T) {
	counts := make(map[uint64]int)
	shingleChars("ab", 4, counts)
	if len(counts) != 0 {
		t.Fatalf("got %v, want empty (input shorter than n)", counts)
	}
}

func TestShingleWordsCountsOverlappingNGrams(t *testing.T) {
	counts := make(map[uint64]int)
	shingleWords("the quick brown fox", 2, counts)
	// "the quick","quick brown","brown fox": 3 distinct bigrams.
	if len(counts) != 3 {
		t.Fatalf("got %d distinct shingles, want 3: %v", len(counts), counts)
	}
}

func TestFeaturesFallsBackToWholeStringBelowShingleSize(t *testing.T) {
	f := Featurizer{CharN: 10}
	feats := f.Features("hi") // shorter than the 10-char shingle
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1 (whole-string fallback): %+v", len(feats), feats)
	}
	if feats[0].Hash != fnv1a(Normalize("hi")) {
		t.Errorf("fallback feature hash = %#x, want fnv1a(Normalize(text))", feats[0].Hash)
	}
}

func TestFeaturesEmptyTextFallsBackToSingleFeature(t *testing.T) {
	f := Default()
	feats := f.Features("")
	if len(feats) != 1 {
		t.Fatalf("got %d features for empty input, want 1: %+v", len(feats), feats)
	}
}

func TestFeaturesCombinesCharAndWordShingles(t *testing.T) {
	f := Featurizer{CharN: 4, WordN: 2}
	feats := f.Features("the quick brown fox jumps")
	if len(feats) == 0 {
		t.Fatal("expected both char and word features, got none")
	}
}
