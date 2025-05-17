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
 * Copyright The KubeVirt Authors.
 *
 */

package controllers_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/network/controllers"
	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("Status Update", func() {
	const (
		testNamespace = "default"

		multusNetworksAnnotation        = `[{"name":"meganet","namespace":"default","interface":"pod7e0055a6880"}]`
		multusOrdinalNetworksAnnotation = `[{"name":"meganet","namespace":"default","interface":"net1"}]`

		multusNetworkStatusWithPrimaryNet = `[{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}}]`

		multusNetworkStatusWithPrimaryNetAndIfaceName = `[` +
			`{"name":"k8s-pod-network","interface":"eth0","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}}` +
			`]`

		multusNetworkStatusWithAlternativePrimaryNet = `[` +
			`{"name":"alternativeNAD", "ips":["10.244.196.146", "fd10:244::c491"], "default":true, "dns":{}}` +
			`]`

		multusNetworkStatusWithAlternativePrimaryNetAndIfaceName = `[` +
			`{"name":"alternativeNAD", "interface":"eth0", "ips":["10.244.196.146", "fd10:244::c491"], "default":true,"dns":{}}` +
			`]`

		multusNetworkStatusWithCustomPrimaryNet = `[` +
			`{"name":"k8s-pod-network", "interface":"eth0", "ips":["10.244.196.146", "fd10:244::c491"], "default":false,"dns":{}}, ` +
			`{"name":"cluster-network", "interface":"custom-iface", "ips":["10.128.0.4"], "mac":"0a:58:0a:80:00:04", "default":true, "dns":{}}` +
			`]`

		multusNetworkStatusWithCustomPrimaryAndSecondaryNets = `[` +
			`{"name":"k8s-pod-network", "interface":"eth0", "ips":["10.244.196.146", "fd10:244::c491"], "default":false,"dns":{}}, ` +
			`{"name":"cluster-network", "interface":"custom-iface", "ips":["10.128.0.4"], "mac":"0a:58:0a:80:00:04", "default":true, "dns":{}}, ` +
			`{"name":"meganet", "interface":"pod7e0055a6880", "mac":"8a:37:d9:e7:0f:18", "dns":{}}` +
			`]`

		multusNetworkStatusWithPrimaryAndSecondaryNets = `[` +
			`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
			`{"name":"meganet","interface":"pod7e0055a6880","mac":"8a:37:d9:e7:0f:18","dns":{}}` +
			`]`

		multusNetworkStatusWithPrimaryAndOrdinalSecondaryNets = `[` +
			`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
			`{"name":"meganet","interface":"net1","mac":"8a:37:d9:e7:0f:18","dns":{}}` +
			`]`

		defaultNetworkName = "default"

		secondaryNetworkName                     = "iface1"
		secondaryNetworkAttachmentDefinitionName = "meganet"

		alternativeNetworkName                     = "alternative"
		alternativeNetworkAttachmentDefinitionName = "alternativeNAD"

		customIfaceName = "custom-iface"
	)

	DescribeTable("Shouldn't generate interface status for a VMI without interfaces", func(podAnnotations map[string]string) {
		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithAutoAttachPodInterface(false),
		)

		Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())
		Expect(vmi.Status.Interfaces).To(BeEmpty())
	},
		Entry("When the Multus network-status annotation is absent", nil),
		Entry("When the Multus network-status annotation exists",
			map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet},
		),
	)

	DescribeTable("Should generate interface status for primary network (not matched on status)", func(podAnnotations map[string]string) {
		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: defaultNetworkName, PodInterfaceName: "eth0"},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	},
		Entry("When Multus network status is absent", map[string]string{}),
		Entry("When Multus network status does not report interface name",
			map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet},
		),
		Entry("When Multus network status reports interface name",
			map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNetAndIfaceName},
		),
	)

	DescribeTable("Should generate interface status for primary network (matched on status)", func(podAnnotations map[string]string) {
		existingInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: defaultNetworkName, PodInterfaceName: "", InfoSource: vmispec.InfoSourceDomainAndGA},
		}

		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmistatus.WithStatus(libvmistatus.New(WithInterfacesStatus(existingInterfacesStatus))),
		)

		Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: defaultNetworkName, PodInterfaceName: "eth0", InfoSource: vmispec.InfoSourceDomainAndGA},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	},
		Entry("When Multus network status is absent", map[string]string{}),
		Entry("When Multus network status does not report interface name",
			map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet},
		),
		Entry("When Multus network status reports interface name",
			map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNetAndIfaceName},
		),
	)

	It("Should report custom pod primary interface name when Multus network status reports it", func() {
		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		annotations := map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithCustomPrimaryNet}
		Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, annotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: defaultNetworkName, PodInterfaceName: customIfaceName},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	})

	DescribeTable("Should generate interface status for Multus default network (not matched on status)",
		func(podAnnotations map[string]string) {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(alternativeNetworkName)),
				libvmi.WithNetwork(&v1.Network{
					Name: alternativeNetworkName,
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: alternativeNetworkAttachmentDefinitionName,
							Default:     true,
						},
					},
				}),
			)

			Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, map[string]string{}))).To(Succeed())

			expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
				{Name: alternativeNetworkName, PodInterfaceName: "eth0"},
			}

			Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
		},
		Entry("when Multus network status is absent",
			map[string]string{multus.DefaultNetworkCNIAnnotation: alternativeNetworkAttachmentDefinitionName},
		),
		Entry("When Multus network status does not report interface name",
			map[string]string{
				multus.DefaultNetworkCNIAnnotation: alternativeNetworkAttachmentDefinitionName,
				networkv1.NetworkStatusAnnot:       multusNetworkStatusWithAlternativePrimaryNet,
			},
		),
		Entry("When Multus network status reports interface name",
			map[string]string{
				multus.DefaultNetworkCNIAnnotation: alternativeNetworkAttachmentDefinitionName,
				networkv1.NetworkStatusAnnot:       multusNetworkStatusWithAlternativePrimaryNetAndIfaceName,
			},
		),
	)

	DescribeTable("Should generate interface status for Multus default network (matched on status)", func(podAnnotations map[string]string) {
		existingInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: alternativeNetworkName, PodInterfaceName: "", InfoSource: vmispec.InfoSourceDomainAndGA},
		}

		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(alternativeNetworkName)),
			libvmi.WithNetwork(&v1.Network{
				Name: alternativeNetworkName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{
						NetworkName: alternativeNetworkAttachmentDefinitionName,
						Default:     true,
					},
				},
			}),
			libvmistatus.WithStatus(libvmistatus.New(WithInterfacesStatus(existingInterfacesStatus))),
		)

		Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: alternativeNetworkName, PodInterfaceName: "eth0", InfoSource: vmispec.InfoSourceDomainAndGA},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	},
		Entry("When Multus network status is absent",
			map[string]string{
				multus.DefaultNetworkCNIAnnotation: alternativeNetworkAttachmentDefinitionName,
			},
		),
		Entry("When Multus network status does not report interface name",
			map[string]string{
				multus.DefaultNetworkCNIAnnotation: alternativeNetworkAttachmentDefinitionName,
				networkv1.NetworkStatusAnnot:       multusNetworkStatusWithAlternativePrimaryNet,
			},
		),
		Entry("When Multus network status reports interface name",
			map[string]string{
				multus.DefaultNetworkCNIAnnotation: alternativeNetworkAttachmentDefinitionName,
				networkv1.NetworkStatusAnnot:       multusNetworkStatusWithAlternativePrimaryNetAndIfaceName,
			},
		),
	)

	It("Shouldn't generate interface status for secondary interface (not matched on status) when Multus network status is absent", func() {
		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, secondaryNetworkAttachmentDefinitionName)),
		)

		podAnnotations := map[string]string{networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation}

		Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		Expect(vmi.Status.Interfaces).To(BeEmpty())
	})

	DescribeTable("Should generate interface status for secondary interface (not matched on status) when Multus network status is reported",
		func(podAnnotations map[string]string, expectedInterfaces []v1.VirtualMachineInstanceNetworkInterface) {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, secondaryNetworkAttachmentDefinitionName)),
			)

			Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

			Expect(vmi.Status.Interfaces).To(Equal(expectedInterfaces))
		},
		Entry("When using hashed naming scheme",
			map[string]string{
				networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation,
				networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndSecondaryNets,
			},
			[]v1.VirtualMachineInstanceNetworkInterface{
				{Name: secondaryNetworkName, PodInterfaceName: "pod7e0055a6880", InfoSource: vmispec.InfoSourceMultusStatus},
			},
		),
		Entry("When using ordinal naming scheme",
			map[string]string{
				networkv1.NetworkAttachmentAnnot: multusOrdinalNetworksAnnotation,
				networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndOrdinalSecondaryNets,
			},
			[]v1.VirtualMachineInstanceNetworkInterface{
				{Name: secondaryNetworkName, PodInterfaceName: "net1", InfoSource: vmispec.InfoSourceMultusStatus},
			},
		),
	)

	DescribeTable("Interface status should be reported correctly for VMI with primary and secondary networks",
		func(podAnnotations map[string]string, expectedPrimaryInterfaceName string) {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, secondaryNetworkAttachmentDefinitionName)),
			)

			Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

			expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
				{Name: defaultNetworkName, PodInterfaceName: expectedPrimaryInterfaceName},
				{Name: secondaryNetworkName, PodInterfaceName: "pod7e0055a6880", InfoSource: vmispec.InfoSourceMultusStatus},
			}

			Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
		},
		Entry("With default primary pod interface name",
			map[string]string{
				networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation,
				networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndSecondaryNets,
			},
			"eth0",
		),
		Entry("With custom primary pod interface name",
			map[string]string{
				networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation,
				networkv1.NetworkStatusAnnot:     multusNetworkStatusWithCustomPrimaryAndSecondaryNets,
			},
			customIfaceName,
		),
	)

	It("Should add the primary interface status for an existing VMI with primary and secondary networks", func() {
		existingInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: secondaryNetworkName, PodInterfaceName: "", InfoSource: vmispec.InfoSourceMultusStatus},
		}

		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, secondaryNetworkAttachmentDefinitionName)),
			libvmistatus.WithStatus(libvmistatus.New(WithInterfacesStatus(existingInterfacesStatus))),
		)

		podAnnotations := map[string]string{
			networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation,
			networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndSecondaryNets,
		}
		Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: defaultNetworkName, PodInterfaceName: "eth0"},
			{Name: secondaryNetworkName, PodInterfaceName: "pod7e0055a6880", InfoSource: vmispec.InfoSourceMultusStatus},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	})

	It("Should keep the Multus info source when VMI.status has an interface and it is reported by Multus network-status", func() {
		existingInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: secondaryNetworkName, InfoSource: vmispec.InfoSourceMultusStatus},
		}

		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, secondaryNetworkAttachmentDefinitionName)),
			libvmistatus.WithStatus(libvmistatus.New(WithInterfacesStatus(existingInterfacesStatus))),
		)

		podAnnotations := map[string]string{
			networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation,
			networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndSecondaryNets,
		}

		Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: secondaryNetworkName, PodInterfaceName: "pod7e0055a6880", InfoSource: vmispec.InfoSourceMultusStatus},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	})

	It("Should remove the Multus info source when VMI.status has an interface but it is not reported by Multus network-status", func() {
		existingInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: secondaryNetworkName, PodInterfaceName: "pod7e0055a6880", InfoSource: vmispec.InfoSourceMultusStatus},
		}

		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, secondaryNetworkAttachmentDefinitionName)),
			libvmistatus.WithStatus(libvmistatus.New(WithInterfacesStatus(existingInterfacesStatus))),
		)

		podAnnotations := map[string]string{
			networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet,
		}

		Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: secondaryNetworkName, PodInterfaceName: "pod7e0055a6880"},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	})

	It("Should keep existing interface status when another info source is reported and Multus network-status is missing", func() {
		existingInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: secondaryNetworkName, InfoSource: vmispec.InfoSourceGuestAgent},
		}

		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, secondaryNetworkAttachmentDefinitionName)),
			libvmistatus.WithStatus(libvmistatus.New(WithInterfacesStatus(existingInterfacesStatus))),
		)

		podAnnotations := map[string]string{
			networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet,
		}

		Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: secondaryNetworkName, PodInterfaceName: "pod7e0055a6880", InfoSource: vmispec.InfoSourceGuestAgent},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	})

	It("Should keep existing interface status for an interface created within the guest", func() {
		existingInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: "", InfoSource: vmispec.InfoSourceGuestAgent, IP: "192.168.50.10"},
		}

		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, secondaryNetworkAttachmentDefinitionName)),
			libvmistatus.WithStatus(libvmistatus.New(WithInterfacesStatus(existingInterfacesStatus))),
		)

		podAnnotations := map[string]string{
			networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation,
			networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndSecondaryNets,
		}

		Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: secondaryNetworkName, PodInterfaceName: "pod7e0055a6880", InfoSource: vmispec.InfoSourceMultusStatus},
			{Name: "", InfoSource: vmispec.InfoSourceGuestAgent, IP: "192.168.50.10"},
		}

		Expect(vmi.Status.Interfaces).To(Equal(expectedInterfacesStatus))
	})

	It("Should keep existing interface status when info source is empty and Multus network-status is missing", func() {
		existingInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: secondaryNetworkName},
		}

		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetworkName)),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetworkName, secondaryNetworkAttachmentDefinitionName)),
			libvmistatus.WithStatus(libvmistatus.New(WithInterfacesStatus(existingInterfacesStatus))),
		)

		podAnnotations := map[string]string{
			networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet,
		}

		Expect(controllers.UpdateVMIStatus(vmi, newPodFromVMI(vmi, podAnnotations))).To(Succeed())

		expectedInterfacesStatus := []v1.VirtualMachineInstanceNetworkInterface{
			{Name: secondaryNetworkName, PodInterfaceName: "pod7e0055a6880"},
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
