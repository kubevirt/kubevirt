/*
 * This file is part of the kubevirt project
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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"fmt"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cd "kubevirt.io/kubevirt/tests/containerdisk"

	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tools/vms-generator/utils"
)

var _ = Describe("[sig-compute]VMI with external kernel boot", func() {

	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		tests.BeforeTestCleanup()
	})

	Context("with external alpine-based kernel & initrd images", func() {
		It("[test_id:7748]ensure successful boot", func() {
			vmi := utils.GetVMIKernelBoot()
			obj, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(obj)
		})

		It("ensure successful boot and deletion when VMI has a disk defined", func() {
			By("Creating VMI with disk and kernel boot")
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")
			utils.AddKernelBootToVMI(vmi)

			Expect(vmi.Spec.Volumes).ToNot(BeEmpty())
			Expect(vmi.Spec.Domain.Devices.Disks).ToNot(BeEmpty())

			By("Ensuring VMI can boot")
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)

			By("Fetching virt-launcher pod")
			virtLauncherPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)

			By("Ensuring VMI is deleted")
			err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(vmi.Name, &v1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() (isVmiDeleted bool) {
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &v1.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				Expect(err).ToNot(HaveOccurred())
				return false
			}, 60*time.Second, 3*time.Second).Should(BeTrue(), "VMI Should be successfully deleted")

			By("Ensuring virt-launcher is deleted")
			Eventually(func() (isVmiDeleted bool) {
				_, err = virtClient.CoreV1().Pods(virtLauncherPod.Namespace).Get(context.Background(), virtLauncherPod.Name, v1.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				Expect(err).ToNot(HaveOccurred())
				return false
			}, 60*time.Second, 3*time.Second).Should(BeTrue(), fmt.Sprintf("virt-launcher pod (%s) Should be successfully deleted", virtLauncherPod.Name))
		})
	})

	Context("with illegal definition ensure rejection of", func() {

		It("[test_id:7750]VMI defined without an image", func() {
			vmi := utils.GetVMIKernelBoot()
			kernelBoot := vmi.Spec.Domain.Firmware.KernelBoot
			kernelBoot.Container.Image = ""
			_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("denied the request: spec.domain.firmware.kernelBoot.container must be defined with an image"))
		})

		It("[test_id:7751]VMI defined with image but without initrd & kernel paths", func() {
			vmi := utils.GetVMIKernelBoot()
			kernelBoot := vmi.Spec.Domain.Firmware.KernelBoot
			kernelBoot.Container.KernelPath = ""
			kernelBoot.Container.InitrdPath = ""
			_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("denied the request: spec.domain.firmware.kernelBoot.container must be defined with at least one of the following: kernelPath, initrdPath"))
		})
	})
})
