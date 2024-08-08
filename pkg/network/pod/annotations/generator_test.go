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

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/pod/annotations"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Annotations Generator", func() {
	Context("Multus", func() {
		const (
			testNamespace = "default"

			defaultNetworkName = "default"
			network1Name       = "test1"
			network2Name       = "other-test1"

			defaultNetworkAttachmentDefinitionName = "default"
			networkAttachmentDefinitionName1       = "test1"
			networkAttachmentDefinitionName2       = "other-namespace/test1"
		)

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
