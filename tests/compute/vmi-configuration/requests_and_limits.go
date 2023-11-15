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
 * Copyright the KubeVirt Authors.
 *
 */

package vmi_configuration

import (
	"context"
	"runtime"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = ConfigDescribe("", func() {
	const enoughMemForSafeBiosEmulation = "32Mi"
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("[rfe_id:140][crit:medium][vendor:cnv-qe@redhat.com][level:component]with CPU request settings", func() {

		It("[test_id:3127]should set CPU request from VMI spec", func() {
			vmi := libvmi.New(
				libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation),
				libvmi.WithResourceCPU("500m"),
			)
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			cpuRequest := computeContainer.Resources.Requests[kubev1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("500m"))
		})

		It("[test_id:3128]should set CPU request when it is not provided", func() {
			vmi := tests.NewRandomVMI()
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			cpuRequest := computeContainer.Resources.Requests[kubev1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("100m"))
		})

		It("[Serial][test_id:3129]should set CPU request from kubevirt-config", Serial, func() {
			kv := util.GetCurrentKv(virtClient)

			config := kv.Spec.Configuration
			configureCPURequest := resource.MustParse("800m")
			config.CPURequest = &configureCPURequest
			tests.UpdateKubeVirtConfigValueAndWait(config)

			vmi := tests.NewRandomVMI()
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			cpuRequest := computeContainer.Resources.Requests[kubev1.ResourceCPU]
			Expect(cpuRequest.String()).To(Equal("800m"))
		})
	})

	Context("[Serial]with automatic CPU limit configured in the CR", Serial, func() {
		BeforeEach(func() {
			By("Adding a label selector to the CR for auto CPU limit")
			kv := util.GetCurrentKv(virtClient)
			config := kv.Spec.Configuration
			config.AutoCPULimitNamespaceLabelSelector = &metav1.LabelSelector{
				MatchLabels: map[string]string{"autocpulimit": "true"},
			}
			tests.UpdateKubeVirtConfigValueAndWait(config)
		})
		It("should not set a CPU limit if the namespace doesn't match the selector", func() {
			By("Creating a running VMI")
			vmi := tests.NewRandomVMI()
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			By("Ensuring no CPU limit is set")
			readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			_, exists := computeContainer.Resources.Limits[kubev1.ResourceCPU]
			Expect(exists).To(BeFalse(), "CPU limit set on the compute container when none was expected")
		})
		It("should set a CPU limit if the namespace matches the selector", func() {
			By("Creating a VMI object")
			vmi := tests.NewRandomVMI()

			By("Adding the right label to VMI namespace")
			namespace, err := virtClient.CoreV1().Namespaces().Get(context.Background(), vmi.Namespace, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			namespace.Labels["autocpulimit"] = "true"
			namespace, err = virtClient.CoreV1().Namespaces().Update(context.Background(), namespace, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Starting the VMI")
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

			By("Ensuring the CPU limit is set to the correct value")
			readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			computeContainer := tests.GetComputeContainerOfPod(readyPod)
			limits, exists := computeContainer.Resources.Limits[kubev1.ResourceCPU]
			Expect(exists).To(BeTrue(), "expected CPU limit not set on the compute container")
			Expect(limits.String()).To(Equal("1"))
		})
	})

	Context("with automatic resource limits FG enabled", decorators.AutoResourceLimitsGate, func() {

		When("there is no ResourceQuota with memory and cpu limits associated with the creation namespace", func() {
			It("should not automatically set memory limits in the virt-launcher pod", func() {
				vmi := libvmi.NewCirros()
				By("Creating a running VMI")
				runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

				By("Ensuring no memory and cpu limits are set")
				readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())
				computeContainer := tests.GetComputeContainerOfPod(readyPod)
				_, exists := computeContainer.Resources.Limits[kubev1.ResourceMemory]
				Expect(exists).To(BeFalse(), "Memory limits set on the compute container when none was expected")
				_, exists = computeContainer.Resources.Limits[kubev1.ResourceCPU]
				Expect(exists).To(BeFalse(), "CPU limits set on the compute container when none was expected")
			})
		})

		When("a ResourceQuota with memory and cpu limits is associated to the creation namespace", func() {
			var (
				vmi                       *virtv1.VirtualMachineInstance
				expectedLauncherMemLimits *resource.Quantity
				expectedLauncherCPULimits resource.Quantity
				vmiRequest                resource.Quantity
			)

			BeforeEach(func() {
				vmiRequest = resource.MustParse("256Mi")
				delta := resource.MustParse("100Mi")
				vmi = libvmi.NewCirros(
					libvmi.WithResourceMemory(vmiRequest.String()),
					libvmi.WithCPUCount(1, 1, 1),
				)
				vmiPodRequest := services.GetMemoryOverhead(vmi, runtime.GOARCH, nil)
				vmiPodRequest.Add(vmiRequest)
				value := int64(float64(vmiPodRequest.Value()) * services.DefaultMemoryLimitOverheadRatio)

				expectedLauncherMemLimits = resource.NewQuantity(value, vmiPodRequest.Format)
				expectedLauncherCPULimits = resource.MustParse("1")

				// Add a delta to not saturate the rq
				rqLimit := expectedLauncherMemLimits.DeepCopy()
				rqLimit.Add(delta)
				By("Creating a Resource Quota with memory limits")
				rq := &kubev1.ResourceQuota{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:    testsuite.GetTestNamespace(nil),
						GenerateName: "test-quota",
					},
					Spec: k8sv1.ResourceQuotaSpec{
						Hard: kubev1.ResourceList{
							k8sv1.ResourceLimitsMemory: resource.MustParse(rqLimit.String()),
							k8sv1.ResourceLimitsCPU:    resource.MustParse("1500m"),
						},
					},
				}
				_, err := virtClient.CoreV1().ResourceQuotas(testsuite.GetTestNamespace(nil)).Create(context.Background(), rq, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should set a memory limit in the virt-launcher pod", func() {
				By("Starting the VMI")
				runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)

				By("Ensuring the memory and cpu limits are set to the correct values")
				readyPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())
				computeContainer := tests.GetComputeContainerOfPod(readyPod)
				memLimits, exists := computeContainer.Resources.Limits[kubev1.ResourceMemory]
				Expect(exists).To(BeTrue(), "expected memory limits set on the compute container")
				Expect(memLimits.Value()).To(BeEquivalentTo(expectedLauncherMemLimits.Value()))
				cpuLimits, exists := computeContainer.Resources.Limits[kubev1.ResourceCPU]
				Expect(exists).To(BeTrue(), "expected cpu limits set on the compute container")
				Expect(cpuLimits.Value()).To(BeEquivalentTo(expectedLauncherCPULimits.Value()))
			})
		})
	})
})
