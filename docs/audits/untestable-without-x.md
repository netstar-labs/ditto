# ditto — untestable-without-X (L2, 2026-09-29)

Exhaustive list of what this pass could not cover, and the exact artifact each needs. There
is exactly one real entry; everything else in the codebase is covered (see `test-ledger.md`).

## `gather`'s I/O-failure branches (app/ditto/main.go)

**Uncovered:** three branches, none exercised by either the in-process (`main_test.go`) or
subprocess (`cli_test.go`) test style:

1. `os.ReadFile` failing on a top-level single-file argument (main.go:146-149) — e.g.
   permission denied.
2. `os.ReadFile` failing inside the `WalkDir` callback on a nested file (main.go:171-174) —
   same failure, different call site.
3. `io.ReadAll(os.Stdin)` failing (main.go:133-136) — e.g. stdin closed/reset mid-read.

**Why not covered:** (1)/(2) need a file that exists but is unreadable — the standard way
(`os.Chmod(path, 0o000)`) is unreliable in CI: **when the test runs as root** (common in
containers), permission bits on a regular file don't block root's own read, so the test would
pass locally and silently stop testing anything once CI runs as a different user, or worse,
flake between environments. (3) needs an `io.Reader` that returns a real error mid-read
substituted for `os.Stdin` — `gather` reads the global `os.Stdin` directly, so there is no
injection seam today.

**Exact artifact needed, if this is worth closing:**
- For (1)/(2): either accept the root-flakiness caveat and add a
  `t.Skip("requires non-root")` guard keyed off `os.Getuid() == 0`, or (cleaner) give `gather`
  a small unexported `readFile` seam that tests can swap for a function returning a
  synthetic error — the same shape as making `os.Stdin` injectable below.
- For (3): change `gather`'s signature to take an `io.Reader` for the no-args/stdin case
  (e.g. `gather(paths []string, stdin io.Reader) ([]doc, error)`), with `main` passing
  `os.Stdin` — then a test can pass an `io.Reader` that errors after N bytes (the same
  `failingWriter`-shaped pattern already used in `main_test.go` for the write side).

**Why not done in this pass:** both are real production-code seam changes (not test-only
additions), and the failure modes involved (disk/permission errors, a broken stdin pipe) are
loud, immediate, single-document failures with no silent-wrong-output risk — they already
return a proper error up through `gather` → `fingerprint`/`cluster` → `main`'s `ditto: <err>`
+ exit 1 path (verified by inspection; the error-propagation shape is identical to the
already-tested nonexistent-path case). The risk this represents is "one bad file skips the
whole batch with a clear error," not silent corruption — judged not to justify a production
API change in this pass. Flagged here as a deliberate, not overlooked, gap.

## Everything else

No other package/symbol in this module has an untestable-without-X gap. `maxSnapshotBytes`
(persist.go)'s 1 GiB bound is inherent to `io.LimitReader` and not independently exercisable
without a >1GiB fixture, which is impractical to keep in a test suite and would not test
anything `io.LimitReader` itself doesn't already guarantee by its own stdlib contract.
