/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"fmt"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"

	v1 "kubevirt.io/api/core/v1"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/dhcp"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("podNIC", func() {
	var (
		mockNetwork          *netdriver.MockNetworkHandler
		baseCacheCreator     tempCacheCreator
		mockDHCPConfigurator *dhcp.MockConfigurator
		ctrl                 *gomock.Controller
	)

	newPhase2PodNICWithMocks := func(vmi *v1.VirtualMachineInstance) *podNIC {
		podnic := newPodNIC(vmi, &vmi.Spec.Networks[0], &vmi.Spec.Domain.Devices.Interfaces[0], mockNetwork, &baseCacheCreator)
		podnic.dhcpConfigurator = mockDHCPConfigurator

		return podnic
	}
	BeforeEach(func() {
		dutils.MockDefaultOwnershipManager()

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
		mockDHCPConfigurator = dhcp.NewMockConfigurator(ctrl)
	})
	AfterEach(func() {
		Expect(baseCacheCreator.New("").Delete()).To(Succeed())
	})

	When("DHCP config is correctly read", func() {
		var (
			podnic *podNIC
			domain *api.Domain
			vmi    *v1.VirtualMachineInstance
		)
		BeforeEach(func() {
			domain = NewDomainWithBridgeInterface()
			vmi = newVMIBridgeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			podnic = newPhase2PodNICWithMocks(vmi)

			const launcherPID = "self"
			Expect(
				cache.WriteDHCPInterfaceCache(
					podnic.cacheCreator,
					launcherPID,
					podnic.podInterfaceName,
					&cache.DHCPConfig{Name: podnic.podInterfaceName},
				),
			).To(Succeed())
		})
		Context("and starting the DHCP server fails", func() {
			BeforeEach(func() {
				mockDHCPConfigurator.EXPECT().Generate().Return(&cache.DHCPConfig{}, nil)
				mockDHCPConfigurator.EXPECT().EnsureDHCPServerStarted(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("Fake EnsureDHCPServerStarted failure"))
				podnic.domainGenerator = &fakeLibvirtSpecGenerator{
					shouldGenerateFail: false,
				}
			})
			It("phase2 should panic", func() {
				Expect(func() { _ = podnic.PlugPhase2(domain) }).To(Panic())
			})
		})
		Context("and starting the DHCP server succeed", func() {
			BeforeEach(func() {
				dhcpConfig := &cache.DHCPConfig{}
				mockDHCPConfigurator.EXPECT().Generate().Return(dhcpConfig, nil)
				mockDHCPConfigurator.EXPECT().EnsureDHCPServerStarted(namescheme.PrimaryPodInterfaceName, *dhcpConfig, vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions).Return(nil)
				podnic.domainGenerator = &fakeLibvirtSpecGenerator{
					shouldGenerateFail: false,
				}
				podnic.podInterfaceName = namescheme.PrimaryPodInterfaceName
			})
			It("phase2 should succeed", func() {
				Expect(podnic.PlugPhase2(domain)).To(Succeed())
			})

		})
	})
})

type fakeLibvirtSpecGenerator struct {
	shouldGenerateFail bool
}

func (b *fakeLibvirtSpecGenerator) Generate() error {
	if b.shouldGenerateFail {
		return fmt.Errorf("Fake LibvirtSpecGenerator.Generate failure")
	}
	return nil

}
