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

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("Prometheus", func() {
	log.Log.SetIOWriter(GinkgoWriter)

	Context("on blocked source", func() {
		It("should handle closed reporting socket", func() {
			ch := make(chan prometheus.Metric)
			close(ch)

			ps := prometheusScraper{ch: ch}

			testReportPanic := func() {
				vmStats := &stats.DomainStats{
					Cpu: &stats.DomainStatsCPU{},
					Memory: &stats.DomainStatsMemory{
						// trigger write on a socket. We need a value set - any value
						RSS:    1024,
						RSSSet: true,
					},
				}
				ps.Report("test", vmStats)
			}
			Expect(testReportPanic).ToNot(Panic())
		})
	})
})
