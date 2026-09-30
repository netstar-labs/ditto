# ditto — architecture

A flat library (`package ditto`) at the repo root, one concern per file, plus a
thin CLI under `app/ditto`. The pipeline is: text → normalize → shingle → hash →
SimHash fingerprint → banded index for near-duplicate lookup.

## Data flow

```
text ─▶ Normalize ─▶ shingle (char/word n-grams) ─▶ fnv1a per shingle ─┐
                                                                        ▼
                                     []Feature{hash,weight} ─▶ Sum (Charikar) ─▶ Fingerprint (uint64)
                                                                        │
                       Index.Add(id, fp) ──▶ banded tables ──▶ Near(fp,k) / Clusters(k,min)
```

## Fingerprinting (`simhash.go`, `featurize.go`, `normalize.go`)

- **Normalize** strips HTML tags (each `<…>` becomes a space so words don't fuse),
  lowercases, and collapses whitespace. It is the shared front-end and part of the
  pipeline contract.
- **Featurizer** shingles the normalized text — character n-grams (default 4) and/
  or word n-grams — and hashes each shingle with 64-bit **FNV-1a** (fast,
  well-distributed, deterministic; no cryptographic strength is needed). Shingles
  are counted, so the weight is the shingle's frequency. A document shorter than
  the shingle size falls back to a single whole-string feature.
- **Sum** applies Charikar's SimHash: 64 signed counters, `+weight` where the
  shingle hash has a bit set and `-weight` where it doesn't; the result bit is 1
  where the counter ends positive. It is **order-independent** — addition commutes
  — so the fingerprint depends only on the multiset of shingles.

Why char n-grams by default: they need no tokenizer, are language- and
encoding-agnostic, and are robust to the light obfuscation (`c4sino`, spacing
tricks) common in adversarial text — where a word tokenizer would fragment
differently and an LLM subword tokenizer would optimise for compression, not
similarity.

## Near-duplicate lookup (`index.go`)

Comparing all pairs is O(n²). The **Index** uses banded lookup — a block-partition
bucketing scheme, per the pigeonhole guarantee used in the Google near-duplicate
crawl paper (no bit permutation is performed): built for a maximum Hamming distance
`k`, it splits each 64-bit fingerprint into `k+1` contiguous blocks. By the
pigeonhole principle, two fingerprints within `k` bits must agree exactly on at
least one block, so each fingerprint is indexed under each of its block values, and
a query only compares candidates that collide on a block — then verifies the true
Hamming distance. `Clusters` runs this over the whole set and unions near pairs
with a disjoint-set forest to yield near-duplicate families.

The sub-quadratic guarantee assumes band buckets stay small — it degrades toward
O(n²) for a corpus dominated by mutually near-duplicate documents (every member
then shares a bucket with every other), which is inherent to banding, not a defect.

Trade-off: the index holds one map entry per block per fingerprint (`k+1` per doc);
memory grows linearly with `k`. For `k=3` that is four 16-bit-keyed tables — the
common setting. An `Index` is build-then-read: not safe for concurrent `Add`.

## Versioning (`version.go`)

A fingerprint is only meaningful relative to the pipeline that produced it —
normalize + shingling + hash. `PipelineVersion` names that pipeline; any change to
it shifts every fingerprint, so persisted fingerprints (e.g. a capture-time
`SimHash` stored on an upstream record) must record the version they were computed
under, and only same-version fingerprints may be compared or pooled.

## Trade-offs

- **64-bit fingerprint.** Standard, one-word compare, `k≈3` resolves web-scale
  near-dups. A larger corpus wanting finer resolution would move to 128-bit
  (double the counters and the block math) — deliberately not done yet.
- **Character shingles by default.** Best robustness-per-line for adversarial
  text; word shingles are available (`WordN`) when token-level similarity is
  wanted and the text is clean.
- **In-memory index, with optional persistence.** The banded tables live in
  memory. `Index.Save`/`LoadIndex` (persist.go) snapshot an index as gob — `k`
  plus the `(id, fingerprint)` pairs, with the banded tables rebuilt on load — so a
  large fleet index can be saved between runs instead of rebuilt from every
  fingerprint each time. gob (not JSON) because a 64-bit fingerprint above 2^53
  would lose precision as a JSON number.
