# ditto — user guide

## Library

```go
import "github.com/netstar-labs/ditto"

f := ditto.Default()                 // character 4-grams; no tokenizer needed
a := f.Of("Your Apple ID has been locked. Please verify your account now.")
b := f.Of("Your Apple ID has been locked. Please verify your account now!")
ditto.Distance(a, b)                 // small — a few bits (here 2)
a.Near(b, 3)                         // true: within 3 bits

// near-duplicate lookup / clustering over a corpus
ix := ditto.NewIndex(3)              // resolve near-dups up to 3 bits
for id, text := range corpus {
    ix.Add(id, f.Of(text))
}
hits := ix.Near(f.Of(query), 3)      // []Match{ID, Fingerprint, Distance}, nearest first
families := ix.Clusters(3, 2)        // [][]string, near-dup families of >=2

// persist a large index between runs instead of rebuilding from every fingerprint
var buf bytes.Buffer
ix.Save(&buf)                        // gob: k + (id, fingerprint) pairs
ix2, _ := ditto.LoadIndex(&buf)      // rebuilds the banded tables; identical results
```

Tuning the featurizer:

```go
ditto.Featurizer{CharN: 4}           // default — robust, language-agnostic
ditto.Featurizer{WordN: 2}           // word 2-shingles — for clean, token-level text
ditto.Featurizer{CharN: 5, WordN: 1} // combine — both feed one fingerprint
```

Storing fingerprints elsewhere (e.g. on an upstream record): also store
`ditto.PipelineVersion`, and only compare fingerprints computed under the same
version.

## CLI

Build: `go build -o ditto ./app/ditto` (standalone: `GOWORK=off`). Inputs are
files, directories (walked), or one document on stdin.

```sh
# fingerprint each input:  <fingerprint> <tab> <id>
ditto fingerprint corpus/*.eml
cat message.eml    | ditto fingerprint          # one document from stdin

# near-duplicate families across a corpus
ditto cluster -k 3 -min 2 corpus/
#   110 document(s), 5 near-duplicate cluster(s) at k=3:
#   [1] 40 members ...

ditto version                                   # version + pipeline id
```

| Command | Flags | Meaning |
|---|---|---|
| `fingerprint` | `-char N` (4), `-word N` (0) | print `fingerprint<tab>id` per input |
| `cluster` | `-k N` (3), `-min M` (2), `-char`, `-word` | group inputs into near-dup families |
| `version` | — | binary version + `PipelineVersion` |

## Choosing k

`k` is the maximum Hamming distance treated as "near-duplicate," and it is fixed
when the index is built (it sets the number of bands). Guidance for 64-bit
fingerprints: `k=0` exact-fingerprint dupes only; `k=3` the standard near-dup
setting (per the Google crawl paper); higher `k` catches looser matches at more
memory (`k+1` band tables) and more false positives. Tune on your corpus by
inspecting clusters at a couple of `k` values.

## Build & deploy

`build/ditto [--cli] [user@host]` cross-compiles the CLI to `linux/amd64` with a
`git describe` version stamp, packages a self-contained installer + `.tgz`, and —
given a host — scp's it over and installs the binary to `/usr/local/bin` via ssh.
With no host it just builds the package under `build/install/`.
