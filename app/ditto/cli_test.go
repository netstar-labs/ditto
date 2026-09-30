package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// These are subprocess (integration) tests: main/usage call os.Exit directly,
// which cannot be exercised in-process without killing the test binary — the
// standard artifact for that is building the real binary once and exec'ing it
// with different args, which is what TestMain below does. Set
// DITTO_TEST_GOCOVERDIR to also attribute real coverage numbers to these
// paths via `go build -cover` + `go tool covdata` (see docs/audits/test-ledger.md).
var (
	binPath string
	covDir  string
)

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "ditto-cli-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	binPath = filepath.Join(tmp, "ditto-bin")
	covDir = os.Getenv("DITTO_TEST_GOCOVERDIR")
	args := []string{"build"}
	if covDir != "" {
		args = append(args, "-cover")
	}
	args = append(args, "-o", binPath, ".")
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic(string(out) + err.Error())
	}
	os.Exit(m.Run())
}

func runCLI(t *testing.T, stdin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if covDir != "" {
		cmd.Env = append(os.Environ(), "GOCOVERDIR="+covDir)
	}
	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return out.String(), errb.String(), ee.ExitCode()
		}
		t.Fatalf("exec %s %v: %v", binPath, args, err)
	}
	return out.String(), errb.String(), 0
}

func TestCLIVersionAndAliases(t *testing.T) {
	for _, arg := range []string{"version", "-v", "-version", "--version"} {
		out, _, code := runCLI(t, "", arg)
		if code != 0 {
			t.Errorf("%s: exit = %d, want 0", arg, code)
		}
		if !strings.HasPrefix(out, "ditto ") || !strings.Contains(out, "pipeline=") {
			t.Errorf("%s: output = %q, want ditto <version> ... pipeline=...", arg, out)
		}
	}
}

func TestCLINoArgsShowsUsageAndExits2(t *testing.T) {
	_, stderr, code := runCLI(t, "")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("stderr = %q, want a usage message", stderr)
	}
}

func TestCLIUnknownCommandShowsUsageAndExits2(t *testing.T) {
	_, stderr, code := runCLI(t, "", "bogus-command")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "usage:") {
		t.Errorf("stderr = %q, want a usage message", stderr)
	}
}

func TestCLIFingerprintFromStdin(t *testing.T) {
	out, stderr, code := runCLI(t, "hello world", "fingerprint")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if !regexp.MustCompile(`^[0-9a-f]{16}\tstdin\n$`).MatchString(out) {
		t.Errorf("output = %q, want <16 hex>\\tstdin\\n", out)
	}
}

func TestCLIFingerprintSingleFileArgument(t *testing.T) {
	// gather's top-level "p is a regular file" branch (as opposed to a
	// directory or stdin) — the documented `ditto fingerprint corpus/*.eml`
	// usage, where the shell expands to individual file arguments.
	dir := t.TempDir()
	file := filepath.Join(dir, "doc.txt")
	mustWrite(t, file, "hello world")

	out, stderr, code := runCLI(t, "", "fingerprint", file)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	want, _, _ := runCLI(t, "hello world", "fingerprint")
	// Same content, so the fingerprint column must match; only the id column
	// (path vs "stdin") differs.
	wantFP := strings.SplitN(want, "\t", 2)[0]
	if !strings.HasPrefix(out, wantFP+"\t") {
		t.Errorf("output = %q, want fingerprint %q for identical content", out, wantFP)
	}
	if !strings.HasSuffix(strings.TrimSpace(out), file) {
		t.Errorf("output = %q, want id column to be the file path %q", out, file)
	}
}

func TestCLIFingerprintAliasFp(t *testing.T) {
	full, _, _ := runCLI(t, "hello world", "fingerprint")
	alias, _, code := runCLI(t, "hello world", "fp")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if alias != full {
		t.Errorf("fp alias output %q != fingerprint output %q", alias, full)
	}
}

func TestCLIClusterOverDirectory(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.txt"), "Your Apple ID has been locked. Please verify your account.")
	mustWrite(t, filepath.Join(dir, "b.txt"), "Your Apple ID has been locked. Please verify your account!")
	mustWrite(t, filepath.Join(dir, "c.txt"), "Completely unrelated weather report for today.")

	out, stderr, code := runCLI(t, "", "cluster", "-k", "8", "-min", "2", dir)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(out, "3 document(s)") {
		t.Errorf("output = %q, want a 3-document summary line", out)
	}
	if !strings.Contains(out, "2 members") {
		t.Errorf("output = %q, want a's/b's 2-member cluster", out)
	}
}

func TestCLINonexistentPathExits1(t *testing.T) {
	for _, cmd := range []string{"fingerprint", "cluster"} {
		_, stderr, code := runCLI(t, "", cmd, "/no/such/path/ditto-test")
		if code != 1 {
			t.Errorf("%s: exit = %d, want 1", cmd, code)
		}
		if !strings.HasPrefix(stderr, "ditto:") {
			t.Errorf("%s: stderr = %q, want the \"ditto: <err>\" prefix from main's error path", cmd, stderr)
		}
	}
}
