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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = DescribeInfra("downwardMetrics", func() {
	var (
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		tests.EnableDownwardMetrics(virtClient)
	})

	DescribeTable("should start a vmi and get the metrics", func(via libvmi.Option, metricsGetter libinfra.MetricsGetter) {
		vmi := libvmifact.NewFedora(via)
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)
		Expect(console.LoginToFedora(vmi)).To(Succeed())

		metrics, err := metricsGetter(vmi)
		Expect(err).ToNot(HaveOccurred())
		timestamp := libinfra.GetTimeFromMetrics(metrics)

		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)
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

	DescribeTable("should not start a VMI if the feature is disabled until it is re-enabled", func(via libvmi.Option) {
		By("disabling downwardMetrics feature")
		tests.DisableDownwardMetrics(virtClient)

		DeferCleanup(func() {
			By("re-enabling the downwardMetrics feature")
			tests.EnableDownwardMetrics(virtClient)
		})

		vmi := libvmifact.NewFedora(via)
		vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("checking the phase of the VMI")
		Eventually(matcher.ThisVMI(vmi), 60*time.Second, 5*time.Second).Should(matcher.BeInPhase(v1.Pending))

		By("re-enabling the downwardMetrics feature")
		tests.EnableDownwardMetrics(virtClient)

		Eventually(matcher.ThisVMI(vmi), 60*time.Second, 5*time.Second).ShouldNot(matcher.BeInPhase(v1.Pending))
	},
		Entry("using a disk", libvmi.WithDownwardMetricsVolume("vhostmd")),
		Entry("using a virtio serial device", libvmi.WithDownwardMetricsChannel()),
	)

	DescribeTable("should update the VMI conditions if the feature is disabled", func(via libvmi.Option) {
		vmi := libvmifact.NewFedora(via)
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)
		Expect(console.LoginToFedora(vmi)).To(Succeed())

		By("disabling downwardMetrics feature")
		tests.DisableDownwardMetrics(virtClient)

		DeferCleanup(func() {
			By("re-enabling the downwardMetrics feature")
			tests.EnableDownwardMetrics(virtClient)
		})

		// Triggering an update to force the reconciliation loop. This will avoid to wait much time.
		Expect(console.LoginToFedora(vmi)).To(Succeed())

		By("checking the VMI conditions")
		Eventually(func(g Gomega) {
			var err error
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			cond := controller.NewVirtualMachineInstanceConditionManager().GetCondition(vmi, v1.VirtualMachineInstanceConfigurationOutOfSync)
			g.Expect(cond).ToNot(BeNil())
			g.Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Type":    Equal(v1.VirtualMachineInstanceConfigurationOutOfSync),
				"Status":  Equal(k8sv1.ConditionFalse),
				"Message": ContainSubstring("The DownwardMetrics feature is disabled but still in use"),
			}))
		}, 60*time.Second, 5*time.Second).Should(Succeed())

	},
		Entry("using a disk", libvmi.WithDownwardMetricsVolume("vhostmd")),
		Entry("using a virtio serial device", libvmi.WithDownwardMetricsChannel()),
	)

	DescribeTable("should update VM conditions if the feature is disabled", func(via libvmi.Option) {
		vm := libvmi.NewVirtualMachine(libvmifact.NewFedora(via))

		By("disabling downwardMetrics feature")
		tests.DisableDownwardMetrics(virtClient)

		DeferCleanup(func() {
			By("re-enabling the downwardMetrics feature")
			tests.EnableDownwardMetrics(virtClient)
		})

		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("checking the VM conditions when the VM is stopped")
		Eventually(func(g Gomega) {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			cond := controller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
			g.Expect(cond).ToNot(BeNil())
			g.Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Type":    Equal(v1.VirtualMachineFailure),
				"Reason":  Equal(controller.FeatureNotEnabled),
				"Status":  Equal(k8sv1.ConditionTrue),
				"Message": ContainSubstring(downwardmetrics.DownwardMetricsNotEnabledError.Error()),
			}))
		}, 60*time.Second, 5*time.Second).Should(Succeed())

		By("starting VM")
		err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("checking the VM conditions when the VM is starting")
		Eventually(func(g Gomega) {
			vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			cond := controller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineFailure)
			g.Expect(cond).ToNot(BeNil())
			g.Expect(*cond).To(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Type":    Equal(v1.VirtualMachineFailure),
				"Reason":  Equal(common.FailedCreateVirtualMachineReason),
				"Status":  Equal(k8sv1.ConditionTrue),
				"Message": ContainSubstring("Failure while starting VMI: %s", downwardmetrics.DownwardMetricsNotEnabledError.Error()),
			}))
		}, 60*time.Second, 5*time.Second).Should(Succeed())
	},
		Entry("using a disk", libvmi.WithDownwardMetricsVolume("vhostmd")),
		Entry("using a virtio serial device", libvmi.WithDownwardMetricsChannel()),
	)

	DescribeTable("should no create a VMI object if the feature is disabled", func(via libvmi.Option) {
		vm := libvmi.NewVirtualMachine(libvmifact.NewFedora(via))

		By("disabling downwardMetrics feature")
		tests.DisableDownwardMetrics(virtClient)

		DeferCleanup(func() {
			By("re-enabling the downwardMetrics feature")
			tests.EnableDownwardMetrics(virtClient)
		})

		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("starting VM")
		err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
		Expect(err).ToNot(HaveOccurred())

		Consistently(func() bool {
			_, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			return errors.IsNotFound(err)
		}, 60*time.Second, 5*time.Second).Should(BeTrue())
	},
		Entry("using a disk", libvmi.WithDownwardMetricsVolume("vhostmd")),
		Entry("using a virtio serial device", libvmi.WithDownwardMetricsChannel()),
	)
})
