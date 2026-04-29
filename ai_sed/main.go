package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"
)

func main() {
	pflag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ai_sed 's/PATTERN/REPLACEMENT/[g]'")
	}
	pflag.Parse()
	args := pflag.Args()
	if len(args) != 1 {
		pflag.Usage()
		os.Exit(2)
	}
	sub, err := parseSubstitute(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "ai_sed:", err)
		os.Exit(1)
	}
	if err := run(sub, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "ai_sed:", err)
		os.Exit(1)
	}
}

func run(sub *substitute, in io.Reader, out io.Writer) error {
	r := bufio.NewReader(in)
	w := bufio.NewWriter(out)
	defer w.Flush()
	for {
		line, err := r.ReadString('\n')
		if len(line) > 0 {
			hadNewline := false
			if line[len(line)-1] == '\n' {
				line = line[:len(line)-1]
				hadNewline = true
			}
			if _, werr := w.WriteString(sub.apply(line)); werr != nil {
				return werr
			}
			if hadNewline {
				if werr := w.WriteByte('\n'); werr != nil {
					return werr
				}
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
