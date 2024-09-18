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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package annotations_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8Scorev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/pod/annotations"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Annotations Generator", func() {
	const testNamespace = "default"

	var clusterConfig *virtconfig.ClusterConfig

	BeforeEach(func() {
		const (
			kubevirtCRName    = "kubevirt"
			kubevirtNamespace = "kubevirt"
		)

		kv := kubecli.NewMinimalKubeVirt(kubevirtCRName)
		kv.Namespace = kubevirtNamespace

		clusterConfig = newClusterConfig(kv)
	})

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

			generator := annotations.NewGenerator(clusterConfig)
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

			generator := annotations.NewGenerator(clusterConfig)
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

			generator := annotations.NewGenerator(clusterConfig)
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

			generator := annotations.NewGenerator(clusterConfig)
			annotations, err := generator.Generate(vmi)
			Expect(err).NotTo(HaveOccurred())

			Expect(annotations).To(HaveKeyWithValue(istio.KubeVirtTrafficAnnotation, "k6t-eth0"))
		})

		DescribeTable("should not generate Istio annotation", func(vmi *v1.VirtualMachineInstance) {
			generator := annotations.NewGenerator(clusterConfig)
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

			generator := annotations.NewGenerator(clusterConfig)
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

			generator := annotations.NewGenerator(clusterConfig)
			annotations, err := generator.GenerateFromSource(vmi, sourcePod)
			Expect(err).ToNot(HaveOccurred())

			Expect(annotations).To(BeEmpty())
		})
	})
})

func newClusterConfig(kv *v1.KubeVirt) *virtconfig.ClusterConfig {
	clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)
	return clusterConfig
}

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
