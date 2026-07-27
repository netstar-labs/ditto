package ditto

import (
	"fmt"
	"math/bits"
)

// Fingerprint is a 64-bit SimHash of a document. Two documents are
// near-duplicates when the Hamming distance between their fingerprints is small.
type Fingerprint uint64

// Feature is one weighted feature of a document — a shingle hash and its weight
// (typically the shingle's frequency). [Sum] folds a document's features into a
// Fingerprint.
type Feature struct {
	Hash   uint64 // a well-distributed hash of the feature
	Weight int    // influence of the feature; <=0 is treated as 1
}

// Sum folds features into a Fingerprint with Charikar's SimHash: each of the 64
// bit positions accumulates +weight when the feature hash has that bit set and
// -weight when it does not; the result bit is 1 wherever the accumulator ends
// positive. Sum is order-independent — the accumulation commutes — so the
// fingerprint depends only on the multiset of features, not their order.
func Sum(features []Feature) Fingerprint {
	var col [64]int64 // 64-bit so accumulation is platform-independent and cannot overflow on any realistic weight total
	for _, f := range features {
		w := int64(f.Weight)
		if w <= 0 {
			w = 1
		}
		h := f.Hash
		for i := 0; i < 64; i++ {
			if h&(uint64(1)<<i) != 0 {
				col[i] += w
			} else {
				col[i] -= w
			}
		}
	}
	var fp uint64
	for i := 0; i < 64; i++ {
		if col[i] > 0 {
			fp |= uint64(1) << i
		}
	}
	return Fingerprint(fp)
}

// Distance is the Hamming distance between two fingerprints (0 = identical).
func Distance(a, b Fingerprint) int { return bits.OnesCount64(uint64(a ^ b)) }

// Near reports whether a and b are within k bits of each other.
func (a Fingerprint) Near(b Fingerprint, k int) bool { return Distance(a, b) <= k }

// String renders the fingerprint as 16 lowercase hex digits.
func (a Fingerprint) String() string { return fmt.Sprintf("%016x", uint64(a)) }
