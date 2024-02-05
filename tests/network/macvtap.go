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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package network

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-config/deprecation"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = SIGDescribe("Macvtap", decorators.Macvtap, Serial, func() {
	const (
		macvtapLowerDevice = "eth0"
		macvtapNetworkName = "net1"
	)

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	BeforeEach(func() {
		tests.EnableFeatureGate(deprecation.MacvtapGate)
	})

	BeforeEach(func() {
		ns := testsuite.GetTestNamespace(nil)
		Expect(libnet.CreateMacvtapNetworkAttachmentDefinition(ns, macvtapNetworkName, macvtapLowerDevice)).To(Succeed(),
			"A macvtap network named %s should be provisioned", macvtapNetworkName)
	})

	Context("a virtual machine with one macvtap interface, with a custom MAC address", func() {
		var serverVMI *v1.VirtualMachineInstance
		var chosenMAC string
		var nodeList *k8sv1.NodeList
		var nodeName string

		BeforeEach(func() {
			nodeList = libnode.GetAllSchedulableNodes(virtClient)
			Expect(nodeList.Items).NotTo(BeEmpty(), "schedulable kubernetes nodes must be present")
			nodeName = nodeList.Items[0].Name
			chosenMACHW, err := GenerateRandomMac()
			Expect(err).ToNot(HaveOccurred())
			chosenMAC = chosenMACHW.String()

			const macvtapNetName = "test-macvtap"
			serverVMI := libvmi.NewAlpineWithTestTooling(
				libvmi.WithInterface(*libvmi.InterfaceWithMac(v1.DefaultMacvtapNetworkInterface(macvtapNetName), chosenMAC)),
				libvmi.WithNetwork(libvmi.MultusNetwork(macvtapNetName, macvtapNetworkName)),
				libvmi.WithNodeAffinityFor(nodeName),
			)
			serverVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(serverVMI)).Create(context.Background(), serverVMI)
			Expect(err).ToNot(HaveOccurred())
			serverVMI = libwait.WaitUntilVMIReady(serverVMI, console.LoginToAlpine)
		})

		It("should have the specified MAC address reported back via the API", func() {
			Expect(serverVMI.Status.Interfaces).To(HaveLen(1), "should have a single interface")
			Expect(serverVMI.Status.Interfaces[0].MAC).To(Equal(chosenMAC), "the expected MAC address should be set in the VMI")
		})
	})
})
