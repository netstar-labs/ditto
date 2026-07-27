# ditto examples

| Example | What it shows | Run |
|---|---|---|
| [cluster](cluster/main.go) | fingerprinting, Hamming distance, and near-duplicate clustering over a small in-memory corpus | `go run ./example/cluster` |
| [mcp](mcp/main.go) | expose ditto as MCP tools over stdio — `ditto_cluster` (stateless) and `ditto_near` (needs a populated index) — for an AI agent host | `cd example/mcp && go run .` |

Build standalone with `GOWORK=off` if the surrounding workspace doesn't list this
module. The `mcp` example is a **separate module** (it imports
`github.com/netstar-labs/mcp`), so the ditto library itself stays dependency-free —
run it from `example/mcp/`.

For the CLI over real files/maildir/stdin, see [../docs/userguide.md](../docs/userguide.md):

```sh
go run ./app/ditto cluster -k 3 corpus/
```
