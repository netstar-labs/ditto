# ditto — harness inventory (L2, 2026-09-29)

L2 Phase 1 asks for fake-external-dependency, fixture-generator, clock-injection, loopback-TLS,
byte-fuzz-driver, and temp-filesystem harnesses, instantiated to what this repo actually needs.
Most don't apply — ditto is a pure, synchronous, stdlib-only computation library with no
network, database, TLS, or wall-clock dependency anywhere in its call graph (`grep -rn
"net\.\|tls\.\|time\.Now\|time\.After"` across all non-test `.go` files: zero hits). What
follows is what actually exists and what each covers.

| Harness | What it simulates | Covers |
|---|---|---|
| `FuzzFingerprint` (fuzz_test.go) | arbitrary/attacker-controlled document text, including invalid UTF-8, mixed scripts, emoji | `Normalize`, `Featurizer.Of` across char/word/combined shingling |
| `FuzzIndex` (fuzz_test.go) | arbitrary document pairs added to a fresh `Index` | `Add`/`Near` self-match invariant, `Clusters` |
| `FuzzLoadIndex` (fuzz_test.go, new this pass) | arbitrary bytes fed to the gob-decode trust boundary | `LoadIndex` never panics on malformed/truncated/random input |
| `failingWriter` (app/ditto/main_test.go) | a disk-full/quota-exceeded `io.Writer` | `writeFingerprints`/`writeClusters`'s Flush-error propagation (the bug this pass fixed) |
| `t.TempDir()` + real symlinks (app/ditto/main_test.go) | a real filesystem tree with nested/top-level symlinks to files and directories, one broken | `gather`'s symlink-handling fix — real filesystem behavior, not mocked |
| Subprocess CLI harness — `TestMain` + `runCLI` (app/ditto/cli_test.go) | the actual compiled binary, invoked with real args/stdin, stdout/stderr/exit-code captured | `main`, `usage`, command dispatch, alias handling — none of which can run inside the test process (`os.Exit`) |
| `sampleDocs` (index_test.go, shared with persist_test.go) | a small fixed corpus: two near-duplicate phishing variants + one exact-duplicate pair | `Near`/`Clusters` semantics, save/load round-trip equivalence |

## Deliberately not built

- **Fake external dependency / server:** no network peer exists to fake.
- **Loopback TLS:** no TLS anywhere in this codebase.
- **Deterministic clock injection:** no TTL, timeout, retry, or cadence logic exists —
  `PipelineVersion` is a static string, not time-derived.
- **Differential/comparison harness against a reference SimHash implementation:** no
  authorized third-party reference was in scope for this pass (would need explicit
  authorization + a nested `bench/compat`-style module per house convention, L2 §"Phase 5" —
  a real option for a future pass if cross-implementation validation becomes a goal, not
  needed to validate this implementation's own internal correctness properties, which the
  existing unit tests and the Google near-dup paper's pigeonhole argument already establish).
