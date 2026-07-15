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
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

func TestFeatureGateReport(t *testing.T) {
	gates := featuregate.GetRegisteredFeatureGates()
	if len(gates) == 0 {
		t.Fatal("expected registered feature gates, got none")
	}

	for name, fg := range gates {
		if fg.State == featuregate.GA || fg.State == featuregate.Discontinued {
			continue
		}
		if name == "" {
			t.Error("found feature gate with empty name")
		}
		if fg.State != featuregate.Alpha && fg.State != featuregate.Beta && fg.State != featuregate.Deprecated {
			t.Errorf("feature gate %q has unexpected state %q", name, fg.State)
		}
	}
}

func TestRenderJSON(t *testing.T) {
	entries := []featureGateEntry{
		{Name: "FeatureA", State: "Alpha"},
		{Name: "FeatureB", State: "Beta"},
	}

	var buf bytes.Buffer
	if err := renderJSON(&buf, entries); err != nil {
		t.Fatalf("renderJSON returned error: %v", err)
	}

	var got []featureGateEntry
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if len(got) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(got))
	}
	for i, e := range entries {
		if got[i] != e {
			t.Errorf("entry %d: expected %+v, got %+v", i, e, got[i])
		}
	}
}

func TestRenderMarkdown(t *testing.T) {
	entries := []featureGateEntry{
		{Name: "FeatureA", State: "Alpha"},
		{Name: "FeatureB", State: "Beta"},
		{Name: "FeatureC", State: "Deprecated"},
	}

	var buf bytes.Buffer
	renderMarkdown(&buf, entries)
	output := buf.String()

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	expectedLines := []string{
		"# Feature Gate Report",
		"",
		"| Feature Gate | State |",
		"|---|---|",
		"| FeatureA | Alpha |",
		"| FeatureB | Beta |",
		"| FeatureC | Deprecated |",
	}

	if len(lines) != len(expectedLines) {
		t.Fatalf("expected %d lines, got %d:\n%s", len(expectedLines), len(lines), output)
	}
	for i, expected := range expectedLines {
		if lines[i] != expected {
			t.Errorf("line %d: expected %q, got %q", i, expected, lines[i])
		}
	}
}
