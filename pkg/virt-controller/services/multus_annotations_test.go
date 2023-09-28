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
})
