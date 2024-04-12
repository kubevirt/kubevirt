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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package testsuite

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo/v2"

	"kubevirt.io/kubevirt/tests/flags"

	qe_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"
)

func GetJunitOutputPath() string {
	junitOutput := filepath.Join(flags.ArtifactsDir, "junit.functest.xml")
	if qe_reporters.JunitOutput != "" {
		junitOutput = qe_reporters.JunitOutput
	}
	return junitOutput
}

func UpdateArtifactsForParallel(artifactsPath string, junitOutput string) (artifactsPathForParallel, junitOutputForParallel string) {
	artifactsPathForParallel = filepath.Join(artifactsPath, strconv.Itoa(GinkgoParallelProcess()))
	junitOutputForParallel = filepath.Join(flags.ArtifactsDir, fmt.Sprintf("partial.junit.functest.%d.xml", GinkgoParallelProcess()))

	return artifactsPathForParallel, junitOutputForParallel
}

func GetPolarionFilename() string {
	return filepath.Join(flags.ArtifactsDir, fmt.Sprintf("partial.polarion.functest.%d.xml", GinkgoParallelProcess()))
}

func GetMaxFailsFromEnv() int {
	const defaultMaxFails = 10
	maxFailsEnv := os.Getenv("REPORTER_MAX_FAILS")
	if maxFailsEnv == "" {
		return defaultMaxFails
	}

	maxFails, err := strconv.Atoi(maxFailsEnv)
	if err != nil { // if the variable is set with a non int value
		fmt.Println("Invalid REPORTER_MAX_FAILS variable, defaulting to 10")
		return defaultMaxFails
	}

	return maxFails
}
