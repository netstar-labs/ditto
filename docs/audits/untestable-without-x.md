# ditto — untestable-without-X (L2, 2026-09-29)

**Update, same day:** the one entry this doc originally listed (`gather`'s I/O-failure
branches) has been closed — see below. Nothing in this module currently has an
untestable-without-X gap.

## Closed: `gather`'s I/O-failure branches (app/ditto/main.go)

Originally flagged as needing a production seam change; on reconsideration, both artifacts
were small enough to justify adding:

1. **Stdin read failure** — `gather` now takes an `io.Reader` parameter instead of reading
   the global `os.Stdin` directly (`gather(paths []string, stdin io.Reader)`, `main` passes
   `os.Stdin`). `TestGatherSurfacesStdinReadFailure` injects an always-erroring reader.
2. **Permission-denied reads** (top-level single-file argument and a nested file inside a
   walked directory) — used the `os.Chmod(path, 0o000)` approach after all, guarded by
   `skipIfCannotDenyOwnRead` (skips under root, where permission bits don't block root's own
   read, and on Windows, where they don't apply the same way) so the test degrades to a clean
   skip instead of a false pass in those environments. `TestGatherTopLevelFilePermissionDenied`
   / `TestGatherNestedFilePermissionDenied`.

Combined with the existing subprocess CLI tests (which exercise the success paths for stdin,
a single top-level file, and a nonexistent path — the exact lines these three new tests don't
touch), every statement in `gather` is now covered by at least one test style. Verified by
inspecting the uncovered-line sets from both `go test -coverprofile` (in-process) and
`go tool covdata` (subprocess/`GOCOVERDIR`) and confirming they don't overlap, rather than by
one merged number (merging text-format and binary coverage profiles into a single percentage
needs extra tooling this pass didn't need to build).

## Everything else

`maxSnapshotBytes` (persist.go)'s 1 GiB bound is inherent to `io.LimitReader` and not
independently exercisable without a >1GiB fixture, which is impractical to keep in a test
suite and would not test anything `io.LimitReader` itself doesn't already guarantee by its
own stdlib contract. This is the only remaining line in the ledger with no dedicated test,
and it is not a gap — it is trusting a stdlib primitive's own documented behavior.
