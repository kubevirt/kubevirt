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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 *
 */

package main

import (
	"encoding/json"
	"fmt"
	"os"

	dto "github.com/prometheus/client_model/go"
)

type Output struct {
	MetricFamilies []*dto.MetricFamily `json:"metricFamilies,omitempty"`
	RecordingRules []RecordingRule     `json:"recordingRules,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("please provide path to the metrics file")
		os.Exit(1)
	}

	path := os.Args[1]
	if _, err := os.Stat(path); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Build recording rules first and filter metrics accordingly
	recRules, err := ExtractRecordingRules()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	recRulesNames := make(map[string]struct{}, len(recRules))
	for _, rr := range recRules {
		recRulesNames[rr.Record] = struct{}{}
	}
	metricFamilies, err := ReadMetrics(path, recRulesNames)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	out := Output{MetricFamilies: metricFamilies, RecordingRules: recRules}

	jsonBytes, err := json.Marshal(out)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(string(jsonBytes)) // Write the JSON string to standard output
}
