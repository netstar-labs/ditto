# ditto — coverage ledger (L2, 2026-09-29)

Every symbol in the module, classified. `unit`/`fuzz`/`integration` mean an actual test of
that kind exists and passes; percentages are `go test -coverprofile` (in-process) or
`go tool covdata` (subprocess binary coverage, via `DITTO_TEST_GOCOVERDIR`) as noted.

## `github.com/netstar-labs/ditto` (core) — 100.0% statements (`go test -coverprofile`)

| Symbol | File | Kind | Status |
|---|---|---|---|
| `Fingerprint`, `Feature`, `Match` (types) | simhash.go, index.go | unit | covered (exercised throughout) |
| `Sum` | simhash.go | unit + fuzz | covered — order-independence, weight<=0 clamp, saturating-overflow (this pass), `FuzzFingerprint` |
| `satAdd64` | simhash.go | unit | covered — table test incl. both overflow directions and the MinInt64+MinInt64 edge (this pass) |
| `Distance`, `Fingerprint.Near`, `Fingerprint.String` | simhash.go | unit | covered |
| `Index`, `NewIndex`, `K`, `Len`, `block` | index.go | unit + fuzz | covered — clamp (`min`/`max`), `FuzzIndex` |
| `Add` | index.go | unit + fuzz | covered |
| `candidates` | index.go | unit + fuzz (indirect via `Near`) | covered |
| `clampK` | index.go | unit | covered (`TestIndexClamp`) |
| `Near` | index.go | unit + fuzz + concurrency | covered; concurrent-read safety independently verified under `-race` (A1 skeptic, 16 goroutines) |
| `Clusters` | index.go | unit + fuzz + perf | covered; degenerate-case scaling documented and benchmarked (`BenchmarkClustersAllDuplicate`) |
| `unionFind`, `newUnionFind`, `find`, `union` | index.go | unit | covered — `TestUnionFind` (this pass) drives all 4 branches (self-union, swap, tie-break, no-swap) |
| `Normalize`, `stripTags`, `isTagStart` | normalize.go | unit + fuzz | covered; `Normalize`'s stdlib rewrite independently re-derived via a 16-case differential test + ~5M-exec differential fuzz against the old implementation before landing |
| `Featurizer`, `Default`, `Of`, `Features` | featurize.go | unit + fuzz | covered — `featurize_test.go` (new this pass; the file had **zero** test coverage before) |
| `shingleChars`, `shingleWords` | featurize.go | unit | covered — overlap counting, repeated-substring counting, shorter-than-n |
| `fnv1a`, `hashWords`, `fnvStep` | featurize.go | unit | covered — known FNV-1a vectors, `hashWords`≡`fnv1a(join)` equivalence (the property `fnvStep`'s extraction depends on) |
| `snapshot` (type), `Save` | persist.go | unit | covered |
| `LoadIndex` | persist.go | unit + fuzz | covered — round-trip, high-fingerprint (>2^53), corrupt-gob, length-mismatch (this pass), `FuzzLoadIndex` (this pass, 60s/~14M execs clean) |
| `maxSnapshotBytes` | persist.go | n/a | a constant; its bounding effect is inherent to `io.LimitReader`, not independently testable without a >1GiB fixture (impractical; noted, not chased) |
| `PipelineVersion` | version.go | n/a | a constant; consumed by `ditto version`, covered via `TestCLIVersionAndAliases` |

## `github.com/netstar-labs/ditto/app/ditto` (CLI) — combined coverage, two styles

In-process (`go test -coverprofile`, `main_test.go`): 28.4% alone — by design, since
`main`/`usage` call `os.Exit` directly and cannot run in the test process. Subprocess/binary
coverage (`cli_test.go` + `GOCOVERDIR`, `go tool covdata`): the complementary view.

| Symbol | In-process | Subprocess | Combined |
|---|---|---|---|
| `main` | 0% | **100%** | covered |
| `usage` | 0% | **100%** | covered |
| `featurizerFlags` | 0% | **100%** | covered |
| `fingerprint` | 0% | **100%** | covered |
| `cluster` | 0% | **100%** | covered |
| `writeFingerprints` | **100%** | 100% | covered (both styles — flush-error test is in-process, happy path in both) |
| `writeClusters` | 57.1% | **100%** | covered (in-process only hit the error path; subprocess exercises the real multi-cluster print loop) |
| `gather` | 87.1% | 77.4% | **covered** — the two views' remaining gaps don't overlap (verified by line inspection, not merged into one number); see `untestable-without-x.md`. `gather` now takes an injected `io.Reader` for stdin, and permission-denied paths are tested via `os.Chmod` guarded against root/Windows false-passes. |
| `doc` (type) | n/a | n/a | a plain struct, no behavior |

## Harness inventory

See `harness-inventory.md`. Short version: no external dependency, network, TLS, or clock
boundary exists in this codebase, so none of the fake-server/fixture/clock-injection
harnesses L2 Phase 1 asks about apply. The one real harness this pass added is the
subprocess CLI harness (`cli_test.go`'s `runCLI`/`TestMain`), which is what made
`main`/`usage`/dispatch testable at all.

## Already-built subsystems (L2 Phase 6)

There is exactly one non-core subsystem: the CLI (`app/ditto`). It is covered above, not
deferred. The two `example/` programs (`example/cluster`, `example/mcp`) are consumer
demonstrations, not shipped subsystems (per house convention — see `docs/audits/audit-findings.md`'s
least-code section) — `example/mcp` is smoke-tested by `go build`/`go vet` in its own module
(it has no test file, and its behavior is "call the public ditto API," already covered by the
core ledger above); `example/cluster` likewise.
