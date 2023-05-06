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

	jsonBytes, err := json.Marshal(metricFamilies)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(string(jsonBytes)) // Write the JSON string to standard output
}
