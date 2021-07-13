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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"time"
)

type inputConfig struct {
	StartTime time.Duration `json:"startTime"`
	EndTime   time.Duration `json:"endTime"`
}

func readInputFile(filePath string) (*inputConfig, error) {
	var cfg *inputConfig
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Unable to read file [%s]: %v", filePath, err)
	}

	if err := json.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("Failed to json unmarshal input config: %v", err)
	}

	return cfg, nil
}

func main() {
	var defaultInputConfigFile = "perf-audit-input.yaml"
	var defaultResultsFile = "perf-audit-results.yaml"

	var inputFile string
	var outputFile string

	flag.StringVar(&inputFile, "config-file", defaultInputConfigFile, "file path to the input config file")
	flag.StringVar(&outputFile, "results-file", defaultResultsFile, "file path for where to store results")

}
