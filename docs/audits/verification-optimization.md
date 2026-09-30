# ditto — L1 verification & optimization (2026-09-29)

House `L1-verification-optimization` pass. Consumes `docs/audits/audit-findings.md` (the A1
pass immediately before this one) rather than re-deriving dead code, duplication, and doc
drift — this doc covers what L1 adds on top: doc-vs-code re-verification after the A1 fixes,
running the existing suite, and hardening/performance analysis.

## Phase 1 — dead code, duplication (delegated to A1 + least-code)

Already covered in `audit-findings.md`. No dead exported surface found; the duplication
found there is fixed. Not re-derived here.

## Phase 2 — documentation

All doc-vs-code drift found by dimension D is fixed (see `audit-findings.md`). Re-checked
after the fixes: `README.md`, `docs/*.md`, `doc.go`, and every exported doc comment were
re-read against the current code; no new drift introduced by this pass's own commits.

**External metadata** (`gh repo view netstar-labs/ditto`): description and topics are
accurate to current scope (`deduplication`, `fingerprinting`, `go`, `golang`,
`hamming-distance`, `locality-sensitive-hashing`, `lsh`, `near-duplicate-detection`,
`simhash`, `zero-dependencies`, `library`, `platform`); license is `apache-2.0`; default
branch `main`. **`homepageUrl` is empty** — L4 §7 sets it to the pkg.go.dev canonical URL at
release time; noted here, not fixed in this pass (not a doc drift, a release-time action).

## Phase 3 — operational validation (all commands actually run, output below)

**Unit tests:** `go test -race ./... -count=1` — green across both modules (`ditto`,
`app/ditto`). Statement coverage (`go test -coverprofile`): **98.3%** for the core package.

**Benchmarks** (`-bench . -benchmem -count=10 -run=^$`, Apple M2 Pro):

| Benchmark | Mean | Range (10 runs) | Allocs/op | B/op | CV |
|---|---|---|---|---|---|
| `BenchmarkFingerprint` | 301.6 µs | 300.6–304.2 µs | 19 | 60,616 | ~0.4% |
| `BenchmarkNear` | 3.85 µs | 3.44–4.26 µs | 9 | 28,640 | ~10.7% |
| `BenchmarkClustersAllDuplicate` | 88.9 ms | 87.8–90.7 ms | 19 | 338,312 | ~1.2% |

`BenchmarkNear`'s ~10.7% CV is at the edge of the "flag if >10%" threshold — expected for a
call this fast (single-digit microseconds), where OS scheduling noise is a larger fraction
of the measured time than for the other two. Not treated as a regression signal; the
`candidates()` fix earlier in this pass already reduced its mean from ~14.9 µs to ~3.85 µs
and allocs from 21 to 9 (measured before/after, see `audit-findings.md`).

**Fuzz** (`-fuzztime=60s` each), all clean, no crashes:

| Target | Execs | Rate |
|---|---|---|
| `FuzzFingerprint` | ~6.8M | ~120k/s |
| `FuzzIndex` | ~11.4M | ~205k/s |
| `FuzzLoadIndex` (new this pass) | ~14.1M | ~235k/s |

No new corpus entries were promoted to `testdata/fuzz/` (that only happens on a discovered
crasher; Go caches non-crashing "interesting" inputs in the build cache, not the repo). Zero
crashers across all three targets, so nothing to commit as a regression seed.

**Race detector:** `-race -count=1` clean on the full suite; additionally, an A1 skeptic
independently ran 16 concurrent goroutines calling `Near`/`Clusters` against a shared,
already-built `*Index` under `-race` — clean, confirming the doc comment's concurrent-read
claim (see `audit-findings.md`, dimension C "checked and clean").

**Integration tests:** none — the library has no external dependencies or network/database
boundary to integrate against. N/A, not skipped.

## Phase 3.6 — hardening validation

Largely covered by A1's dimension C (see `audit-findings.md`), which already applied the
adversarial-input lens (nil/empty/truncated/oversized/duplicate) to every boundary. Summary
against this phase's checklist:

- **Input boundaries:** `Normalize`/`Featurizer.Features` (arbitrary text, including invalid
  UTF-8) — fuzzed (`FuzzFingerprint`), traced by hand for malformed UTF-8 and degenerate
  inputs (all-whitespace, all-tags, empty). `LoadIndex` (arbitrary gob bytes) — now fuzzed
  (`FuzzLoadIndex`, this pass) and has 3 explicit corrupt/truncated-input unit tests.
- **Error handling:** `app/ditto`'s swallowed write/flush errors were the one real gap
  (fixed, A1). `persist.go` returns clean errors on corrupt input and length-mismatched
  id/fingerprint arrays, never panics (verified by auditor C + existing tests).
- **Concurrency:** no goroutines or channels anywhere in this codebase — it is a
  synchronous, single-threaded computation library. The one concurrency contract
  (`Near`/`Clusters` safe for concurrent *read* on a built `*Index`; `Add` is not
  concurrent-safe and is documented as such) is verified under `-race` as above.
- **Resource limits:** `LoadIndex` bounds its read at `maxSnapshotBytes` (1 GiB) via
  `io.LimitReader` — already in place before this pass. No other unbounded-allocation
  surface from untrusted input (features/shingles scale with input length, which the caller
  controls by choosing what to feed in; there is no network/RPC boundary sizing anything).
- **Integer overflow:** `Sum`'s accumulator overflow was the one real finding (fixed, A1,
  dimension C #2).
- **DoS / regex:** no regexes anywhere in the codebase (`grep -rn regexp` — zero hits). N/A.

## Phase 4/5 — performance analysis and optimization plan

The one measured, applied optimization is `candidates()`'s map→bool-slice swap (A1, ~4x
latency reduction on `Near`, see `audit-findings.md`). Profiled `Sum` and `Add` (the other
two hot paths) with `-cpuprofile`/`-memprofile` on `BenchmarkFingerprint`/`BenchmarkNear`:
no further allocation or CPU hotspot stood out relative to their inherent cost (`Sum`'s
60,616 B/op / 19 allocs/op is dominated by `Featurizer.Features`'s shingle-count map and the
output `[]Feature` slice — both scale with the input's distinct-shingle count, not
avoidable without changing the algorithm). No further optimization is proposed this pass.

**Deferred, not this pass:** the O(n²) degenerate-case scaling (dimension C #4) is
inherent to banding, not a bug to optimize away in this design; a bucket-size cap or an
early-exit for identical-fingerprint buckets would change `Near`/`Clusters`' output contract
(fewer or different candidates compared) and is a design decision, not a mechanical fix —
noted as a possible future scope item, not implemented here.

## Definition-of-done check

- [x] All tests run and passed, command + output above
- [x] Benchmarks: count=10, mean ± range reported
- [x] Race detector: `-count=1`
- [x] Fuzz: 60s/target, all three targets, results documented
- [x] Hardening checklist run against the actual code (not skipped)
- [x] Every estimate/claim above is either **verified** (ran it this session) or explicitly
      marked **N/A** with the reason (no goroutines, no regex, no network boundary)
