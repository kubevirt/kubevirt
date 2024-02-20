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

package network

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("getResourceNameForNetwork", func() {
	It("should return empty string when resource name is not specified", func() {
		network := &networkv1.NetworkAttachmentDefinition{}
		Expect(getResourceNameForNetwork(network)).To(Equal(""))
	})

	It("should return resource name if specified", func() {
		network := &networkv1.NetworkAttachmentDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					MULTUS_RESOURCE_NAME_ANNOTATION: "fake.com/fakeResource",
				},
			},
		}
		Expect(getResourceNameForNetwork(network)).To(Equal("fake.com/fakeResource"))
	})
})

var _ = Describe("getNamespaceAndNetworkName", func() {
	It("should return vmi namespace when namespace is implicit", func() {
		vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "testns"}}
		namespace, networkName := getNamespaceAndNetworkName(vmi.Namespace, "testnet")
		Expect(namespace).To(Equal("testns"))
		Expect(networkName).To(Equal("testnet"))
	})

	It("should return namespace from networkName when namespace is explicit", func() {
		vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "testns"}}
		namespace, networkName := getNamespaceAndNetworkName(vmi.Namespace, "otherns/testnet")
		Expect(namespace).To(Equal("otherns"))
		Expect(networkName).To(Equal("testnet"))
	})
})
