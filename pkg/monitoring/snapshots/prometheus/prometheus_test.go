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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/log"
)

var _ = BeforeSuite(func() {
	log.Log.SetIOWriter(GinkgoWriter)
})

var _ = Describe("Prometheus", func() {
	Context("Observing snapshots", func() {
		It("should expose snapshot duration metrics", func() {
			snapshotMetrics := NewSnapshotMetrics()
			prometheus.MustRegister(snapshotMetrics.SnapshotDuration)

			// Add a fake sample to the histogram
			snapshotMetrics.ObserveSnapshotDuration(float64(1), "failed")

			reg := prometheus.NewRegistry()
			reg.MustRegister(snapshotMetrics.SnapshotDuration)

			metrics, _ := reg.Gather()

			// Assure correct metric was registred
			Expect(metrics[0].GetName()).To(BeEquivalentTo("kubevirt_snapshot_duration_seconds"))
			// Assure sample was observed
			Expect(metrics[0].Metric[0].Histogram.GetSampleCount()).To(BeEquivalentTo(1))
			// Assure label was added
			Expect(metrics[0].Metric[0].GetLabel()).To(HaveLen(1))
		})
	})
})
