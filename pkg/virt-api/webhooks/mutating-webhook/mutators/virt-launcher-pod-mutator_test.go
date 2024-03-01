// ginkgo --focus "Virt Launcher Pod Mutator"
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
 */

package mutators

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/testutils"
	"net"
)

const (
	networkName          = "name"
	networkNamespace     = "namespace"
	networkInterfaceName = "eth0"
	networkMacAddress    = "00:00:00:00:00:01"
	networkIp            = "192.168.0.1"
)

var _ = Describe("Virt Launcher Pod Mutator", func() {

	DescribeTable("should build valid network key", func(network networkv1.NetworkSelectionElement, expected string) {
		result := networkKey(network)
		Expect(result).To(Equal(expected))
	},
		Entry("with network only", networkv1.NetworkSelectionElement{
			Name: networkName,
		}, networkName),
		Entry("with network and namespace", networkv1.NetworkSelectionElement{
			Name:      networkName,
			Namespace: networkNamespace,
		}, networkNamespace+"/"+networkName),
	)

	It("should deserialize Multus networks", func() {
		payload := `[
			{
				"name":"name1",
				"namespace":"namespace1",
				"ips":["10.10.0.110/24"],
				"mac":"00:00:00:00:00:01",
				"interface":"pod890360ec7a1"
			},
			{
				"name":"name2",
				"namespace":"namespace2",
				"ips":["10.100.2.101/24"],
				"mac":"00:00:00:00:00:02",
				"interface":"pod0b696b333de"
			}
		]`
		_, err := deserializeMultusAnnotation(payload)
		Expect(err).To(BeNil())
	})

	It("should merge networks", func() {
		infinibandGuid := uuid.New().String()
		portMappings := []*networkv1.PortMapEntry{{HostPort: 56000, ContainerPort: 8080, Protocol: "TCP", HostIP: "192.168.0.1"}}
		bandwithEntries := networkv1.BandwidthEntry{IngressRate: 1, IngressBurst: 10, EgressRate: 1, EgressBurst: 10}
		CNIArgs := map[string]interface{}{"key": "value"}
		gateways := []net.IP{net.ParseIP("192.168.0.1"), net.ParseIP("192.168.0.2")}

		network1 := networkv1.NetworkSelectionElement{
			Name:             networkName,
			Namespace:        networkNamespace,
			InterfaceRequest: networkInterfaceName,
			MacRequest:       networkMacAddress,
		}
		network2 := networkv1.NetworkSelectionElement{
			Name:                  networkName,
			Namespace:             networkNamespace,
			IPRequest:             []string{networkIp},
			InfinibandGUIDRequest: infinibandGuid,
			PortMappingsRequest:   portMappings,
			BandwidthRequest:      &bandwithEntries,
			CNIArgs:               &CNIArgs,
			GatewayRequest:        gateways,
		}

		result, err := mergeNetworks(network1, network2)

		Expect(err).To(BeNil())
		Expect(result.InterfaceRequest).To(Equal(networkInterfaceName))
		Expect(result.IPRequest).To(Equal([]string{networkIp}))
		Expect(result.MacRequest).To(Equal(networkMacAddress))
		Expect(result.InfinibandGUIDRequest).To(Equal(infinibandGuid))
		Expect(result.PortMappingsRequest).To(Equal(portMappings))
		Expect(result.BandwidthRequest).To(Equal(&bandwithEntries))
		Expect(result.CNIArgs).To(Equal(&CNIArgs))
		Expect(result.GatewayRequest).To(Equal(gateways))
	})

	It("should fail on networks merge with interface name not customizable", func() {
		network1 := networkv1.NetworkSelectionElement{
			Name:             networkName,
			Namespace:        networkNamespace,
			InterfaceRequest: networkInterfaceName,
			MacRequest:       networkMacAddress,
		}
		network2 := networkv1.NetworkSelectionElement{
			Name:             networkName,
			Namespace:        networkNamespace,
			InterfaceRequest: networkInterfaceName,
		}
		_, err := mergeNetworks(network1, network2)
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal(fmt.Sprintf("interface name can't be modified for '%s' - value is managed by virt-controller", networkName)))
	})

	It("should fail on networks merge with mac address not customizable", func() {
		network1 := networkv1.NetworkSelectionElement{
			Name:             networkName,
			Namespace:        networkNamespace,
			InterfaceRequest: networkInterfaceName,
			MacRequest:       networkMacAddress,
		}
		network2 := networkv1.NetworkSelectionElement{
			Name:       networkName,
			Namespace:  networkNamespace,
			MacRequest: networkMacAddress,
		}
		_, err := mergeNetworks(network1, network2)
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal(fmt.Sprintf("mac address should be set for '%s' through the interface definition in the VMI spec", networkName)))
	})

	It("should match networks by key and merge network elements", func() {
		extraNetwork := "extraNetwork"
		networkIps := []string{networkIp}
		networks1 := []networkv1.NetworkSelectionElement{
			{Name: extraNetwork},
			{Name: networkName, Namespace: networkNamespace},
		}
		networks2 := []networkv1.NetworkSelectionElement{
			{Name: networkName, Namespace: networkNamespace, IPRequest: []string{networkIp}},
		}

		result, err := enrichMultusAnnotation(networks1, networks2)
		Expect(err).To(BeNil())
		Expect(len(result)).To(Equal(2))
		Expect(result[0].Name).To(Equal(extraNetwork))
		Expect(result[0].IPRequest).To(BeNil())
		Expect(result[1].Name).To(Equal(networkName))
		Expect(result[1].IPRequest).To(Equal(networkIps))
	})

	It("should mutate Multus pod annotation", func() {
		mutator := &VirtLauncherPodsMutator{}
		var kvInformer cache.SharedIndexInformer
		mutator.ClusterConfig, _, kvInformer = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		mutator.VirtLauncherPodInformer = kvInformer

		pod := corev1.Pod{}
		pod.Labels = map[string]string{
			v1.AppLabel: "virt-launcher",
		}
		pod.Annotations = map[string]string{
			networkv1.NetworkAttachmentAnnot: `[{
				"name": "name",
				"namespace": "namespace",
				"mac": "00:00:00:00:00:01",
				"interface":"pod890360ec7a1"
			}]`,
			MultusCustomizationAnnotation: `[{
				"name": "name",
				"namespace": "namespace",
				"ips": ["10.10.0.110/24"]
			}]`,
		}

		podBytes, _ := json.Marshal(pod)
		admissionReview := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: k8smetav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
				Object: runtime.RawExtension{
					Raw: podBytes,
				},
			},
		}

		admissionResponse := mutator.Mutate(admissionReview)

		Expect(admissionResponse).ToNot(BeNil())
		Expect(admissionResponse.Allowed).To(BeTrue())
		expected := []uint8(`[{"op":"replace","path":"/metadata/annotations/k8s.v1.cni.cncf.io~1networks","value":"[{\"name\":\"name\",\"namespace\":\"namespace\",\"ips\":[\"10.10.0.110/24\"],\"mac\":\"00:00:00:00:00:01\",\"interface\":\"pod890360ec7a1\"}]"}]`)
		Expect(admissionResponse.Patch).To(Equal(expected))
	})
})
