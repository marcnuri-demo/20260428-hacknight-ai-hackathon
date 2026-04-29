package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/pflag"
)

const (
	exitMatch    = 0
	exitNoMatch  = 1
	exitError    = 2
	maxLineBytes = 16 * 1024 * 1024 // 16 MiB — well above any realistic log line.
)

func main() {
	os.Exit(realMain(os.Stdin, os.Stdout, os.Stderr, os.Args[1:]))
}

func realMain(stdin io.Reader, stdout, stderr io.Writer, args []string) int {
	fs := pflag.NewFlagSet("ai_grep", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	extended := fs.BoolP("extended-regexp", "E", false, "interpret pattern as POSIX extended regex")
	onlyMatching := fs.BoolP("only-matching", "o", false, "print only the matched parts of each line")

	if err := fs.Parse(args); err != nil {
		return exitError
	}
	rest := fs.Args()
	if len(rest) < 1 {
		fmt.Fprintln(stderr, "ai_grep: missing pattern")
		return exitError
	}
	pattern := rest[0]

	var re *regexp.Regexp
	if *extended {
		r, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Fprintf(stderr, "ai_grep: invalid regex: %v\n", err)
			return exitError
		}
		re = r
	}

	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 64*1024), maxLineBytes)
	bw := bufio.NewWriter(stdout)
	defer bw.Flush()

	matched := false
	for scanner.Scan() {
		line := scanner.Text()
		if *onlyMatching {
			if re == nil {
				// Literal -o: each non-overlapping occurrence on its own line.
				idx := 0
				for {
					i := strings.Index(line[idx:], pattern)
					if i < 0 {
						break
					}
					matched = true
					bw.WriteString(pattern)
					bw.WriteByte('\n')
					idx += i + len(pattern)
					if len(pattern) == 0 {
						break
					}
				}
			} else {
				for _, m := range re.FindAllString(line, -1) {
					matched = true
					bw.WriteString(m)
					bw.WriteByte('\n')
				}
			}
			continue
		}
		var hit bool
		if re != nil {
			hit = re.MatchString(line)
		} else {
			hit = strings.Contains(line, pattern)
		}
		if hit {
			matched = true
			bw.WriteString(line)
			bw.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "ai_grep: read error: %v\n", err)
		return exitError
	}
	if matched {
		return exitMatch
	}
	return exitNoMatch
}
