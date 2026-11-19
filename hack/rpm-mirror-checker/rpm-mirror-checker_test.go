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
	"slices"
	"testing"
)

// sha256 values as the WORKSPACE generator writes them: plain hex
const (
	shaHealthy         = "dd11bab2ea0abdfa310362eace871422a003340bf223135626500f8f5a985f6b"
	shaSingleBuilddeps = "6d65bb2098442639ae5199ec4c8ee342b965dbc31f73277df63f04b4273f0278"
	shaSingleMirror    = "153742d20ad378247d8256619d2fca5666565befd8dbf0743966da331c993540"
	shaSingleURLAttr   = "042bf32e5b511a8b2a346f34e79876e0ee252fea03a6c73bd16d23e5785b298c"
	shaNoURLs          = "c7a09c660f998150305b32bdc619820ec1c7b4a8be153901cb1e32c3ba0e44e9"
)

const (
	healthyMirrorURL    = "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/acl-2.3.1-4.el9.x86_64.rpm"
	singleMirrorURL     = "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/acl-2.4.0-2.el10.x86_64.rpm"
	singleURLAttrURL    = "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/attr-2.5.1-3.el9.x86_64.rpm"
	healthyBuilddepsURL = builddepsURLPattern + shaHealthy
	singleBuilddepsURL  = builddepsURLPattern + shaSingleBuilddeps
)

const testWorkspace = `
rpm(
    name = "acl-0__2.3.1-4.el9.x86_64",
    sha256 = "` + shaHealthy + `",
    # a ) that must not end the rule
    urls = [
        "` + healthyMirrorURL + `",
        "` + healthyBuilddepsURL + `",
    ],
)

rpm(
    name = "acl-0__2.4.0-2.el10.x86_64",
    sha256 = "` + shaSingleBuilddeps + `",
    urls = ["` + singleBuilddepsURL + `"],
)

rpm(
    name = "acl-0__2.4.0-2.el10.aarch64",
    sha256 = "` + shaSingleBuilddeps + `",
    url = "` + singleBuilddepsURL + `",
)

rpm(
    name = "attr-0__2.5.1-3.el9.x86_64",
    sha256 = "` + shaSingleMirror + `",
    urls = [
        "` + singleMirrorURL + `",
    ],
)

rpm(
    name = "attr-0__2.5.1-3.el9.aarch64",
    sha256 = "` + shaSingleURLAttr + `",
    url = "` + singleURLAttrURL + `",
)

rpm(
    name = "bzip2-0__1.0.8-10.el9.x86_64",
    sha256 = "` + shaNoURLs + `",
)

rpm(
    name = "bzip2-0__1.0.8-10.el9.aarch64",
)

http_archive(
    name = "not-an-rpm",
    urls = ["https://example.com/single-url"],
)
`

func testRules(t *testing.T) []rpmRule {
	t.Helper()
	rules, err := parseRPMRules([]byte(testWorkspace))
	if err != nil {
		t.Fatalf("failed to parse test WORKSPACE: %v", err)
	}
	return rules
}

func TestParseRPMRules(t *testing.T) {
	rules := testRules(t)

	if len(rules) != 7 {
		t.Fatalf("expected 7 rpm rules, got %d: %v", len(rules), rules)
	}
	if rules[0].Name != "acl-0__2.3.1-4.el9.x86_64" || rules[0].SHA256 != shaHealthy {
		t.Errorf("unexpected first rule: %v", rules[0])
	}
	if !slices.Equal(rules[0].URLs, []string{healthyMirrorURL, healthyBuilddepsURL}) {
		t.Errorf("expected both urls of the first rule to be captured, got %v", rules[0].URLs)
	}
	if !slices.Equal(rules[1].URLs, []string{singleBuilddepsURL}) {
		t.Errorf("expected the single url of the second rule to be captured, got %v", rules[1].URLs)
	}
	if !slices.Equal(rules[2].URLs, []string{singleBuilddepsURL}) {
		t.Errorf("expected the url attribute of the third rule to be captured, got %v", rules[2].URLs)
	}
	if rules[6].Name != "bzip2-0__1.0.8-10.el9.aarch64" || len(rules[6].URLs) != 0 || rules[6].SHA256 != "" {
		t.Errorf("unexpected last rule: %v", rules[6])
	}
}

func TestParseRPMRulesUnterminated(t *testing.T) {
	if _, err := parseRPMRules([]byte("rpm(\n    name = \"x\",\n")); err == nil {
		t.Error("expected an error for an unterminated rpm() rule")
	}
}

func TestBuilddepsURL(t *testing.T) {
	if url := builddepsURL(""); url != "" {
		t.Errorf("expected no url for an empty sha256, got %q", url)
	}
	if url := builddepsURL(shaHealthy); url != healthyBuilddepsURL {
		t.Errorf("expected %q, got %q", healthyBuilddepsURL, url)
	}
}

func TestURLFailures(t *testing.T) {
	expectedFailures := []string{
		"rpm attr-0__2.5.1-3.el9.x86_64: only 1 url, not a builddeps url: " + singleMirrorURL + " (add " + builddepsURLPattern + shaSingleMirror + ")",
		"rpm attr-0__2.5.1-3.el9.aarch64: only 1 url, not a builddeps url: " + singleURLAttrURL + " (add " + builddepsURLPattern + shaSingleURLAttr + ")",
		"rpm bzip2-0__1.0.8-10.el9.x86_64: no urls (add " + builddepsURLPattern + shaNoURLs + ")",
		"rpm bzip2-0__1.0.8-10.el9.aarch64: no urls",
	}
	failures := urlFailures(testRules(t), 2)
	if !slices.Equal(failures, expectedFailures) {
		t.Errorf("expected failures %v, got %v", expectedFailures, failures)
	}
}

func TestURLFailuresMinURLs(t *testing.T) {
	// a single builddeps url stays acceptable, whatever the minimum is
	expectedFailures := []string{
		"rpm acl-0__2.3.1-4.el9.x86_64: only 2 urls: [" + healthyMirrorURL + " " + healthyBuilddepsURL + "]",
		"rpm attr-0__2.5.1-3.el9.x86_64: only 1 url, not a builddeps url: " + singleMirrorURL + " (add " + builddepsURLPattern + shaSingleMirror + ")",
		"rpm attr-0__2.5.1-3.el9.aarch64: only 1 url, not a builddeps url: " + singleURLAttrURL + " (add " + builddepsURLPattern + shaSingleURLAttr + ")",
		"rpm bzip2-0__1.0.8-10.el9.x86_64: no urls (add " + builddepsURLPattern + shaNoURLs + ")",
		"rpm bzip2-0__1.0.8-10.el9.aarch64: no urls",
	}
	failures := urlFailures(testRules(t), 3)
	if !slices.Equal(failures, expectedFailures) {
		t.Errorf("expected failures %v, got %v", expectedFailures, failures)
	}
}
