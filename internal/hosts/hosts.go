// Package hosts parses host lists: ranges such as "web[01-03,07]", and host files.
package hosts

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// MaxExpansion limits how many hosts one pattern may expand to, to catch typos like
// "web[1-1000000]".
const MaxExpansion = 100_000

// Expand expands each pattern and returns the hosts in order, without duplicates.
//
// A pattern may contain bracketed ranges: "web[01-03]" is web01, web02 and web03 (leading zeros
// set the width), "db[1,4-5]" is db1, db4 and db5, and "r[1-2]n[1-2]" is r1n1, r1n2, r2n1 and r2n2.
// IPv6 literals in brackets ("[::1]:22") are left alone.
func Expand(patterns ...string) ([]string, error) {
	var out []string
	seen := make(map[string]struct{})
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		expanded, err := expand(p)
		if err != nil {
			return nil, fmt.Errorf("host pattern %q: %w", p, err)
		}
		for _, h := range expanded {
			if _, ok := seen[h]; !ok {
				seen[h] = struct{}{}
				out = append(out, h)
			}
		}
		if len(out) > MaxExpansion {
			return nil, fmt.Errorf("more than %d hosts", MaxExpansion)
		}
	}
	return out, nil
}

func expand(p string) ([]string, error) {
	open := strings.IndexByte(p, '[')
	if open < 0 {
		return []string{p}, nil
	}
	end := strings.IndexByte(p[open:], ']')
	if end < 0 {
		return nil, fmt.Errorf("unclosed '['")
	}
	end += open
	body := p[open+1 : end]
	if strings.Contains(body, ":") {
		// An IPv6 literal, not a range. Expand anything after it.
		rest, err := expand(p[end+1:])
		if err != nil {
			return nil, err
		}
		return prefixAll(p[:end+1], rest), nil
	}

	items, err := parseRange(body)
	if err != nil {
		return nil, err
	}
	rest, err := expand(p[end+1:])
	if err != nil {
		return nil, err
	}
	if len(items)*len(rest) > MaxExpansion {
		return nil, fmt.Errorf("expands to more than %d hosts", MaxExpansion)
	}
	var out []string
	for _, it := range items {
		out = append(out, prefixAll(p[:open]+it, rest)...)
	}
	return out, nil
}

func prefixAll(prefix string, suffixes []string) []string {
	out := make([]string, len(suffixes))
	for i, s := range suffixes {
		out[i] = prefix + s
	}
	return out
}

// parseRange parses the inside of brackets: comma-separated numbers or lo-hi ranges.
func parseRange(body string) ([]string, error) {
	var out []string
	for part := range strings.SplitSeq(body, ",") {
		part = strings.TrimSpace(part)
		lo, hi, isRange := strings.Cut(part, "-")
		if !isRange {
			hi = lo
		}
		a, errA := strconv.Atoi(lo)
		b, errB := strconv.Atoi(hi)
		if errA != nil || errB != nil || a < 0 || b < a {
			return nil, fmt.Errorf("bad range %q", part)
		}
		if b-a >= MaxExpansion {
			return nil, fmt.Errorf("range %q is too large", part)
		}
		width := 0
		if len(lo) > 1 && lo[0] == '0' {
			width = len(lo)
		}
		for n := a; n <= b; n++ {
			out = append(out, fmt.Sprintf("%0*d", width, n))
		}
	}
	return out, nil
}

// Read returns the host patterns in r: one or more per line, separated by spaces or commas.
// Text after '#' is a comment.
func Read(r io.Reader) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line, _, _ := strings.Cut(sc.Text(), "#")
		out = append(out, strings.FieldsFunc(line, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t'
		})...)
	}
	return out, sc.Err()
}
