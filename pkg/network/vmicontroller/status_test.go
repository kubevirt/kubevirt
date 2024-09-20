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
 * Copyright 2024 The KubeVirt Authors.
 *
 */

package vmicontroller_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/network/vmicontroller"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("Status Update", func() {
	const (
		testNamespace = "default"

		multusNetworksAnnotation        = `[{"name":"meganet","namespace":"default","interface":"pod7e0055a6880"}]`
		multusOrdinalNetworksAnnotation = `[{"name":"meganet","namespace":"default","interface":"net1"}]`

		multusNetworkStatusWithPrimaryNet = `[{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}}]`

		multusNetworkStatusWithPrimaryAndSecondaryNets = `[` +
			`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
			`{"name":"meganet","interface":"pod7e0055a6880","mac":"8a:37:d9:e7:0f:18","dns":{}}` +
			`]`

		multusNetworkStatusWithPrimaryAndOrdinalSecondaryNets = `[` +
			`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
			`{"name":"meganet","interface":"net1","mac":"8a:37:d9:e7:0f:18","dns":{}}` +
			`]`

		networkName                     = "iface1"
		networkAttachmentDefinitionName = "meganet"
	)

	DescribeTable("Shouldn't generate interface status for a VMI without interfaces", func(podAnnotations map[string]string) {
		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithAutoAttachPodInterface(false),
		)

		err := vmicontroller.UpdateStatus(vmi, newPodFromVMI(vmi, podAnnotations))
		Expect(err).NotTo(HaveOccurred())

		Expect(vmi.Status.Interfaces).To(BeEmpty())
	},
		Entry("When the Multus network-status annotation does not exist", nil),
		Entry("When the Multus network-status annotation exists",
			map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet},
		),
	)

	It("Shouldn't generate interface status for secondary interface (not matched on status) when Multus network status is absent", func() {
		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(networkName, networkAttachmentDefinitionName)),
		)

		podAnnotations := map[string]string{networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation}

		Expect(vmicontroller.UpdateStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		Expect(vmi.Status.Interfaces).To(BeEmpty())
	})

	DescribeTable("Should generate interface status for secondary interface (not matched on status) when Multus network status is reported",
		func(podAnnotations map[string]string) {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(networkName, networkAttachmentDefinitionName)),
			)

			Expect(vmicontroller.UpdateStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

			expectedInterfaces := []v1.VirtualMachineInstanceNetworkInterface{
				{Name: networkName, InfoSource: vmispec.InfoSourceMultusStatus},
			}

			Expect(vmi.Status.Interfaces).To(Equal(expectedInterfaces))
		},
		Entry("When using hashed naming scheme",
			map[string]string{
				networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation,
				networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndSecondaryNets,
			},
		),
		Entry("When using ordinal naming scheme",
			map[string]string{
				networkv1.NetworkAttachmentAnnot: multusOrdinalNetworksAnnotation,
				networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndOrdinalSecondaryNets,
			},
		),
	)

	It("Should keep the Multus info source when VMI.status has an interface and it is reported by Multus network-status", func() {
		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(networkName, networkAttachmentDefinitionName)),
		)

		existingInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: networkName, InfoSource: vmispec.InfoSourceMultusStatus},
		}

		vmi.Status = libvmistatus.New(WithInterfacesStatus(existingInterfacesStatus))

		podAnnotations := map[string]string{
			networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation,
			networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndSecondaryNets,
		}

		Expect(vmicontroller.UpdateStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: networkName, InfoSource: vmispec.InfoSourceMultusStatus},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	})

	It("Should remove the Multus info source when VMI.status has an interface but it is not reported by Multus network-status", func() {
		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(networkName, networkAttachmentDefinitionName)),
		)

		existingInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: networkName, InfoSource: vmispec.InfoSourceMultusStatus},
		}

		vmi.Status = libvmistatus.New(WithInterfacesStatus(existingInterfacesStatus))

		podAnnotations := map[string]string{
			networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet,
		}

		Expect(vmicontroller.UpdateStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: networkName},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	})

	It("Should keep existing interface status when another info source is reported and Multus network-status is missing", func() {
		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(networkName, networkAttachmentDefinitionName)),
		)

		existingInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: networkName, InfoSource: vmispec.InfoSourceGuestAgent},
		}

		vmi.Status = libvmistatus.New(WithInterfacesStatus(existingInterfacesStatus))

		podAnnotations := map[string]string{
			networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet,
		}

		Expect(vmicontroller.UpdateStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: networkName, InfoSource: vmispec.InfoSourceGuestAgent},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	})

	It("Should keep existing interface status when info source is empty and Multus network-status is missing", func() {
		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(networkName, networkAttachmentDefinitionName)),
		)

		existingInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: networkName},
		}

		vmi.Status = libvmistatus.New(WithInterfacesStatus(existingInterfacesStatus))

		podAnnotations := map[string]string{
			networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet,
		}

		Expect(vmicontroller.UpdateStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: networkName},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	})
})

func newPodFromVMI(vmi *v1.VirtualMachineInstance, annotations map[string]string) *k8scorev1.Pod {
	return &k8scorev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "virt-launcher-" + vmi.Name,
			Namespace:   vmi.Namespace,
			Annotations: annotations,
		},
	}
}

func WithInterfacesStatus(interfaces []v1.VirtualMachineInstanceNetworkInterface) libvmistatus.Option {
	return func(vmiStatus *v1.VirtualMachineInstanceStatus) {
		vmiStatus.Interfaces = interfaces
	}
}
