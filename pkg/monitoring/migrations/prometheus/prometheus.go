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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"

	v1 "kubevirt.io/client-go/api/v1"
)

const (
	migrationNamespace = "kubevirt"
	migrationSubsystem = "migration"
	statusLabel        = "status"
	targetNodeLabel    = "target_node"
	sourceNodeLabel    = "source_node"
	vmiNameLabel       = "name"
	vmiNamespaceLabel  = "namespace"
)

var migrationLabels = []string{statusLabel, targetNodeLabel, sourceNodeLabel, vmiNameLabel, vmiNamespaceLabel}

type MigrationMetrics struct {
	MigrationDuration *prometheus.HistogramVec
}

func NewMigrationMetrics(migrationMetrics *v1.HistogramsConfig) *MigrationMetrics {
	return &MigrationMetrics{
		MigrationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: migrationNamespace,
				Subsystem: migrationSubsystem,
				Name:      "duration_seconds",
				Help:      "Time spent on migrations.",
				Buckets:   migrationMetrics.DurationHistogram.BucketValues,
			}, migrationLabels,
		),
	}
}

// ObserveMigrationDuration will add the time spent on a particular migration to the correct bucket
func (m *MigrationMetrics) ObserveMigrationDuration(duration float64, target, source, vmiName, vmiNamespace, status string) {
	m.MigrationDuration.With(
		prometheus.Labels{
			statusLabel:       status,
			targetNodeLabel:   target,
			sourceNodeLabel:   source,
			vmiNameLabel:      vmiName,
			vmiNamespaceLabel: vmiNamespace,
		},
	).Observe(duration)
}
