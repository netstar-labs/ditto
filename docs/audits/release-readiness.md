# ditto — L4 pre-flight (2026-09-29, re-verified after two follow-up fixes)

Pre-flight gate per house `L4-release-packaging` §0, run on `audit/a1-l1-l2-hardening`
(branched from `main`@`ff992e9`, the `v0.1.0` release commit — this is the only commit in the
repo's history). **Not yet pushed or opened as a PR** — see "What's left" below. Re-run after
two follow-up commits resolved the pass's remaining deferred items (zero-value `Index` guard,
`gather`'s untested I/O-failure branches — see `defects-log.md`); nothing below changed as a
result except the release notes gaining one more line.

## §0 pre-flight gate — all green

- `go build ./...`, `go vet ./...`, `staticcheck ./...`, `deadcode -test ./...`, `gofmt -l .`
  — clean, all fourteen commits' worth of changes.
- `go test -race ./... -count=1` — green (core + `app/ditto`).
- Golden/round-trip suite: `TestIndexSaveLoad`, `TestLoadIndexHighFingerprint` — green.
- Fuzz smoke: `FuzzFingerprint`, `FuzzIndex`, `FuzzLoadIndex` — 60s each, ~7-14M execs, clean.
- `go mod tidy` — no diff (re-checked after the follow-up commits). `govulncheck ./...` — no
  vulnerabilities found (re-checked). Zero third-party dependencies (`go list -deps ./...` is
  stdlib-only; confirmed above and in `audit-findings.md`'s doc-drift check).
- Tree is clean on this branch; branched from current `main` tip (single-commit history, so
  trivially "synced").
- Re-audit: this whole branch **is** the A1 pass (`audit-findings.md`) plus L1
  (`verification-optimization.md`) plus L2 (`test-ledger.md` et al.) plus two follow-up fixes
  from a final independent skeptic pass — there is no separate "diff since assess" to re-run;
  the assess pass and the fix pass are the same commit history, already re-validated after
  each one.

## §1 version & tag — recommendation: `v0.1.1` (patch)

**Exported API surface diff against `v0.1.0`** (every `func`/`type`/`const`/`var` at column 0
in each core file, `git show v0.1.0:<file>` vs current, re-run after the follow-up commits):
the **only** additions in the core `ditto` package are `satAdd64`, `clampK`, `fnvStep` — all
lowercase, unexported. The follow-up commits added `Index.Add`'s new panic condition (no
signature change) and changed unexported `app/ditto` internals (`gather`'s new `io.Reader`
parameter, `package main` — not importable, has no exported surface to consumers). Nothing
exported was added, removed, or changed signature anywhere. **Zero breaking change**,
confirmed by diff, not assumed.

Per the house v0 policy (patch for fixes; minor for everything else including breaking
changes; never `v1`): this pass is four bug fixes (defects-log.md — the three original plus
the zero-value `Index` panic guard) plus doc corrections, simplifications, and dedup with no
behavior or API change, plus test/coverage additions — squarely a **patch**: `v0.1.1`.

## §2 release notes (drafted, not published)

```
## v0.1.1

### Fixes
- fingerprint/cluster no longer silently truncate output with exit code 0 on a write
  failure (disk full/quota exceeded) — the error is now surfaced and the process exits
  non-zero.
- Sum's accumulator no longer silently overflows and flips a fingerprint bit under an
  extreme Feature.Weight; it now saturates instead of wrapping.
- A symlink to a directory, anywhere in a walked tree or as the top-level argument, no
  longer aborts the entire fingerprint/cluster batch — it's skipped; a symlink to a
  regular file is still followed as before.
- Index.Add now panics with a clear message if called on a zero-value Index (one not
  constructed via NewIndex/LoadIndex), instead of silently accepting entries that could
  never be found by Near/Clusters.

### Additions
- Documentation corrections: the near-dup lookup's actual mechanism (block-partition
  bucketing, not a "permutation table"), the O(n²) degenerate-case caveat for a
  mass-duplicate corpus, corrected CLI alias/default documentation, an untagged
  example/mcp dependency pinned to the real v0.1.0 tag.
- Test coverage: 98.3% -> 100.0% statements (core package); new subprocess CLI test
  harness; new FuzzLoadIndex fuzz target; new featurize_test.go (previously untested file).

### Breaking
- None. Exported API surface is unchanged (diffed against v0.1.0; only unexported helpers
  were added).

Verification: go build/vet/staticcheck/deadcode/gofmt clean; go test -race -count=1 green;
go mod tidy no-diff; govulncheck clean; 3 fuzz targets clean at 60s each. Full detail in
docs/audits/.
```

## §3-6 — not applicable to this release

- **Reproducible build / container image:** ditto is a library plus a CLI (`app/ditto`) that
  consumers build themselves; this pass ships no binary artifact or image. `build/ditto`
  (the cross-compiling installer) is unaffected by this pass's changes.
- **Third-party attribution:** zero third-party dependencies (confirmed above) — nothing to
  attribute.
- **SBOM:** trivial (stdlib only); not generated for a patch release with no dependency
  change — would reuse `A4-dependency-supply-chain` if ever needed.

## §7 — repository metadata

Checked (`gh repo view`), already accurate, **no change needed** for a patch release:
description, topics (`deduplication`, `fingerprinting`, `go`, `golang`, `hamming-distance`,
`locality-sensitive-hashing`, `lsh`, `near-duplicate-detection`, `simhash`,
`zero-dependencies`, `library`, `platform`), license (`apache-2.0`), default branch (`main`).
**`homepageUrl` is empty** — not a regression from this pass, but worth setting to
`https://pkg.go.dev/github.com/netstar-labs/ditto` whenever a release is actually cut (a
one-line `gh repo edit`, not gated on anything in this pass).

## What's left (needs your go-ahead, not done in this pass)

Per the repo's own `.claude/claude.md` workflow ("push on request," "always ask before
merging") and general practice for anything visible to others:

1. **Push `audit/a1-l1-l2-hardening` and open a PR** — not done; branch is local-only.
2. **Merge** — only after you review, per the repo's standing rule.
3. **Tag `v0.1.1` and publish the GitHub release** — only after merge, using the drafted
   notes above; also set `homepageUrl` at that point (§7).

Nothing above requires re-running the gate — it's already green on this branch tip.
