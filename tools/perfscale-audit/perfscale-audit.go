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
	"log"

	audit_api "kubevirt.io/kubevirt/tools/perfscale-audit/api"
	metric_client "kubevirt.io/kubevirt/tools/perfscale-audit/metric-client"
)

func main() {
	var defaultInputConfigFile = "perf-audit-input.json"
	var defaultResultsFile = "perf-audit-results.json"

	var inputFile string
	var outputFile string

	flag.StringVar(&inputFile, "config-file", defaultInputConfigFile, "file path to the input config file")
	flag.StringVar(&outputFile, "results-file", defaultResultsFile, "file path for where to store results")
	flag.Parse()

	inputCfg, err := audit_api.ReadInputFile(inputFile)
	if err != nil {
		log.Fatal(err)
	}

	metricClient, err := metric_client.NewMetricClient(inputCfg)
	if err != nil {
		log.Fatal(err)
	}

	result, err := metricClient.GenerateResults()
	if err != nil {
		log.Fatal(err)
	}

	err = result.DumpToFile(outputFile)
	if err != nil {
		log.Fatal(err)
	}

	err = result.DumpToStdout()
	if err != nil {
		log.Fatal(err)
	}
}
