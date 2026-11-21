package check

import (
	"regexp"
	"strings"
)

var (
	macUserHomeRe   = regexp.MustCompile(`(?:^|[^\w./-]|file://)(/Users/([a-zA-Z0-9_.-]+))`)
	linuxUserHomeRe = regexp.MustCompile(`(?:^|[^\w./-]|file://)(/home/([a-zA-Z0-9_.-]+))`)
	winUserHomeRe   = regexp.MustCompile(`(?i)(?:^|[^\w./-]|file:///)([a-z]:[/\\]Users[/\\]([a-zA-Z0-9_.-]+))`)
)

var exemptUsers = map[string]bool{
	"username": true,
	"example":  true,
	"user":     true,
	"yourname": true,
	"runner":   true,
	"admin":    true,
	"default":  true,
	"public":   true, // Windows universal built-in shared profile (C:\Users\Public)
}

func isExemptUser(u string) bool {
	lower := strings.ToLower(u)
	if lower == "..." || strings.HasSuffix(lower, "...") {
		return true
	}
	return exemptUsers[lower]
}

type pathSpan struct {
	start int
	end   int
}

// checkPathHygiene scans markdown content line-by-line for local developer home directory paths per Rule F-002.
func (c *checker) checkPathHygiene(rel, text string) {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lineNum := i + 1

		// Skip lines discussing Rule F-002, path hygiene, or path-hygiene test references
		lower := strings.ToLower(line)
		if strings.Contains(lower, "f-002") ||
			strings.Contains(lower, "path-hygiene") ||
			strings.Contains(lower, "pathhygiene") ||
			strings.Contains(lower, "path hygiene") ||
			strings.Contains(lower, "path_hygiene") {
			continue
		}

		seen := make(map[string]bool)
		var matchedSpans []pathSpan

		checkRegex := func(re *regexp.Regexp) {
			matches := re.FindAllStringSubmatchIndex(line, -1)
			for _, loc := range matches {
				if len(loc) < 6 {
					continue
				}

				mStart, mEnd := loc[2], loc[3]
				overlaps := false
				for _, s := range matchedSpans {
					if mStart < s.end && mEnd > s.start {
						overlaps = true
						break
					}
				}
				if overlaps {
					continue
				}

				rawUser := line[loc[4]:loc[5]]
				user := rawUser
				if user != "..." && strings.HasSuffix(user, ".") {
					user = strings.TrimRight(user, ".")
				}
				if isExemptUser(user) {
					continue
				}

				matchedPath := line[mStart:mEnd]
				if user != rawUser {
					matchedPath = strings.TrimSuffix(matchedPath, rawUser) + user
				}

				matchedSpans = append(matchedSpans, pathSpan{start: mStart, end: mEnd})

				if seen[matchedPath] {
					continue
				}
				seen[matchedPath] = true

				c.add(Warn, rel, lineNum, "path-hygiene", "local developer home path detected (%s); use repository-relative paths or generic clone URLs per Rule F-002", matchedPath)
			}
		}

		// Check Windows first so drive-letter paths like D:/Users/... claim their span before macUserHomeRe
		checkRegex(winUserHomeRe)
		checkRegex(macUserHomeRe)
		checkRegex(linuxUserHomeRe)
	}
}
