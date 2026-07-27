// Package ditto detects near-duplicate documents with Charikar's SimHash: it
// maps each document to a 64-bit [Fingerprint] such that near-duplicates land a
// small Hamming distance apart, then finds them in better-than-O(n²) time with a
// banded [Index].
//
// Unlike a cryptographic hash — where one changed input bit flips ~half the
// output — SimHash is a locality-sensitive hash: similar inputs produce similar
// fingerprints, so "are these two nearly the same?" becomes a bit-count.
//
// It is pure Go, standard library only, and needs no tokenizer: the default
// [Featurizer] shingles character 4-grams, which is language-agnostic and robust
// to light obfuscation. ditto answers only "is this a near-duplicate?" — it does
// not classify or categorize content.
package ditto
