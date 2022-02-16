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

var (
	RunPerfTests                bool
	cyclicTestDurationInSeconds uint
	RunPerfRealtime             bool
	realtimeThreshold           uint
)

func init() {
	flag.BoolVar(&RunPerfTests, "performance-test", false, "run performance tests. If false, all performance tests will be skiped.")
	if ptest, _ := strconv.ParseBool(os.Getenv("KUBEVIRT_E2E_PERF_TEST")); ptest {
		RunPerfTests = true
	}
	flag.BoolVar(&RunPerfRealtime, "realtime-test", false, "run realtime performance tests only.")
	if run, _ := strconv.ParseBool(os.Getenv("KUBEVIRT_E2E_REALTIME_PERF_TEST")); run {
		RunPerfRealtime = true
	}
	flag.UintVar(&cyclicTestDurationInSeconds, "cyclictest-duration", 60, "time in seconds to run the cyclic test (for realtime performance)")
	if duration, err := strconv.ParseUint(os.Getenv("KUBEVIRT_E2E_CYCLIC_DURATION_IN_SECONDS"), 10, 64); err != nil {
		cyclicTestDurationInSeconds = uint(duration)
	}
	flag.UintVar(&realtimeThreshold, "realtime-threshold", 40, "sets the threshold for maximum cpu latency time in microseconds (for realtime performance)")
	if threshold, err := strconv.ParseUint(os.Getenv("KUBEVIRT_E2E_REALTIME_LATENCY_THRESHOLD_IN_MICROSECONDS"), 10, 64); err != nil {
		realtimeThreshold = uint(threshold)
	}
}

func SIGDescribe(text string, body func()) bool {
	return Describe("[sig-performance][Serial] "+text, body)
}

func FSIGDescribe(text string, body func()) bool {
	return FDescribe("[sig-performance][Serial] "+text, body)
}

func skipIfNoPerformanceTests() {
	if !RunPerfTests {
		Skip("Performance tests are not enabled.")
	}
}

func skipIfNoRealtimePerformanceTests() {
	if !RunPerfRealtime {
		Skip("Realtime performance tests are not enabled")
	}
}
