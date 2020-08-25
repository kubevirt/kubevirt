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
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = SIGDescribe("[rfe_id:694][crit:medium][vendor:cnv-qe@redhat.com][level:component]Primary Pod Network", func() {
	Describe("Status", func() {
		AssertReportedIP := func(vmi *v1.VirtualMachineInstance) {
			By("Getting pod of the VMI")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.GetNamespace())

			By("Making sure IP/s reported on the VMI matches the ones on the pod")
			Expect(ValidateVMIandPodIPMatch(vmi, vmiPod)).To(Succeed(), "Should have matching IP/s between pod and vmi")
		}

		Context("VMI connected to the pod network using the default (implicit) binding", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = libvmi.SetupVMI(vmiWithImplicitBinding(), tests.LoggedInCirrosExpecter)
			})

			AfterEach(func() {
				libvmi.CleanupVMI(vmi)
			})

			It("should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
		})

		Context("VMI connected to the pod network using bridge binding", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = libvmi.SetupVMI(vmiWithBridgeBinding(), tests.LoggedInCirrosExpecter)
			})

			AfterEach(func() {
				libvmi.CleanupVMI(vmi)
			})

			It("should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
		})

		Context("VMI connected to the pod network using masquerade binding", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = libvmi.SetupVMI(vmiWithMasqueradeBinding(), tests.LoggedInCirrosExpecter)
			})

			AfterEach(func() {
				libvmi.CleanupVMI(vmi)
			})

			It("should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
		})
	})

	Describe("Connectivity", func() {
		const servicePort = 1500
		const ipv4HeaderSize = 20
		const icmpHeaderSize = 8
		const ipv4IcmpHeaderSize = ipv4HeaderSize + icmpHeaderSize

		AssertConnectivity := func(outboundVMI, inboundVMI *v1.VirtualMachineInstance) {
			inboundVMIAddress := inboundVMI.Status.Interfaces[0].IP

			By("Checking connectivity via ICMP")
			Expect(tests.PingFromVMConsole(outboundVMI, inboundVMIAddress)).To(Succeed())

			By("Checking connectivity via ICMP with MTU-sized frames")
			mtu := getPodNetworkMTU(outboundVMI)
			Expect(tests.PingFromVMConsole(outboundVMI, inboundVMIAddress, "-c5", "-w5", "-s", strconv.Itoa(mtu-ipv4IcmpHeaderSize))).To(Succeed())

			By("Checking connectivity via HTTP")
			Expect(tests.PingHTTPFromVMConsole(outboundVMI, inboundVMIAddress, servicePort)).To(Succeed())
		}

		Context("2 VMIs connected to the pod network using the bridge binding", func() {
			vmis := libvmi.NewVMIPool()

			BeforeEach(func() {
				vmis["outbound"] = vmiWithBridgeBinding()
				vmis["inbound"] = vmiWithBridgeBinding()
				vmis.Setup(tests.LoggedInCirrosExpecter)
				tests.HTTPServer.Start(vmis["inbound"], servicePort)
			})

			AfterEach(func() {
				vmis.Cleanup()
			})

			It("[test_id:1540]should be able to reach from one to another", func() { AssertConnectivity(vmis["outbound"], vmis["inbound"]) })
		})

		Context("2 VMIs connected to the pod network using the masquerade binding", func() {
			vmis := libvmi.NewVMIPool()

			BeforeEach(func() {
				vmis["outbound"] = vmiWithMasqueradeBinding()
				vmis["inbound"] = vmiWithMasqueradeBinding()
				vmis.Setup(tests.LoggedInCirrosExpecter)
				tests.HTTPServer.Start(vmis["inbound"], servicePort)
			})

			AfterEach(func() {
				vmis.Cleanup()
			})

			It("[test_id:1539]should be able to reach from one to another", func() { AssertConnectivity(vmis["outbound"], vmis["inbound"]) })
		})
	})
})

func vmiWithImplicitBinding() *v1.VirtualMachineInstance {
	return libvmi.NewCirros()
}

func vmiWithBridgeBinding() *v1.VirtualMachineInstance {
	return libvmi.NewCirros(
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding()),
	)
}

func vmiWithMasqueradeBinding() *v1.VirtualMachineInstance {
	return libvmi.NewCirros(
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
	)
}

func getPodNetworkMTU(vmi *v1.VirtualMachineInstance) int {
	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Should initialize KubeVirt client")

	vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.GetNamespace())
	output, err := tests.ExecuteCommandOnPod(
		virtClient,
		vmiPod,
		"compute",
		[]string{"cat", "/sys/class/net/k6t-eth0/mtu"},
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Should be able to read virt-launcher MTU")

	mtu, err := strconv.Atoi(strings.TrimSuffix(output, "\n"))
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Obtained MTU should be convertable to int")

	return mtu
}
