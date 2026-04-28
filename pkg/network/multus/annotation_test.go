/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package multus_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/multus"
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
})
