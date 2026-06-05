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
	"fmt"
	"os"
	"sort"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

type featureGateEntry struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

func main() {
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

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
