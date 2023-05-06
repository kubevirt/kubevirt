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
	"flag"
	"fmt"
	"os"

	"github.com/prometheus/client_golang/prometheus/testutil/promlint"
)

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

	metricFamilies, err := ReadMetrics(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Call customRules function to apply the custom rules to the linter
	linter := promlint.NewWithMetricFamilies(metricFamilies)

	problems, err := linter.Lint()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Set the operator and sub-operator names
	operatorName := flag.String("operator-name", "kubevirt", "")
	subOperatorName := flag.String("sub-operator-name", "kubevirt", "")
	flag.Parse()

	for _, family := range metricFamilies {
		problems = CustomLinterRules(problems, family, *operatorName, *subOperatorName)
	}

	for _, problem := range problems {
		fmt.Printf("%s: %s\n", problem.Metric, problem.Text)
	}
}
