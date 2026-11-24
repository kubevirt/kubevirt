/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// A single url is acceptable when it points at the kubevirt builddeps
// bucket, which is owned by the project and content-addressed by sha256.
const builddepsURLPattern = "https://storage.googleapis.com/builddeps/"

// The WORKSPACE file is machine-generated, so rpm() rules start at
// column 0 and their attributes are plain string assignments and
// string lists.
var (
	rpmRuleStart  = regexp.MustCompile(`(?m)^rpm\(`)
	namePattern   = regexp.MustCompile(`\bname\s*=\s*"([^"]*)"`)
	sha256Pattern = regexp.MustCompile(`\bsha256\s*=\s*"([^"]*)"`)
	urlPattern    = regexp.MustCompile(`\burl\s*=\s*"([^"]*)"`)
	urlsPattern   = regexp.MustCompile(`(?s)\burls\s*=\s*\[(.*?)\]`)
	stringPattern = regexp.MustCompile(`"([^"]*)"`)
)

type options struct {
	workspacePath string
	minURLs       int
}

func flagOptions() options {
	o := options{}
	flag.StringVar(&o.workspacePath, "workspace", "WORKSPACE", "The path of the WORKSPACE file to check")
	flag.IntVar(&o.minURLs, "min-urls", 2, "The minimum number of urls every rpm rule must have, unless it has a single builddeps url")
	flag.Parse()
	return o
}

// rpmRule is an rpm() rule of a WORKSPACE file
type rpmRule struct {
	Name   string
	SHA256 string
	URLs   []string
}

// parseRPMRules parses the rpm() rules of a WORKSPACE file
func parseRPMRules(workspaceData []byte) ([]rpmRule, error) {
	workspace := string(workspaceData)
	var rules []rpmRule
	for _, loc := range rpmRuleStart.FindAllStringIndex(workspace, -1) {
		end, ok := matchParen(workspace, loc[1]-1)
		if !ok {
			return nil, fmt.Errorf("unterminated rpm() rule at offset %d", loc[0])
		}
		body := workspace[loc[1]:end]
		rules = append(rules, rpmRule{
			Name:   attrString(body, namePattern),
			SHA256: attrString(body, sha256Pattern),
			URLs:   attrURLs(body),
		})
	}
	return rules, nil
}

// matchParen returns the index of the ')' matching the '(' at open,
// skipping over quoted strings and comments, or ok=false when the rule
// is not terminated
func matchParen(s string, open int) (int, bool) {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '"':
			end := skipString(s, i)
			if end < 0 {
				return 0, false
			}
			i = end
		case '#':
			nl := strings.IndexByte(s[i:], '\n')
			if nl < 0 {
				return 0, false
			}
			i += nl - 1
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

// skipString returns the index of the closing quote of the string that
// starts at s[open], or -1 when it is not terminated
func skipString(s string, open int) int {
	for i := open + 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '"':
			return i
		}
	}
	return -1
}

// attrString returns the value of the string attribute matching the
// pattern, or an empty string when it is absent
func attrString(body string, pattern *regexp.Regexp) string {
	if m := pattern.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	return ""
}

// attrURLs returns the urls of a rule, from its urls list or its
// single url attribute
func attrURLs(body string) []string {
	if m := urlPattern.FindStringSubmatch(body); m != nil {
		return []string{m[1]}
	}
	if m := urlsPattern.FindStringSubmatch(body); m != nil {
		var urls []string
		for _, s := range stringPattern.FindAllStringSubmatch(m[1], -1) {
			urls = append(urls, s[1])
		}
		return urls
	}
	return nil
}

// builddepsURL returns the url under which the kubevirt builddeps bucket
// serves an rpm, or an empty string when its sha256 is unknown
func builddepsURL(sha256 string) string {
	if sha256 == "" {
		return ""
	}
	return builddepsURLPattern + sha256
}

// urlFailures returns a failure message for every rpm rule that has fewer
// than minURLs urls and is not a single builddeps url
func urlFailures(rules []rpmRule, minURLs int) []string {
	var failures []string
	for _, rule := range rules {
		if len(rule.URLs) >= minURLs {
			continue
		}
		if len(rule.URLs) == 1 && strings.HasPrefix(rule.URLs[0], builddepsURLPattern) {
			continue
		}
		hint := ""
		if url := builddepsURL(rule.SHA256); url != "" {
			hint = fmt.Sprintf(" (add %s)", url)
		}
		switch len(rule.URLs) {
		case 0:
			failures = append(failures, fmt.Sprintf("rpm %s: no urls%s", rule.Name, hint))
		case 1:
			failures = append(failures, fmt.Sprintf("rpm %s: only 1 url, not a builddeps url: %s%s", rule.Name, rule.URLs[0], hint))
		default:
			failures = append(failures, fmt.Sprintf("rpm %s: only %d urls: %v", rule.Name, len(rule.URLs), rule.URLs))
		}
	}
	return failures
}

func main() {
	opts := flagOptions()

	workspaceData, err := os.ReadFile(opts.workspacePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to read %s: %v\n", opts.workspacePath, err)
		os.Exit(1)
	}

	rules, err := parseRPMRules(workspaceData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	failures := urlFailures(rules, opts.minURLs)
	for _, failure := range failures {
		fmt.Printf("  %s\n", failure)
	}
	fmt.Printf("Checked %d rpm rules in %s\n", len(rules), opts.workspacePath)

	if len(failures) > 0 {
		fmt.Fprintf(os.Stderr, "FAILURE: %d of %d rpm rules depend on a single non-builddeps url in %s\n", len(failures), len(rules), opts.workspacePath)
		os.Exit(1)
	}
	fmt.Printf("SUCCESS: all rpm rules have at least %d urls or a single builddeps url\n", opts.minURLs)
}
