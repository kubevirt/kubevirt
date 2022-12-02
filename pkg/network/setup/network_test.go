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
	"errors"
	"fmt"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/network/infraconfigurators"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("VMNetworkConfigurator", func() {
	var (
		baseCacheCreator tempCacheCreator
	)
	AfterEach(func() {
		Expect(baseCacheCreator.New("").Delete()).To(Succeed())
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
				vmNetworkConfigurator = NewVMNetworkConfigurator(vmi, &baseCacheCreator)
			})
			It("should propagate errors when phase1 is called", func() {
				launcherPID := 0
				err := vmNetworkConfigurator.SetupPodNetworkPhase1(launcherPID, vmi.Spec.Networks)
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

				vmNetworkConfigurator := NewVMNetworkConfigurator(vm, &baseCacheCreator)
				iface := v1.DefaultBridgeNetworkInterface()
				defaultNet := v1.DefaultPodNetwork()
				launcherPID := 0
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID, vm.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(ConsistOf([]podNIC{{
					vmi:              vm,
					podInterfaceName: namescheme.PrimaryPodInterfaceName,
					vmiSpecIface:     iface,
					vmiSpecNetwork:   defaultNet,
					handler:          vmNetworkConfigurator.handler,
					cacheCreator:     vmNetworkConfigurator.cacheCreator,
					launcherPID:      &launcherPID,
					infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
						vm,
						iface,
						generateInPodBridgeInterfaceName(namescheme.PrimaryPodInterfaceName),
						launcherPID,
						vmNetworkConfigurator.handler),
				}}))
			})
			It("should accept empty network list", func() {
				vmi := api2.NewMinimalVMIWithNS("testnamespace", "testVmName")
				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, &baseCacheCreator)
				launcherPID := 0
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID, vmi.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(BeEmpty())
			})
			It("should configure networking with multus", func() {
				const multusInterfaceName = "37a8eec1ce1"
				vmi := newVMIBridgeInterface("testnamespace", "testVmName")
				iface := v1.DefaultBridgeNetworkInterface()
				cniNet := vmiPrimaryNetwork()
				vmi.Spec.Networks = []v1.Network{*cniNet}
				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, &baseCacheCreator)
				launcherPID := 0
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID, vmi.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(ConsistOf([]podNIC{{
					vmi:              vmi,
					vmiSpecIface:     iface,
					vmiSpecNetwork:   cniNet,
					podInterfaceName: multusInterfaceName,
					handler:          vmNetworkConfigurator.handler,
					cacheCreator:     vmNetworkConfigurator.cacheCreator,
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

				vmNetworkConfigurator := NewVMNetworkConfigurator(vm, &baseCacheCreator)
				launcherPID := 0
				nics, err := vmNetworkConfigurator.getPhase1NICs(&launcherPID, vm.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(ContainElements([]podNIC{
					{
						vmi:              vm,
						vmiSpecIface:     &vm.Spec.Domain.Devices.Interfaces[0],
						vmiSpecNetwork:   additionalCNINet1,
						podInterfaceName: "e56ef68384a",
						handler:          vmNetworkConfigurator.handler,
						cacheCreator:     vmNetworkConfigurator.cacheCreator,
						launcherPID:      &launcherPID,
						infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
							vm,
							&vm.Spec.Domain.Devices.Interfaces[0],
							generateInPodBridgeInterfaceName("e56ef68384a"),
							launcherPID,
							vmNetworkConfigurator.handler),
					},
					{
						vmi:              vm,
						vmiSpecIface:     &vm.Spec.Domain.Devices.Interfaces[1],
						vmiSpecNetwork:   cniNet,
						podInterfaceName: "eth0",
						handler:          vmNetworkConfigurator.handler,
						cacheCreator:     vmNetworkConfigurator.cacheCreator,
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
						podInterfaceName: "9f531ef99d2",
						handler:          vmNetworkConfigurator.handler,
						cacheCreator:     vmNetworkConfigurator.cacheCreator,
						launcherPID:      &launcherPID,
						infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
							vm,
							&vm.Spec.Domain.Devices.Interfaces[2],
							generateInPodBridgeInterfaceName("9f531ef99d2"),
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

				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, &baseCacheCreator)
				launcherPID := 0

				const expectedPodIfaceName = "45b3499a170"
				Expect(vmNetworkConfigurator.getPhase1NICs(
					&launcherPID,
					[]v1.Network{networkToHotplug(ifaceToHotplug)},
				)).To(ConsistOf(podNIC{
					vmi:              vmi,
					podInterfaceName: expectedPodIfaceName,
					launcherPID:      &launcherPID,
					vmiSpecIface:     &hotplugInterface,
					vmiSpecNetwork:   &hotplugNetwork,
					handler:          vmNetworkConfigurator.handler,
					cacheCreator:     vmNetworkConfigurator.cacheCreator,
					infraConfigurator: infraconfigurators.NewBridgePodNetworkConfigurator(
						vmi,
						&hotplugInterface,
						generateInPodBridgeInterfaceName(expectedPodIfaceName),
						launcherPID,
						vmNetworkConfigurator.handler,
					),
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

				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, nil)
				nics, err := vmNetworkConfigurator.getPhase1NICs(pointer.P(0), vmi.Spec.Networks)
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
			mockNetworkH *netdriver.MockNetworkHandler

			vmi                   *v1.VirtualMachineInstance
			vmNetworkConfigurator *VMNetworkConfigurator
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockNetworkH = netdriver.NewMockNetworkHandler(ctrl)

			dutils.MockDefaultOwnershipManager()

			vmi = newVMIBridgeInterface("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmNetworkConfigurator = newVMNetworkConfiguratorWithHandlerAndCache(vmi, mockNetworkH, &baseCacheCreator)
		})

		It("fails setup during network discovery", func() {
			mockNetworkH.EXPECT().ReadIPAddressesFromLink(gomock.Any()).Return("", "", fmt.Errorf("discovery error"))

			err := vmNetworkConfigurator.SetupPodNetworkPhase1(0, vmi.Spec.Networks)
			Expect(err).To(HaveOccurred())
			var errCritical *neterrors.CriticalNetworkError
			Expect(errors.As(err, &errCritical)).To(BeFalse(), "expected a non-critical error, but got %v", err)
		})

		It("fails (critically) setup during network preparation (config)", func() {
			mockNetworkH.EXPECT().ReadIPAddressesFromLink(gomock.Any()).Return("1.2.3.4", "2001::1", nil)
			mockNetworkH.EXPECT().IsIpv4Primary().Return(true, nil)
			mockNetworkH.EXPECT().LinkByName(gomock.Any()).Return(&netlink.Bridge{}, nil)
			mockNetworkH.EXPECT().AddrList(gomock.Any(), gomock.Any()).Return([]netlink.Addr{}, nil)

			mockNetworkH.EXPECT().LinkSetDown(gomock.Any()).Return(fmt.Errorf("config error"))

			err := vmNetworkConfigurator.SetupPodNetworkPhase1(0, vmi.Spec.Networks)
			Expect(err).To(HaveOccurred())
			var errCritical *neterrors.CriticalNetworkError
			Expect(errors.As(err, &errCritical)).To(BeTrue(), "expected critical error, but got %v", err)
		})

		It("is passing setup successfully (and persists cache data)", func() {
			linkIP4, linkIP6 := "1.2.3.4", "2001::1"
			mockNetworkH.EXPECT().ReadIPAddressesFromLink(gomock.Any()).Return(linkIP4, linkIP6, nil)
			mockNetworkH.EXPECT().IsIpv4Primary().Return(true, nil)
			mockNetworkH.EXPECT().LinkByName(gomock.Any()).Return(&netlink.Bridge{}, nil)
			mockNetworkH.EXPECT().AddrList(gomock.Any(), gomock.Any()).Return([]netlink.Addr{}, nil)
			mockNetworkH.EXPECT().LinkSetDown(gomock.Any()).Return(nil)
			mockNetworkH.EXPECT().LinkAdd(gomock.Any()).Return(nil)
			mockNetworkH.EXPECT().LinkByName(gomock.Any()).Return(&netlink.Bridge{}, nil)
			mockNetworkH.EXPECT().LinkSetHardwareAddr(gomock.Any(), gomock.Any()).Return(nil)
			mockNetworkH.EXPECT().LinkSetMaster(gomock.Any(), gomock.Any()).Return(nil)
			mockNetworkH.EXPECT().LinkSetUp(gomock.Any()).Return(nil)
			mockNetworkH.EXPECT().ParseAddr(gomock.Any()).Return(&netlink.Addr{}, nil)
			mockNetworkH.EXPECT().AddrAdd(gomock.Any(), gomock.Any()).Return(nil)
			mockNetworkH.EXPECT().DisableTXOffloadChecksum(gomock.Any()).Return(nil)
			mockNetworkH.EXPECT().CreateTapDevice(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockNetworkH.EXPECT().BindTapDeviceToBridge(gomock.Any(), gomock.Any()).Return(nil)
			mockNetworkH.EXPECT().LinkSetUp(gomock.Any()).Return(nil)
			mockNetworkH.EXPECT().LinkSetLearningOff(gomock.Any()).Return(nil)

			Expect(vmNetworkConfigurator.SetupPodNetworkPhase1(0, vmi.Spec.Networks)).To(Succeed())

			var podData *cache.PodIfaceCacheData
			podData, err := cache.ReadPodInterfaceCache(&baseCacheCreator, string(vmi.UID), "default")
			Expect(err).ToNot(HaveOccurred())
			Expect(podData.PodIP).To(Equal(linkIP4))
			Expect(podData.PodIPs).To(ConsistOf(linkIP4, linkIP6))
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
