package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/spf13/pflag"
)

func main() {
	reverse := pflag.BoolP("reverse", "r", false, "reverse the result of comparisons")
	numeric := pflag.BoolP("numeric-sort", "n", false, "compare according to string numerical value")
	unique := pflag.BoolP("unique", "u", false, "output only the first of an equal run")
	pflag.Parse()

	if err := run(os.Stdin, os.Stdout, *reverse, *numeric, *unique); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer, reverse, numeric, unique bool) error {
	lines, err := readLines(in)
	if err != nil {
		return err
	}

	cmp := compareBytes
	if numeric {
		cmp = compareNumeric
	}

	sort.SliceStable(lines, func(i, j int) bool {
		c := cmp(lines[i], lines[j])
		if c == 0 && !unique {
			c = bytes.Compare(lines[i], lines[j])
		}
		if reverse {
			return c > 0
		}
		return c < 0
	})

	if unique {
		lines = dedupe(lines, cmp)
	}

	w := bufio.NewWriter(out)
	for _, line := range lines {
		w.Write(line)
		w.WriteByte('\n')
	}
	return w.Flush()
}

func readLines(in io.Reader) ([][]byte, error) {
	data, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	if data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	return bytes.Split(data, []byte{'\n'}), nil
}

func compareBytes(a, b []byte) int {
	return bytes.Compare(a, b)
}

// compareNumeric mirrors GNU sort -n: parse a leading numeric prefix
// (optional whitespace, optional sign, digits with optional fractional part).
// Lines without a numeric prefix sort as zero. Comparison is by value.
func compareNumeric(a, b []byte) int {
	va := parseNumericPrefix(a)
	vb := parseNumericPrefix(b)
	if va < vb {
		return -1
	}
	if va > vb {
		return 1
	}
	return 0
}

func parseNumericPrefix(s []byte) float64 {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	var val float64
	hasDigits := false
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		val = val*10 + float64(s[i]-'0')
		hasDigits = true
		i++
	}
	if i < len(s) && s[i] == '.' {
		i++
		frac := 0.1
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			val += float64(s[i]-'0') * frac
			frac /= 10
			hasDigits = true
			i++
		}
	}
	if !hasDigits {
		return 0
	}
	if neg {
		return -val
	}
	return val
}

func dedupe(lines [][]byte, cmp func(a, b []byte) int) [][]byte {
	if len(lines) == 0 {
		return lines
	}
	out := lines[:1]
	for i := 1; i < len(lines); i++ {
		if cmp(lines[i], out[len(out)-1]) != 0 {
			out = append(out, lines[i])
		}
	}
	return out
}
