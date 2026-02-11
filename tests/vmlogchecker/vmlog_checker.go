package vmlogchecker

import (
	"regexp"
	"strings"
)

var VirtLauncherErrorAllowlist = []*regexp.Regexp{
	// TODO: Add patterns here
}

// errorKeywordPatterns provides broad keyword-based error detection for the
// CLI tool's --all-levels mode, which scans lines regardless of JSON level.
// The e2e reporter pre-filters on "level":"error" via IsErrorLevel instead.
var errorKeywordPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\berror\b`),
	regexp.MustCompile(`\bfailed\b`),
	regexp.MustCompile(`\bpanic\b`),
	regexp.MustCompile(`\bfatal\b`),
}

type ErrorClassification int

const (
	NotAnError ErrorClassification = iota
	AllowlistedError
	UnexpectedError
)

// IsErrorLevel returns true if the log line contains a JSON "level":"error" field.
// Use this to pre-filter lines before classification when only error-level lines matter.
func IsErrorLevel(line string) bool {
	return strings.Contains(line, `"level":"error"`)
}

func ClassifyLogLine(line string) ErrorClassification {
	if line == "" || !containsErrorKeyword(line) {
		return NotAnError
	}

	if IsAllowlisted(line) {
		return AllowlistedError
	}

	return UnexpectedError
}

func containsErrorKeyword(line string) bool {
	lineLower := strings.ToLower(line)
	for _, pattern := range errorKeywordPatterns {
		if pattern.MatchString(lineLower) {
			return true
		}
	}
	return false
}

func IsAllowlisted(errorLine string) bool {
	for _, pattern := range VirtLauncherErrorAllowlist {
		if pattern.MatchString(errorLine) {
			return true
		}
	}
	return false
}
