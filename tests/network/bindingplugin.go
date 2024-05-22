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
 * Copyright 2023 Red Hat, Inc.
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

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkvconfig"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/kubevirt/pkg/libvmi"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = SIGDescribe("[Serial]network binding plugin", Serial, decorators.NetCustomBindingPlugins, func() {

	BeforeEach(func() {
		tests.EnableFeatureGate(virtconfig.NetworkBindingPlugingsGate)
	})

	Context("with CNI and Sidecar", func() {
		BeforeEach(func() {
			const passtBindingName = "passt"
			passtSidecarImage := libregistry.GetUtilityImageFromRegistry("network-passt-binding")

			err := libkvconfig.WithNetBindingPlugin(passtBindingName, v1.InterfaceBindingPlugin{
				SidecarImage:                passtSidecarImage,
				NetworkAttachmentDefinition: libnet.PasstNetAttDef,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		BeforeEach(func() {
			Expect(libnet.CreatePasstNetworkAttachmentDefinition(testsuite.GetTestNamespace(nil))).To(Succeed())
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

	Context("with domain attachment type", func() {
		const (
			macvtapNetworkConfNAD = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s", "annotations": {"k8s.v1.cni.cncf.io/resourceName": "macvtap.network.kubevirt.io/%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"%s\", \"type\": \"macvtap\"}"}}`
			macvtapBindingName    = "macvtap"
			macvtapLowerDevice    = "eth0"
			macvtapNetworkName    = "net1"
		)

		BeforeEach(func() {
			macvtapNad := fmt.Sprintf(macvtapNetworkConfNAD, macvtapNetworkName, testsuite.GetTestNamespace(nil), macvtapLowerDevice, macvtapNetworkName)
			namespace := testsuite.GetTestNamespace(nil)
			Expect(libnet.CreateNetworkAttachmentDefinition(macvtapNetworkName, namespace, macvtapNad)).
				To(Succeed(), "A macvtap network named %s should be provisioned", macvtapNetworkName)
		})

		BeforeEach(func() {
			err := libkvconfig.WithNetBindingPlugin(macvtapBindingName, v1.InterfaceBindingPlugin{DomainAttachmentType: v1.Tap})
			Expect(err).NotTo(HaveOccurred())
		})

		It("can run a virtual machine with one macvtap interface", func() {
			var vmi *v1.VirtualMachineInstance
			var chosenMAC string

			chosenMACHW, err := GenerateRandomMac()
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
})
