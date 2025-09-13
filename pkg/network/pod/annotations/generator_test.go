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

package annotations_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8Scorev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/network/downwardapi"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/pod/annotations"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("Annotations Generator", func() {
	const testNamespace = "default"

	Context("Multus", func() {
		const (
			defaultNetworkName = "default"
			network1Name       = "test1"
			network2Name       = "other-test1"

			defaultNetworkAttachmentDefinitionName = "default"
			networkAttachmentDefinitionName1       = "test1"
			networkAttachmentDefinitionName2       = "other-namespace/test1"
		)

		It("should generate the Multus networks annotation", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(defaultNetworkName)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(network1Name)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(network2Name)),
				libvmi.WithNetwork(libvmi.MultusNetwork(defaultNetworkName, defaultNetworkAttachmentDefinitionName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, networkAttachmentDefinitionName1)),
				libvmi.WithNetwork(libvmi.MultusNetwork(network2Name, networkAttachmentDefinitionName2)),
			)

			generator := annotations.NewGenerator(stubClusterConfig{})
			annotations, err := generator.Generate(vmi)
			Expect(err).NotTo(HaveOccurred())

			expectedValue := "[" +
				"{\"name\":\"default\",\"namespace\":\"default\",\"interface\":\"pod37a8eec1ce1\"}," +
				"{\"name\":\"test1\",\"namespace\":\"default\",\"interface\":\"pod1b4f0e98519\"}," +
				"{\"name\":\"test1\",\"namespace\":\"other-namespace\",\"interface\":\"pod49dba5c72f0\"}" +
				"]"

			Expect(annotations).To(HaveKeyWithValue(networkv1.NetworkAttachmentAnnot, expectedValue))
		})

		It("should generate the Multus networks and Multus default network annotations", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(defaultNetworkName)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(network1Name)),
				libvmi.WithNetwork(newMultusDefaultPodNetwork(defaultNetworkName, defaultNetworkAttachmentDefinitionName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, networkAttachmentDefinitionName1)),
			)

			generator := annotations.NewGenerator(stubClusterConfig{})
			annotations, err := generator.Generate(vmi)
			Expect(err).NotTo(HaveOccurred())

			expectedValue := "[{\"name\":\"test1\",\"namespace\":\"default\",\"interface\":\"pod1b4f0e98519\"}]"

			Expect(annotations).To(HaveKeyWithValue(networkv1.NetworkAttachmentAnnot, expectedValue))
			Expect(annotations).To(HaveKeyWithValue(multus.DefaultNetworkCNIAnnotation, defaultNetworkAttachmentDefinitionName))
		})

		It("should generate the Multus networks annotation when an interface has a custom MAC address", func() {
			const customMACAddress = "de:ad:00:00:be:af"

			sriovNIC := libvmi.InterfaceDeviceWithSRIOVBinding(network1Name)

			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(defaultNetworkName)),
				libvmi.WithInterface(*libvmi.InterfaceWithMac(&sriovNIC, customMACAddress)),
				libvmi.WithNetwork(libvmi.MultusNetwork(defaultNetworkName, defaultNetworkAttachmentDefinitionName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, networkAttachmentDefinitionName1)),
			)

			generator := annotations.NewGenerator(stubClusterConfig{})
			annotations, err := generator.Generate(vmi)
			Expect(err).NotTo(HaveOccurred())

			expectedValue := "[" +
				"{\"name\":\"default\",\"namespace\":\"default\",\"interface\":\"pod37a8eec1ce1\"}," +
				"{\"name\":\"test1\",\"namespace\":\"default\",\"mac\":\"de:ad:00:00:be:af\",\"interface\":\"pod1b4f0e98519\"}" +
				"]"

			Expect(annotations).To(HaveKeyWithValue(networkv1.NetworkAttachmentAnnot, expectedValue))
		})
	})

	Context("Istio annotations", func() {
		It("should generate Istio annotation when VMI is connected to pod network using masquerade binding", func() {
			vmi := libvmi.New(
				libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			generator := annotations.NewGenerator(stubClusterConfig{})
			annotations, err := generator.Generate(vmi)
			Expect(err).NotTo(HaveOccurred())

			Expect(annotations).To(HaveKeyWithValue(istio.KubeVirtTrafficAnnotation, "k6t-eth0"))
		})

		DescribeTable("should not generate Istio annotation", func(vmi *v1.VirtualMachineInstance) {
			generator := annotations.NewGenerator(stubClusterConfig{})
			annotations, err := generator.Generate(vmi)
			Expect(err).NotTo(HaveOccurred())

			Expect(annotations).To(Not(HaveKey(istio.KubeVirtTrafficAnnotation)))
		},
			Entry("when VMI is not connected any network", libvmi.New()),
			Entry("when VMI is connected to pod network and not using masquerade binding",
				libvmi.New(
					libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				),
			),
		)
	})

	Context("Network naming scheme conversion during migration", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			const (
				networkName1                     = "blue"
				networkAttachmentDefinitionName1 = "test1"

				networkName2                     = "red"
				networkAttachmentDefinitionName2 = "other-namespace/test1"
			)

			vmi = libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName1)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName2)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(networkName1, networkAttachmentDefinitionName1)),
				libvmi.WithNetwork(libvmi.MultusNetwork(networkName2, networkAttachmentDefinitionName2)),
			)
		})

		It("should convert the naming scheme when source pod has ordinal naming", func() {
			sourcePodAnnotations := map[string]string{}
			sourcePodAnnotations[networkv1.NetworkStatusAnnot] = `[
							{"interface":"eth0", "name":"default"},
							{"interface":"net1", "name":"test1", "namespace":"default"},
							{"interface":"net2", "name":"test1", "namespace":"other-namespace"}
						]`

			sourcePod := newStubVirtLauncherPod(vmi, sourcePodAnnotations)

			generator := annotations.NewGenerator(stubClusterConfig{})
			convertedAnnotations, err := generator.GenerateFromSource(vmi, sourcePod)
			Expect(err).ToNot(HaveOccurred())

			expectedMultusNetworksAnnotation := `[
							{"interface":"net1", "name":"test1", "namespace":"default"},
							{"interface":"net2", "name":"test1", "namespace":"other-namespace"}
						]`

			Expect(convertedAnnotations[networkv1.NetworkAttachmentAnnot]).To(MatchJSON(expectedMultusNetworksAnnotation))
		})

		It("should not convert the naming scheme when source pod does not have ordinal naming", func() {
			sourcePodAnnotations := map[string]string{}
			sourcePodAnnotations[networkv1.NetworkStatusAnnot] = `[
							{"interface":"pod16477688c0e", "name":"test1", "namespace":"default"},
							{"interface":"podb1f51a511f1", "name":"test1", "namespace":"other-namespace"}
						]`

			sourcePod := newStubVirtLauncherPod(vmi, sourcePodAnnotations)

			generator := annotations.NewGenerator(stubClusterConfig{})
			annotations, err := generator.GenerateFromSource(vmi, sourcePod)
			Expect(err).ToNot(HaveOccurred())

			Expect(annotations).To(BeEmpty())
		})
	})

	Context("Network Info annotation", func() {
		const (
			networkName1                     = "boo"
			networkAttachmentDefinitionName1 = "default/no-device-info"
			networkName2                     = "foo"
			networkAttachmentDefinitionName2 = "default/with-device-info"
			networkName3                     = "doo"
			networkAttachmentDefinitionName3 = "default/sriov"
			networkName4                     = "goo"
			networkAttachmentDefinitionName4 = "default/br-net"

			deviceInfoPlugin    = "deviceinfo"
			nonDeviceInfoPlugin = "non_deviceinfo"
		)

		const (
			multusNetworkStatusWithPrimaryAndSecondaryNetsWithoutDeviceInfo = `[` +
				`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
				`{"name":"default/no-device-info","interface":"pod6446d58d6df","mac":"8a:37:d9:e7:0f:18","dns":{}}` +
				`]`

			multusNetworkStatusEntryForDeviceInfo = `
{
  "name": "default/with-device-info",
  "interface": "pod2c26b46b68f",
  "dns": {},
  "device-info": {
    "type": "pci",
    "version": "1.0.0",
    "pci": {
      "pci-address": "0000:65:00.2"
    }
  }
}`

			multusNetworkStatusEntryForSRIOV = `
{
  "name": "default/sriov",
  "interface": "pod778c553efa0",
  "dns": {},
  "device-info": {
    "type": "pci",
    "version": "1.0.0",
    "pci": {
      "pci-address": "0000:65:00.3"
    }
  }
}`
		)

		var clusterConfig stubClusterConfig

		BeforeEach(func() {
			clusterConfig.registeredPlugins = map[string]v1.InterfaceBindingPlugin{
				deviceInfoPlugin:    {DownwardAPI: v1.DeviceInfo},
				nonDeviceInfoPlugin: {},
			}
		})

		It("Should not generate the network info annotation when there are no networks", func() {
			vmi := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithAutoAttachPodInterface(false))

			const multusNetworkStatusWithPrimaryNet = `[` +
				`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}}` +
				`]`

			podAnnotations := map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet}

			generator := annotations.NewGenerator(clusterConfig)
			actualAnnotations := generator.GenerateFromActivePod(vmi, newStubVirtLauncherPod(vmi, podAnnotations))

			Expect(actualAnnotations).To(Not(HaveKey(downwardapi.NetworkInfoAnnot)))
		})

		It("Should not generate the network info annotation when there are no networks with binding plugin / SR-IOV", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName1)),
				libvmi.WithNetwork(libvmi.MultusNetwork(networkName1, networkAttachmentDefinitionName1)),
			)

			podAnnotations := map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryAndSecondaryNetsWithoutDeviceInfo}

			generator := annotations.NewGenerator(clusterConfig)
			actualAnnotations := generator.GenerateFromActivePod(vmi, newStubVirtLauncherPod(vmi, podAnnotations))

			Expect(actualAnnotations).To(Not(HaveKey(downwardapi.NetworkInfoAnnot)))
		})

		It("Should not generate the network info annotation when there are networks with binding plugin but none with device-info", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(libvmi.InterfaceWithBindingPlugin(networkName1, v1.PluginBinding{Name: nonDeviceInfoPlugin})),
				libvmi.WithNetwork(libvmi.MultusNetwork(networkName1, networkAttachmentDefinitionName1)),
			)

			podAnnotations := map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryAndSecondaryNetsWithoutDeviceInfo}

			generator := annotations.NewGenerator(clusterConfig)
			actualAnnotations := generator.GenerateFromActivePod(vmi, newStubVirtLauncherPod(vmi, podAnnotations))

			Expect(actualAnnotations).To(Not(HaveKey(downwardapi.NetworkInfoAnnot)))
		})

		It("Should generate the network info annotation when there is one binding plugin with device info", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(libvmi.InterfaceWithBindingPlugin(networkName2, v1.PluginBinding{Name: deviceInfoPlugin})),
				libvmi.WithNetwork(libvmi.MultusNetwork(networkName2, networkAttachmentDefinitionName2)),
			)

			const multusNetworkStatusWithPrimaryAndSecondaryNetsWithDeviceInfo = `[` +
				`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
				multusNetworkStatusEntryForDeviceInfo +
				`]`

			podAnnotations := map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryAndSecondaryNetsWithDeviceInfo}

			generator := annotations.NewGenerator(clusterConfig)
			actualAnnotations := generator.GenerateFromActivePod(vmi, newStubVirtLauncherPod(vmi, podAnnotations))

			Expect(actualAnnotations).To(HaveKeyWithValue(
				downwardapi.NetworkInfoAnnot,
				`{"interfaces":[{"network":"foo","deviceInfo":{"type":"pci","version":"1.0.0","pci":{"pci-address":"0000:65:00.2"}}}]}`,
			))
		})

		It("Should generate the network info annotation when there is an SR-IOV interface", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(networkName3)),
				libvmi.WithNetwork(libvmi.MultusNetwork(networkName3, networkAttachmentDefinitionName3)),
			)

			const multusNetworkStatusWithPrimaryAndSRIOVSecondaryNet = `[` +
				`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
				multusNetworkStatusEntryForSRIOV +
				`]`

			podAnnotations := map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryAndSRIOVSecondaryNet}

			generator := annotations.NewGenerator(clusterConfig)
			actualAnnotations := generator.GenerateFromActivePod(vmi, newStubVirtLauncherPod(vmi, podAnnotations))

			Expect(actualAnnotations).To(HaveKeyWithValue(
				downwardapi.NetworkInfoAnnot,
				`{"interfaces":[{"network":"doo","deviceInfo":{"type":"pci","version":"1.0.0","pci":{"pci-address":"0000:65:00.3"}}}]}`,
			))
		})

		It("Should generate the network info annotation when there is SR-IOV interface and binding plugin interface with device-info", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(libvmi.InterfaceWithBindingPlugin(networkName2, v1.PluginBinding{Name: deviceInfoPlugin})),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(networkName3)),
				libvmi.WithInterface(libvmi.InterfaceWithBindingPlugin(networkName1, v1.PluginBinding{Name: nonDeviceInfoPlugin})),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName4)),
				libvmi.WithNetwork(libvmi.MultusNetwork(networkName2, networkAttachmentDefinitionName2)),
				libvmi.WithNetwork(libvmi.MultusNetwork(networkName3, networkAttachmentDefinitionName3)),
				libvmi.WithNetwork(libvmi.MultusNetwork(networkName1, networkAttachmentDefinitionName1)),
				libvmi.WithNetwork(libvmi.MultusNetwork(networkName4, networkAttachmentDefinitionName4)),
			)

			const multusNetworkStatusWithPrimaryAndFourSecondaryNets = `[` +
				`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
				multusNetworkStatusEntryForDeviceInfo + "," +
				multusNetworkStatusEntryForSRIOV + "," +
				`{"name":"default/no-device-info","interface":"pod6446d58d6df","mac":"8a:37:d9:e7:0f:18","dns":{}},` +
				`{"name":"default/br-net","interface":"podeeea394806a","mac":"6a:1f:28:23:58:40","dns":{}}` +
				`]`

			podAnnotations := map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryAndFourSecondaryNets}

			generator := annotations.NewGenerator(clusterConfig)
			actualAnnotations := generator.GenerateFromActivePod(vmi, newStubVirtLauncherPod(vmi, podAnnotations))

			Expect(actualAnnotations).To(HaveKey(downwardapi.NetworkInfoAnnot))

			var actualNetInfo downwardapi.NetworkInfo
			Expect(json.Unmarshal([]byte(actualAnnotations[downwardapi.NetworkInfoAnnot]), &actualNetInfo)).To(Succeed())

			expectedNetInfo := []downwardapi.Interface{
				{
					Network: networkName3,
					DeviceInfo: &networkv1.DeviceInfo{
						Type:    "pci",
						Version: "1.0.0",
						Pci: &networkv1.PciDevice{
							PciAddress: "0000:65:00.3",
						},
					},
				},
				{
					Network: networkName2,
					DeviceInfo: &networkv1.DeviceInfo{
						Type:    "pci",
						Version: "1.0.0",
						Pci: &networkv1.PciDevice{
							PciAddress: "0000:65:00.2",
						},
					},
				},
			}

			Expect(actualNetInfo.Interfaces).To(Equal(expectedNetInfo))
		})
	})

	Context("NIC Hotplug / Hotunplug", func() {
		const (
			network1Name                     = "red"
			networkAttachmentDefinitionName1 = "some-net"

			network2Name                     = "blue"
			networkAttachmentDefinitionName2 = "other-net"

			multusNetworksAnnotation = `[{"name":"some-net","namespace":"default","interface":"podb1f51a511f1"}]`

			multusOrdinalNetworksAnnotation = `[{"name":"some-net","namespace":"default","interface":"net1"}]`

			multusNetworksAnnotationWithTwoNets = `[` +
				`{"name":"some-net","namespace":"default","interface":"podb1f51a511f1"},` +
				`{"name":"other-net","namespace":"default","interface":"pod16477688c0e"}` +
				`]`

			multusOrdinalNetworksAnnotationWithTwoNets = `[` +
				`{"name":"some-net","namespace":"default","interface":"net1"},` +
				`{"name":"other-net","namespace":"default","interface":"net2"}` +
				`]`

			multusNetworkStatusWithPrimaryNet = `[` +
				`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
				`]`

			multusNetworkStatusWithPrimaryAndSecondaryNets = `[` +
				`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
				`{"name":"some-net","interface":"podb1f51a511f1","mac":"8a:37:d9:e7:0f:18","dns":{}}` +
				`]`

			multusOrdinalNetworkStatusWithPrimaryAndSecondaryNets = `[` +
				`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
				`{"name":"some-net","interface":"net1","mac":"8a:37:d9:e7:0f:18","dns":{}}` +
				`]`

			multusNetworkStatusWithPrimaryAndTwoSecondaryNets = `[` +
				`{"name":"k8s-pod-network","ips":["10.244.196.146","fd10:244::c491"],"default":true,"dns":{}},` +
				`{"name":"some-net","interface":"podb1f51a511f1","mac":"8a:37:d9:e7:0f:18","dns":{}}` +
				`{"name":"other-net","interface":"pod16477688c0e","mac":"25:bb:e2:a3:e8:4d","dns":{}}` +
				`]`
		)

		It("Should not generate network attachment annotation when there are no networks", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithAutoAttachPodInterface(false),
			)

			pod := newStubVirtLauncherPod(vmi, map[string]string{
				networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet,
			})

			generator := annotations.NewGenerator(stubClusterConfig{})

			annotations := generator.GenerateFromActivePod(vmi, pod)
			Expect(annotations).ToNot(HaveKey(networkv1.NetworkAttachmentAnnot))
		})

		DescribeTable("Should not generate network attachment annotation", func(podAnnotations map[string]string) {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			pod := newStubVirtLauncherPod(vmi, map[string]string{})
			generator := annotations.NewGenerator(stubClusterConfig{})

			annotations := generator.GenerateFromActivePod(vmi, pod)
			Expect(annotations).ToNot(HaveKey(networkv1.NetworkAttachmentAnnot))
		},
			Entry("when network-status annotation is missing", map[string]string{}),
			Entry("when network-status annotation has just the primary network",
				map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet},
			),
		)

		DescribeTable("Should not generate network attachment annotation when all spec interfaces are present",
			func(podAnnotations map[string]string) {
				vmi := libvmi.New(
					libvmi.WithNamespace(testNamespace),
					libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(network1Name)),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, networkAttachmentDefinitionName1)),
				)

				pod := newStubVirtLauncherPod(vmi, podAnnotations)
				generator := annotations.NewGenerator(stubClusterConfig{})

				annotations := generator.GenerateFromActivePod(vmi, pod)
				Expect(annotations).ToNot(HaveKey(networkv1.NetworkAttachmentAnnot))
			},
			Entry("with hashed naming scheme",
				map[string]string{
					networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation,
					networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndSecondaryNets,
				},
			),
			Entry("with ordinal naming scheme",
				map[string]string{
					networkv1.NetworkAttachmentAnnot: multusOrdinalNetworksAnnotation,
					networkv1.NetworkStatusAnnot:     multusOrdinalNetworkStatusWithPrimaryAndSecondaryNets,
				},
			),
		)

		It("Should not generate network attachment annotation when all spec interfaces are present in status", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(network1Name)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, networkAttachmentDefinitionName1)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: "default", PodInterfaceName: "eth0"}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       network1Name,
							InfoSource: vmispec.InfoSourceMultusStatus,
						}),
					),
				),
			)

			pod := newStubVirtLauncherPod(vmi, map[string]string{
				networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation,
				networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndSecondaryNets,
			})
			generator := annotations.NewGenerator(stubClusterConfig{})

			annotations := generator.GenerateFromActivePod(vmi, pod)
			Expect(annotations).ToNot(HaveKey(networkv1.NetworkAttachmentAnnot))
		})

		It("Should generate network attachment annotation when VMI is not connected to secondary networks and an interface is hot plugged",
			func() {
				vmi := libvmi.New(
					libvmi.WithNamespace(testNamespace),
					libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(network1Name)),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, networkAttachmentDefinitionName1)),
				)

				podAnnotations := map[string]string{
					networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet,
				}
				generator := annotations.NewGenerator(stubClusterConfig{})
				annotations := generator.GenerateFromActivePod(vmi, newStubVirtLauncherPod(vmi, podAnnotations))

				Expect(annotations[networkv1.NetworkAttachmentAnnot]).To(MatchJSON(multusNetworksAnnotation))
			})

		DescribeTable(
			"Should generate network attachment annotation when VMI is connected to a secondary network and an interface is hot plugged",
			func(podAnnotations map[string]string, expectedMultusAnnotation string) {
				vmi := libvmi.New(
					libvmi.WithNamespace(testNamespace),
					libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(network1Name)),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(network2Name)),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, networkAttachmentDefinitionName1)),
					libvmi.WithNetwork(libvmi.MultusNetwork(network2Name, networkAttachmentDefinitionName2)),
				)

				generator := annotations.NewGenerator(stubClusterConfig{})
				annotations := generator.GenerateFromActivePod(vmi, newStubVirtLauncherPod(vmi, podAnnotations))

				Expect(annotations[networkv1.NetworkAttachmentAnnot]).To(MatchJSON(expectedMultusAnnotation))
			},
			Entry("with hashed naming scheme",
				map[string]string{networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryAndSecondaryNets},
				multusNetworksAnnotationWithTwoNets,
			),
			Entry("with ordinal naming scheme",
				map[string]string{networkv1.NetworkStatusAnnot: multusOrdinalNetworkStatusWithPrimaryAndSecondaryNets},
				multusOrdinalNetworksAnnotationWithTwoNets,
			),
		)

		It("Should generate network attachment annotation when multiple secondary interfaces are hot plugged", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(network1Name)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(network2Name)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, networkAttachmentDefinitionName1)),
				libvmi.WithNetwork(libvmi.MultusNetwork(network2Name, networkAttachmentDefinitionName2)),
			)

			podAnnotations := map[string]string{
				networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet,
			}

			generator := annotations.NewGenerator(stubClusterConfig{})
			annotations := generator.GenerateFromActivePod(vmi, newStubVirtLauncherPod(vmi, podAnnotations))

			Expect(annotations[networkv1.NetworkAttachmentAnnot]).To(MatchJSON(multusNetworksAnnotationWithTwoNets))
		})

		It("Should not generate network attachment annotation when an SR-IOV iface is hot plugged", func() {
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(network1Name)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, networkAttachmentDefinitionName1)),
				libvmistatus.WithStatus(libvmistatus.New(
					libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: "default"}),
				)),
			)

			podAnnotations := map[string]string{
				networkv1.NetworkStatusAnnot: multusNetworkStatusWithPrimaryNet,
			}

			generator := annotations.NewGenerator(stubClusterConfig{})
			annotations := generator.GenerateFromActivePod(vmi, newStubVirtLauncherPod(vmi, podAnnotations))

			Expect(annotations).ToNot(HaveKey(networkv1.NetworkAttachmentAnnot))
		})

		It("Should generate network attachment annotation when a secondary interface is hot unplugged", func() {
			ifaceWithStateAbsent := libvmi.InterfaceDeviceWithBridgeBinding(network1Name)
			ifaceWithStateAbsent.State = v1.InterfaceStateAbsent
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(ifaceWithStateAbsent),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(network2Name)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, networkAttachmentDefinitionName1)),
				libvmi.WithNetwork(libvmi.MultusNetwork(network2Name, networkAttachmentDefinitionName2)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: "default", PodInterfaceName: "eth0"}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       network1Name,
							InfoSource: vmispec.InfoSourceMultusStatus,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       network2Name,
							InfoSource: vmispec.InfoSourceMultusStatus,
						}),
					),
				),
			)

			podAnnotations := map[string]string{
				networkv1.NetworkAttachmentAnnot: multusNetworksAnnotationWithTwoNets,
				networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndTwoSecondaryNets,
			}

			generator := annotations.NewGenerator(stubClusterConfig{})
			annotations := generator.GenerateFromActivePod(vmi, newStubVirtLauncherPod(vmi, podAnnotations))

			const expectedMultusNetAttach = `[{"name":"other-net","namespace":"default","interface":"pod16477688c0e"}]`
			Expect(annotations[networkv1.NetworkAttachmentAnnot]).To(MatchJSON(expectedMultusNetAttach))
		})

		It("Should remove the Multus network attachment annotation when the last secondary interface is hot unplugged", func() {
			ifaceWithStateAbsent := libvmi.InterfaceDeviceWithBridgeBinding(network1Name)
			ifaceWithStateAbsent.State = v1.InterfaceStateAbsent
			vmi := libvmi.New(
				libvmi.WithNamespace(testNamespace),
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(ifaceWithStateAbsent),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, networkAttachmentDefinitionName1)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: "default", PodInterfaceName: "eth0"}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       network1Name,
							InfoSource: vmispec.InfoSourceMultusStatus,
						}),
					),
				),
			)

			podAnnotations := map[string]string{
				networkv1.NetworkAttachmentAnnot: multusNetworksAnnotation,
				networkv1.NetworkStatusAnnot:     multusNetworkStatusWithPrimaryAndSecondaryNets,
			}

			generator := annotations.NewGenerator(stubClusterConfig{})
			annotations := generator.GenerateFromActivePod(vmi, newStubVirtLauncherPod(vmi, podAnnotations))

			Expect(annotations).To(HaveKeyWithValue(networkv1.NetworkAttachmentAnnot, ""))
		})
	})
})

func newMultusDefaultPodNetwork(name, networkAttachmentDefinitionName string) *v1.Network {
	return &v1.Network{
		Name: name,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{
				NetworkName: networkAttachmentDefinitionName,
				Default:     true,
			},
		},
	}
}

func newStubVirtLauncherPod(vmi *v1.VirtualMachineInstance, podAnnotations map[string]string) *k8Scorev1.Pod {
	return &k8Scorev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "virt-launcher-" + vmi.Name,
			Namespace:   vmi.Namespace,
			Annotations: podAnnotations,
		},
	}
}

type stubClusterConfig struct {
	registeredPlugins map[string]v1.InterfaceBindingPlugin
}

func (s stubClusterConfig) GetNetworkBindings() map[string]v1.InterfaceBindingPlugin {
	return s.registeredPlugins
}
