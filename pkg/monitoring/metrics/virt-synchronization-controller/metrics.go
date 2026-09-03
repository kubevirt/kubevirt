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

package synccontrollermetrics

import (
	"sync"
	"sync/atomic"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var (
	registerOnce sync.Once
	registerErr  error
)

// EnsureProxyMetrics registers proxy metrics and starts the throughput reporter.
// Safe to call multiple times; registration happens once. Call this when the
// migration tunnel proxy is actually initialized (networks available).
func EnsureProxyMetrics(stop <-chan struct{}) error {
	registerOnce.Do(func() {
		registerErr = operatormetrics.RegisterMetrics(proxyMetrics)
		if registerErr != nil {
			return
		}
		StartThroughputReporter(stop)
	})
	return registerErr
}

// SetupMetrics registers synchronization-controller proxy metrics without
// starting the throughput reporter. Prefer EnsureProxyMetrics when the proxy runs.
func SetupMetrics() error {
	return operatormetrics.RegisterMetrics(proxyMetrics)
}

// ListMetrics returns registered metrics for this process.
func ListMetrics() []operatormetrics.Metric {
	return operatormetrics.ListMetrics()
}

// resetRegistrationForTesting clears sync.Once state so tests can re-register.
func resetRegistrationForTesting() {
	registerOnce = sync.Once{}
	registerErr = nil
	throughputReporter = sync.Once{}
	throughputMu.Lock()
	throughputBytes = map[throughputKey]*atomic.Uint64{}
	throughputMu.Unlock()
}
