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

package infrastructure

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe(SIG("downwardMetrics", func() {
	const vmiStartTimeout = libvmops.StartupTimeoutSecondsXLarge

	DescribeTable("should start a vmi and get the metrics", func(via libvmi.Option, metricsGetter libinfra.MetricsGetter) {
		vmi := libvmifact.NewFedora(via)
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, vmiStartTimeout)
		Expect(console.LoginToFedora(vmi)).To(Succeed())

		metrics, err := metricsGetter(vmi)
		Expect(err).ToNot(HaveOccurred())
		timestamp := libinfra.GetTimeFromMetrics(metrics)

		vmi, err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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
		vmi := libvmifact.NewFedora(libvmi.WithCPUCount(1, 1, 1), libvmi.WithDownwardMetricsVolume("vhostmd"))
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, vmiStartTimeout)
		Expect(console.LoginToFedora(vmi)).To(Succeed())

		metrics, err := libinfra.GetDownwardMetricsDisk(vmi)
		Expect(err).ToNot(HaveOccurred())

		Expect(metrics.Metrics).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Name":  Equal("ResourceProcessorLimit"),
			"Value": Equal("1"),
		})))
	})
}))
