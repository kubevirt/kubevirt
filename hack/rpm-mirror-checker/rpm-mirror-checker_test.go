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

const testRpmsJSON = `{
	"name": "kubevirt-rpms-test",
	"rpms": [
		{
			"name": "healthy",
			"integrity": "sha256-3RG6suoKvfoxA2LqzocUIqADNAvyIxNWJlAPj1qYX2s=",
			"urls": [
				"http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/healthy.rpm",
				"https://storage.googleapis.com/builddeps/dd11bab2ea0abdfa310362eace871422a003340bf223135626500f8f5a985f6b"
			]
		},
		{
			"name": "single-builddeps",
			"integrity": "sha256-bWW7IJhEJjmuUZnsTI7jQrll28Mfcyd99j8EtCc/Ang=",
			"urls": ["https://storage.googleapis.com/builddeps/6d65bb2098442639ae5199ec4c8ee342b965dbc31f73277df63f04b4273f0278"]
		},
		{
			"name": "single-mirror",
			"integrity": "sha256-FTdC0grTeCR9glZhnS/KVmZWW+/Y2/B0OWbaMxyZNUA=",
			"urls": [
				"http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/single-mirror.rpm"
			]
		},
		{
			"name": "no-urls",
			"integrity": "sha256-nmqgxgIEu0sVLOVByjqfXCiwIO1VHdQX05NqiyFT8N8="
		}
	]
}`

func TestParseRPMRules(t *testing.T) {
	rules, err := parseRPMRules([]byte(testRpmsJSON))
	if err != nil {
		t.Fatalf("failed to parse test rpms.json: %v", err)
	}

	if len(rules) != 4 {
		t.Fatalf("expected 4 rpms, got %d: %v", len(rules), rules)
	}
	if rules[0].Name != "healthy" || len(rules[0].URLs) != 2 {
		t.Errorf("unexpected first rpm: %v", rules[0])
	}
	if rules[1].Name != "single-builddeps" || !slices.Equal(rules[1].URLs, []string{"https://storage.googleapis.com/builddeps/6d65bb2098442639ae5199ec4c8ee342b965dbc31f73277df63f04b4273f0278"}) {
		t.Errorf("unexpected second rpm: %v", rules[1])
	}
	if rules[2].Integrity != "sha256-FTdC0grTeCR9glZhnS/KVmZWW+/Y2/B0OWbaMxyZNUA=" {
		t.Errorf("expected integrity of third rpm to be captured, got %q", rules[2].Integrity)
	}
	if rules[3].Name != "no-urls" || len(rules[3].URLs) != 0 {
		t.Errorf("unexpected last rpm: %v", rules[3])
	}
}

func TestBuilddepsURL(t *testing.T) {
	for integrity, expectedURL := range map[string]string{
		"sha256-3RG6suoKvfoxA2LqzocUIqADNAvyIxNWJlAPj1qYX2s=": "https://storage.googleapis.com/builddeps/dd11bab2ea0abdfa310362eace871422a003340bf223135626500f8f5a985f6b",
		// not sha256-prefixed
		"3RG6suoKvfoxA2LqzocUIqADNAvyIxNWJlAPj1qYX2s=": "",
		// not base64
		"sha256-not-base64!!": "",
		// not a sha256-sized digest
		"sha256-c2VjcmV0": "",
		// empty
		"": "",
	} {
		if url := builddepsURL(integrity); url != expectedURL {
			t.Errorf("builddepsURL(%q) = %q, expected %q", integrity, url, expectedURL)
		}
	}
}

func TestURLFailures(t *testing.T) {
	rules, err := parseRPMRules([]byte(testRpmsJSON))
	if err != nil {
		t.Fatalf("failed to parse test rpms.json: %v", err)
	}

	expectedFailures := []string{
		"rpm single-mirror: only 1 url, not a builddeps url: http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/single-mirror.rpm (add https://storage.googleapis.com/builddeps/153742d20ad378247d8256619d2fca5666565befd8dbf0743966da331c993540)",
		"rpm no-urls: no urls (add https://storage.googleapis.com/builddeps/9e6aa0c60204bb4b152ce541ca3a9f5c28b020ed551dd417d3936a8b2153f0df)",
	}
	failures := urlFailures(rules, 2)
	if !slices.Equal(failures, expectedFailures) {
		t.Errorf("expected failures %v, got %v", expectedFailures, failures)
	}
}
