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
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/cache"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/cache/fake"

	v1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("Network", func() {
	var mockpodNIC *MockpodNIC
	var ctrl *gomock.Controller
	var pid int
	var cacheFactory cache.InterfaceCacheFactory

	BeforeEach(func() {
		pid = os.Getpid()
		ctrl = gomock.NewController(GinkgoT())
		mockpodNIC = NewMockpodNIC(ctrl)
		cacheFactory = fake.NewFakeInMemoryNetworkCacheFactory()
		podNICFactory = func(handler NetworkHandler, cacheFactory cache.InterfaceCacheFactory) podNIC {
			return mockpodNIC
		}
	})
	AfterEach(func() {
		podNICFactory = newpodNIC
	})

	Context("interface configuration", func() {
		It("should configure bridged pod networking by default", func() {
			vm := newVMIBridgeInterface("testnamespace", "testVmName")
			iface := v1.DefaultBridgeNetworkInterface()
			defaultNet := v1.DefaultPodNetwork()

			mockpodNIC.EXPECT().PlugPhase1(vm, iface, defaultNet, primaryPodInterfaceName, pid)
			err := SetupPodNetworkPhase1(vm, pid, cacheFactory)
			Expect(err).To(BeNil())
		})
		It("should accept empty network list", func() {
			vmi := newVMI("testnamespace", "testVmName")
			err := SetupPodNetworkPhase1(vmi, pid, cacheFactory)
			Expect(err).To(BeNil())
		})
		It("should configure networking with multus", func() {
			const multusInterfaceName = "net1"
			vm := newVMIBridgeInterface("testnamespace", "testVmName")
			iface := v1.DefaultBridgeNetworkInterface()
			cniNet := &v1.Network{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "default"},
				},
			}
			vm.Spec.Networks = []v1.Network{*cniNet}

			mockpodNIC.EXPECT().PlugPhase1(vm, iface, cniNet, multusInterfaceName, pid)
			err := SetupPodNetworkPhase1(vm, pid, cacheFactory)
			Expect(err).To(BeNil())
		})
		It("should configure networking with multus and a default multus network", func() {
			vm := newVMIBridgeInterface("testnamespace", "testVmName")

			// We plug three multus interfaces in, with the default being second, to ensure the netN
			// interfaces are numbered correctly
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				{
					Name: "additional1",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Bridge: &v1.InterfaceBridge{},
					},
				},
				{
					Name: "default",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Bridge: &v1.InterfaceBridge{},
					},
				},
				{
					Name: "additional2",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Bridge: &v1.InterfaceBridge{},
					},
				},
			}

			cniNet := &v1.Network{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "default", Default: true},
				},
			}
			additionalCNINet1 := &v1.Network{
				Name: "additional1",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "additional1"},
				},
			}
			additionalCNINet2 := &v1.Network{
				Name: "additional2",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "additional2"},
				},
			}

			vm.Spec.Networks = []v1.Network{*additionalCNINet1, *cniNet, *additionalCNINet2}

			mockpodNIC.EXPECT().PlugPhase1(vm, &vm.Spec.Domain.Devices.Interfaces[0], additionalCNINet1, "net1", pid)
			mockpodNIC.EXPECT().PlugPhase1(vm, &vm.Spec.Domain.Devices.Interfaces[1], cniNet, "eth0", pid)
			mockpodNIC.EXPECT().PlugPhase1(vm, &vm.Spec.Domain.Devices.Interfaces[2], additionalCNINet2, "net2", pid)
			err := SetupPodNetworkPhase1(vm, pid, cacheFactory)
			Expect(err).To(BeNil())
		})
	})
})
