package blackbox_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	binOnce sync.Once
	binPath string
	binErr  error
)

func aiSedBinary(t *testing.T) string {
	t.Helper()
	binOnce.Do(func() {
		dir, err := os.MkdirTemp("", "ai_sed_bin_*")
		if err != nil {
			binErr = err
			return
		}
		binPath = filepath.Join(dir, "ai_sed")
		cmd := exec.Command("go", "build", "-o", binPath, "..")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			binErr = err
		}
	})
	if binErr != nil {
		t.Fatalf("build ai_sed: %v", binErr)
	}
	return binPath
}

func mustRun(t *testing.T, name string, args []string, input string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(input)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v: %v\nstderr: %s", name, args, err, errb.String())
	}
	return out.String()
}

func errorLines(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../server.log")
	if err != nil {
		t.Fatalf("read server.log: %v", err)
	}
	var b strings.Builder
	for _, line := range strings.SplitAfter(string(data), "\n") {
		if strings.Contains(line, "ERROR") {
			b.WriteString(line)
		}
	}
	return b.String()
}

func TestSubstituteScenarios(t *testing.T) {
	bin := aiSedBinary(t)

	cases := []struct {
		name   string
		script string
		input  string
	}{
		{
			name:   "simple substitute without g",
			script: "s/foo/bar/",
			input:  "foofoo\nfoo bar foo\nno match\n",
		},
		{
			name:   "substitute with g",
			script: "s/foo/bar/g",
			input:  "foofoo\nfoo bar foo\nno match\n",
		},
		{
			name:   "README capture groups",
			script: `s/.*\[user=\([^]]*\)\] - \(.*\)/\2|\1/`,
			input:  errorLines(t),
		},
		{
			name:   "README strip after pipe",
			script: "s/|.*//",
			input:  "Authentication token expired|user020\nNullPointerException in main thread|user003\nno pipe here\n",
		},
		{
			name:   "no match passthrough",
			script: "s/zzz/xxx/",
			input:  "abc\ndef\nghi\n",
		},
		{
			name:   "empty input",
			script: "s/foo/bar/g",
			input:  "",
		},
		{
			name:   "input without trailing newline",
			script: "s/foo/bar/",
			input:  "foofoo",
		},
		{
			name:   "ampersand in replacement",
			script: "s/foo/[&]/g",
			input:  "foofoo bar foo\n",
		},
		{
			name:   "escaped ampersand in replacement",
			script: `s/foo/x\&y/`,
			input:  "foo\n",
		},
		{
			name:   "anchors mid-pattern are literal",
			script: "s/a^b/X/",
			input:  "a^b\nab\n",
		},
		{
			name:   "backslash in input",
			script: `s/\\/X/g`,
			input:  "a\\b\\c\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustRun(t, bin, []string{tc.script}, tc.input)
			want := mustRun(t, "sed", []string{tc.script}, tc.input)
			if got != want {
				t.Errorf("output mismatch\nscript: %s\nwant: %q\n got: %q", tc.script, want, got)
			}
		})
	}
}

func TestPipelineProblem2(t *testing.T) {
	bin := aiSedBinary(t)
	input := errorLines(t)

	want := mustRun(t, "sed", []string{`s/.*\[user=\([^]]*\)\] - \(.*\)/\2|\1/`}, input)
	got := mustRun(t, bin, []string{`s/.*\[user=\([^]]*\)\] - \(.*\)/\2|\1/`}, input)
	if got != want {
		t.Fatalf("first sed stage mismatch")
	}

	want2 := mustRun(t, "sed", []string{"s/|.*//"}, want)
	got2 := mustRun(t, bin, []string{"s/|.*//"}, got)
	if got2 != want2 {
		t.Fatalf("second sed stage mismatch")
	}
}
