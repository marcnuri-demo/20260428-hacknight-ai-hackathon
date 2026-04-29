package test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ai_sort_bin")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	binPath = filepath.Join(dir, "ai_sort")
	build := exec.Command("go", "build", "-o", binPath, "..")
	build.Stderr = os.Stderr
	build.Stdout = os.Stdout
	if err := build.Run(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func runPipe(t *testing.T, name string, input string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v failed: %v\nstderr: %s", name, args, err, stderr.String())
	}
	return stdout.Bytes()
}

func assertSameAsNative(t *testing.T, input string, args ...string) {
	t.Helper()
	native := runPipe(t, "sort", input, args...)
	ai := runPipe(t, binPath, input, args...)
	if !bytes.Equal(native, ai) {
		t.Errorf("output differs for args=%v\ninput=%q\nnative=%q\nai_sort=%q", args, input, native, ai)
	}
}

func TestPlainAlphabetic(t *testing.T) {
	input := "banana\napple\ncherry\nbanana\ndate\n"
	assertSameAsNative(t, input)
}

func TestReverse(t *testing.T) {
	input := "banana\napple\ncherry\nbanana\ndate\n"
	assertSameAsNative(t, input, "-r")
}

func TestNumeric(t *testing.T) {
	input := "  3 NetworkError\n 12 DatabaseError\n  1 AuthError\n  7 TimeoutError\n 12 DiskError\n"
	assertSameAsNative(t, input, "-n")
}

func TestUnique(t *testing.T) {
	input := "alpha\nbeta\nalpha\ngamma\nbeta\nalpha\n"
	assertSameAsNative(t, input, "-u")
}

func TestReverseNumericBundled(t *testing.T) {
	input := "  3 NetworkError\n 12 DatabaseError\n  1 AuthError\n  7 TimeoutError\n 12 DiskError\n"
	assertSameAsNative(t, input, "-rn")
}

func TestEmpty(t *testing.T) {
	assertSameAsNative(t, "")
	assertSameAsNative(t, "", "-r")
	assertSameAsNative(t, "", "-n")
	assertSameAsNative(t, "", "-u")
	assertSameAsNative(t, "", "-rn")
}

func TestSingleLine(t *testing.T) {
	assertSameAsNative(t, "only-line\n")
	assertSameAsNative(t, "only-line\n", "-r")
	assertSameAsNative(t, "only-line\n", "-n")
	assertSameAsNative(t, "only-line\n", "-u")
	assertSameAsNative(t, "only-line\n", "-rn")
}

func TestInputWithoutTrailingNewline(t *testing.T) {
	assertSameAsNative(t, "no-newline")
	assertSameAsNative(t, "zebra\napple")
}

func TestEmbeddedBlankLines(t *testing.T) {
	input := "banana\n\napple\n\ncherry\n\n"
	assertSameAsNative(t, input)
	assertSameAsNative(t, input, "-r")
	assertSameAsNative(t, input, "-u")
}
