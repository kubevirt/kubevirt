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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package network

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Network", func() {
	var mockNetworkInterface *MockNetworkInterface
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockNetworkInterface = NewMockNetworkInterface(ctrl)
	})
	AfterEach(func() {
		NetworkInterfaceFactory = getNetworkClass
	})

	Context("interface configuration", func() {
		It("should configure bridged pod networking by default", func() {
			NetworkInterfaceFactory = func(network *v1.Network) (NetworkInterface, error) {
				return mockNetworkInterface, nil
			}
			domain := &api.Domain{}
			vm := newVMIBridgeInterface("testnamespace", "testVmName")
			api.SetObjectDefaults_Domain(domain)
			iface := v1.DefaultNetworkInterface()
			defaultNet := v1.DefaultPodNetwork()

			mockNetworkInterface.EXPECT().Plug(iface, defaultNet, domain, podInterface)
			err := SetupNetworkInterfaces(vm, domain)
			Expect(err).To(BeNil())
		})
		It("should accept empty network list", func() {
			vmi := newVMI("testnamespace", "testVmName")
			domain := &api.Domain{}
			err := SetupNetworkInterfaces(vmi, domain)
			Expect(err).To(BeNil())
		})
		It("should configure networking with multus", func() {
			NetworkInterfaceFactory = func(network *v1.Network) (NetworkInterface, error) {
				return mockNetworkInterface, nil
			}
			const multusInterfaceName = "net1"
			domain := &api.Domain{}
			vm := newVMIBridgeInterface("testnamespace", "testVmName")
			api.SetObjectDefaults_Domain(domain)
			iface := v1.DefaultNetworkInterface()
			cniNet := &v1.Network{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.CniNetwork{NetworkName: "default"},
				},
			}
			vm.Spec.Networks = []v1.Network{*cniNet}

			mockNetworkInterface.EXPECT().Plug(iface, cniNet, domain, "net1")
			err := SetupNetworkInterfaces(vm, domain)
			Expect(err).To(BeNil())
		})
		It("should configure networking with genie", func() {
			NetworkInterfaceFactory = func(network *v1.Network) (NetworkInterface, error) {
				return mockNetworkInterface, nil
			}
			const genieInterfaceName = "eth0"
			domain := &api.Domain{}
			vm := newVMIBridgeInterface("testnamespace", "testVmName")
			api.SetObjectDefaults_Domain(domain)
			iface := v1.DefaultNetworkInterface()
			cniNet := &v1.Network{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Genie: &v1.CniNetwork{NetworkName: "default"},
				},
			}
			vm.Spec.Networks = []v1.Network{*cniNet}

			mockNetworkInterface.EXPECT().Plug(iface, cniNet, domain, genieInterfaceName)
			err := SetupNetworkInterfaces(vm, domain)
			Expect(err).To(BeNil())
		})
	})
})
