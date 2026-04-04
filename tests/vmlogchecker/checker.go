package vmlogchecker

import (
	"regexp"
	"strings"
)

// SIGMask is a bitmask identifying which SIG areas are affected by an allowlist entry.
// Entries may affect multiple SIGs; combine with |, e.g. SIGNetwork | SIGStorage.
type SIGMask uint8

const (
	SIGCompute     SIGMask = 1 << iota // 0x01
	SIGNetwork                         // 0x02
	SIGOperator                        // 0x04
	SIGPerformance                     // 0x08
	SIGStorage                         // 0x10
)

// AllowlistEntry describes a known/expected error pattern in virt-launcher logs.
// When adding a new entry, always use the last entry's ID + 1.
// Never reuse an ID after deletion.
type AllowlistEntry struct {
	// ID is a stable unique identifier. Set to last entry's ID + 1 on insert.
	ID int
	// Regex is matched against the full log line.
	Regex *regexp.Regexp
	// SIGs is the bitmask of affected SIG areas for triage routing.
	SIGs SIGMask
}

// VirtLauncherErrorAllowlist lists known error patterns that are expected and
// should not fail tests. Add new entries at the end with ID = last ID + 1.
var VirtLauncherErrorAllowlist = []AllowlistEntry{
	// Example:
	// {
	// 	ID:               1,
	// 	Regex:            regexp.MustCompile(`"level":"error","msg":"End of file while reading data: Input/output error","pos":"virNetSocketReadWire`),
	// 	SIGs:             SIGCompute | SIGNetwork,
	// },
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

	if MatchAllowlist(line) != nil {
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

// MatchAllowlist returns the first AllowlistEntry whose Regex matches the given
// line, or nil if the line is not allowlisted.
func MatchAllowlist(errorLine string) *AllowlistEntry {
	for i := range VirtLauncherErrorAllowlist {
		if VirtLauncherErrorAllowlist[i].Regex.MatchString(errorLine) {
			return &VirtLauncherErrorAllowlist[i]
		}
	}
	return nil
}
