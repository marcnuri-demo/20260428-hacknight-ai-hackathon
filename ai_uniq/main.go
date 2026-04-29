// ai_uniq — minimal stdin/stdout clone of GNU uniq.
//
// Scope (per issue #4): collapse adjacent duplicate lines (default) and emit
// counts in GNU's `%7d %s\n` format under `-c`. No file args, no other flags.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"
)

func main() {
	count := pflag.BoolP("count", "c", false, "prefix lines by the number of occurrences")
	pflag.Parse()
	if err := run(os.Stdin, os.Stdout, *count); err != nil {
		fmt.Fprintln(os.Stderr, "ai_uniq:", err)
		os.Exit(1)
	}
}

// run reads `in`, collapses adjacent duplicate lines, and writes results to
// `out`. Output is always `\n`-terminated, matching GNU uniq's normalization
// even when the input's final line lacks a terminator.
func run(in io.Reader, out io.Writer, count bool) (retErr error) {
	r := bufio.NewReader(in)
	w := bufio.NewWriter(out)
	defer func() {
		if err := w.Flush(); err != nil && retErr == nil {
			retErr = err
		}
	}()

	var prev string
	var have bool
	var n uint64

	emit := func() error {
		if !have {
			return nil
		}
		if count {
			_, err := fmt.Fprintf(w, "%7d %s\n", n, prev)
			return err
		}
		if _, err := w.WriteString(prev); err != nil {
			return err
		}
		_, err := w.WriteString("\n")
		return err
	}

	for {
		line, err := r.ReadString('\n')
		if line != "" || err == nil {
			content := line
			if len(content) > 0 && content[len(content)-1] == '\n' {
				content = content[:len(content)-1]
			}
			switch {
			case !have:
				prev, n, have = content, 1, true
			case content == prev:
				n++
			default:
				if err := emit(); err != nil {
					return err
				}
				prev, n = content, 1
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return emit()
}
