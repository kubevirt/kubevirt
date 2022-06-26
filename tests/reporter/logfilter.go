package reporter

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// filterBySinceTimestamp filters lines by matching timestamp where each line timestamp is after 'since' timestamp.
// Each line timestamp is parsed by the 'parseFn'.
func filterBySinceTimestamp(content string, timestampRegex *regexp.Regexp, since time.Time, parseFn func(string) (time.Time, error)) string {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "\nfilterBySinceTimestamp: panic: %v", r)
		}
	}()

	filtered := strings.Builder{}
	scanner := bufio.NewScanner(bytes.NewBufferString(content))
	for scanner.Scan() {
		line := scanner.Text()
		match := timestampRegex.FindString(line)
		if match == "" {
			continue
		}

		lineTimestamp, err := parseFn(match)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nfailed to parse line timestamp: %v\n", err)
			continue
		}

		if lineTimestamp.UTC().After(since.UTC()) {
			filtered.WriteString(line)
			filtered.WriteString("\n")
		}
	}

	return filtered.String()
}
