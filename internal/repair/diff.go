package repair

import (
	"fmt"
	"strings"
)

// Diff generates a standard unified diff between original and modified text.
func Diff(original, modified, path string) string {
	if original == modified {
		return ""
	}
	origLines := splitLines(original)
	modLines := splitLines(modified)

	n, m := len(origLines), len(modLines)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			if origLines[i] == modLines[j] {
				dp[i+1][j+1] = dp[i][j] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i+1][j+1] = dp[i+1][j]
			} else {
				dp[i+1][j+1] = dp[i][j+1]
			}
		}
	}

	type diffLine struct {
		op   byte // ' ', '-', '+'
		text string
	}
	var diffs []diffLine
	i, j := n, m
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && origLines[i-1] == modLines[j-1] {
			diffs = append(diffs, diffLine{' ', origLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			diffs = append(diffs, diffLine{'+', modLines[j-1]})
			j--
		} else if i > 0 && (j == 0 || dp[i][j-1] < dp[i-1][j]) {
			diffs = append(diffs, diffLine{'-', origLines[i-1]})
			i--
		}
	}
	// Reverse to forward order
	for k, l := 0, len(diffs)-1; k < l; k, l = k+1, l-1 {
		diffs[k], diffs[l] = diffs[l], diffs[k]
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- a/%s\n", path)
	fmt.Fprintf(&sb, "+++ b/%s\n", path)
	fmt.Fprintf(&sb, "@@ -1,%d +1,%d @@\n", n, m)
	for _, dl := range diffs {
		sb.WriteByte(dl.op)
		sb.WriteString(dl.text)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
