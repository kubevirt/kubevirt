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

package network

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = SIGDescribe("Primary Pod Network", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).NotTo(HaveOccurred(), "Should successfully initialize an API client")
	})

	Describe("Status", func() {
		AssertReportedIP := func(vmi *v1.VirtualMachineInstance) {
			By("Getting pod of the VMI")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)

			By("Making sure IP reported on the VMI matches the one on the pod")
			Expect(vmi.Status.Interfaces[0].IP).To(Equal(vmiPod.Status.PodIP))
			Expect(vmi.Status.Interfaces[0].IPs).NotTo(BeEmpty())
			Expect(vmi.Status.Interfaces[0].IPs[0]).To(Equal(vmiPod.Status.PodIP))
		}

		Context("VMI connected to the pod network using the default (implicit) binding", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = setupVMI(virtClient, vmiWithDefaultBinding())
			})

			AfterEach(func() {
				cleanupVMI(virtClient, vmi)
			})

			It("should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
		})

		Context("VMI connected to the pod network using bridge binding", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = setupVMI(virtClient, vmiWithBridgeBinding())
			})

			AfterEach(func() {
				cleanupVMI(virtClient, vmi)
			})

			It("should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
		})

		Context("VMI connected to the pod network using masquerade binding", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = setupVMI(virtClient, vmiWithMasqueradeBinding())
			})

			AfterEach(func() {
				cleanupVMI(virtClient, vmi)
			})

			It("should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
		})
	})
})

func setupVMI(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
	By("Creating the VMI")
	var err error
	vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
	Expect(err).NotTo(HaveOccurred(), "VMI should be successfully created")

	By("Waiting until the VMI gets ready")
	vmi = tests.WaitUntilVMIReady(vmi, tests.LoggedInAlpineExpecter)

	return vmi
}

func cleanupVMI(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) {
	if vmi != nil {
		By("Deleting the VMI")
		Expect(virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.GetName(), &metav1.DeleteOptions{})).To(Succeed())

		By("Waiting for the VMI to be gone")
		Eventually(func() error {
			_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.GetName(), &metav1.GetOptions{})
			return err
		}, 2*time.Minute, time.Second).Should(SatisfyAll(HaveOccurred(), WithTransform(errors.IsNotFound, BeTrue())), "The VMI should be gone within the given timeout")
	}
}

func vmiWithDefaultBinding() *v1.VirtualMachineInstance {
	vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
	vmi.Spec.Domain.Devices.Interfaces = nil
	vmi.Spec.Networks = nil
	return vmi
}

func vmiWithBridgeBinding() *v1.VirtualMachineInstance {
	vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	return vmi
}

func vmiWithMasqueradeBinding() *v1.VirtualMachineInstance {
	vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	return vmi
}
