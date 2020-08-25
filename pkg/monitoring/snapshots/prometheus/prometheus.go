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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

const snapshotNamespace = "kubevirt"
const snapshotSubsystem = "snapshot"

type SnapshotMetrics struct {
	SnapshotDuration *prometheus.HistogramVec
}

func NewSnapshotMetrics() *SnapshotMetrics {
	return &SnapshotMetrics{
		SnapshotDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: snapshotNamespace,
				Subsystem: snapshotSubsystem,
				Name:      "duration_seconds",
				Help:      "Time spent on snapshots.",
				// Will create 12 buckets, starting with snapshots that take less then 2 seconds
				// up to 2048 seconds(34,13 minutes), plus an extra for +Inf
				Buckets: prometheus.ExponentialBuckets(2, 2, 11),
				// TODO: Make possible for users to manage buckets length and values
			}, []string{"status"},
		),
	}
}

// ObserveSnapshotDuration will add the time spent on a particular snapshot to the correct bucket
func (m *SnapshotMetrics) ObserveSnapshotDuration(duration float64, status string) {
	m.SnapshotDuration.With(
		prometheus.Labels{
			"status": status,
		},
	).Observe(duration)
}
