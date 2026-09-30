# ditto — benchmark baselines (L2, 2026-09-29)

`go test -bench . -benchmem -count=10 -run=^$ .`, Apple M2 Pro (darwin/arm64), go1.27.0,
`GOWORK=off`. Recorded after this pass's `candidates()` optimization (see `audit-findings.md`).

| Benchmark | Mean | Range (10 runs) | Allocs/op | B/op | CV |
|---|---|---|---|---|---|
| `BenchmarkFingerprint` (4-char shingling, 120-word doc) | 301.6 µs | 300.6–304.2 µs | 19 | 60,616 | ~0.4% |
| `BenchmarkNear` (20,000-entry index, k=3) | 3.85 µs | 3.44–4.26 µs | 9 | 28,640 | ~10.7% |
| `BenchmarkClustersAllDuplicate` (4,000 identical fingerprints, k=3, new this pass) | 88.9 ms | 87.8–90.7 ms | 19 | 338,312 | ~1.2% |

**Regression watch:**
- `BenchmarkNear` is the one to compare across future changes to `candidates()`/`Add`/`block`
  — this pass took it from 14.9 µs / 21 allocs to 3.85 µs / 9 allocs by swapping a per-call
  map for a per-call `[]bool` (measured before/after, not estimated).
- `BenchmarkClustersAllDuplicate` exists specifically to keep the degenerate O(n²) case
  (dimension C finding #4 in `audit-findings.md`) visible under `-bench`, not to be optimized
  against — the quadratic behavior is inherent to banding on a mass-duplicate corpus, not a
  regression target.
- `BenchmarkNear`'s ~10.7% CV is at the edge of the "flag if >10%" threshold, expected for a
  call this fast (scheduling noise is a larger fraction of a few-microsecond measurement);
  not itself evidence of a problem.

No CPU/memory profile hotspot outside of `Sum`/`Featurizer.Features`'s inherent
input-proportional allocation was found (see `docs/audits/verification-optimization.md`
Phase 4/5) — no further optimization proposed this pass.
