// Command ditto fingerprints documents and finds near-duplicates with SimHash.
//
//	ditto fingerprint [-char N] [-word N] [files/dirs...]   # fingerprint <tab> id per input (alias: fp)
//	ditto cluster     [-k N] [-min M] [-char N] [-word N] [files/dirs...]  # near-duplicate families
//	ditto version                                           # aliases: -v, -version, --version
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

// featurizerFlags registers the -char/-word flags shared by the fingerprint and
// cluster subcommands. Call the returned func after fs.Parse, once the flag
// values are populated.
func featurizerFlags(fs *flag.FlagSet) func() ditto.Featurizer {
	char := fs.Int("char", 4, "character n-gram length (0 disables)")
	word := fs.Int("word", 0, "word n-gram length (0 disables)")
	return func() ditto.Featurizer { return ditto.Featurizer{CharN: *char, WordN: *word} }
}

func fingerprint(args []string) error {
	fs := flag.NewFlagSet("fingerprint", flag.ExitOnError)
	featurizer := featurizerFlags(fs)
	fs.Parse(args)
	f := featurizer()

	docs, err := gather(fs.Args(), os.Stdin)
	if err != nil {
		return err
	}
	return writeFingerprints(os.Stdout, docs, f)
}

func writeFingerprints(out io.Writer, docs []doc, f ditto.Featurizer) error {
	w := bufio.NewWriter(out)
	for _, d := range docs {
		fmt.Fprintf(w, "%s\t%s\n", f.Of(d.text), d.id)
	}
	// bufio.Writer latches the first write error and returns it from every
	// subsequent call including Flush, so checking Flush's return here is
	// sufficient to surface a failed write instead of silently truncating.
	return w.Flush()
}

func cluster(args []string) error {
	fs := flag.NewFlagSet("cluster", flag.ExitOnError)
	k := fs.Int("k", 3, "max Hamming distance for a near-duplicate")
	min := fs.Int("min", 2, "minimum members to report a cluster")
	featurizer := featurizerFlags(fs)
	fs.Parse(args)
	f := featurizer()

	docs, err := gather(fs.Args(), os.Stdin)
	if err != nil {
		return err
	}
	ix := ditto.NewIndex(*k)
	for _, d := range docs {
		ix.Add(d.id, f.Of(d.text))
	}
	clusters := ix.Clusters(*k, *min)
	return writeClusters(os.Stdout, ix, clusters, *k)
}

func writeClusters(out io.Writer, ix *ditto.Index, clusters [][]string, k int) error {
	w := bufio.NewWriter(out)
	fmt.Fprintf(w, "%d document(s), %d near-duplicate cluster(s) at k=%d:\n", ix.Len(), len(clusters), k)
	for i, g := range clusters {
		fmt.Fprintf(w, "\n[%d] %d members\n", i+1, len(g))
		for _, id := range g {
			fmt.Fprintf(w, "  %s\n", id)
		}
	}
	// See writeFingerprints' Flush comment: this surfaces a failed write instead
	// of silently truncating output with exit code 0.
	return w.Flush()
}

// ---- input gathering -------------------------------------------------------

type doc struct {
	id   string
	text string
}

// gather reads each path as one document; directories are walked (files only).
// With no paths it reads a single document from stdin.
func gather(paths []string, stdin io.Reader) ([]doc, error) {
	if len(paths) == 0 {
		b, err := io.ReadAll(stdin)
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
			if d.Type()&fs.ModeSymlink != 0 {
				// A symlink's DirEntry is never IsDir (WalkDir doesn't follow
				// it to classify), so a symlink to a directory would otherwise
				// reach ReadFile below and fail with "is a directory" — which
				// aborts the WHOLE walk, discarding every doc already
				// gathered, for one link anywhere in the tree. Resolve it: a
				// symlink to a regular file is still read (matches a plain
				// file entry); a symlink to a directory, or a broken one, is
				// skipped rather than treated as fatal.
				target, statErr := os.Stat(path)
				if statErr != nil || target.IsDir() {
					return nil
				}
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
