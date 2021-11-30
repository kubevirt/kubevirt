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
	"github.com/onsi/ginkgo/config"
	ginkgo_reporters "github.com/onsi/ginkgo/reporters"

	"kubevirt.io/kubevirt/tests/flags"

	"kubevirt.io/kubevirt/tests/reporter"

	"kubevirt.io/kubevirt/tests"
	qe_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	_ "kubevirt.io/kubevirt/tests/network"
	_ "kubevirt.io/kubevirt/tests/numa"
	_ "kubevirt.io/kubevirt/tests/performance"
	_ "kubevirt.io/kubevirt/tests/realtime"
	_ "kubevirt.io/kubevirt/tests/storage"
	_ "kubevirt.io/kubevirt/tests/virtoperatorbasic"
)

var justAfterEachReporter = []reporter.JustAfterEachReporter{}

func TestTests(t *testing.T) {
	flags.NormalizeFlags()
	tests.CalculateNamespaces()
	maxFails := getMaxFailsFromEnv()
	artifactsPath := path.Join(flags.ArtifactsDir, "k8s-reporter")
	junitOutput := path.Join(flags.ArtifactsDir, "junit.functest.xml")
	if qe_reporters.JunitOutput != "" {
		junitOutput = qe_reporters.JunitOutput
	}
	if config.GinkgoConfig.ParallelTotal > 1 {
		artifactsPath = path.Join(artifactsPath, strconv.Itoa(config.GinkgoConfig.ParallelNode))
		junitOutput = path.Join(flags.ArtifactsDir, fmt.Sprintf("partial.junit.functest.%d.xml", config.GinkgoConfig.ParallelNode))
	}

	outputEnricherReporter := reporter.NewCapturedOutputEnricher(
		ginkgo_reporters.NewJUnitReporter(junitOutput),
	)
	k8sReporter := reporter.NewKubernetesReporter(artifactsPath, maxFails)
	justAfterEachReporter = append(justAfterEachReporter, outputEnricherReporter, k8sReporter)
	reporters := []Reporter{
		outputEnricherReporter,
		k8sReporter,
	}
	if qe_reporters.Polarion.Run {
		reporters = append(reporters, &qe_reporters.Polarion)
	}

	RunSpecsWithDefaultAndCustomReporters(t, "Tests Suite", reporters)
}

var _ = SynchronizedBeforeSuite(tests.SynchronizedBeforeTestSetup, tests.BeforeTestSuitSetup)

var _ = SynchronizedAfterSuite(tests.AfterTestSuitCleanup, tests.SynchronizedAfterTestSuiteCleanup)

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

// Collect info directly after each `It` execution with our reporters
// to collect the state directly after the spec.
var _ = JustAfterEach(func() {
	for _, reporter := range justAfterEachReporter {
		reporter.JustAfterEach(CurrentGinkgoTestDescription())
	}
})
