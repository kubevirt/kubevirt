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

// Command kv-coverage-collector runs the E2E coverage collector server. It is
// deployed as a test image into the kubevirt-coverage namespace when functests
// run with --cov-report; instrumented components POST their coverage files to
// it and it merges/serves the combined report.
package main

import (
	"net/http"
	"os"

	klog "kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/coveragecollector"
)

func main() {
	klog.InitializeLogging("kv-coverage-collector")

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	dataDir := os.Getenv("COVERAGE_DIR")
	if dataDir == "" {
		dataDir = "/coverage-data"
	}

	c := coveragecollector.New(dataDir)
	klog.Log.Infof("kv-coverage-collector listening on %s, storing to %s", addr, dataDir)
	if err := http.ListenAndServe(addr, c.Handler()); err != nil {
		klog.Log.Reason(err).Critical("coverage collector server stopped")
	}
}
