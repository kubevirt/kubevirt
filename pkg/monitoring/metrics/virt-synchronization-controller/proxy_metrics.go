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
	"time"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

const (
	channelTypeState = "state"
	channelTypeDisk  = "disk"

	// Protocol ports for libvirt migration DATA channels (not control-plane sync RPCs).
	libvirtDirectMigrationPort = 49152 // state / memory
	libvirtBlockMigrationPort  = 49153 // disk / NBD

	// throughputWindow is the timescale used to compute bytes/sec gauges.
	throughputWindow = 10 * time.Second
)

var (
	proxyMetrics = []operatormetrics.Metric{
		proxyActiveConnections,
		proxyBytesTransferredTotal,
		proxyMigrationBytes,
		proxyThroughputBytesPerSecond,
		proxyErrors,
	}

	proxyActiveConnections = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_decentralized_migration_proxy_active_connections",
			Help: "Number of active connections through the migration proxy",
		},
		[]string{"proxy_type"},
	)

	proxyBytesTransferredTotal = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_decentralized_migration_proxy_bytes_transferred_total",
			Help: "Total state/disk bytes transferred through the migration proxy across all migrations",
		},
		[]string{"proxy_type", "channel_type"},
	)

	proxyMigrationBytes = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_decentralized_migration_proxy_migration_bytes",
			Help: "State/disk bytes transferred for an active migration through the proxy",
		},
		[]string{"migration_id", "proxy_type", "channel_type"},
	)

	proxyThroughputBytesPerSecond = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_decentralized_migration_proxy_throughput_bytes_per_second",
			Help: "Approximate state/disk proxy throughput in bytes per second over a short window",
		},
		[]string{"proxy_type", "channel_type"},
	)

	proxyErrors = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_decentralized_migration_proxy_errors_total",
			Help: "Total number of migration proxy errors",
		},
		[]string{"proxy_type", "error_type"},
	)
)

type throughputKey struct {
	proxyType   string
	channelType string
}

var (
	throughputMu       sync.Mutex
	throughputBytes    = map[throughputKey]*atomic.Uint64{}
	throughputReporter sync.Once
)

// ChannelTypeFromID maps a migration protocol channel/port to a metric label.
// Returns empty string for channels that should not be counted (control / unknown).
func ChannelTypeFromID(channelID int32) string {
	switch channelID {
	case libvirtDirectMigrationPort:
		return channelTypeState
	case libvirtBlockMigrationPort:
		return channelTypeDisk
	default:
		return ""
	}
}

func throughputCounter(proxyType, channelType string) *atomic.Uint64 {
	key := throughputKey{proxyType: proxyType, channelType: channelType}
	throughputMu.Lock()
	defer throughputMu.Unlock()
	if c, ok := throughputBytes[key]; ok {
		return c
	}
	c := &atomic.Uint64{}
	throughputBytes[key] = c
	return c
}

// StartThroughputReporter periodically updates throughput gauges from byte deltas.
// Safe to call once; subsequent calls are no-ops.
func StartThroughputReporter(stop <-chan struct{}) {
	throughputReporter.Do(func() {
		go reportThroughput(stop)
	})
}

func reportThroughput(stop <-chan struct{}) {
	ticker := time.NewTicker(throughputWindow)
	defer ticker.Stop()

	last := map[throughputKey]uint64{}
	lastTime := time.Now()

	known := []throughputKey{
		{proxyType: "source", channelType: channelTypeState},
		{proxyType: "source", channelType: channelTypeDisk},
		{proxyType: "target", channelType: channelTypeState},
		{proxyType: "target", channelType: channelTypeDisk},
	}

	for {
		select {
		case <-stop:
			return
		case now := <-ticker.C:
			elapsed := now.Sub(lastTime).Seconds()
			if elapsed <= 0 {
				continue
			}
			for _, key := range known {
				cur := throughputCounter(key.proxyType, key.channelType).Load()
				prev := last[key]
				var rate float64
				if cur >= prev {
					rate = float64(cur-prev) / elapsed
				}
				proxyThroughputBytesPerSecond.WithLabelValues(key.proxyType, key.channelType).Set(rate)
				last[key] = cur
			}
			lastTime = now
		}
	}
}

// ActiveConnectionsInc increments the active connections gauge.
func ActiveConnectionsInc(proxyType string) {
	proxyActiveConnections.WithLabelValues(proxyType).Inc()
}

// ActiveConnectionsDec decrements the active connections gauge.
func ActiveConnectionsDec(proxyType string) {
	proxyActiveConnections.WithLabelValues(proxyType).Dec()
}

// BytesTransferredAdd records state/disk DATA bytes once per forwarded payload.
// channelType must be "state" or "disk"; other values are ignored.
func BytesTransferredAdd(migrationID, proxyType, channelType string, bytes float64) {
	if bytes <= 0 || (channelType != channelTypeState && channelType != channelTypeDisk) {
		return
	}
	proxyBytesTransferredTotal.WithLabelValues(proxyType, channelType).Add(bytes)
	proxyMigrationBytes.WithLabelValues(migrationID, proxyType, channelType).Add(bytes)
	throughputCounter(proxyType, channelType).Add(uint64(bytes))
}

// ClearMigrationBytes removes per-migration series when a tunnel stops.
func ClearMigrationBytes(migrationID string) {
	for _, proxyType := range []string{"source", "target"} {
		for _, channelType := range []string{channelTypeState, channelTypeDisk} {
			proxyMigrationBytes.DeleteLabelValues(migrationID, proxyType, channelType)
		}
	}
}

// ErrorsInc increments the proxy errors counter.
func ErrorsInc(proxyType, errorType string) {
	proxyErrors.WithLabelValues(proxyType, errorType).Inc()
}
