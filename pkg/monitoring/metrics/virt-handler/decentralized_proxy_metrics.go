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

package virthandler

import "github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

var (
	decentralizedMigrationProxyMetrics = []operatormetrics.Metric{
		decentralizedMigrationProxyActiveConnections,
		decentralizedMigrationProxyBytesTransferred,
		decentralizedMigrationProxyErrors,
	}

	decentralizedMigrationProxyActiveConnections = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_decentralized_migration_proxy_active_connections",
			Help: "Number of active connections through the migration proxy",
		},
		[]string{"proxy_type"},
	)

	decentralizedMigrationProxyBytesTransferred = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_decentralized_migration_proxy_bytes_transferred_total",
			Help: "Total bytes transferred through the migration proxy",
		},
		[]string{"proxy_type", "direction"},
	)

	decentralizedMigrationProxyErrors = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_decentralized_migration_proxy_errors_total",
			Help: "Total number of migration proxy errors",
		},
		[]string{"proxy_type", "error_type"},
	)
)

// ProxyActiveConnectionsInc increments the active connections gauge for the given proxy type
func DecentralizedMigrationProxyActiveConnectionsInc(proxyType string) {
	decentralizedMigrationProxyActiveConnections.WithLabelValues(proxyType).Inc()
}

// ProxyActiveConnectionsDec decrements the active connections gauge for the given proxy type
func DecentralizedMigrationProxyActiveConnectionsDec(proxyType string) {
	decentralizedMigrationProxyActiveConnections.WithLabelValues(proxyType).Dec()
}

// DecentralizedMigrationProxyBytesTransferredAdd adds to the bytes transferred counter
func DecentralizedMigrationProxyBytesTransferredAdd(proxyType, direction string, bytes float64) {
	decentralizedMigrationProxyBytesTransferred.WithLabelValues(proxyType, direction).Add(bytes)
}

// ProxyErrorsInc increments the proxy errors counter
func DecentralizedMigrationProxyErrorsInc(proxyType, errorType string) {
	decentralizedMigrationProxyErrors.WithLabelValues(proxyType, errorType).Inc()
}
