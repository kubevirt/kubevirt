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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package infrastructure

import (
	"context"
	"time"

	"kubevirt.io/kubevirt/tests/libinfra"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/libvmi"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
)

var _ = DescribeInfra("downwardMetrics", func() {
	var (
		virtClient kubecli.KubevirtClient
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	DescribeTable("should start a vmi and get the metrics", func(via libvmi.Option, metricsGetter libinfra.MetricsGetter) {

		vmi := libvmi.NewFedora(via)
		vmi = tests.RunVMIAndExpectLaunch(vmi, 180)
		Expect(console.LoginToFedora(vmi)).To(Succeed())

		metrics, err := metricsGetter(vmi)
		Expect(err).ToNot(HaveOccurred())
		timestamp := libinfra.GetTimeFromMetrics(metrics)

		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() int {
			metrics, err = metricsGetter(vmi)
			Expect(err).ToNot(HaveOccurred())
			return libinfra.GetTimeFromMetrics(metrics)
		}, 10*time.Second, 1*time.Second).ShouldNot(Equal(timestamp))
		Expect(libinfra.GetHostnameFromMetrics(metrics)).To(Equal(vmi.Status.NodeName))

	},
		Entry("[test_id:6535]using a disk", libvmi.WithDownwardMetricsVolume("vhostmd"), libinfra.GetDownwardMetricsDisk),
		Entry("using a virtio serial device", libvmi.WithDownwardMetricsChannel(), libinfra.GetDownwardMetricsVirtio),
	)

	It("metric ResourceProcessorLimit should be present", func() {
		vmi := libvmi.NewFedora(libvmi.WithCPUCount(1, 1, 1), libvmi.WithDownwardMetricsVolume("vhostmd"))
		vmi = tests.RunVMIAndExpectLaunch(vmi, 180)
		Expect(console.LoginToFedora(vmi)).To(Succeed())

		metrics, err := libinfra.GetDownwardMetricsDisk(vmi)
		Expect(err).ToNot(HaveOccurred())

		//let's try to find the ResourceProcessorLimit metric
		found := false
		j := 0
		for i, metric := range metrics.Metrics {
			if metric.Name == "ResourceProcessorLimit" {
				j = i
				found = true
				break
			}
		}
		Expect(found).To(BeTrue())
		Expect(metrics.Metrics[j].Value).To(Equal("1"))
	})
})
