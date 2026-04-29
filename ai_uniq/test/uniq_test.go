// Black-box behavioral tests for ai_uniq.
//
// These tests build the real ai_uniq binary, pipe real input through it AND
// through the system's native `uniq`, and assert byte-identical stdout.
// No mocks, no stubs, no fakes.
//
// The `-c` output format is pinned to GNU (`%7d %s\n`). macOS/BSD `uniq -c`
// uses a 4-wide prefix, so on BSD systems the `-c` byte-identity subtests
// skip. CI runs on Linux (GNU) and exercises every case.
package test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var (
	binPath  string
	gnuUniq  bool
	repoRoot string
)

func TestMain(m *testing.M) {
	// Resolve repo paths from this test file's location so `go test ./...`
	// works regardless of cwd.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("could not resolve test file path")
	}
	testDir := filepath.Dir(thisFile)
	toolDir := filepath.Dir(testDir)
	repoRoot = filepath.Dir(toolDir)

	tmp, err := os.MkdirTemp("", "ai_uniq_bin_")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	binPath = filepath.Join(tmp, "ai_uniq")

	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = toolDir
	if out, err := build.CombinedOutput(); err != nil {
		panic("go build failed: " + err.Error() + "\n" + string(out))
	}

	// Detect GNU uniq once (7-wide right-aligned `-c` prefix).
	probe := exec.Command("uniq", "-c")
	probe.Stdin = strings.NewReader("a\na\n")
	out, err := probe.Output()
	if err == nil && bytes.HasPrefix(out, []byte("      2 a")) {
		gnuUniq = true
	}

	os.Exit(m.Run())
}

// pipe runs cmdName with args, feeding `input` on stdin, and returns stdout.
func pipe(t *testing.T, cmdName, input string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(cmdName, args...)
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v failed: %v\nstderr: %s", cmdName, args, err, stderr.String())
	}
	return stdout.Bytes()
}

func assertBytesEq(t *testing.T, ai, native []byte, label string) {
	t.Helper()
	if !bytes.Equal(ai, native) {
		t.Fatalf("%s: ai_uniq output differs from native uniq.\n--- ai (%d bytes) ---\n%q\n--- native (%d bytes) ---\n%q",
			label, len(ai), ai, len(native), native)
	}
}

// checkPair runs the input through ai_uniq and through native uniq with the
// same args and asserts byte-identity. `-c` runs are skipped when native
// uniq is BSD.
func checkPair(t *testing.T, name, input string, args ...string) {
	t.Helper()
	t.Run(name+"/default", func(t *testing.T) {
		ai := pipe(t, binPath, input)
		native := pipe(t, "uniq", input)
		assertBytesEq(t, ai, native, name+" default")
	})
	t.Run(name+"/count", func(t *testing.T) {
		if !gnuUniq {
			t.Skip("native uniq is not GNU; skipping -c byte-identity check")
		}
		ai := pipe(t, binPath, input, "-c")
		native := pipe(t, "uniq", input, "-c")
		assertBytesEq(t, ai, native, name+" -c")
	})
	_ = args // reserved for future flag combos; default + -c are the contract
}

func TestAdjacentDuplicatesCollapsed(t *testing.T) {
	checkPair(t, "mixed", "a\na\nb\nb\nb\nc\na\na\n")
}

func TestEmptyInput(t *testing.T) {
	checkPair(t, "empty", "")
}

func TestSingleLine(t *testing.T) {
	checkPair(t, "single", "only\n")
}

func TestNoDuplicates(t *testing.T) {
	checkPair(t, "no-dupes", "a\nb\nc\nd\ne\n")
}

func TestAllDuplicates(t *testing.T) {
	checkPair(t, "all-dupes", "same\nsame\nsame\nsame\nsame\n")
}

// Regression: GNU uniq normalizes output to be `\n`-terminated even when the
// final input line lacked a terminator. An earlier draft propagated the
// missing terminator instead. BSD uniq preserves the missing terminator —
// since our contract is GNU, this test runs only against GNU.
func TestUnterminatedFinalLine(t *testing.T) {
	if !gnuUniq {
		t.Skip("native uniq is not GNU; trailing-newline normalization differs")
	}
	for _, tc := range []struct{ name, input string }{
		{"no-trailing-newline", "a\nb"},
		{"single-no-newline", "a"},
		{"dupes-no-trailing", "a\na"},
	} {
		t.Run(tc.name+"/default", func(t *testing.T) {
			ai := pipe(t, binPath, tc.input)
			native := pipe(t, "uniq", tc.input)
			assertBytesEq(t, ai, native, tc.name+" default")
		})
		t.Run(tc.name+"/count", func(t *testing.T) {
			ai := pipe(t, binPath, tc.input, "-c")
			native := pipe(t, "uniq", tc.input, "-c")
			assertBytesEq(t, ai, native, tc.name+" -c")
		})
	}
}

func TestBlankLines(t *testing.T) {
	checkPair(t, "blank-lines", "\n\n\nfoo\nfoo\n")
}

// Regression: lines exceeding bufio's initial buffer must be read whole.
// bufio.Reader.ReadString grows as needed; bufio.Scanner would not.
func TestVeryLongLine(t *testing.T) {
	long := strings.Repeat("x", 200_000)
	input := long + "\n" + long + "\nshort\n"
	checkPair(t, "very-long-line", input)
}

func TestCountMixedDigitWidths(t *testing.T) {
	if !gnuUniq {
		t.Skip("native uniq is not GNU; skipping -c byte-identity check")
	}
	var b strings.Builder
	b.WriteString("alpha\n")
	for i := 0; i < 12; i++ {
		b.WriteString("beta\n")
	}
	for i := 0; i < 123; i++ {
		b.WriteString("gamma\n")
	}
	b.WriteString("delta\n")
	ai := pipe(t, binPath, b.String(), "-c")
	native := pipe(t, "uniq", b.String(), "-c")
	assertBytesEq(t, ai, native, "-c mixed widths")
}

// errorTypesSorted produces the same input shape that feeds `uniq` in
// README Problem 1: server.log -> grep ERROR -> grep -oE -> sort. We use
// native helpers for the prefix so this test stays a uniq-specific check
// (the end-to-end Problem 1 diff is enforced by CI's validate.sh).
func errorTypesSorted(t *testing.T) string {
	t.Helper()
	logBytes, err := os.ReadFile(filepath.Join(repoRoot, "server.log"))
	if err != nil {
		t.Fatalf("read server.log: %v", err)
	}
	cmd := exec.Command("sh", "-c", `grep "ERROR" | grep -oE "[A-Za-z ]+$" | sort`)
	cmd.Stdin = bytes.NewReader(logBytes)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("error-types pipeline failed: %v\nstderr: %s", err, stderr.String())
	}
	return stdout.String()
}

func TestReadmePipelineErrorTypes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires sh")
	}
	checkPair(t, "readme-error-types", errorTypesSorted(t))
}
