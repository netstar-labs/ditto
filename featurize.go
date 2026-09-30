package ditto

import "strings"

// Featurizer turns text into the weighted features [Sum] folds into a
// [Fingerprint]. Use [Default] (character 4-grams) unless you have a reason to
// change the shingle sizes; whichever you pick must be identical when fingerprints
// are compared, so a change is a pipeline change (see [PipelineVersion]).
type Featurizer struct {
	CharN int // character n-gram length (0 disables character shingles)
	WordN int // word n-gram length (0 disables word shingles)
}

// Default featurizes with character 4-grams: no tokenizer, language-agnostic, and
// robust to light obfuscation (c4sino still shares grams with casino).
func Default() Featurizer { return Featurizer{CharN: 4} }

// Of normalizes text and returns its Fingerprint.
func (f Featurizer) Of(text string) Fingerprint { return Sum(f.Features(text)) }

// Features normalizes text and returns its weighted shingle features. A document
// shorter than the shingle size falls back to a single whole-string feature so it
// still gets a stable, comparable fingerprint.
func (f Featurizer) Features(text string) []Feature {
	norm := Normalize(text)
	counts := make(map[uint64]int)
	if n := f.CharN; n > 0 {
		shingleChars(norm, n, counts)
	}
	if n := f.WordN; n > 0 {
		shingleWords(norm, n, counts)
	}
	if len(counts) == 0 {
		counts[fnv1a(norm)] = 1
	}
	feats := make([]Feature, 0, len(counts))
	for h, c := range counts {
		feats = append(feats, Feature{Hash: h, Weight: c})
	}
	return feats
}

func shingleChars(s string, n int, counts map[uint64]int) {
	// Byte offset of each rune start, plus len(s) as the final boundary, so a
	// window of n runes hashes the byte substring s[off[i]:off[i+n]] directly —
	// no per-shingle string allocation. Byte-identical to hashing string(runes)
	// for valid UTF-8, which Normalize (run first) always produces.
	off := make([]int, 0, len(s)+1)
	for i := range s {
		off = append(off, i)
	}
	off = append(off, len(s))
	runes := len(off) - 1
	for i := 0; i+n <= runes; i++ {
		counts[fnv1a(s[off[i]:off[i+n]])]++
	}
}

func shingleWords(s string, n int, counts map[uint64]int) {
	w := strings.Fields(s)
	for i := 0; i+n <= len(w); i++ {
		counts[hashWords(w[i:i+n])]++
	}
}

// fnv1a is the 64-bit FNV-1a hash of s — fast, well-distributed, and deterministic
// (all a SimHash feature hash needs; no cryptographic strength required).
func fnv1a(s string) uint64 {
	h := uint64(fnvOffset)
	for i := 0; i < len(s); i++ {
		h = fnvStep(h, s[i])
	}
	return h
}

// hashWords hashes a space-joined word n-gram without allocating the join —
// byte-identical to fnv1a(strings.Join(words, " ")), expressed through the same
// fnvStep so that equivalence is structural rather than two hand-copied loops.
func hashWords(words []string) uint64 {
	h := uint64(fnvOffset)
	for i, wd := range words {
		if i > 0 {
			h = fnvStep(h, ' ')
		}
		for j := 0; j < len(wd); j++ {
			h = fnvStep(h, wd[j])
		}
	}
	return h
}

// fnvStep is one FNV-1a byte-mix step. A plain top-level function inlines
// reliably, so this costs nothing over the hand-inlined version.
func fnvStep(h uint64, b byte) uint64 { return (h ^ uint64(b)) * fnvPrime }

const (
	fnvOffset = 14695981039346656037
	fnvPrime  = 1099511628211
)
