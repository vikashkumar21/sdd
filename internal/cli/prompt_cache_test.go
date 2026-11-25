package cli

import (
	"strings"
	"testing"
)

func TestPromptCache_EstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		minToken int
		maxToken int
	}{
		{
			name:     "empty string",
			input:    "",
			minToken: 0,
			maxToken: 0,
		},
		{
			name:     "short string",
			input:    "Hello world",
			minToken: 1,
			maxToken: 5,
		},
		{
			name:     "4096 bytes of text",
			input:    strings.Repeat("abcdefghij", 410), // 4100 chars
			minToken: 1024,
			maxToken: 1200,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EstimateTokens(tc.input)
			if got < tc.minToken || got > tc.maxToken {
				t.Errorf("EstimateTokens() = %d, want between %d and %d", got, tc.minToken, tc.maxToken)
			}
		})
	}
}

func TestPromptCache_PromptParts_FullPrompt(t *testing.T) {
	prefix := "# Invariant Static Prefix\nRole and rubric content."
	suffix := "## Dynamic Suffix\nTask details and changed files."

	parts := PromptParts{
		StaticPrefix:  prefix,
		DynamicSuffix: suffix,
	}

	full := parts.FullPrompt()
	if !strings.Contains(full, prefix) {
		t.Errorf("FullPrompt missing prefix")
	}
	if !strings.Contains(full, suffix) {
		t.Errorf("FullPrompt missing suffix")
	}
	if !strings.Contains(full, CacheBoundaryMarker) {
		t.Errorf("FullPrompt missing CacheBoundaryMarker")
	}

	split := SplitPromptAtCacheBoundary(full)
	if split.StaticPrefix != prefix {
		t.Errorf("SplitPromptAtCacheBoundary static prefix mismatch:\ngot: %q\nwant: %q", split.StaticPrefix, prefix)
	}
	if split.DynamicSuffix != suffix {
		t.Errorf("SplitPromptAtCacheBoundary dynamic suffix mismatch:\ngot: %q\nwant: %q", split.DynamicSuffix, suffix)
	}

	// Edge case 1: empty StaticPrefix
	onlySuffix := PromptParts{DynamicSuffix: "suffix only"}.FullPrompt()
	if onlySuffix != "suffix only" {
		t.Errorf("expected suffix only, got %q", onlySuffix)
	}

	// Edge case 2: empty DynamicSuffix
	onlyPrefix := PromptParts{StaticPrefix: "prefix only"}.FullPrompt()
	if onlyPrefix != "prefix only" {
		t.Errorf("expected prefix only, got %q", onlyPrefix)
	}

	// Edge case 3: both empty
	bothEmpty := PromptParts{}.FullPrompt()
	if bothEmpty != "" {
		t.Errorf("expected empty string, got %q", bothEmpty)
	}

	// Edge case 4: missing boundary marker in SplitPromptAtCacheBoundary
	noMarker := SplitPromptAtCacheBoundary("raw text with no marker")
	if noMarker.StaticPrefix != "raw text with no marker" || noMarker.DynamicSuffix != "" {
		t.Errorf("unexpected split result when marker missing: %+v", noMarker)
	}
}

func TestPromptCache_ValidateCacheBoundary(t *testing.T) {
	// 1. Empty prefix
	if err := ValidateCacheBoundary(PromptParts{}); err == nil {
		t.Errorf("expected error on empty prefix, got nil")
	}

	// 2. Below threshold (100 bytes)
	shortParts := PromptParts{
		StaticPrefix:  strings.Repeat("a", 100),
		DynamicSuffix: "task info",
	}
	if err := ValidateCacheBoundary(shortParts); err == nil {
		t.Errorf("expected error on prefix < 4096 bytes, got nil")
	}

	// 3. Exact boundary: 4095 bytes (fails)
	boundaryFail := PromptParts{
		StaticPrefix:  strings.Repeat("a", 4095),
		DynamicSuffix: "task info",
	}
	if err := ValidateCacheBoundary(boundaryFail); err == nil {
		t.Errorf("expected error on 4095 bytes (< 4096), got nil")
	}

	// 4. Exact boundary: 4096 bytes (passes)
	boundaryPass := PromptParts{
		StaticPrefix:  strings.Repeat("a", 4096),
		DynamicSuffix: "task info",
	}
	if err := ValidateCacheBoundary(boundaryPass); err != nil {
		t.Errorf("expected clean validation on 4096 bytes, got %v", err)
	}

	// 5. Meeting safety target threshold (TargetStaticPrefixBytes = 4200)
	targetParts := PromptParts{
		StaticPrefix:  strings.Repeat("a", TargetStaticPrefixBytes),
		DynamicSuffix: "task info",
	}
	if err := ValidateCacheBoundary(targetParts); err != nil {
		t.Errorf("expected clean validation on TargetStaticPrefixBytes (4200), got %v", err)
	}

	// 6. Multi-byte token floor divergence (high byte count, low token count)
	// 300 4-byte runes = 1200 bytes (> 1000) but only ~81 estimated tokens (< 1024)
	multiByteParts := PromptParts{
		StaticPrefix:  strings.Repeat("🌍", 1050), // 4200 bytes, but only 1050 runes -> ~283 tokens
		DynamicSuffix: "task info",
	}
	if err := ValidateCacheBoundary(multiByteParts); err == nil {
		t.Errorf("expected error when estimated tokens < 1024 even if byteLen >= 4096, got nil")
	}
}
