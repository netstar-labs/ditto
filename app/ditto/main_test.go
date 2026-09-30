package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/netstar-labs/ditto"
)

// failingWriter returns errAfter once its n-th Write is reached, simulating a
// full disk or exceeded quota on the real os.Stdout.
type failingWriter struct {
	n   int
	err error
}

func (w *failingWriter) Write(p []byte) (int, error) {
	if w.n <= 0 {
		return 0, w.err
	}
	w.n--
	return len(p), nil
}

func TestWriteFingerprintsSurfacesWriteError(t *testing.T) {
	wantErr := errors.New("disk full")
	docs := []doc{{id: "a", text: "hello world"}, {id: "b", text: "goodbye world"}}
	err := writeFingerprints(&failingWriter{n: 0, err: wantErr}, docs, ditto.Featurizer{CharN: 4})
	if !errors.Is(err, wantErr) {
		t.Fatalf("writeFingerprints error = %v, want %v", err, wantErr)
	}
}

func TestWriteClustersSurfacesWriteError(t *testing.T) {
	wantErr := errors.New("disk full")
	ix := ditto.NewIndex(3)
	ix.Add("a", 0)
	err := writeClusters(&failingWriter{n: 0, err: wantErr}, ix, nil, 3)
	if !errors.Is(err, wantErr) {
		t.Fatalf("writeClusters error = %v, want %v", err, wantErr)
	}
}

func TestGatherSkipsSymlinkedDirectoryWithoutAbortingWalk(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a_before.txt"), "before")
	mustWrite(t, filepath.Join(root, "z_after.txt"), "after")

	// targetDir lives OUTSIDE root: the only path to it from the walk is
	// through the symlink, so if gather's skip logic were a no-op and it
	// instead descended into target_dir like a real subdirectory, this test
	// would still (wrongly) pass. Keeping it external makes the assertion
	// meaningful.
	targetDir := t.TempDir()
	mustWrite(t, filepath.Join(targetDir, "inside.txt"), "inside")

	if err := os.Symlink(targetDir, filepath.Join(root, "link_to_dir")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	docs, err := gather([]string{root})
	if err != nil {
		t.Fatalf("gather returned error, the symlink should have been skipped: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("got %d docs, want 2 (the symlinked dir's contents must not be read, but a_before/z_after must survive): %+v", len(docs), docs)
	}
}

func TestGatherFollowsSymlinkToRegularFile(t *testing.T) {
	root := t.TempDir()
	realFile := filepath.Join(root, "real.txt")
	mustWrite(t, realFile, "real content")

	if err := os.Symlink(realFile, filepath.Join(root, "link_to_file")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	docs, err := gather([]string{root})
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("got %d docs, want 2 (real.txt and link_to_file, both readable): %+v", len(docs), docs)
	}
}

func TestGatherSkipsTopLevelSymlinkToDirectory(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "target_dir")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(targetDir, "inside.txt"), "inside")

	link := filepath.Join(root, "link_to_dir")
	if err := os.Symlink(targetDir, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	docs, err := gather([]string{link})
	if err != nil {
		t.Fatalf("gather returned error for a top-level symlink-to-dir, want a clean skip: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("got %d docs, want 0 (a symlinked directory argument is skipped, not followed): %+v", len(docs), docs)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
