// Package doc validates and stages PDF and document specifications
// for multimodal AI coding agents.
package doc

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vikashkumar21/sdd/internal/context"
)

// ValidatePDF verifies that data begins with the PDF magic header `%PDF-`,
// does not exceed context.MaxInputBytes, and extracts the PDF version.
func ValidatePDF(data []byte) (version string, err error) {
	if len(data) == 0 {
		return "", errors.New("doc: PDF data is empty")
	}
	if len(data) > context.MaxInputBytes {
		return "", fmt.Errorf("doc: PDF size %d bytes exceeds maximum allowed (%d bytes)", len(data), context.MaxInputBytes)
	}

	headerLimit := 1024
	if len(data) < headerLimit {
		headerLimit = len(data)
	}
	idx := bytes.Index(data[:headerLimit], []byte("%PDF-"))
	if idx == -1 {
		return "", errors.New("doc: missing %PDF- header signature")
	}

	rest := data[idx+5:]
	var verBuilder strings.Builder
	for _, b := range rest {
		if (b >= '0' && b <= '9') || b == '.' {
			verBuilder.WriteByte(b)
		} else {
			break
		}
	}
	ver := verBuilder.String()
	if ver == "" {
		ver = "unknown"
	}

	return ver, nil
}

// IsPDF checks whether filename has a .pdf extension or begins with %PDF-.
func IsPDF(filename string, data []byte) bool {
	if strings.EqualFold(filepath.Ext(filename), ".pdf") {
		return true
	}
	if len(data) >= 5 && bytes.HasPrefix(data, []byte("%PDF-")) {
		return true
	}
	return false
}

// InjectDocReference adds a reference to the staged document into requirements.md
// under "## 2. Context and references".
func InjectDocReference(requirementsMarkdown, stagedPath, origin, format string) string {
	refLine := fmt.Sprintf("- Reference document: `%s` (Format: %s, Source: %s)", stagedPath, format, origin)
	contextHeader := "## 2. Context and references\n"
	if strings.Contains(requirementsMarkdown, contextHeader) {
		parts := strings.SplitN(requirementsMarkdown, contextHeader, 2)
		return parts[0] + contextHeader + refLine + "\n" + parts[1]
	}
	return requirementsMarkdown + "\n" + refLine + "\n"
}
