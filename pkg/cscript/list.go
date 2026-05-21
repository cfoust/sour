package cscript

import (
	"strings"
)

// parseList splits a CubeScript list into its elements.
// Lists are space-separated, with support for "quoted strings" and [bracketed blocks].
func parseList(s string) []string {
	var result []string
	i := 0

	for {
		// Skip whitespace and comments
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\n') {
			i++
		}
		if i < len(s) && s[i] == '/' && i+1 < len(s) && s[i+1] == '/' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}

		if i >= len(s) {
			break
		}

		switch s[i] {
		case '"':
			// Quoted string
			i++ // skip opening "
			start := i
			for i < len(s) && s[i] != '"' && s[i] != '\n' {
				if s[i] == '^' {
					i++
				}
				i++
			}
			result = append(result, s[start:i])
			if i < len(s) && s[i] == '"' {
				i++
			}

		case '(', '[':
			// Bracketed block
			open := s[i]
			close := byte(']')
			if open == '(' {
				close = ')'
			}
			i++ // skip opener
			start := i
			depth := 1
			for i < len(s) && depth > 0 {
				if s[i] == open {
					depth++
				} else if s[i] == close {
					depth--
				} else if s[i] == '"' {
					i++
					for i < len(s) && s[i] != '"' && s[i] != '\n' {
						if s[i] == '^' {
							i++
						}
						i++
					}
				}
				if depth > 0 {
					i++
				}
			}
			result = append(result, s[start:i])
			if i < len(s) {
				i++ // skip closer
			}

		case ')':
			return result
		case ']':
			return result
		case 0:
			return result
		case ';':
			i++
			continue

		default:
			// Unquoted word
			start := i
			depth := 0
			for i < len(s) {
				c := s[i]
				switch c {
				case ' ', '\t', '\r', '\n', ';':
					if depth == 0 {
						goto wordDone
					}
				case '"':
					if depth == 0 {
						goto wordDone
					}
				case '/':
					if depth == 0 && i+1 < len(s) && s[i+1] == '/' {
						goto wordDone
					}
				case '[', '(':
					depth++
				case ']':
					if depth <= 0 {
						goto wordDone
					}
					depth--
				case ')':
					if depth <= 0 {
						goto wordDone
					}
					depth--
				}
				i++
			}
		wordDone:
			if i > start {
				result = append(result, s[start:i])
			}
		}
	}

	return result
}

// listLen returns the number of elements in a CubeScript list.
func listLen(s string) int {
	return len(parseList(s))
}

// listAt returns the element at position pos in a CubeScript list.
func listAt(s string, pos int) string {
	items := parseList(s)
	if pos >= 0 && pos < len(items) {
		return items[pos]
	}
	return ""
}

// listIndexOf returns the index of elem in the list, or -1 if not found.
func listIndexOf(list, elem string) int {
	items := parseList(list)
	for i, item := range items {
		if item == elem {
			return i
		}
	}
	return -1
}

// subList extracts a sublist starting at skip, with up to count elements.
func subList(list string, skip, count int) string {
	items := parseList(list)
	if skip < 0 {
		skip = 0
	}
	if skip >= len(items) {
		return ""
	}
	end := len(items)
	if count >= 0 && skip+count < end {
		end = skip + count
	}
	return strings.Join(items[skip:end], " ")
}

// listDel removes all elements from list that appear in del.
func listDel(list, del string) string {
	items := parseList(list)
	delItems := parseList(del)
	delSet := make(map[string]bool)
	for _, d := range delItems {
		delSet[d] = true
	}

	var result []string
	for _, item := range items {
		if !delSet[item] {
			result = append(result, item)
		}
	}
	return strings.Join(result, " ")
}

// listSplice inserts vals into list at position skip, replacing count elements.
func listSplice(list, vals string, skip, count int) string {
	items := parseList(list)
	if skip < 0 {
		skip = 0
	}
	if skip > len(items) {
		skip = len(items)
	}
	if count < 0 {
		count = 0
	}
	end := skip + count
	if end > len(items) {
		end = len(items)
	}

	var result []string
	result = append(result, items[:skip]...)
	if vals != "" {
		result = append(result, parseList(vals)...)
	}
	result = append(result, items[end:]...)
	return strings.Join(result, " ")
}

// prettyList joins list elements with commas and an optional conjunction.
func prettyList(list, conj string) string {
	items := parseList(list)
	n := len(items)
	if n == 0 {
		return ""
	}
	if n == 1 {
		return items[0]
	}
	if conj == "" {
		return strings.Join(items, ", ")
	}
	if n == 2 {
		return items[0] + " " + conj + " " + items[1]
	}
	return strings.Join(items[:n-1], ", ") + ", " + conj + " " + items[n-1]
}
