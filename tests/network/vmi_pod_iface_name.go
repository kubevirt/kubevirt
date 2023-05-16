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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = SIGDescribe("pod interface name", func() {

	Context("VMI with default pod network and two secondary networks", func() {
		var testVMI *kvv1.VirtualMachineInstance

		const (
			testNetAttachDefName = "blue-net"
			blueNetworkName      = "blue-network"
			redNetworkName       = "red-network"
		)

		BeforeEach(func() {
			By("Create NetworkAttachmentDefinition")
			Expect(libnet.CreateNAD(util.NamespaceTestDefault, testNetAttachDefName)).To(Succeed())

			testVMI = libvmi.NewAlpineWithTestTooling(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(kvv1.DefaultPodNetwork()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(blueNetworkName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(blueNetworkName, testNetAttachDefName)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(redNetworkName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(redNetworkName, testNetAttachDefName)),
			)

			By("Starting VMI")
			var err error
			testVMI, err = kubevirt.Client().VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), testVMI)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for VMI status pod iface name update")
			Eventually(func() bool {
				var err error
				testVMI, err = kubevirt.Client().VirtualMachineInstance(testVMI.Namespace).Get(context.Background(), testVMI.Name, &metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				return podIfaceNameExistInStatus(testVMI)
			}, 10*time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("should have secondary networks pod interface name in status", func() {
			vmiPodIfaceNames := map[string]string{}
			for _, iface := range testVMI.Status.Interfaces {
				vmiPodIfaceNames[iface.Name] = iface.PodInterfaceName
			}

			Expect(vmiPodIfaceNames).To(HaveKeyWithValue(blueNetworkName, "podd05c0e2a83b"))
			Expect(vmiPodIfaceNames).To(HaveKeyWithValue(redNetworkName, "pod3c12b9d89fa"))
		})
	})
})

func podIfaceNameExistInStatus(vmi *kvv1.VirtualMachineInstance) bool {
	for i := range vmi.Status.Interfaces {
		if vmi.Status.Interfaces[i].PodInterfaceName != "" {
			return true
		}
	}
	return false
}
