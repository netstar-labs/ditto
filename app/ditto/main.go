// Command ditto fingerprints documents and finds near-duplicates with SimHash.
//
//	ditto fingerprint [-char N] [-word N] [files/dirs...]   # fingerprint <tab> id per input
//	ditto cluster     [-k N] [-min M] [-char N] [-word N] [files/dirs...]  # near-duplicate families
//	ditto version
//
// Inputs are files, directories (walked), or a single document on stdin. Each
// input file is one document; its id is its path (or "stdin").
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/netstar-labs/ditto"
)

// stamped by build/ditto via -ldflags -X.
var (
	version = "dev"
	build   = "none"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "fingerprint", "fp":
		err = fingerprint(os.Args[2:])
	case "cluster":
		err = cluster(os.Args[2:])
	case "version", "-version", "--version", "-v":
		fmt.Printf("ditto %s (%s) pipeline=%s\n", version, build, ditto.PipelineVersion)
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ditto:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ditto <fingerprint|cluster|version> [flags] [files/dirs...]")
	os.Exit(2)
}

func fingerprint(args []string) error {
	fs := flag.NewFlagSet("fingerprint", flag.ExitOnError)
	char := fs.Int("char", 4, "character n-gram length (0 disables)")
	word := fs.Int("word", 0, "word n-gram length (0 disables)")
	fs.Parse(args)
	f := ditto.Featurizer{CharN: *char, WordN: *word}

	docs, err := gather(fs.Args())
	if err != nil {
		return err
	}
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for _, d := range docs {
		fmt.Fprintf(w, "%s\t%s\n", f.Of(d.text), d.id)
	}
	return nil
}

func cluster(args []string) error {
	fs := flag.NewFlagSet("cluster", flag.ExitOnError)
	k := fs.Int("k", 3, "max Hamming distance for a near-duplicate")
	min := fs.Int("min", 2, "minimum members to report a cluster")
	char := fs.Int("char", 4, "character n-gram length (0 disables)")
	word := fs.Int("word", 0, "word n-gram length (0 disables)")
	fs.Parse(args)
	f := ditto.Featurizer{CharN: *char, WordN: *word}

	docs, err := gather(fs.Args())
	if err != nil {
		return err
	}
	ix := ditto.NewIndex(*k)
	for _, d := range docs {
		ix.Add(d.id, f.Of(d.text))
	}
	clusters := ix.Clusters(*k, *min)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	fmt.Fprintf(w, "%d document(s), %d near-duplicate cluster(s) at k=%d:\n", ix.Len(), len(clusters), *k)
	for i, g := range clusters {
		fmt.Fprintf(w, "\n[%d] %d members\n", i+1, len(g))
		for _, id := range g {
			fmt.Fprintf(w, "  %s\n", id)
		}
	}
	return nil
}

// ---- input gathering -------------------------------------------------------

type doc struct {
	id   string
	text string
}

// gather reads each path as one document; directories are walked (files only).
// With no paths it reads a single document from stdin.
func gather(paths []string) ([]doc, error) {
	if len(paths) == 0 {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		return []doc{{id: "stdin", text: string(b)}}, nil
	}
	var docs []doc
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			b, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			docs = append(docs, doc{id: p, text: string(b)})
			continue
		}
		err = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			docs = append(docs, doc{id: path, text: string(b)})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return docs, nil
}
