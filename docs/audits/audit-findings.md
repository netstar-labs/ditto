# ditto — audit findings (A1 adversarial audit, 2026-09-29)

Torture-chamber pass per the house `A1-adversarial-audit` process: baseline gate, four
concurrent report-only auditors (simplify / dedup / correctness+perf / doc-drift) plus a
`least-code` pass, then an independent skeptic assigned to REFUTE (not confirm) every
correctness/perf/doc-drift candidate before anything was applied. All fixes below survived
the skeptic pass and the full re-validation gate (`go build`, `go vet`, `staticcheck`,
`deadcode -test`, `gofmt -l`, `go test -race -count=1`) after every commit.

**Baseline (before this pass):** clean — `go vet`, `staticcheck`, `deadcode -test`,
`gofmt -l` all empty; `go test -race ./... -count=1` green. The codebase was already small
and tight (~1177 non-test LOC across 9 files), so findings below are real but modest.

## Correctness + performance (dimension C) — the load-bearing findings

All five were independently reproduced by a skeptic given only the claim and the code, not
the auditor's reasoning, before being trusted.

| # | Finding | Verdict | Fix |
|---|---|---|---|
| 1 | `app/ditto`: `fingerprint`/`cluster` discarded every `Fprintf` error and `Flush`'s error via a bare `defer w.Flush()` — a write failure on stdout (disk full/quota) produced silently truncated output, exit code 0. Reproduced with `ulimit -f`: 2000-line expected output truncated to 7 lines, no stderr, exit 0. | **CONFIRMED** | Fixed — `bufio.Writer` latches its first error and returns it from every later call including `Flush`, so returning `w.Flush()` directly is sufficient. Extracted `writeFingerprints`/`writeClusters` (io.Writer-parameterized) so this is actually testable. Regression tests in `app/ditto/main_test.go`. |
| 2 | `simhash.go` `Sum`: `[64]int64` accumulator overflows and wraps (two's complement) when two `Feature`s share a hash bit and each carry `Weight: math.MaxInt`, flipping the output bit — contradicting the doc comment's "cannot overflow on any realistic weight total". Reproduced directly. Not reachable through the shipped `Featurizer` (shingle counts never approach `MaxInt`), but `Feature`/`Sum` are general-purpose public API. | **CONFIRMED** | Fixed — `satAdd64` saturates at `int64`'s bounds instead of wrapping; only the accumulator's sign is ever read, so saturation cannot produce a wrong sign. `TestSumLargeWeightsDoNotFlipBit` + `TestSatAdd64` (which itself caught a real edge case: `MinInt64 + MinInt64` wraps to exactly 0, not a positive number). |
| 3 | `index.go`: the zero-value `Index` (`var ix ditto.Index`, not `NewIndex`) accepts `Add` calls (`Len()` grows) but never indexes anything into a band table (`tables == nil`, loop is a no-op) — `Near` then reports zero matches forever, and `Clusters` at `min<=1` returns spurious all-singleton output rather than an obvious error. Reproduced directly. Grepped every call site in the repo: none constructs a zero-value `Index` today. | **CONFIRMED**, currently unreached | Doc-only fix applied (cheapest, no behavior change): `Index`'s comment now states the zero value is not usable. **Deferred decision, flagged for you:** a louder guard (panic on `Add` when `tables == nil`, or a lazy default `k=0`) is a real option if this library grows external callers who might zero-value it (e.g. via a containing struct) — not applied here since it's an API-behavior choice, not a bug fix. |
| 4 | `index.go`: the "better-than-O(n²)" doc claim is false for ditto's own headline use case — a corpus of mass-identical/near-identical documents (a spam blast) makes every entry share every band bucket, so `Near`/`Clusters` degrade to true O(n²). Measured: ~4x time per doubling at n=2000→16000 (textbook quadratic; sub-quadratic would be ~2x). | **CONFIRMED** | Doc fix — `index.go` and `docs/architecture.md` now state the sub-quadratic guarantee depends on band buckets staying small and degrades for a mass-duplicate corpus. `BenchmarkClustersAllDuplicate` added to keep the regression visible. This is inherent to banding, not a bug; no code change. |
| 5 | `app/ditto` `gather()`: a symlink to a directory, nested anywhere in a walked tree **or as the top-level argument itself**, reached `os.ReadFile` and failed with "is a directory" — and that error aborted the **entire** `WalkDir`, discarding every doc already gathered, for one link anywhere in the tree. | **CONFIRMED** | Fixed — a symlink to a regular file is still read (unchanged, already worked); a symlink to a directory, or a broken one, is skipped instead of treated as fatal. Regression tests cover nested and top-level cases, plus a control proving symlink-to-file is unaffected. |

**PERF (measured, applied):** `candidates()` (used by `Near`) swapped a per-call
`map[int]struct{}` dedup for a per-call `[]bool` — call-local, so `Near`'s "safe for
concurrent use" property is unaffected. `BenchmarkNear`: 14934 ns/op, 21 allocs/op →
~3500–5300 ns/op, 9 allocs/op (measured before/after, not estimated).

**Considered and NOT applied:** Auditor B's `forEachCandidate` helper to unify the
band-iteration shape across `Add`/`candidates`/`Clusters` — the auditor itself flagged risk
of a closure defeating escape analysis on `Add`'s hot path and recommended excluding `Add`.
Given the `candidates()` optimization above already changed that code and the remaining LOC
saved is marginal, this was skipped: not worth the risk to a path just proven perf-sensitive,
per the least-code ladder's "does this need to exist" question.

**Checked and clean (not just skimmed) — from auditor C's report:** the "Near keeps its own
map/slice, stays safe for concurrent use" claim (verified with `-race` under 16 concurrent
goroutines calling `Near`/`Clusters`), `Clusters`' generation-stamp correctness including
n=0, block-boundary math at k=0 (relies on correct `uint64(1)<<64 == 0` wraparound — verified,
not a bug), duplicate ids/hashes (has a dedicated test), `persist.go`'s round-trip and
corrupt-input handling, `normalize.go`/`featurize.go` on malformed UTF-8 and degenerate
inputs.

