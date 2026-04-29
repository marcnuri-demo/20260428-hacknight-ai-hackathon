package blackbox_test

import (
	"bytes"
	"errors"
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

func runWithStatus(t *testing.T, name string, args []string, input string) (stdout, stderr string, exit int) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(input)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return outb.String(), errb.String(), ee.ExitCode()
		}
		t.Fatalf("%s %v: %v", name, args, err)
	}
	return outb.String(), errb.String(), 0
}

func TestSubstituteErrors(t *testing.T) {
	bin := aiSedBinary(t)

	cases := []struct {
		name      string
		args      []string
		input     string
		stderrSub string
	}{
		{
			name:      "non-substitute command rejected",
			args:      []string{"d"},
			stderrSub: "only 's' command supported",
		},
		{
			name:      "non-slash delimiter rejected",
			args:      []string{"s|foo|bar|"},
			stderrSub: "only '/' delimiter supported",
		},
		{
			name:      "missing trailing delimiter rejected",
			args:      []string{"s/foo/bar"},
			stderrSub: "malformed substitute",
		},
		{
			name:      "incomplete script rejected",
			args:      []string{"s/foo"},
			stderrSub: "malformed substitute",
		},
		{
			name:      "unsupported flag rejected",
			args:      []string{"s/foo/bar/x"},
			stderrSub: "not supported",
		},
		{
			name:      "backreference in pattern rejected",
			args:      []string{`s/\(foo\)\1/x/`},
			stderrSub: "backreferences in pattern not supported",
		},
		{
			name:      "trailing backslash in replacement rejected",
			args:      []string{`s/foo/bar\/`},
			stderrSub: "malformed substitute",
		},
		{
			name:      "empty script rejected",
			args:      []string{""},
			stderrSub: "only 's' command supported",
		},
		{
			name:      "no positional argument exits with usage",
			args:      []string{},
			stderrSub: "usage:",
		},
		{
			name:      "two positional arguments rejected",
			args:      []string{"s/a/b/", "extra"},
			stderrSub: "usage:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exit := runWithStatus(t, bin, tc.args, tc.input)
			if exit == 0 {
				t.Fatalf("expected non-zero exit, got 0\nstdout: %q\nstderr: %q", stdout, stderr)
			}
			if !strings.Contains(stderr, tc.stderrSub) {
				t.Errorf("stderr does not contain %q\nstderr: %q", tc.stderrSub, stderr)
			}
			if stdout != "" {
				t.Errorf("expected empty stdout on error, got %q", stdout)
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
