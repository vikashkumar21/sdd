package cli

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// MinStaticPrefixTokens defines the minimum token count required to activate
// provider prefix caching across Anthropic, OpenAI, and Gemini (1,024 tokens).
const MinStaticPrefixTokens = 1024

// MinStaticPrefixBytes defines the deterministic byte floor (4,096 UTF-8 bytes,
// with a 4,200 byte safety threshold) guaranteeing >= 1,024 tokens across standard LLM tokenizers.
const MinStaticPrefixBytes = 4096

// TargetStaticPrefixBytes is the recommended minimum byte size ensuring comfortable margin above 1,024 tokens.
const TargetStaticPrefixBytes = 4200

// CacheBoundaryMarker is the comment marking the exact transition from invariant static prefix to dynamic suffix.
const CacheBoundaryMarker = "<!-- sdd:cache-boundary:end-prefix -->"

// PromptParts represents a partitioned prompt separated at the cache boundary.
type PromptParts struct {
	StaticPrefix  string
	DynamicSuffix string
}

// FullPrompt concatenates the static prefix and dynamic suffix with canonical newline separation.
func (p PromptParts) FullPrompt() string {
	prefix := strings.TrimRight(p.StaticPrefix, "\r\n")
	suffix := strings.TrimLeft(p.DynamicSuffix, "\r\n")
	if prefix == "" {
		return suffix
	}
	if suffix == "" {
		return prefix
	}
	return prefix + "\n\n" + CacheBoundaryMarker + "\n\n" + suffix
}

// EstimateTokens calculates an approximate token count from UTF-8 string content
// using deterministic character heuristics (~3.7 chars per token for markdown/code).
func EstimateTokens(s string) int {
	if s == "" {
		return 0
	}
	runes := utf8.RuneCountInString(s)
	tokens := (runes * 10) / 37
	if tokens == 0 && runes > 0 {
		return 1
	}
	return tokens
}

// ValidateCacheBoundary verifies that a generated prompt possesses an invariant
// prefix meeting MinStaticPrefixBytes and MinStaticPrefixTokens.
func ValidateCacheBoundary(parts PromptParts) error {
	if strings.TrimSpace(parts.StaticPrefix) == "" {
		return errors.New("prompt static prefix is empty")
	}
	byteLen := len(parts.StaticPrefix)
	estimatedTokens := EstimateTokens(parts.StaticPrefix)
	if byteLen < MinStaticPrefixBytes {
		return fmt.Errorf("static prefix length (%d bytes, ~%d estimated tokens) is below the minimum caching floor (%d bytes, %d tokens)",
			byteLen, estimatedTokens, MinStaticPrefixBytes, MinStaticPrefixTokens)
	}
	if estimatedTokens < MinStaticPrefixTokens {
		return fmt.Errorf("static prefix estimated tokens (~%d tokens, %d bytes) is below the minimum token floor (%d tokens, %d bytes)",
			estimatedTokens, byteLen, MinStaticPrefixTokens, MinStaticPrefixBytes)
	}
	return nil
}

// SplitPromptAtCacheBoundary splits a rendered prompt at the CacheBoundaryMarker.
// If the marker is missing, the entire content is returned as StaticPrefix.
func SplitPromptAtCacheBoundary(rendered string) PromptParts {
	idx := strings.Index(rendered, CacheBoundaryMarker)
	if idx == -1 {
		return PromptParts{StaticPrefix: rendered}
	}
	prefix := strings.TrimRight(rendered[:idx], "\r\n")
	suffix := strings.TrimLeft(rendered[idx+len(CacheBoundaryMarker):], "\r\n")
	return PromptParts{
		StaticPrefix:  prefix,
		DynamicSuffix: suffix,
	}
}
