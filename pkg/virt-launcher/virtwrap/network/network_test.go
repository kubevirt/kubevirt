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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/cache/fake"

	v1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("VMNetworkConfigurator", func() {
	Context("interface configuration", func() {
		It("should configure bridged pod networking by default", func() {
			vm := newVMIBridgeInterface("testnamespace", "testVmName")

			vmNetworkConfigurator := NewVMNetworkConfigurator(vm, fake.NewFakeInMemoryNetworkCacheFactory())
			iface := v1.DefaultBridgeNetworkInterface()
			defaultNet := v1.DefaultPodNetwork()
			nics, err := vmNetworkConfigurator.getNICs()
			Expect(err).ToNot(HaveOccurred())
			Expect(nics).To(ConsistOf([]podNIC{{
				vmi:              vm,
				podInterfaceName: primaryPodInterfaceName,
				iface:            iface,
				network:          defaultNet,
				handler:          vmNetworkConfigurator.handler,
				cacheFactory:     vmNetworkConfigurator.cacheFactory,
			}}))
		})
		It("should accept empty network list", func() {
			vmi := newVMI("testnamespace", "testVmName")
			vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, fake.NewFakeInMemoryNetworkCacheFactory())
			nics, err := vmNetworkConfigurator.getNICs()
			Expect(err).ToNot(HaveOccurred())
			Expect(nics).To(BeEmpty())
		})
		It("should configure networking with multus", func() {
			const multusInterfaceName = "net1"
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			iface := v1.DefaultBridgeNetworkInterface()
			cniNet := &v1.Network{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "default"},
				},
			}
			vmi.Spec.Networks = []v1.Network{*cniNet}
			vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, fake.NewFakeInMemoryNetworkCacheFactory())
			nics, err := vmNetworkConfigurator.getNICs()
			Expect(err).ToNot(HaveOccurred())
			Expect(nics).To(ConsistOf([]podNIC{{
				vmi:              vmi,
				iface:            iface,
				network:          cniNet,
				podInterfaceName: multusInterfaceName,
				handler:          vmNetworkConfigurator.handler,
				cacheFactory:     vmNetworkConfigurator.cacheFactory,
			}}))
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

			vmNetworkConfigurator := NewVMNetworkConfigurator(vm, fake.NewFakeInMemoryNetworkCacheFactory())
			nics, err := vmNetworkConfigurator.getNICs()
			Expect(err).ToNot(HaveOccurred())
			Expect(nics).To(ContainElements([]podNIC{
				{
					vmi:              vm,
					iface:            &vm.Spec.Domain.Devices.Interfaces[0],
					network:          additionalCNINet1,
					podInterfaceName: "net1",
					handler:          vmNetworkConfigurator.handler,
					cacheFactory:     vmNetworkConfigurator.cacheFactory,
				},
				{
					vmi:              vm,
					iface:            &vm.Spec.Domain.Devices.Interfaces[1],
					network:          cniNet,
					podInterfaceName: "eth0",
					handler:          vmNetworkConfigurator.handler,
					cacheFactory:     vmNetworkConfigurator.cacheFactory,
				},
				{
					vmi:              vm,
					iface:            &vm.Spec.Domain.Devices.Interfaces[2],
					network:          additionalCNINet2,
					podInterfaceName: "net2",
					handler:          vmNetworkConfigurator.handler,
					cacheFactory:     vmNetworkConfigurator.cacheFactory,
				},
			}))
		})
	})
})
