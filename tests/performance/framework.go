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

package performance

import (
	"flag"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo"
)

var RunPerfTests = false

func init() {
	flag.BoolVar(&RunPerfTests, "performance-test", false, "run performance tests. If false, all performance tests will be skiped.")
	if ptest, _ := strconv.ParseBool(os.Getenv("KUBEVIRT_E2E_PERF_TEST")); ptest {
		RunPerfTests = true
	}
}

func SIGDescribe(text string, body func()) bool {
	BeforeEach(func() {
		if !RunPerfTests {
			Skip("Performance tests are not enabled.")
		}
	})
	return Describe("[sig-performance][Serial] "+text, body)
}

func FSIGDescribe(text string, body func()) bool {
	BeforeEach(func() {
		if !RunPerfTests {
			Skip("Performance tests are not enabled.")
		}
	})
	return FDescribe("[sig-performance][Serial] "+text, body)
}
