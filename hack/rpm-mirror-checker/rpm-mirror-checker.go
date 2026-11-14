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
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// A single url is acceptable when it points at the kubevirt builddeps
// bucket, which is owned by the project and content-addressed by sha256.
var builddepsURLPattern = regexp.MustCompile(`^https://storage\.googleapis\.com/builddeps/`)

type options struct {
	rpmsPath string
	minURLs  int
}

func flagOptions() options {
	o := options{}
	flag.StringVar(&o.rpmsPath, "rpms", "rpms.json", "The path of the rpms.json file to check")
	flag.IntVar(&o.minURLs, "min-urls", 2, "The minimum number of urls every rpm must have, unless it has a single builddeps url")
	flag.Parse()
	return o
}

// rpmsFile is a rpms.json file as generated for the WORKSPACE
type rpmsFile struct {
	Name string    `json:"name"`
	RPMs []rpmRule `json:"rpms"`
}

// rpmRule is a single rpm of a rpms.json file
type rpmRule struct {
	Name      string   `json:"name"`
	Integrity string   `json:"integrity"`
	URLs      []string `json:"urls"`
}

// parseRPMRules parses the rpms of a rpms.json file
func parseRPMRules(rpmsData []byte) ([]rpmRule, error) {
	var rpms rpmsFile
	if err := json.Unmarshal(rpmsData, &rpms); err != nil {
		return nil, fmt.Errorf("failed to parse rpms.json file: %w", err)
	}
	return rpms.RPMs, nil
}

// builddepsURL returns the url under which the kubevirt builddeps bucket
// serves an rpm, or an empty string when its sha256 is unknown or malformed
func builddepsURL(integrity string) string {
	b64, ok := strings.CutPrefix(integrity, "sha256-")
	if !ok {
		return ""
	}
	sum, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(sum) != 32 {
		return ""
	}
	return "https://storage.googleapis.com/builddeps/" + hex.EncodeToString(sum)
}

// urlFailures returns a failure message for every rpm that has fewer than
// minURLs urls and is not a single builddeps url
func urlFailures(rules []rpmRule, minURLs int) []string {
	var failures []string
	for _, rule := range rules {
		if len(rule.URLs) >= minURLs {
			continue
		}
		if len(rule.URLs) == 1 && builddepsURLPattern.MatchString(rule.URLs[0]) {
			continue
		}
		hint := ""
		if url := builddepsURL(rule.Integrity); url != "" {
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

	rpmsData, err := os.ReadFile(opts.rpmsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to read %s: %v\n", opts.rpmsPath, err)
		os.Exit(1)
	}

	rules, err := parseRPMRules(rpmsData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	failures := urlFailures(rules, opts.minURLs)
	for _, failure := range failures {
		fmt.Printf("  %s\n", failure)
	}
	fmt.Printf("Checked %d rpms in %s\n", len(rules), opts.rpmsPath)

	if len(failures) > 0 {
		fmt.Fprintf(os.Stderr, "FAILURE: %d of %d rpms depend on a single non-builddeps url in %s\n", len(failures), len(rules), opts.rpmsPath)
		os.Exit(1)
	}
	fmt.Printf("SUCCESS: all rpms have at least %d urls or a single builddeps url\n", opts.minURLs)
}
