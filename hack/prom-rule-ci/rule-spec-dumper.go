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
 */

package main

import (
	"encoding/json"

	"kubevirt.io/kubevirt/pkg/monitoring/rules"

	"fmt"
	"os"
)

func verifyArgs(args []string) error {
	numOfArgs := len(os.Args[1:])
	if numOfArgs != 1 {
		return fmt.Errorf("expected exactly 1 argument, got: %d", numOfArgs)
	}
	return nil
}

func main() {
	if err := verifyArgs(os.Args); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	targetFile := os.Args[1]

	if err := rules.SetupRules("ci"); err != nil {
		panic(err)
	}

	promRule, err := rules.BuildPrometheusRule("ci")
	if err != nil {
		panic(err)
	}
	b, err := json.Marshal(promRule.Spec)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(targetFile, b, 0644)
	if err != nil {
		panic(err)
	}
}
