package main

import (
	"fmt"
	"regexp"
	"strings"
)

type substitute struct {
	re     *regexp.Regexp
	repl   []replPart
	global bool
}

type replPart struct {
	literal string
	group   int // -1 = literal, 0 = whole match (&), 1..9 = backreference
}

func parseSubstitute(script string) (*substitute, error) {
	if len(script) < 2 || script[0] != 's' {
		return nil, fmt.Errorf("only 's' command supported")
	}
	delim := script[1]
	if delim != '/' {
		return nil, fmt.Errorf("only '/' delimiter supported")
	}
	parts, err := splitScript(script[2:], delim)
	if err != nil {
		return nil, err
	}
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed substitute: expected s/PATTERN/REPLACEMENT/FLAGS")
	}
	rePat, err := bre2re2(parts[0])
	if err != nil {
		return nil, err
	}
	re, err := regexp.Compile(rePat)
	if err != nil {
		return nil, fmt.Errorf("compile regex %q: %w", parts[0], err)
	}
	repl, err := parseReplacement(parts[1])
	if err != nil {
		return nil, err
	}
	sub := &substitute{re: re, repl: repl}
	for _, c := range parts[2] {
		switch c {
		case 'g':
			sub.global = true
		default:
			return nil, fmt.Errorf("flag %q not supported", c)
		}
	}
	return sub, nil
}

func splitScript(s string, delim byte) ([]string, error) {
	var parts []string
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			cur.WriteByte('\\')
			cur.WriteByte(s[i+1])
			i++
			continue
		}
		if c == delim {
			parts = append(parts, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	parts = append(parts, cur.String())
	return parts, nil
}

// bre2re2 translates a POSIX Basic Regular Expression into RE2 syntax.
// Handled BRE features: \(...\), \{m,n\}, \+, \?, ^/$ anchors at start/end of
// pattern, ., *, [...], [^...], POSIX [:class:], \n, \t, \\, \/.
func bre2re2(bre string) (string, error) {
	var out strings.Builder
	for i := 0; i < len(bre); {
		c := bre[i]
		switch c {
		case '\\':
			if i+1 >= len(bre) {
				return "", fmt.Errorf("trailing backslash in pattern")
			}
			next := bre[i+1]
			switch next {
			case '(', ')', '{', '}', '+', '?', '|':
				out.WriteByte(next)
			case '\\':
				out.WriteString(`\\`)
			case '/':
				out.WriteByte('/')
			case '.', '*', '^', '$', '[', ']':
				out.WriteByte('\\')
				out.WriteByte(next)
			case 'n':
				out.WriteString(`\n`)
			case 't':
				out.WriteString(`\t`)
			case '1', '2', '3', '4', '5', '6', '7', '8', '9':
				return "", fmt.Errorf("backreferences in pattern not supported")
			default:
				if isRE2Meta(next) {
					out.WriteByte('\\')
				}
				out.WriteByte(next)
			}
			i += 2
		case '[':
			end, err := findCharClassEnd(bre, i)
			if err != nil {
				return "", err
			}
			out.WriteString(bre[i : end+1])
			i = end + 1
		case '(', ')', '{', '}', '+', '?', '|':
			out.WriteByte('\\')
			out.WriteByte(c)
			i++
		case '^':
			if i == 0 {
				out.WriteByte('^')
			} else {
				out.WriteString(`\^`)
			}
			i++
		case '$':
			if i == len(bre)-1 {
				out.WriteByte('$')
			} else {
				out.WriteString(`\$`)
			}
			i++
		default:
			out.WriteByte(c)
			i++
		}
	}
	return out.String(), nil
}

func findCharClassEnd(s string, start int) (int, error) {
	i := start + 1
	if i < len(s) && s[i] == '^' {
		i++
	}
	if i < len(s) && s[i] == ']' {
		i++
	}
	for i < len(s) {
		if s[i] == '[' && i+1 < len(s) && s[i+1] == ':' {
			j := i + 2
			for j+1 < len(s) && !(s[j] == ':' && s[j+1] == ']') {
				j++
			}
			if j+1 >= len(s) {
				return 0, fmt.Errorf("unclosed POSIX character class")
			}
			i = j + 2
			continue
		}
		if s[i] == ']' {
			return i, nil
		}
		i++
	}
	return 0, fmt.Errorf("unclosed character class")
}

func isRE2Meta(c byte) bool {
	switch c {
	case '\\', '.', '+', '*', '?', '(', ')', '|', '[', ']', '{', '}', '^', '$':
		return true
	}
	return false
}

func parseReplacement(s string) ([]replPart, error) {
	var parts []replPart
	var lit strings.Builder
	flush := func() {
		if lit.Len() > 0 {
			parts = append(parts, replPart{literal: lit.String(), group: -1})
			lit.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' {
			if i+1 >= len(s) {
				return nil, fmt.Errorf("trailing backslash in replacement")
			}
			next := s[i+1]
			switch next {
			case '&', '\\', '/':
				lit.WriteByte(next)
			case 'n':
				lit.WriteByte('\n')
			case 't':
				lit.WriteByte('\t')
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				flush()
				parts = append(parts, replPart{group: int(next - '0')})
			default:
				lit.WriteByte(next)
			}
			i++
			continue
		}
		if c == '&' {
			flush()
			parts = append(parts, replPart{group: 0})
			continue
		}
		lit.WriteByte(c)
	}
	flush()
	return parts, nil
}

func (s *substitute) apply(line string) string {
	if !s.global {
		loc := s.re.FindStringSubmatchIndex(line)
		if loc == nil {
			return line
		}
		var out strings.Builder
		out.WriteString(line[:loc[0]])
		out.WriteString(s.expand(line, loc))
		out.WriteString(line[loc[1]:])
		return out.String()
	}
	matches := s.re.FindAllStringSubmatchIndex(line, -1)
	if matches == nil {
		return line
	}
	var out strings.Builder
	last := 0
	for _, loc := range matches {
		out.WriteString(line[last:loc[0]])
		out.WriteString(s.expand(line, loc))
		last = loc[1]
	}
	out.WriteString(line[last:])
	return out.String()
}

func (s *substitute) expand(line string, loc []int) string {
	var out strings.Builder
	for _, p := range s.repl {
		if p.group < 0 {
			out.WriteString(p.literal)
			continue
		}
		idx := p.group * 2
		if idx+1 >= len(loc) || loc[idx] < 0 {
			continue
		}
		out.WriteString(line[loc[idx]:loc[idx+1]])
	}
	return out.String()
}
