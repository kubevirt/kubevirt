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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package services

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Multus annotations", func() {
	var multusAnnotationPool multusNetworkAnnotationPool
	var vmi v1.VirtualMachineInstance
	var network v1.Network

	BeforeEach(func() {
		vmi = v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testvmi", Namespace: "namespace1", UID: "1234",
			},
		}
		network = v1.Network{
			NetworkSource: v1.NetworkSource{
				Multus: &v1.MultusNetwork{NetworkName: "test1"},
			},
		}
	})

	Context("a multus annotation pool with no elements", func() {
		BeforeEach(func() {
			multusAnnotationPool = multusNetworkAnnotationPool{}
		})

		It("is empty", func() {
			Expect(multusAnnotationPool.isEmpty()).To(BeTrue())
		})

		It("when added an element, is no longer empty", func() {
			podIfaceName := "net1"
			multusAnnotationPool.add(newMultusAnnotationData(vmi.Namespace, vmi.Spec.Domain.Devices.Interfaces, network, podIfaceName))
			Expect(multusAnnotationPool.isEmpty()).To(BeFalse())
		})

		It("generate a null string", func() {
			Expect(multusAnnotationPool.toString()).To(BeIdenticalTo("null"))
		})
	})

	Context("a multus annotation pool with elements", func() {
		BeforeEach(func() {
			multusAnnotationPool = multusNetworkAnnotationPool{
				pool: []networkv1.NetworkSelectionElement{
					newMultusAnnotationData(vmi.Namespace, vmi.Spec.Domain.Devices.Interfaces, network, "net1"),
				},
			}
		})

		It("is not empty", func() {
			Expect(multusAnnotationPool.isEmpty()).To(BeFalse())
		})

		It("generates a json serialized string representing the annotation", func() {
			expectedString := `[{"name":"test1","namespace":"namespace1","interface":"net1"}]`
			Expect(multusAnnotationPool.toString()).To(BeIdenticalTo(expectedString))
		})
	})

	Context("Generate Multus network selection annotation", func() {
		When("NetworkBindingPlugins feature enabled", func() {
			It("should fail if the specified network binding plugin is not registered (specified in Kubevirt config)", func() {
				vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default"}}
				vmi.Spec.Networks = []v1.Network{
					{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					{Name: "default", Binding: &v1.PluginBinding{Name: "test-binding"}}}

				config := testsClusterConfig(map[string]v1.InterfaceBindingPlugin{
					"another-test-binding": {NetworkAttachmentDefinition: "another-test-binding-net"},
				})

				_, err := GenerateMultusCNIAnnotation(vmi.Namespace, vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, config)

				Expect(err).To(HaveOccurred())
			})

			It("should add network binding plugin net-attach-def to multus annotation", func() {
				vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default"}}
				vmi.Spec.Networks = []v1.Network{
					{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
					{Name: "blue", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "test1"}}},
					{Name: "red", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "other-namespace/test1"}}},
				}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					{Name: "default", Binding: &v1.PluginBinding{Name: "test-binding"}},
					{Name: "blue"}, {Name: "red"}}

				config := testsClusterConfig(map[string]v1.InterfaceBindingPlugin{
					"test-binding": {NetworkAttachmentDefinition: "test-binding-net"},
				})

				Expect(GenerateMultusCNIAnnotation(vmi.Namespace, vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, config)).To(MatchJSON(
					`[
						{"name": "test-binding-net","namespace": "default", "cni-args": {"logicNetworkName": "default"}},						
						{"name": "test1","namespace": "default","interface": "pod16477688c0e"},
						{"name": "test1","namespace": "other-namespace","interface": "podb1f51a511f1"}
					]`,
				))
			})

			DescribeTable("should parse NetworkAttachmentDefinition name and namespace correctly, given",
				func(netAttachDefRawName, expectedAnnot string) {
					vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default"}}
					vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
					vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
						{Name: "default", Binding: &v1.PluginBinding{Name: "test-binding"}}}

					config := testsClusterConfig(map[string]v1.InterfaceBindingPlugin{
						"test-binding": {NetworkAttachmentDefinition: netAttachDefRawName},
					})

					Expect(GenerateMultusCNIAnnotation(vmi.Namespace, vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, config)).To(MatchJSON(expectedAnnot))
				},
				Entry("name with no namespace", "my-binding",
					`[{"namespace": "default", "name": "my-binding", "cni-args": {"logicNetworkName": "default"}}]`),
				Entry("name with namespace", "namespace1/my-binding",
					`[{"namespace": "namespace1", "name": "my-binding", "cni-args": {"logicNetworkName": "default"}}]`),
			)
		})
	})
})

func testsClusterConfig(plugins map[string]v1.InterfaceBindingPlugin) *virtconfig.ClusterConfig {
	kvConfig := &v1.KubeVirtConfiguration{
		DeveloperConfiguration: &v1.DeveloperConfiguration{
			FeatureGates: []string{"NetworkBindingPlugins"},
		},
		NetworkConfiguration: &v1.NetworkConfiguration{
			Binding: plugins,
		},
	}
	config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(kvConfig)

	return config
}
