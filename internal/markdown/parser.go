// Package markdown provides a deterministic, lightweight scanner for Markdown documents,
// focusing on code fences, inline code spans, and table column extraction.
package markdown

import (
	"strings"
	"unicode"
)

// Fence tracks an open code block fence.
type Fence struct {
	Char   rune // '`' or '~'
	Length int  // number of backticks or tildes (>= 3)
	Line   int  // 1-based line number where fence opened
}

// FenceMismatch describes an unclosed or improperly nested code fence.
type FenceMismatch struct {
	Line    int
	Message string
}

// CheckFences scans text and returns findings for any unclosed code fences.
func CheckFences(text string) []FenceMismatch {
	lines := strings.Split(text, "\n")
	var current *Fence

	for i, rawLine := range lines {
		lineNo := i + 1
		trimmed := strings.TrimLeftFunc(rawLine, unicode.IsSpace)
		char, count := parseFenceMarker(trimmed)
		if count < 3 {
			continue
		}

		if current == nil {
			// Opening fence
			current = &Fence{
				Char:   char,
				Length: count,
				Line:   lineNo,
			}
		} else {
			// In a fence: only a closing fence of the SAME character and AT LEAST the same length closes it
			if char == current.Char && count >= current.Length {
				// Must not have info string on closing fence (per CommonMark)
				rest := strings.TrimSpace(trimmed[count:])
				if rest == "" {
					current = nil
				}
			}
		}
	}

	if current != nil {
		return []FenceMismatch{
			{
				Line:    current.Line,
				Message: "unclosed code fence (opened with " + strings.Repeat(string(current.Char), current.Length) + ")",
			},
		}
	}
	return nil
}

func parseFenceMarker(s string) (rune, int) {
	if len(s) == 0 {
		return 0, 0
	}
	first := rune(s[0])
	if first != '`' && first != '~' {
		return 0, 0
	}
	count := 0
	for _, r := range s {
		if r == first {
			count++
		} else {
			break
		}
	}
	return first, count
}

// TableRow represents a parsed table row.
type TableRow struct {
	LineNo  int
	Columns int
	IsDelim bool
}

// TableBlock represents a contiguous markdown table block.
type TableBlock struct {
	Header   TableRow
	Rows     []TableRow
	HasDelim bool
}

// ExtractTables scans text and returns all markdown table blocks outside of code fences.
func ExtractTables(text string) []TableBlock {
	lines := strings.Split(text, "\n")
	var currentFence *Fence
	var tables []TableBlock
	var currentRows []TableRow

	flush := func() {
		if len(currentRows) >= 2 {
			// Check if row 1 is a delimiter row (|---|---|)
			hasDelim := len(currentRows) >= 2 && currentRows[1].IsDelim
			tables = append(tables, TableBlock{
				Header:   currentRows[0],
				Rows:     currentRows[1:],
				HasDelim: hasDelim,
			})
		}
		currentRows = nil
	}

	for i, rawLine := range lines {
		lineNo := i + 1
		trimmed := strings.TrimLeftFunc(rawLine, unicode.IsSpace)

		// Check code fence state
		char, count := parseFenceMarker(trimmed)
		if count >= 3 {
			if currentFence == nil {
				flush()
				currentFence = &Fence{Char: char, Length: count, Line: lineNo}
				continue
			} else if char == currentFence.Char && count >= currentFence.Length {
				rest := strings.TrimSpace(trimmed[count:])
				if rest == "" {
					currentFence = nil
					continue
				}
			}
		}

		if currentFence != nil {
			continue
		}

		// Outside code fence: check for table row
		cols, isTable, isDelim := parseTableRow(trimmed)
		if isTable {
			currentRows = append(currentRows, TableRow{
				LineNo:  lineNo,
				Columns: cols,
				IsDelim: isDelim,
			})
		} else {
			flush()
		}
	}
	flush()

	return tables
}

// parseTableRow inspects a line. It returns the number of structural columns,
// whether it represents a table row (starts with '|'), and whether it is a delimiter row.
func parseTableRow(line string) (cols int, isTable bool, isDelim bool) {
	if !strings.HasPrefix(line, "|") {
		return 0, false, false
	}

	// Tokenize line: skip escaped pipes (\|) and pipes inside inline code spans (`...`)
	structuralPipes := 0
	inCode := false
	codeDelimLen := 0
	escaped := false

	runes := []rune(line)
	n := len(runes)

	for i := 0; i < n; i++ {
		r := runes[i]
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}

		if r == '`' {
			// Count consecutive backticks
			backticks := 1
			for i+1 < n && runes[i+1] == '`' {
				backticks++
				i++
			}
			if !inCode {
				inCode = true
				codeDelimLen = backticks
			} else if backticks == codeDelimLen {
				inCode = false
				codeDelimLen = 0
			}
			continue
		}

		if !inCode && r == '|' {
			structuralPipes++
		}
	}

	if structuralPipes < 2 {
		return 0, false, false
	}

	// Columns is structuralPipes - 1 (for standard | col1 | col2 |)
	// If trailing pipe is omitted (| col1 | col2), cols is structuralPipes
	trailingPipe := strings.HasSuffix(strings.TrimRightFunc(line, unicode.IsSpace), "|")
	if trailingPipe {
		cols = structuralPipes - 1
	} else {
		cols = structuralPipes
	}

	// Check if this row is a delimiter row: cells contain only -, :, and spaces
	isDelim = checkIsDelimiterRow(line)

	return cols, true, isDelim
}

func checkIsDelimiterRow(line string) bool {
	trimmed := strings.Trim(line, "| \t\r")
	if trimmed == "" {
		return false
	}
	cells := strings.Split(trimmed, "|")
	for _, cell := range cells {
		c := strings.TrimSpace(cell)
		if len(c) == 0 {
			return false
		}
		for _, r := range c {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return true
}
