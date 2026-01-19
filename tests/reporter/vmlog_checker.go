package reporter

import (
	"fmt"
	"regexp"
	"strings"
)

var VirtLauncherErrorAllowlist = []*regexp.Regexp{
	// TODO: Add patterns here
}

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

func ClassifyLogLine(line string) ErrorClassification {
	if line == "" {
		return NotAnError
	}

	lineLower := strings.ToLower(line)
	hasErrorKeyword := false
	for _, pattern := range errorKeywordPatterns {
		if pattern.MatchString(lineLower) {
			hasErrorKeyword = true
			break
		}
	}

	if !hasErrorKeyword {
		return NotAnError
	}

	if IsAllowlisted(line) {
		return AllowlistedError
	}

	return UnexpectedError
}

func IsAllowlisted(errorLine string) bool {
	for _, pattern := range VirtLauncherErrorAllowlist {
		if pattern.MatchString(errorLine) {
			return true
		}
	}
	return false
}

func AnalyzeVMLogs(logs string) (totalLines, allowlistedCount, unexpectedCount int) {
	for _, line := range strings.Split(logs, "\n") {
		if line == "" {
			continue
		}
		totalLines++

		classification := ClassifyLogLine(line)
		switch classification {
		case AllowlistedError:
			allowlistedCount++
		case UnexpectedError:
			unexpectedCount++
		}
	}
	return
}

func FormatErrorLine(vmiName string, line string) string {
	if vmiName != "" {
		return fmt.Sprintf("[%s] %s", vmiName, line)
	}
	return line
}
