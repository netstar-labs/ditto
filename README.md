# ditto

Near-duplicate detection for Go — **Charikar SimHash**, pure standard library, no
tokenizer, no model. Fingerprint a document to a 64-bit value where
near-duplicates land a few bits apart, then find them across a corpus in better
than O(n²). ditto answers one question — *is this a near-duplicate?* — and nothing
else. The name is the ditto mark (`″`): *same as above*.

```
text ─▶ Normalize ─▶ shingle (char/word n-grams) ─▶ FNV-1a ─▶ Sum (Charikar)
                                                                    │
                                                                    ▼
                                                            Fingerprint (uint64)
                                                                    │
                          Index.Add(id,fp) ─▶ banded tables ─▶ Near(fp,k) · Clusters(k,min)
```

## Quick start

```go
f := ditto.Default()                       // character 4-grams
a := f.Of("Your Apple ID has been locked. Please verify your account now.")
b := f.Of("Your Apple ID has been locked. Please verify your account now!")
a.Near(b, 3)                               // true — near-duplicate (differ by ~2 bits)

ix := ditto.NewIndex(3)
for id, text := range corpus { ix.Add(id, f.Of(text)) }
ix.Clusters(3, 2)                          // near-duplicate families
```

```sh
go run ./app/ditto cluster -k 3 corpus/    # near-dup families over files/maildir
```

## Documentation

- **Start here** — [docs/introduction.md](docs/introduction.md) ·
  [docs/executive-summary.md](docs/executive-summary.md)
- **Deep dive** — [docs/architecture.md](docs/architecture.md)
- **Operations** — [docs/userguide.md](docs/userguide.md)
- **Examples** — [example/README.md](example/README.md)

## Layout

| File | Purpose |
|---|---|
| [simhash.go](simhash.go) | `Fingerprint` (+ `String`), `Feature`, `Sum` (Charikar accumulation), `Distance`, `Near` |
| [featurize.go](featurize.go) | `Featurizer` — normalize → char/word shingles → FNV-1a features |
| [normalize.go](normalize.go) | `Normalize` — HTML-strip, lowercase, whitespace-collapse |
| [index.go](index.go) | `Index` (`Add`/`Near`/`Clusters`/`K`/`Len`/`Save`) — banded near-duplicate lookup; `Clusters` via union-find |
| [persist.go](persist.go) | `Index.Save` / `LoadIndex` — gob snapshot (k + id/fp pairs); rebuilds the banded tables on load |
| [version.go](version.go) | `PipelineVersion` — the fingerprint-pipeline identity |
| [app/ditto/](app/ditto/main.go) | the CLI — `fingerprint` · `cluster` · `version` |

## Notes

- Go module `github.com/netstar-labs/ditto`. **Standard library only** — no
  dependencies. Build standalone with `GOWORK=off`.
- Deterministic: a fingerprint is a pure function of the versioned pipeline
  ([version.go](version.go)); store `PipelineVersion` alongside any persisted
  fingerprint and compare only within a version.
- ditto is *not* a classifier. For "what is this content about?" that is a
  categorizer's job; ditto only answers sameness.

## License

Licensed under the Apache License, Version 2.0 — see [LICENSE](LICENSE).
