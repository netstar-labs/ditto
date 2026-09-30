// Example mcp exposes ditto as Model Context Protocol tools over stdio, so an AI
// agent host can dedupe a batch of documents it already holds (ditto_cluster) or
// ask whether a document is a near-duplicate of something in a prewarmed corpus
// (ditto_near).
//
//	go run .          # from this directory — it is a separate module so the
//	                  # ditto library itself stays dependency-free
//
// This is deliberately two tools with very different shapes:
//
//   - ditto_cluster is STATELESS: everything it needs arrives in the call, so it
//     is useful the moment the server starts.
//   - ditto_near is STATEFUL: it answers against an Index, and an EMPTY index
//     matches nothing. See the note on nearIndex below for what it needs to be
//     useful.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/netstar-labs/ditto"
	"github.com/netstar-labs/mcp"
)

func main() {
	srv := mcp.New(mcp.Options{Name: "ditto", Version: ditto.PipelineVersion})

	// Tool 1 — stateless batch dedup. Hand it documents, get back near-duplicate
	// families. No backing state, so it is immediately useful.
	must(srv.AddTool(mcp.Tool{
		Name: "ditto_cluster",
		Description: "Group near-duplicate documents. Input: docs (a list of {id, text}), " +
			"optional k (max Hamming distance, default 3) and min (smallest cluster to report, default 2). " +
			"Returns the near-duplicate families as arrays of ids.",
		InputSchema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"docs":{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"},"text":{"type":"string"}},"required":["id","text"]}},
				"k":{"type":"integer","minimum":0},
				"min":{"type":"integer","minimum":1}
			},
			"required":["docs"]
		}`),
	}, func(ctx context.Context, s *mcp.Session, raw json.RawMessage) (*mcp.CallToolResult, error) {
		var in struct {
			Docs []struct {
				ID   string `json:"id"`
				Text string `json:"text"`
			} `json:"docs"`
			K   *int `json:"k"`
			Min *int `json:"min"`
		}
		if err := json.Unmarshal(raw, &in); err != nil {
			return mcp.ToolError("invalid arguments: %v", err), nil
		}
		k, min := intOr(in.K, 3), intOr(in.Min, 2)
		f := ditto.Default()
		ix := ditto.NewIndex(k)
		for _, d := range in.Docs {
			ix.Add(d.ID, f.Of(d.Text))
		}
		out, err := json.Marshal(ix.Clusters(k, min))
		if err != nil {
			return mcp.ToolError("encode result: %v", err), nil
		}
		return mcp.ToolText("%s", out), nil
	}))

	// Tool 2 — stateful near-duplicate lookup.
	//
	// WHAT THIS NEEDS TO BE USEFUL: a POPULATED nearIndex. Below it is seeded with
	// two example documents purely so a query actually returns a hit and this
	// tool demonstrates something real out of the box. In production that corpus
	// is owned and kept warm elsewhere and injected the same way:
	//
	//   - a service that aggregates fingerprints across your corpus into one
	//     shared Index, or
	//   - a snapshot is restored at startup with ditto.LoadIndex(f).
	nearIndex := ditto.NewIndex(3)
	seedF := ditto.Default()
	nearIndex.Add("example-phishing-1", seedF.Of("Your Apple ID has been locked. Please verify your account at the link below."))
	nearIndex.Add("example-newsletter-1", seedF.Of("Weekly newsletter: this week's top stories from around the web, curated for you."))
	must(srv.AddTool(mcp.Tool{
		Name: "ditto_near",
		Description: "Find near-duplicates of a document within a prewarmed corpus. Input: text, " +
			"optional k (max Hamming distance, default 3). Returns matches (id, distance), nearest first. " +
			"Only useful when the server's corpus index is populated.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"},"k":{"type":"integer","minimum":0}},"required":["text"]}`),
	}, func(ctx context.Context, s *mcp.Session, raw json.RawMessage) (*mcp.CallToolResult, error) {
		var in struct {
			Text string `json:"text"`
			K    *int   `json:"k"`
		}
		if err := json.Unmarshal(raw, &in); err != nil {
			return mcp.ToolError("invalid arguments: %v", err), nil
		}
		matches := nearIndex.Near(ditto.Default().Of(in.Text), intOr(in.K, 3))
		if len(matches) == 0 {
			return mcp.ToolText("no near-duplicates in the (small, example-seeded) corpus"), nil
		}
		var b strings.Builder
		for _, m := range matches {
			fmt.Fprintf(&b, "%s\tdistance=%d\n", m.ID, m.Distance)
		}
		return mcp.ToolText("%s", b.String()), nil
	}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := srv.ServeStdio(ctx); err != nil {
		panic(err)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// intOr returns *p if p is non-nil, else def — for an optional JSON int field
// with a default.
func intOr(p *int, def int) int {
	if p != nil {
		return *p
	}
	return def
}
