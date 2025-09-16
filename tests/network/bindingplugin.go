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
 * Copyright The KubeVirt Authors.
 *
 */

package network

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("network binding plugin", Serial, decorators.NetCustomBindingPlugins, func() {
	Context("with CNI and Sidecar", func() {
		const passtNetAttDefName = "netbindingpasst"

		BeforeEach(func() {
			const passtBindingName = "passt"
			passtSidecarImage := libregistry.GetUtilityImageFromRegistry("network-passt-binding")

			err := config.RegisterKubevirtConfigChange(
				config.WithNetBindingPlugin(passtBindingName, v1.InterfaceBindingPlugin{
					SidecarImage:                passtSidecarImage,
					NetworkAttachmentDefinition: passtNetAttDefName,
				}),
			)
			Expect(err).NotTo(HaveOccurred())
		})

		BeforeEach(func() {
			netAttachDef := libnet.NewPasstNetAttachDef(passtNetAttDefName)
			_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil), netAttachDef)
			Expect(err).NotTo(HaveOccurred())
		})

		It("can be used by a VMI as its primary network", func() {
			const (
				macAddress = "02:00:00:00:00:02"
			)
			passtIface := libvmi.InterfaceWithPasstBindingPlugin()
			passtIface.MacAddress = macAddress
			vmi := libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(passtIface),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			var err error
			namespace := testsuite.GetTestNamespace(nil)
			vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi = libwait.WaitUntilVMIReady(
				vmi,
				console.LoginToAlpine,
				libwait.WithFailOnWarnings(false),
				libwait.WithTimeout(180),
			)

			Expect(vmi.Status.Interfaces).To(HaveLen(1))
			Expect(vmi.Status.Interfaces[0].IPs).NotTo(BeEmpty())
			Expect(vmi.Status.Interfaces[0].IP).NotTo(BeEmpty())
			Expect(vmi.Status.Interfaces[0].MAC).To(Equal(macAddress))
		})
	})

	Context("with domain attachment tap type", func() {
		const (
			macvtapBindingName = "macvtap"
			macvtapLowerDevice = "eth0"
			macvtapNetworkName = "net1"
		)

		BeforeEach(func() {
			netAttachDef := libnet.NewMacvtapNetAttachDef(macvtapNetworkName, macvtapLowerDevice)
			_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil), netAttachDef)
			Expect(err).NotTo(HaveOccurred())
		})

		BeforeEach(func() {
			err := config.RegisterKubevirtConfigChange(
				config.WithNetBindingPlugin(macvtapBindingName, v1.InterfaceBindingPlugin{
					DomainAttachmentType: v1.Tap,
				}),
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("can run a virtual machine with one macvtap interface", func() {
			var vmi *v1.VirtualMachineInstance
			var chosenMAC string

			chosenMACHW, err := libnet.GenerateRandomMac()
			Expect(err).ToNot(HaveOccurred())
			chosenMAC = chosenMACHW.String()

			ifaceName := "macvtapIface"
			macvtapIface := libvmi.InterfaceWithBindingPlugin(
				ifaceName, v1.PluginBinding{Name: macvtapBindingName},
			)
			vmi = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(
					*libvmi.InterfaceWithMac(&macvtapIface, chosenMAC)),
				libvmi.WithNetwork(libvmi.MultusNetwork(ifaceName, macvtapNetworkName)))

			vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmi = libwait.WaitUntilVMIReady(
				vmi,
				console.LoginToAlpine)

			Expect(vmi.Status.Interfaces).To(HaveLen(1), "should have a single interface")
			Expect(vmi.Status.Interfaces[0].MAC).To(Equal(chosenMAC), "the expected MAC address should be set in the VMI")
		})
	})

	Context("with domain attachment managedTap type", func() {
		const (
			bindingName = "managed-tap"
			networkName = "default"
		)

		BeforeEach(func() {
			err := config.RegisterKubevirtConfigChange(
				config.WithNetBindingPlugin(bindingName, v1.InterfaceBindingPlugin{
					DomainAttachmentType: v1.ManagedTap,
				}),
			)
			Expect(err).NotTo(HaveOccurred())
		})

		It("can run a virtual machine with one primary managed-tap interface", func() {
			var vmi *v1.VirtualMachineInstance

			primaryIface := libvmi.InterfaceWithBindingPlugin(
				networkName, v1.PluginBinding{Name: bindingName},
			)
			vmi = libvmifact.NewAlpineWithTestTooling(
				libvmi.WithInterface(primaryIface),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine, libwait.WithTimeout(30))

			Expect(vmi.Status.Interfaces).To(HaveLen(1))
			Expect(vmi.Status.Interfaces[0].Name).To(Equal(primaryIface.Name))
		})

		It("can establish communication between two VMs", func() {
			const (
				guestIfaceName = "eth0"
				serverIPAddr   = "10.1.1.102"
				serverCIDR     = serverIPAddr + "/24"
				clientCIDR     = "10.1.1.101/24"
			)
			nodeList := libnode.GetAllSchedulableNodes(kubevirt.Client())
			Expect(nodeList.Items).NotTo(BeEmpty(), "schedulable kubernetes nodes must be present")
			nodeName := nodeList.Items[0].Name

			const linuxBridgeNADName = "bridge0"
			namespace := testsuite.GetTestNamespace(nil)
			netAttachDef := libnet.NewBridgeNetAttachDef(
				linuxBridgeNADName,
				"br10",
				libnet.WithIPAM(map[string]string{
					"type":   "host-local",
					"subnet": "10.1.1.0/24",
				}),
			)
			_, err := libnet.CreateNetAttachDef(context.Background(), namespace, netAttachDef)
			Expect(err).ToNot(HaveOccurred())

			primaryIface := libvmi.InterfaceWithBindingPlugin(
				"mynet1", v1.PluginBinding{Name: bindingName},
			)
			primaryNetwork := v1.Network{
				Name: "mynet1",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{
						NetworkName: fmt.Sprintf("%s/%s", namespace, linuxBridgeNADName),
						Default:     true,
					},
				},
			}
			primaryIface.MacAddress = "de:ad:00:00:be:af"
			opts := []libvmi.Option{
				libvmi.WithInterface(primaryIface),
				libvmi.WithNetwork(&primaryNetwork),
				libvmi.WithNodeAffinityFor(nodeName),
			}
			serverVMI := libvmifact.NewAlpineWithTestTooling(opts...)

			primaryIface.MacAddress = "de:ad:00:00:be:aa"
			opts = []libvmi.Option{
				libvmi.WithInterface(primaryIface),
				libvmi.WithNetwork(&primaryNetwork),
				libvmi.WithNodeAffinityFor(nodeName),
			}
			clientVMI := libvmifact.NewAlpineWithTestTooling(opts...)

			ns := testsuite.GetTestNamespace(nil)
			serverVMI, err = kubevirt.Client().VirtualMachineInstance(ns).Create(context.Background(), serverVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			clientVMI, err = kubevirt.Client().VirtualMachineInstance(ns).Create(context.Background(), clientVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			serverVMI = libwait.WaitUntilVMIReady(serverVMI, console.LoginToAlpine)
			clientVMI = libwait.WaitUntilVMIReady(clientVMI, console.LoginToAlpine)

			Expect(libnet.AddIPAddress(serverVMI, guestIfaceName, serverCIDR)).To(Succeed())
			Expect(libnet.AddIPAddress(clientVMI, guestIfaceName, clientCIDR)).To(Succeed())

			Expect(libnet.PingFromVMConsole(clientVMI, serverIPAddr)).To(Succeed())
		})
	})
}))
