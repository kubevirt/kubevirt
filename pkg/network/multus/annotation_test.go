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

package multus_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/multus"
)

const (
	testNetworkAnnotation  = `[{"name":"test","namespace":"default"}]`
	ibPfNetworkAnnotation  = `[{"name":"ib-pf-network","namespace":"default","interface":"net1"}]`
	ibPfNetworkWithCNIArgs = `[{"name":"ib-pf-network","namespace":"default","cni-args":{"deviceID":"0000:01:00.0"}}]`
)

var _ = Describe("Multus annotations", func() {
	Context("Generate Multus network selection annotation", func() {
		When("NetworkBindingPlugins feature enabled", func() {
			It("should fail if the specified network binding plugin is not registered (specified in Kubevirt config)", func() {
				vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default"}}
				vmi.Spec.Networks = []v1.Network{
					{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
				}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					{Name: "default", Binding: &v1.PluginBinding{Name: "test-binding"}},
				}

				registeredBindinPlugins := map[string]v1.InterfaceBindingPlugin{
					"another-test-binding": {NetworkAttachmentDefinition: "another-test-binding-net"},
				}

				_, err := multus.GenerateCNIAnnotation(vmi.Namespace, vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, registeredBindinPlugins)

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
					{Name: "blue"},
					{Name: "red"},
				}

				registeredBindinPlugins := map[string]v1.InterfaceBindingPlugin{
					"test-binding": {NetworkAttachmentDefinition: "test-binding-net"},
				}

				Expect(multus.GenerateCNIAnnotation(
					vmi.Namespace,
					vmi.Spec.Domain.Devices.Interfaces,
					vmi.Spec.Networks,
					registeredBindinPlugins)).To(MatchJSON(
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
						{Name: "default", Binding: &v1.PluginBinding{Name: "test-binding"}},
					}

					registeredBindinPlugins := map[string]v1.InterfaceBindingPlugin{
						"test-binding": {NetworkAttachmentDefinition: netAttachDefRawName},
					}

					Expect(
						multus.GenerateCNIAnnotation(vmi.Namespace, vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, registeredBindinPlugins),
					).To(MatchJSON(expectedAnnot))
				},
				Entry("name with no namespace", "my-binding",
					`[{"namespace": "default", "name": "my-binding", "cni-args": {"logicNetworkName": "default"}}]`),
				Entry("name with namespace", "namespace1/my-binding",
					`[{"namespace": "namespace1", "name": "my-binding", "cni-args": {"logicNetworkName": "default"}}]`),
			)
		})
	})

	Context("Merge Multus annotations", func() {
		Describe("MergeMultusAnnotations", func() {
			It("should return new annotation when existing is empty", func() {
				existing := ""
				new := testNetworkAnnotation

				result, err := multus.MergeMultusAnnotations(existing, new)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(new))
			})

			It("should return existing annotation when new is empty", func() {
				existing := testNetworkAnnotation
				new := ""

				result, err := multus.MergeMultusAnnotations(existing, new)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(existing))
			})

			It("should preserve CNI arguments from existing annotation", func() {
				existing := `[{"name":"ib-pf-network","namespace":"default","cni-args":{"deviceID":"0000:01:00.0","resourceName":"nvidia.com/ib_pf"}}]`
				new := ibPfNetworkAnnotation

				result, err := multus.MergeMultusAnnotations(existing, new)
				Expect(err).ToNot(HaveOccurred())

				var networks []map[string]interface{}
				err = json.Unmarshal([]byte(result), &networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(networks).To(HaveLen(1))

				network := networks[0]
				Expect(network["name"]).To(Equal("ib-pf-network"))
				Expect(network["namespace"]).To(Equal("default"))
				Expect(network["interface"]).To(Equal("net1"))
				Expect(network["cni-args"]).ToNot(BeNil())
				cniArgs := network["cni-args"].(map[string]interface{})
				Expect(cniArgs["deviceID"]).To(Equal("0000:01:00.0"))
				Expect(cniArgs["resourceName"]).To(Equal("nvidia.com/ib_pf"))
			})

			It("should merge CNI arguments from both annotations", func() {
				existing := ibPfNetworkWithCNIArgs
				new := `[{"name":"ib-pf-network","namespace":"default","cni-args":{"resourceName":"nvidia.com/ib_pf"}}]`

				result, err := multus.MergeMultusAnnotations(existing, new)
				Expect(err).ToNot(HaveOccurred())

				var networks []map[string]interface{}
				err = json.Unmarshal([]byte(result), &networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(networks).To(HaveLen(1))

				network := networks[0]
				Expect(network["cni-args"]).ToNot(BeNil())
				cniArgs := network["cni-args"].(map[string]interface{})
				Expect(cniArgs["deviceID"]).To(Equal("0000:01:00.0"))
				Expect(cniArgs["resourceName"]).To(Equal("nvidia.com/ib_pf"))
			})

			It("should preserve other fields from existing annotation", func() {
				existing := `[{"name":"ib-pf-network","namespace":"default","mac":"00:11:22:33:44:55","ips":["192.168.1.100"]}]`
				new := ibPfNetworkAnnotation

				result, err := multus.MergeMultusAnnotations(existing, new)
				Expect(err).ToNot(HaveOccurred())

				var networks []map[string]interface{}
				err = json.Unmarshal([]byte(result), &networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(networks).To(HaveLen(1))

				network := networks[0]
				Expect(network["name"]).To(Equal("ib-pf-network"))
				Expect(network["namespace"]).To(Equal("default"))
				Expect(network["interface"]).To(Equal("net1"))
				Expect(network["mac"]).To(Equal("00:11:22:33:44:55"))
				Expect(network["ips"]).To(Equal([]interface{}{"192.168.1.100"}))
			})

			It("should handle multiple networks correctly", func() {
				existing := `[
					{"name":"ib-pf-network","namespace":"default","cni-args":{"deviceID":"0000:01:00.0"}},
					{"name":"gpu-network","namespace":"default","cni-args":{"gpuID":"0"}}
				]`
				new := `[
					{"name":"ib-pf-network","namespace":"default","interface":"net1"},
					{"name":"gpu-network","namespace":"default","interface":"net2"}
				]`

				result, err := multus.MergeMultusAnnotations(existing, new)
				Expect(err).ToNot(HaveOccurred())

				var networks []map[string]interface{}
				err = json.Unmarshal([]byte(result), &networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(networks).To(HaveLen(2))

				// Check first network
				ibNetwork := networks[0]
				Expect(ibNetwork["name"]).To(Equal("ib-pf-network"))
				Expect(ibNetwork["interface"]).To(Equal("net1"))
				cniArgs := ibNetwork["cni-args"].(map[string]interface{})
				Expect(cniArgs["deviceID"]).To(Equal("0000:01:00.0"))

				// Check second network
				gpuNetwork := networks[1]
				Expect(gpuNetwork["name"]).To(Equal("gpu-network"))
				gpuCniArgs := gpuNetwork["cni-args"].(map[string]interface{})
				Expect(gpuNetwork["interface"]).To(Equal("net2"))
				Expect(gpuCniArgs["gpuID"]).To(Equal("0"))
			})

			It("should handle new networks not in existing annotation", func() {
				existing := ibPfNetworkWithCNIArgs
				new := `[
					{"name":"ib-pf-network","namespace":"default","interface":"net1"},
					{"name":"new-network","namespace":"default","interface":"net2"}
				]`

				result, err := multus.MergeMultusAnnotations(existing, new)
				Expect(err).ToNot(HaveOccurred())

				var networks []map[string]interface{}
				err = json.Unmarshal([]byte(result), &networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(networks).To(HaveLen(2))

				// Check existing network is preserved
				ibNetwork := networks[0]
				Expect(ibNetwork["name"]).To(Equal("ib-pf-network"))
				Expect(ibNetwork["interface"]).To(Equal("net1"))
				cniArgs := ibNetwork["cni-args"].(map[string]interface{})
				Expect(cniArgs["deviceID"]).To(Equal("0000:01:00.0"))

				// Check new network is added
				newNetwork := networks[1]
				Expect(newNetwork["name"]).To(Equal("new-network"))
				Expect(newNetwork["interface"]).To(Equal("net2"))
			})

			It("should handle invalid JSON gracefully", func() {
				existing := testNetworkAnnotation
				new := `invalid json`

				_, err := multus.MergeMultusAnnotations(existing, new)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to parse new multus annotation"))
			})
		})
	})
})
