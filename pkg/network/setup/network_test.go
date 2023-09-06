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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"

	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/infraconfigurators"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("VMNetworkConfigurator", func() {
	var baseCacheCreator tempCacheCreator

	const launcherPID = 0

	AfterEach(func() {
		Expect(baseCacheCreator.New("").Delete()).To(Succeed())
	})
	Context("interface configuration", func() {

		Context("when vm has no network source", func() {
			var (
				vmi                   *v1.VirtualMachineInstance
				vmNetworkConfigurator *VMNetworkConfigurator
				configState           ConfigState
			)

			BeforeEach(func() {
				vmi = newVMIBridgeInterface("testnamespace", "testVmName")
				vmi.Spec.Networks = []v1.Network{{
					Name:          "default",
					NetworkSource: v1.NetworkSource{},
				}}
				vmNetworkConfigurator = NewVMNetworkConfigurator(vmi, &baseCacheCreator, WithNetSetup(netpodStub{}), WithLauncherPid(0))
				stateCache := NewConfigStateCache(string(vmi.UID), vmNetworkConfigurator.cacheCreator)
				configState = NewConfigState(&stateCache, nsExecutorStub{})
			})
			It("should propagate errors when phase1 is called", func() {
				launcherPID := 0
				err := vmNetworkConfigurator.SetupPodNetworkPhase1(launcherPID, vmi.Spec.Networks, &configState)
				Expect(err).To(MatchError("Network not implemented"))
			})
			It("should propagate errors when phase2 is called", func() {
				var domain *api.Domain
				err := vmNetworkConfigurator.SetupPodNetworkPhase2(domain, vmi.Spec.Networks)
				Expect(err).To(MatchError("Network not implemented"))
			})
		})
		Context("when calling []podNIC factory functions", func() {
			It("should configure bridged pod networking by default", func() {
				vm := newVMIBridgeInterface("testnamespace", "testVmName")

				launcherPID := 0
				vmNetworkConfigurator := NewVMNetworkConfigurator(vm, &baseCacheCreator, WithNetSetup(netpodStub{}), WithLauncherPid(launcherPID))
				iface := v1.DefaultBridgeNetworkInterface()
				defaultNet := v1.DefaultPodNetwork()
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID, vm.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(ConsistOf([]podNIC{{
					vmi:            vm,
					vmiSpecIface:   iface,
					vmiSpecNetwork: defaultNet,
					handler:        vmNetworkConfigurator.handler,
					cacheCreator:   vmNetworkConfigurator.cacheCreator,
					launcherPID:    &launcherPID,
					infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
						vm,
						iface,
						launcherPID,
						vmNetworkConfigurator.handler),
				}}))
			})
			It("should accept empty network list", func() {
				vmi := api2.NewMinimalVMIWithNS("testnamespace", "testVmName")
				launcherPID := 0
				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, &baseCacheCreator, WithNetSetup(netpodStub{}), WithLauncherPid(launcherPID))
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID, vmi.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(BeEmpty())
			})
			It("should configure networking with multus", func() {
				vmi := newVMIBridgeInterface("testnamespace", "testVmName")
				iface := v1.DefaultBridgeNetworkInterface()
				cniNet := vmiPrimaryNetwork()
				vmi.Spec.Networks = []v1.Network{*cniNet}
				launcherPID := 0
				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, &baseCacheCreator, WithNetSetup(netpodStub{}), WithLauncherPid(launcherPID))
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID, vmi.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(ConsistOf([]podNIC{{
					vmi:            vmi,
					vmiSpecIface:   iface,
					vmiSpecNetwork: cniNet,
					handler:        vmNetworkConfigurator.handler,
					cacheCreator:   vmNetworkConfigurator.cacheCreator,
					launcherPID:    &launcherPID,
					infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
						vmi,
						iface,
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

				launcherPID := 0
				vmNetworkConfigurator := NewVMNetworkConfigurator(vm, &baseCacheCreator, WithNetSetup(netpodStub{}), WithLauncherPid(launcherPID))
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID, vm.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(ContainElements([]podNIC{
					{
						vmi:            vm,
						vmiSpecIface:   &vm.Spec.Domain.Devices.Interfaces[0],
						vmiSpecNetwork: additionalCNINet1,
						handler:        vmNetworkConfigurator.handler,
						cacheCreator:   vmNetworkConfigurator.cacheCreator,
						launcherPID:    &launcherPID,
						infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
							vm,
							&vm.Spec.Domain.Devices.Interfaces[0],
							launcherPID,
							vmNetworkConfigurator.handler),
					},
					{
						vmi:            vm,
						vmiSpecIface:   &vm.Spec.Domain.Devices.Interfaces[1],
						vmiSpecNetwork: cniNet,
						handler:        vmNetworkConfigurator.handler,
						cacheCreator:   vmNetworkConfigurator.cacheCreator,
						launcherPID:    &launcherPID,
						infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
							vm,
							&vm.Spec.Domain.Devices.Interfaces[1],
							launcherPID,
							vmNetworkConfigurator.handler),
					},
					{
						vmi:            vm,
						vmiSpecIface:   &vm.Spec.Domain.Devices.Interfaces[2],
						vmiSpecNetwork: additionalCNINet2,
						handler:        vmNetworkConfigurator.handler,
						cacheCreator:   vmNetworkConfigurator.cacheCreator,
						launcherPID:    &launcherPID,
						infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
							vm, &vm.Spec.Domain.Devices.Interfaces[2],
							launcherPID,
							vmNetworkConfigurator.handler),
					},
				}))
			})

			It("should configure networking for an hotplugged interface", func() {
				const ifaceToHotplug = "newnet1"

				vmi := newVMIBridgeInterface("testnamespace", "testVmName")

				hotplugNetwork := networkToHotplug(ifaceToHotplug)
				vmi.Spec.Networks = append(vmi.Spec.Networks, hotplugNetwork)

				hotplugInterface := v1.Interface{
					Name:                   ifaceToHotplug,
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
				}
				vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, hotplugInterface)

				launcherPID := 0
				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, &baseCacheCreator, WithNetSetup(netpodStub{}), WithLauncherPid(launcherPID))

				Expect(vmNetworkConfigurator.getPhase1NICs(
					&launcherPID,
					[]v1.Network{networkToHotplug(ifaceToHotplug)},
				)).To(ConsistOf(podNIC{
					vmi:            vmi,
					launcherPID:    &launcherPID,
					vmiSpecIface:   &hotplugInterface,
					vmiSpecNetwork: &hotplugNetwork,
					handler:        vmNetworkConfigurator.handler,
					cacheCreator:   vmNetworkConfigurator.cacheCreator,
					infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
						vmi,
						&hotplugInterface,
						launcherPID,
						vmNetworkConfigurator.handler),
				}))
			})

			It("should not process SR-IOV networks", func() {
				vmi := api2.NewMinimalVMIWithNS("testnamespace", "testVmName")
				const networkName = "sriov"
				vmi.Spec.Networks = []v1.Network{{
					Name: networkName,
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "sriov-nad"},
					},
				}}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
					Name: networkName, InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
				}}

				launcherPID := 0
				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, nil, WithNetSetup(netpodStub{}), WithLauncherPid(launcherPID))
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID, vmi.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(BeEmpty())

				nics, err = vmNetworkConfigurator.getPhase2NICs(&api.Domain{}, vmi.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(BeEmpty())
			})
		})
	})

	Context("pod network phase1", func() {
		var (
			vmi         *v1.VirtualMachineInstance
			configState ConfigState
		)

		BeforeEach(func() {
			dutils.MockDefaultOwnershipManager()

			vmi = newVMIBridgeInterface("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			stateCache := NewConfigStateCache(string(vmi.UID), &baseCacheCreator)
			configState = NewConfigState(&stateCache, nsExecutorStub{})
		})

		It("fails setup during network setup", func() {
			netPodWithError := netpodStub{errSetup: fmt.Errorf("config error")}
			vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, &baseCacheCreator, WithNetSetup(netPodWithError))
			err := vmNetworkConfigurator.SetupPodNetworkPhase1(0, vmi.Spec.Networks, &configState)
			Expect(err).To(HaveOccurred())
		})

		It("is passing setup successfully", func() {
			vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, &baseCacheCreator, WithNetSetup(netpodStub{}), WithLauncherPid(0))
			Expect(vmNetworkConfigurator.SetupPodNetworkPhase1(0, vmi.Spec.Networks, &configState)).To(Succeed())
		})
	})
	Context("UnplugPodNetworksPhase1", func() {
		var (
			vmi                   *v1.VirtualMachineInstance
			vmNetworkConfigurator *VMNetworkConfigurator
			configState           ConfigStateExecutor
		)

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{UID: "123"}}
			vmi.Spec.Networks = []v1.Network{}
			vmNetworkConfigurator = NewVMNetworkConfigurator(vmi, nil, WithNetSetup(netpodStub{}), WithLauncherPid(0))
		})
		It("should succeed on successful Unplug", func() {
			configState = &ConfigStateStub{}
			Expect(vmNetworkConfigurator.UnplugPodNetworksPhase1(vmi, vmi.Spec.Networks, configState)).To(Succeed())
		})
		It("should fail on failing Unplug", func() {
			configState = &ConfigStateStub{UnplugShouldFail: true}
			Expect(vmNetworkConfigurator.UnplugPodNetworksPhase1(vmi, vmi.Spec.Networks, configState)).ToNot(Succeed())
		})
	})

	Context("filter out ordinal interfaces", func() {
		var (
			vmi                   *v1.VirtualMachineInstance
			vmNetworkConfigurator *VMNetworkConfigurator
			mockNetworkH          *netdriver.MockNetworkHandler
		)
		const (
			ordinalPodIfaceName = "net0"
			hashPodIfaceName    = "pod123"
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockNetworkH = netdriver.NewMockNetworkHandler(ctrl)
			vmi = &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{UID: "123"}}
			vmi.Spec.Networks = []v1.Network{{
				Name: testNet0,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{},
				},
			}}
			vmNetworkConfigurator = NewVMNetworkConfigurator(vmi, &baseCacheCreator, WithNetSetup(netpodStub{}), WithNetUtilsHandler(mockNetworkH), WithLauncherPid(0))
		})
		It("shouldn't filter the network, it has non-ordinal name", func() {
			mockNetworkH.EXPECT().LinkByName(gomock.Any()).Return(&netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: hashPodIfaceName}}, nil)
			Expect(vmNetworkConfigurator.filterOutOrdinalInterfaces(vmi.Spec.Networks, vmi)).To(ConsistOf([]string{testNet0}))
		})
		It("shouldn't filter the ordinal network", func() {
			mockNetworkH.EXPECT().LinkByName(gomock.Any()).Return(&netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: ordinalPodIfaceName}}, nil)
			Expect(vmNetworkConfigurator.filterOutOrdinalInterfaces(vmi.Spec.Networks, vmi)).To(BeEmpty())
		})
		It("shouldn't filter a network with no link", func() {
			mockNetworkH.EXPECT().LinkByName(gomock.Any()).Return(nil, netlink.LinkNotFoundError{}).AnyTimes()
			Expect(vmNetworkConfigurator.filterOutOrdinalInterfaces(vmi.Spec.Networks, vmi)).To(ConsistOf([]string{testNet0}))
		})
	})
})

func vmiPrimaryNetwork() *v1.Network {
	return &v1.Network{
		Name: "default",
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{NetworkName: "default"},
		},
	}
}

func networkToHotplug(name string) v1.Network {
	const nadName = "mynad"
	return v1.Network{
		Name: name,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{
				NetworkName: nadName,
			},
		},
	}
}

type netpodStub struct {
	errSetup error
}

func (n netpodStub) Setup(_ func() error) error {
	return n.errSetup
}
