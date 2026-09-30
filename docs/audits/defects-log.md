# ditto — defects found and fixed (this pass, 2026-09-29)

Each root-caused in production code, with a regression test. See `audit-findings.md` for the
adversarial-skeptic verification each one survived before being trusted.

| # | Defect | Root cause | Fix | Regression test |
|---|---|---|---|---|
| 1 | `fingerprint`/`cluster` silently truncated output with exit 0 on a write failure | `defer w.Flush()` discarded every `Fprintf`/`Flush` error | Return `w.Flush()` directly — `bufio.Writer` latches its first error and returns it from every later call | `TestWriteFingerprintsSurfacesWriteError`, `TestWriteClustersSurfacesWriteError` |
| 2 | `Sum`'s accumulator silently overflowed and flipped an output bit under large `Weight` | `[64]int64` accumulator with unbounded caller-supplied `int` weights, plain `+=`/`-=` | `satAdd64`: saturate at `int64` bounds instead of wrapping | `TestSumLargeWeightsDoNotFlipBit`, `TestSatAdd64` |
| 3 | One symlink to a directory, anywhere in a walked tree (or as the top-level argument), aborted the entire batch, discarding every already-gathered doc | `WalkDir`'s callback fell through to `os.ReadFile` for a symlink `DirEntry` (never `IsDir()`), which fails with "is a directory," and that error propagates out of the whole walk | Resolve the symlink's target: skip it if a directory or broken, still read it if a regular file | `TestGatherSkipsSymlinkedDirectoryWithoutAbortingWalk`, `TestGatherFollowsSymlinkToRegularFile`, `TestGatherSkipsTopLevelSymlinkToDirectory` |

## Not a defect, but a doc-accuracy fix (no code change)

- "Better-than-O(n²)" claim doesn't hold for ditto's own headline use case (mass-duplicate
  corpus) — documented as inherent to banding, backed by `BenchmarkClustersAllDuplicate`.
- "Permutation-table trick" naming was wrong for what the code does (direct block-hash
  bucketing) — renamed in both `index.go` and `docs/architecture.md`.

## Resolved after initial deferral

- **Zero-value `Index` was silently non-functional** (`Add` accepted entries that never
  reached a band table). Originally left as a doc-only warning pending a decision; now fixed —
  `Add` panics with a clear message when called on a zero-value `Index`. `NewIndex`/`LoadIndex`
  results never trip the guard (verified: `TestConstructedIndexesNeverTripZeroValueGuard`). See
  `audit-findings.md` dimension C #3.
- **`gather`'s permission-denied and stdin-read-failure branches had no test.** Originally
  deferred as needing a production seam change; the seam (an injectable `io.Reader` for stdin)
  and a root/Windows-guarded `os.Chmod` test were added. See `untestable-without-x.md`.
