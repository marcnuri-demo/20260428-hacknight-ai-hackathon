package blackbox_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// nativeGrep is the GNU grep binary used as the oracle. On Linux (and CI) this
// is `grep`; on macOS BSD grep ships as `grep` and GNU grep is `ggrep` under
// Homebrew. Differences between BSD and GNU grep can mask real regressions, so
// prefer GNU when available.
var nativeGrep string

// aiGrepBin is the compiled ai_grep binary, built once per `go test` run.
var aiGrepBin string

func TestMain(m *testing.M) {
	if env := os.Getenv("NATIVE_GREP"); env != "" {
		nativeGrep = env
	} else if p, err := exec.LookPath("ggrep"); err == nil {
		nativeGrep = p
	} else {
		nativeGrep = "grep"
	}
	// Surface which oracle binary the suite picked so a failure on a Mac dev
	// box without GNU grep installed is obviously diagnosable.
	fmt.Fprintf(os.Stderr, "blackbox tests: native grep oracle = %s\n", nativeGrep)

	tmp, err := os.MkdirTemp("", "ai_grep_bin_")
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: mkdir temp failed: %v\n", err)
		os.Exit(2)
	}
	defer os.RemoveAll(tmp)
	aiGrepBin = filepath.Join(tmp, "ai_grep")
	cmd := exec.Command("go", "build", "-o", aiGrepBin, ".")
	cmd.Dir = ".."
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: go build failed: %v\n%s", err, out)
		os.Exit(2)
	}
	os.Exit(m.Run())
}

// run executes cmd with stdin, returns stdout, stderr, exit code.
func run(t *testing.T, stdin string, name string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			t.Fatalf("failed to run %s: %v", name, err)
		}
	}
	return stdout.String(), stderr.String(), code
}

// compare runs both ai_grep and native grep with the same stdin and args and
// asserts byte-identical stdout and matching exit codes (0/1). Returns the
// shared stdout so callers can chain it into a follow-up stage.
func compare(t *testing.T, stdin string, args ...string) string {
	t.Helper()
	aiOut, _, aiCode := run(t, stdin, aiGrepBin, args...)
	natOut, _, natCode := run(t, stdin, nativeGrep, args...)
	if aiOut != natOut {
		t.Errorf("stdout mismatch for args %v\nai_grep: %q\nnative : %q", args, aiOut, natOut)
	}
	if aiCode != natCode {
		t.Errorf("exit-code mismatch for args %v\nai_grep: %d\nnative : %d", args, aiCode, natCode)
	}
	return aiOut
}

func TestLiteralMatch(t *testing.T) {
	compare(t, "a\nERROR x\nb\n", "ERROR")
}

func TestExtendedRegex(t *testing.T) {
	compare(t, "abc 123\nXYZ\n", "-E", "[A-Z]+")
}

func TestOnlyMatchingBundled(t *testing.T) {
	compare(t, "abc 123 def\n", "-oE", "[0-9]+")
}

func TestOnlyMatchingMultiplePerLine(t *testing.T) {
	// "-o" prints each non-overlapping match on its own line.
	compare(t, "1 a 2 b 3\nno digits\n4\n", "-oE", "[0-9]+")
}

func TestNoMatchExitCode1(t *testing.T) {
	out, _, code := run(t, "alpha\nbeta\n", aiGrepBin, "ZZZZZ")
	if out != "" {
		t.Errorf("expected empty stdout, got %q", out)
	}
	if code != 1 {
		t.Errorf("expected exit code 1 on no match, got %d", code)
	}
}

func TestMissingPatternExitCode2(t *testing.T) {
	_, _, code := run(t, "", aiGrepBin)
	if code != 2 {
		t.Errorf("expected exit code 2 on missing pattern, got %d", code)
	}
}

func TestInvalidRegexExitCode2(t *testing.T) {
	_, _, code := run(t, "abc\n", aiGrepBin, "-E", "[")
	if code != 2 {
		t.Errorf("expected exit code 2 on invalid regex, got %d", code)
	}
}

// readServerLog reads the repo-root server.log fixture for end-to-end checks.
func readServerLog(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "server.log"))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read server.log: %v", err)
	}
	return string(b)
}

func TestServerLogERROR(t *testing.T) {
	compare(t, readServerLog(t), "ERROR")
}

func TestServerLogProblem1Pipeline(t *testing.T) {
	// Chain ai_grep's own output through both stages so that a regression in
	// either stage surfaces here (don't seed stage 2 with native grep output).
	stage1 := compare(t, readServerLog(t), "ERROR")
	compare(t, stage1, "-oE", "[A-Za-z ]+$")
}

func TestServerLogProblem3Pipeline(t *testing.T) {
	stage1 := compare(t, readServerLog(t), "ERROR")
	compare(t, stage1, "-E", `^\[2026-04-16 (0[0-8]|1[7-9]|2[0-3]):`)
}

func TestLiteralOnlyMatching(t *testing.T) {
	// Literal `-o` (no `-E`) — covers the non-regex branch in main.go.
	compare(t, "foo bar foo baz foo\nno match here\nfoofoo\n", "-o", "foo")
}
