package urfavehelp

import (
	"fmt"
	"io"
	"strings"
	"unicode"
)

// capitalizeFirst returns s with its first rune capitalized.
func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// wrapWords reflows text at word boundaries to fit width.
func wrapWords(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	var current strings.Builder

	for _, word := range words {
		if current.Len() == 0 {
			current.WriteString(word)
			continue
		}
		if current.Len()+1+len(word) > width {
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(word)
		} else {
			current.WriteString(" ")
			current.WriteString(word)
		}
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

// printPaddedEntry writes an aligned label and description with hanging indents for continuation lines.
func printPaddedEntry(w io.Writer, label string, desc string, gutter int, maxWidth int, prefixGap int, minDescWidth int) {
	desc = capitalizeFirst(desc)
	if desc == "" {
		_, _ = fmt.Fprintf(w, "  %s\n", label)
		return
	}

	avail := max(minDescWidth, maxWidth-(gutter+prefixGap))

	lines := wrapWords(desc, avail)
	if len(lines) == 0 {
		_, _ = fmt.Fprintf(w, "  %s\n", label)
		return
	}

	_, _ = fmt.Fprintf(w, "  %-*s  %s\n", gutter, label, lines[0])

	hangingIndent := strings.Repeat(" ", gutter+prefixGap)
	for _, line := range lines[1:] {
		_, _ = fmt.Fprintf(w, "%s%s\n", hangingIndent, line)
	}
}