## Doc / comment vs code drift (dimension D)

| # | Finding | Verdict | Fix |
|---|---|---|---|
| 1 | `index.go` and `docs/architecture.md` named the lookup scheme "the permutation-table trick/method from the Google near-dup crawl paper." The actual code does direct block-hash bucketing (fixed contiguous bit ranges, `map[uint64][]int` per block) — no bit permutation, no sorted tables, no binary search, which is what that paper's named technique actually is. | **CONFIRMED** | Reworded in both places to "a block-partition bucketing scheme, per the pigeonhole guarantee ... (no bit permutation is performed)". |
| 2 | `example/mcp/go.mod` required `github.com/netstar-labs/ditto v0.2.0`, a tag that was never created (`git tag -l` / `git ls-remote --tags` both show only `v0.1.0`) — masked today by a local `replace` directive; removing it fails `go mod tidy` with "unknown revision v0.2.0". | **CONFIRMED** | Pinned to the real tag, `v0.1.0`. |
| 3 | `docs/userguide.md`'s CLI table showed `-char`/`-word` defaults for `fingerprint` but omitted them for `cluster`, though both subcommands set identical defaults. | cosmetic | Fixed — both rows now show defaults. |
| 4 | The `fp` alias for `fingerprint` and the `-v`/`-version`/`--version` aliases for `version` are wired in `app/ditto/main.go` but undocumented anywhere. | cosmetic | Documented in the CLI's own doc comment and `userguide.md`. |
| 5 | `persist.go` claimed the reloaded index is "byte-for-byte equivalent" to the original — no byte-level serialization of the in-memory struct is ever compared; the claim is really functional equivalence (which the same sentence's parenthetical already specifies). | cosmetic | Reworded to "functionally equivalent". |

**Checked and clean:** the README/executive-summary worked example's claimed Hamming
distance (recomputed: matches exactly), "pure Go, stdlib-only" (verified via
`go list -deps`), `stripTags`'s specific behavioral claims (`x < y`, `<3`, etc. — all
verified), `Feature.Weight`'s "<=0 treated as 1" claim, the FNV-1a constants, `doc.go`'s
stdin-id claim, both example READMEs, the `build/ditto` script description.

## Simplification (dimension A) + least-code

| Finding | Verdict | Applied? |
|---|---|---|
| `normalize.go` `Normalize`'s hand-rolled per-rune loop → `strings.Join(strings.Fields(strings.ToLower(stripTags(text))), " ")` | Independently re-derived (not just accepted from the auditor's trace): differential test against the old implementation across 16 adversarial cases (invalid UTF-8, Turkish dotless I, titlecase digraphs, combining marks, multiple Unicode whitespace categories) plus a ~5M-exec differential fuzz run — zero mismatches. Case-folding never changes which runes are whitespace, so the two passes commute. | Applied |
| `index.go` `NewIndex`'s two-if clamp → `min(max(k, 0), 63)` (go 1.25.0) | Trivially equivalent | Applied |
| `app/ditto` `gather()`: drop `os.Stat`, call `filepath.WalkDir` unconditionally | Claimed behavior-preserving for all cases; **REFUTED** by skeptic for the nonexistent-path case — `WalkDir`'s internal `Lstat` produces an `"lstat: ..."` error instead of today's `"stat: ..."`, a real (if narrow) user-visible text change | **Not applied** — the underlying symlink bug this was meant to help with was fixed directly in the `WalkDir` callback instead (see dimension C #5), without touching the top-level branch or its error text. |
| `featurize.go`: `fnv1a`/`hashWords` each hand-copied the same FNV-1a byte-mix step | Cross-validated (surfaced independently by both the least-code pass and auditor B) | Applied — extracted `fnvStep`; `hashWords`'s "byte-identical to fnv1a(join)" claim is now structural, locked in by a new `featurize_test.go` (this file had **no test coverage at all** before this pass — flagged for L2). |
| `app/ditto`: `fingerprint`/`cluster` registered identical `-char`/`-word` flags | Cross-validated | Applied — `featurizerFlags(fs)` helper. |
| `index_test.go`/`persist_test.go`: identical 4-doc fixture duplicated verbatim | Zero risk | Applied — shared `sampleDocs`. |
| `index.go`: `k` clamp to `ix.k` duplicated in `Near`/`Clusters` | Zero risk | Applied — `ix.clampK`. |
| `example/mcp`: optional-int-with-default unwrap repeated 3x | Zero risk, example-only | Applied — `intOr`. |
| `example/mcp`: `ditto_near` shipped wired but permanently non-functional (empty index, never populated) | Real — flagged by least-code as failing ladder rung 1 | Applied — seeded with two example docs so the tool demonstrates a real hit, matching `example/cluster`'s pattern. |

**No dead exported surface found** — every exported identifier has a live caller in tests,
the CLI, or an example (grepped across the whole module).

## Followups deferred to later passes

- **Zero-value `Index` guard** (dimension C #3 above) — a decision for you, not applied.
- **`featurize.go` had zero test coverage before this pass** — closed with a baseline
  `featurize_test.go`, but exhaustive edge-case/fuzz coverage for `shingleChars`/
  `shingleWords` belongs to the L2 pass (coverage ledger).
- L1/L2 (verification, optimization, exhaustive test/fuzz/bench/harden suite) follow this
  pass per the house lifecycle; see `docs/audits/verification-optimization.md` and
  `docs/audits/test-ledger.md`.
