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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"

	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("VMNetworkConfigurator", func() {
	var baseCacheCreator tempCacheCreator

	const launcherPID = 0

	AfterEach(func() {
		Expect(baseCacheCreator.New("").Delete()).To(Succeed())
	})
	Context("interface configuration", func() {

		It("when vm has no network source should propagate errors when phase2 is called", func() {
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			vmi.Spec.Networks = []v1.Network{{
				Name:          "default",
				NetworkSource: v1.NetworkSource{},
			}}
			vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, &baseCacheCreator, WithLauncherPid(0))
			var domain *api.Domain
			err := vmNetworkConfigurator.SetupPodNetworkPhase2(domain, vmi.Spec.Networks)
			Expect(err).To(MatchError("Network not implemented"))
		})

		Context("when calling []podNIC factory functions", func() {
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
				vmNetworkConfigurator := NewVMNetworkConfigurator(vmi, nil, WithLauncherPid(launcherPID))

				nics, err := vmNetworkConfigurator.getPhase2NICs(&api.Domain{}, vmi.Spec.Networks)
				Expect(err).ToNot(HaveOccurred())
				Expect(nics).To(BeEmpty())
			})
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
			vmNetworkConfigurator = NewVMNetworkConfigurator(vmi, nil, WithLauncherPid(0))
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
			vmNetworkConfigurator = NewVMNetworkConfigurator(vmi, &baseCacheCreator, WithNetUtilsHandler(mockNetworkH), WithLauncherPid(0))
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

type netpodStub struct {
	errSetup error
}

func (n netpodStub) Setup(_ func() error) error {
	return n.errSetup
}
