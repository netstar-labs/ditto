# Audits

House audit trail, per the org's internal `A1`/`L1`/`L2` engineering process.

| Doc | Pass | Content |
|---|---|---|
| [audit-findings.md](audit-findings.md) | A1 (adversarial audit) | Four-dimension audit (simplify/dedup/correctness+perf/doc-drift) + least-code, each correctness/perf/doc claim independently skeptic-verified before being trusted. |
| [verification-optimization.md](verification-optimization.md) | L1 (verify & optimize) | Doc re-verification, full test/bench/fuzz/race run, hardening checklist, performance analysis. |
| [test-ledger.md](test-ledger.md) | L2 (test/harden) | Every symbol classified and its coverage, both in-process and subprocess-binary. |
| [defects-log.md](defects-log.md) | L2 | The three real bugs found and fixed this pass, root cause -> fix -> regression test. |
| [harness-inventory.md](harness-inventory.md) | L2 | What test harnesses exist, what they cover, and what deliberately wasn't built (no network/TLS/clock dependency exists). |
| [benchmark-baselines.md](benchmark-baselines.md) | L2 | Recorded benchmark numbers and what to watch for regression. |
| [untestable-without-x.md](untestable-without-x.md) | L2 | The one real coverage gap (`gather`'s I/O-failure branches) and the exact artifact needed to close it. |

2026-09-29 pass: baseline was already clean (no prior audit doc existed). Three real bugs
found and fixed (silent CLI write-truncation, `Sum` accumulator overflow, one-symlink-aborts-
the-batch), all confirmed by an independent skeptic with a real reproduction before landing.
Statement coverage 98.3% -> 100.0% for the core package. See `defects-log.md` for the short
version.
