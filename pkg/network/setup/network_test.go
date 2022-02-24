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
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/infraconfigurators"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("VMNetworkConfigurator", func() {
	var (
		tmpDir                string
		interfaceCacheFactory cache.InterfaceCacheFactory
	)
	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("/tmp", "interface-cache")
		Expect(err).ToNot(HaveOccurred())
		interfaceCacheFactory = cache.NewInterfaceCacheFactoryWithBasePath(tmpDir)
	})
	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})
	Context("interface configuration", func() {

		Context("when vm has no network source", func() {
			var (
				vmi                   *v1.VirtualMachineInstance
				vmNetworkConfigurator *VMNetworkConfigurator
			)
			BeforeEach(func() {
				vmi = newVMIBridgeInterface("testnamespace", "testVmName")
				vmi.Spec.Networks = []v1.Network{{
					Name:          "default",
					NetworkSource: v1.NetworkSource{},
				}}
				vmNetworkConfigurator = NewVMNetworkConfigurator(vmi, interfaceCacheFactory)
			})
			It("should propagate errors when phase1 is called", func() {
				launcherPID := 0
				err := vmNetworkConfigurator.SetupPodNetworkPhase1(launcherPID)
				Expect(err).To(MatchError("Network not implemented"))
			})
			It("should propagate errors when phase2 is called", func() {
				var domain *api.Domain
				err := vmNetworkConfigurator.SetupPodNetworkPhase2(domain)
				Expect(err).To(MatchError("Network not implemented"))
			})
		})
		Context("when calling []podNIC factory functions", func() {
			It("should configure bridged pod networking by default", func() {
				vm := newVMIBridgeInterface("testnamespace", "testVmName")

				vmNetworkConfigurator := NewVMNetworkConfigurator(vm, interfaceCacheFactory)
				iface := v1.DefaultBridgeNetworkInterface()
				defaultNet := v1.DefaultPodNetwork()
				launcherPID := 0
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(ConsistOf([]podNIC{{
					vmi:              vm,
					podInterfaceName: primaryPodInterfaceName,
					vmiSpecIface:     iface,
					vmiSpecNetwork:   defaultNet,
					handler:          vmNetworkConfigurator.handler,
					cacheFactory:     vmNetworkConfigurator.cacheFactory,
					launcherPID:      &launcherPID,
					infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
						vm,
						iface,
						generateInPodBridgeInterfaceName(primaryPodInterfaceName),
						launcherPID,
						vmNetworkConfigurator.handler),
				}}))
			})
			It("should accept empty network list", func() {
				vmi := newVMI("testnamespace", "testVmName")
				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, interfaceCacheFactory)
				launcherPID := 0
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID)
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
				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, interfaceCacheFactory)
				launcherPID := 0
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(ConsistOf([]podNIC{{
					vmi:              vmi,
					vmiSpecIface:     iface,
					vmiSpecNetwork:   cniNet,
					podInterfaceName: multusInterfaceName,
					handler:          vmNetworkConfigurator.handler,
					cacheFactory:     vmNetworkConfigurator.cacheFactory,
					launcherPID:      &launcherPID,
					infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
						vmi,
						iface,
						generateInPodBridgeInterfaceName(multusInterfaceName),
						launcherPID,
						vmNetworkConfigurator.handler),
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

				vmNetworkConfigurator := NewVMNetworkConfigurator(vm, interfaceCacheFactory)
				launcherPID := 0
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(ContainElements([]podNIC{
					{
						vmi:              vm,
						vmiSpecIface:     &vm.Spec.Domain.Devices.Interfaces[0],
						vmiSpecNetwork:   additionalCNINet1,
						podInterfaceName: "net1",
						handler:          vmNetworkConfigurator.handler,
						cacheFactory:     vmNetworkConfigurator.cacheFactory,
						launcherPID:      &launcherPID,
						infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
							vm,
							&vm.Spec.Domain.Devices.Interfaces[0],
							generateInPodBridgeInterfaceName("net1"),
							launcherPID,
							vmNetworkConfigurator.handler),
					},
					{
						vmi:              vm,
						vmiSpecIface:     &vm.Spec.Domain.Devices.Interfaces[1],
						vmiSpecNetwork:   cniNet,
						podInterfaceName: "eth0",
						handler:          vmNetworkConfigurator.handler,
						cacheFactory:     vmNetworkConfigurator.cacheFactory,
						launcherPID:      &launcherPID,
						infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
							vm,
							&vm.Spec.Domain.Devices.Interfaces[1],
							generateInPodBridgeInterfaceName("eth0"),
							launcherPID,
							vmNetworkConfigurator.handler),
					},
					{
						vmi:              vm,
						vmiSpecIface:     &vm.Spec.Domain.Devices.Interfaces[2],
						vmiSpecNetwork:   additionalCNINet2,
						podInterfaceName: "net2",
						handler:          vmNetworkConfigurator.handler,
						cacheFactory:     vmNetworkConfigurator.cacheFactory,
						launcherPID:      &launcherPID,
						infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
							vm,
							&vm.Spec.Domain.Devices.Interfaces[2],
							generateInPodBridgeInterfaceName("net2"),
							launcherPID,
							vmNetworkConfigurator.handler),
					},
				}))
			})
		})
	})
})
