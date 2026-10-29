// Package block locates the managed contract block that sdd writes into agent
// files (CLAUDE.md, GEMINI.md, AGENTS.md). Text outside the markers belongs to
// the human and is never touched.
package block

import "strings"

const (
	beginPrefix = "<!-- sdd:begin contract"
	// End is the marker that closes a managed block.
	End = "<!-- sdd:end contract -->"
)

// Begin returns the opening marker for the given template version.
func Begin(templatesVersion string) string {
	return beginPrefix + " templates=" + templatesVersion + " -->"
}

// Locate returns the byte offsets of a managed block: blockStart is the start
// of the opening marker, innerStart/innerEnd delimit the text between the
// markers, and blockEnd is the offset just after the closing marker. ok is
// false when no well-formed block exists.
func Locate(content string) (blockStart, innerStart, innerEnd, blockEnd int, ok bool) {
	b := strings.Index(content, beginPrefix)
	if b < 0 {
		return 0, 0, 0, 0, false
	}
	closeRel := strings.Index(content[b:], "-->")
	if closeRel < 0 {
		return 0, 0, 0, 0, false
	}
	innerStart = b + closeRel + len("-->")
	endRel := strings.Index(content[innerStart:], End)
	if endRel < 0 {
		return 0, 0, 0, 0, false
	}
	innerEnd = innerStart + endRel
	return b, innerStart, innerEnd, innerEnd + len(End), true
}

// Inner returns the text between the markers with surrounding blank lines
// removed, or false when the content has no managed block.
func Inner(content string) (string, bool) {
	_, is, ie, _, ok := Locate(content)
	if !ok {
		return "", false
	}
	return strings.Trim(content[is:ie], "\r\n"), true
}

// Has reports whether the content contains a well-formed managed block.
func Has(content string) bool {
	_, _, _, _, ok := Locate(content)
	return ok
}
