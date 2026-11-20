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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	_ "kubevirt.io/kubevirt/pkg/virt-config/featuregate/compute"
	_ "kubevirt.io/kubevirt/pkg/virt-config/featuregate/legacy"
	_ "kubevirt.io/kubevirt/pkg/virt-config/featuregate/network"
	_ "kubevirt.io/kubevirt/pkg/virt-config/featuregate/storage"
)

type featureGateEntry struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

func collectEntries() []featureGateEntry {
	gates := featuregate.GetRegisteredFeatureGates()

	var entries []featureGateEntry
	for _, fg := range gates {
		if fg.State == featuregate.GA || fg.State == featuregate.Discontinued {
			continue
		}
		entries = append(entries, featureGateEntry{
			Name:  fg.Name,
			State: string(fg.State),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].State != entries[j].State {
			return entries[i].State < entries[j].State
		}
		return entries[i].Name < entries[j].Name
	})

	return entries
}

func renderJSON(w io.Writer, entries []featureGateEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	if _, err = fmt.Fprintln(w, string(data)); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}

func renderMarkdown(w io.Writer, entries []featureGateEntry) {
	fmt.Fprintln(w, "# Feature Gate Report")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Feature Gate | State |")
	fmt.Fprintln(w, "|---|---|")
	for _, e := range entries {
		fmt.Fprintf(w, "| %s | %s |\n", e.Name, e.State)
	}
}

func main() {
	outputFormat := flag.String("output-format", "json", "Output format: json or md")
	outputFile := flag.String("output-file", "", "Output file path (default: stdout)")
	flag.Parse()

	entries := collectEntries()

	w := io.Writer(os.Stdout)
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing file: %v\n", err)
				os.Exit(1)
			}
		}()
		w = f
	}

	switch *outputFormat {
	case "json":
		if err := renderJSON(w, entries); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "md":
		renderMarkdown(w, entries)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown output format %q (supported: json, md)\n", *outputFormat)
		os.Exit(1)
	}
}
