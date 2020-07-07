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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"

	"kubevirt.io/kubevirt/tests/reporter"

	"kubevirt.io/kubevirt/tests"
	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"
)

func TestTests(t *testing.T) {
	maxFails := getMaxFailsFromEnv()
	reporters := []Reporter{
		reporter.NewKubernetesReporter(path.Join(os.Getenv("ARTIFACTS"), "k8s-reporter"), maxFails),
	}
	if ginkgo_reporters.Polarion.Run {
		reporters = append(reporters, &ginkgo_reporters.Polarion)
	}
	if ginkgo_reporters.JunitOutput != "" {
		reporters = append(reporters, ginkgo_reporters.NewJunitReporter())
	}
	RunSpecsWithDefaultAndCustomReporters(t, "Tests Suite", reporters)
}

var _ = BeforeSuite(func() {
	tests.BeforeTestSuitSetup()
})

var _ = AfterSuite(func() {
	tests.AfterTestSuitCleanup()
})

func getMaxFailsFromEnv() int {
	maxFailsEnv := os.Getenv("REPORTER_MAX_FAILS")
	if maxFailsEnv == "" {
		return 10
	}

	maxFails, err := strconv.Atoi(maxFailsEnv)
	if err != nil { // if the variable is set with a non int value
		fmt.Println("Invalid REPORTER_MAX_FAILS variable, defaulting to 10")
		return 10
	}

	return maxFails
}
