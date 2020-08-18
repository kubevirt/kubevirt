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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"

	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = SIGDescribe("Primary Pod Network", func() {
	Describe("Status", func() {
		AssertReportedIP := func(vmi *v1.VirtualMachineInstance) {
			By("Getting pod of the VMI")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)

			By("Making sure IP/s reported on the VMI matches the ones on the pod")
			Expect(ValidateVMIandPodIPMatch(vmi, vmiPod)).To(Succeed(), "Should have matching IP/s between pod and vmi")
		}

		Context("VMI connected to the pod network using the default (implicit) binding", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = libvmi.SetupVMI(vmiWithImplicitBinding())
			})

			AfterEach(func() {
				libvmi.CleanupVMI(vmi)
			})

			It("should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
		})

		Context("VMI connected to the pod network using bridge binding", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = libvmi.SetupVMI(vmiWithBridgeBinding())
			})

			AfterEach(func() {
				libvmi.CleanupVMI(vmi)
			})

			It("should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
		})

		Context("VMI connected to the pod network using masquerade binding", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = libvmi.SetupVMI(vmiWithMasqueradeBinding())
			})

			AfterEach(func() {
				libvmi.CleanupVMI(vmi)
			})

			It("should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
		})
	})
})

func vmiWithImplicitBinding() *v1.VirtualMachineInstance {
	vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
	vmi.Spec.Domain.Devices.Interfaces = nil
	vmi.Spec.Networks = nil
	return vmi
}

func vmiWithBridgeBinding() *v1.VirtualMachineInstance {
	vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	return vmi
}

func vmiWithMasqueradeBinding() *v1.VirtualMachineInstance {
	vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	return vmi
}
