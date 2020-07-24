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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/log"
)

var _ = BeforeSuite(func() {
	log.Log.SetIOWriter(GinkgoWriter)
})

var _ = Describe("Prometheus", func() {
	Context("Observing migrations", func() {
		It("should expose migration duration metrics", func() {
			migrationMetricsConfig := &v1.HistogramsConfig{
				DurationHistogram: &v1.HistogramMetric{
					BucketValues: []float64{1, 2, 3},
				},
			}

			migrationMetrics := NewMigrationMetrics(migrationMetricsConfig)
			// Add a fake sample to the histogram
			migrationMetrics.ObserveMigrationDuration(float64(1), "fakeTarget", "fakeSource", "fakeVMI", "fakeNamespace", "failed")

			// Register and collect metrics from registry
			reg := prometheus.NewRegistry()
			reg.MustRegister(migrationMetrics.MigrationDuration)
			metrics, _ := reg.Gather()

			// Assure correct metric was registred
			Expect(metrics[0].GetName()).To(BeEquivalentTo("kubevirt_migration_duration_seconds"))
			// Assure sample was observed
			Expect(metrics[0].Metric[0].Histogram.GetSampleCount()).To(BeEquivalentTo(1))
			// Assure labels were added
			Expect(metrics[0].Metric[0].GetLabel()).To(HaveLen(len(migrationLabels)))
		})
	})

	Context("Setting different configurations", func() {
		It("should create custom buckets", func() {
			buckets := []float64{60, 180, 300, 1800, 3600}
			migrationMetricsConfig := &v1.HistogramsConfig{
				DurationHistogram: &v1.HistogramMetric{
					BucketValues: buckets,
				},
			}

			migrationMetrics := NewMigrationMetrics(migrationMetricsConfig)

			// Add a fake sample to the histogram
			migrationMetrics.ObserveMigrationDuration(float64(1), "fakeTarget", "fakeSource", "fakeVMI", "fakeNamespace", "failed")

			// Register and collect metrics from registry
			reg := prometheus.NewRegistry()
			reg.MustRegister(migrationMetrics.MigrationDuration)
			metrics, _ := reg.Gather()

			for i := 0; i < len(buckets); i++ {
				// Assure correct bucket values
				Expect(metrics[0].Metric[0].Histogram.Bucket[i].GetUpperBound()).To(BeEquivalentTo(buckets[i]))
			}
		})
	})
})
