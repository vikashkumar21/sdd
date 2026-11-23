// Package markdown provides lightweight, deterministic scanners for Markdown documents.
package markdown

import (
	"fmt"
	"strings"
	"unicode"
)

// MermaidFinding describes a syntactic or structural error in a Mermaid code block.
type MermaidFinding struct {
	Line    int
	Rule    string
	Message string
}

// knownDiagramTypes lists standard Mermaid diagram declarations.
var knownDiagramTypes = []string{
	"flowchart",
	"graph",
	"sequencediagram",
	"erdiagram",
	"statediagram-v2",
	"statediagram",
	"classdiagram",
	"classdiagram-v2",
	"gantt",
	"pie",
	"gitgraph",
	"c4context",
	"c4container",
	"c4component",
	"c4dynamic",
	"c4deployment",
	"journey",
	"quadrantchart",
	"xychart-beta",
	"block-beta",
	"mindmap",
	"timeline",
	"sankey-beta",
	"requirementdiagram",
	"packet-beta",
	"kanban",
	"architecture-beta",
}

// CheckMermaid scans markdown text for fenced mermaid code blocks and validates their syntax.
func CheckMermaid(text string) []MermaidFinding {
	var findings []MermaidFinding
	lines := strings.Split(text, "\n")

	var inMermaid bool
	var openLine int
	var blockLines []string
	var blockLineNums []int

	for i, rawLine := range lines {
		lineNo := i + 1
		trimmed := strings.TrimLeftFunc(rawLine, unicode.IsSpace)
		char, count := parseFenceMarker(trimmed)

		if !inMermaid {
			if count >= 3 && char == '`' || count >= 3 && char == '~' {
				info := strings.TrimSpace(trimmed[count:])
				parts := strings.Fields(info)
				if len(parts) > 0 && strings.EqualFold(parts[0], "mermaid") {
					inMermaid = true
					openLine = lineNo
					blockLines = nil
					blockLineNums = nil
				}
			}
		} else {
			// Inside mermaid block
			if count >= 3 && (char == '`' || char == '~') {
				rest := strings.TrimSpace(trimmed[count:])
				if rest == "" {
					// End of mermaid block: validate accumulated block
					findings = append(findings, validateMermaidBlock(openLine, blockLines, blockLineNums)...)
					inMermaid = false
					blockLines = nil
					blockLineNums = nil
					continue
				}
			}
			blockLines = append(blockLines, rawLine)
			blockLineNums = append(blockLineNums, lineNo)
		}
	}

	return findings
}

func validateMermaidBlock(openLine int, lines []string, lineNums []int) []MermaidFinding {
	var findings []MermaidFinding

	// 1. Check for non-empty content and valid header
	var headerLine string
	var headerLineNo int
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			continue
		}
		headerLine = trimmed
		headerLineNo = lineNums[idx]
		break
	}

	if headerLine == "" {
		findings = append(findings, MermaidFinding{
			Line:    openLine,
			Rule:    "mermaid-syntax",
			Message: "empty Mermaid code block",
		})
		return findings
	}

	// Validate diagram header declaration
	fields := strings.Fields(headerLine)
	diagType := strings.ToLower(fields[0])
	validType := false
	for _, kt := range knownDiagramTypes {
		if diagType == kt {
			validType = true
			break
		}
	}

	if !validType {
		findings = append(findings, MermaidFinding{
			Line:    headerLineNo,
			Rule:    "mermaid-syntax",
			Message: fmt.Sprintf("unrecognized or missing Mermaid diagram type %q (expected flowchart, sequenceDiagram, erDiagram, etc.)", fields[0]),
		})
	}

	isER := diagType == "erdiagram"
	isSequence := diagType == "sequencediagram"
	isFlowchart := diagType == "flowchart" || diagType == "graph"

	// 2. Bracket balancing
	type bracketOpen struct {
		char rune
		line int
	}
	var stack []bracketOpen

	for idx, line := range lines {
		lineNo := lineNums[idx]
		runes := []rune(line)
		inString := false
		var stringDelim rune
		inEdgeLabel := false

		// Check if line is a comment
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "%%") {
			continue
		}

		// In sequenceDiagram, skip message text after colon
		if isSequence {
			colonIdx := -1
			for ci, r := range runes {
				if r == ':' && !inString {
					colonIdx = ci
					break
				}
			}
			if colonIdx != -1 {
				runes = runes[:colonIdx]
			}
		}

		// In erDiagram, check if line is a relationship (contains -- or ..)
		isERRelationship := isER && (strings.Contains(line, "--") || strings.Contains(line, ".."))

		for i := 0; i < len(runes); i++ {
			r := runes[i]

			// Handle comments %% to end of line
			if !inString && r == '%' && i+1 < len(runes) && runes[i+1] == '%' {
				break
			}

			// Handle string literals ("..." or '...')
			if inString {
				if r == '\\' && i+1 < len(runes) {
					i++ // Skip escaped char
					continue
				}
				if r == stringDelim {
					inString = false
				}
				continue
			}

			if r == '"' || r == '\'' {
				inString = true
				stringDelim = r
				continue
			}

			// Handle flowchart edge labels |...|
			if isFlowchart && r == '|' {
				inEdgeLabel = !inEdgeLabel
				continue
			}
			if inEdgeLabel {
				continue
			}

			// Skip ER diagram crow's foot relationship connectors like o{, |{, }|, }o
			if isERRelationship && (r == '{' || r == '}') {
				continue
			}

			// Bracket matching
			switch r {
			case '(', '[', '{':
				stack = append(stack, bracketOpen{char: r, line: lineNo})
			case ')':
				if len(stack) == 0 || stack[len(stack)-1].char != '(' {
					findings = append(findings, MermaidFinding{
						Line:    lineNo,
						Rule:    "mermaid-syntax",
						Message: "unexpected closing parenthesis ')' without matching '('",
					})
				} else {
					stack = stack[:len(stack)-1]
				}
			case ']':
				if len(stack) == 0 || stack[len(stack)-1].char != '[' {
					findings = append(findings, MermaidFinding{
						Line:    lineNo,
						Rule:    "mermaid-syntax",
						Message: "unexpected closing bracket ']' without matching '['",
					})
				} else {
					stack = stack[:len(stack)-1]
				}
			case '}':
				if len(stack) == 0 || stack[len(stack)-1].char != '{' {
					findings = append(findings, MermaidFinding{
						Line:    lineNo,
						Rule:    "mermaid-syntax",
						Message: "unexpected closing brace '}' without matching '{'",
					})
				} else {
					stack = stack[:len(stack)-1]
				}
			}
		}
	}

	for _, unclosed := range stack {
		findings = append(findings, MermaidFinding{
			Line:    unclosed.line,
			Rule:    "mermaid-syntax",
			Message: fmt.Sprintf("unclosed delimiter '%c'", unclosed.char),
		})
	}

	return findings
}
