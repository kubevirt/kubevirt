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
 * Copyright The KubeVirt Authors.
 *
 */

package launcher_test

import (
	"fmt"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/cache"
	dhcpconfigurator "kubevirt.io/kubevirt/pkg/network/dhcp"
	"kubevirt.io/kubevirt/pkg/network/setup/launcher"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("SetupPodNetworkPhase2", func() {
	const podIfaceName = "eth0"

	DescribeTable("should skip non-tap-based interfaces",
		func(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
			expectedDomain := domain.DeepCopy()
			configurator := launcher.NewVMNetworkConfigurator(vmi, nil,
				launcher.WithNetworkHandler(stubNetworkHandler{}),
				launcher.WithDomainAttachments(map[string]string{}),
			)

			Expect(configurator.SetupPodNetworkPhase2(domain, vmi.Spec.Networks)).To(Succeed())
			Expect(domain).To(Equal(expectedDomain))
		},
		Entry("SR-IOV", libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding("sriov")),
			libvmi.WithNetwork(libvmi.MultusNetwork("sriov", "sriov-nad")),
		), domainWithSRIOVHostDevice()),
		Entry("Passt", libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding(v1.DefaultPodNetwork().Name)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		), domainWithPasstInterface()),
		Entry("binding plugin without tap domain attachment", libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceWithBindingPlugin("foo", v1.PluginBinding{Name: "foo"})),
			libvmi.WithNetwork(libvmi.MultusNetwork("foo", "foo-nad")),
		), &api.Domain{}),
	)

	DescribeTable("should enrich the domain interface with MTU, MAC and target",
		func(vmi *v1.VirtualMachineInstance, dhcpFactory launcher.DHCPConfiguratorFactory) {
			domain := domainWithDefaultInterface()
			configurator := launcher.NewVMNetworkConfigurator(vmi, nil,
				launcher.WithNetworkHandler(newStubNetworkHandler(podIfaceName)),
				launcher.WithDomainAttachments(map[string]string{v1.DefaultPodNetwork().Name: string(v1.Tap)}),
				launcher.WithDHCPConfiguratorFactory(dhcpFactory),
			)

			Expect(configurator.SetupPodNetworkPhase2(domain, vmi.Spec.Networks)).To(Succeed())

			expectedDomain := domainWithDefaultInterface()
			expectedDomain.Spec.Devices.Interfaces[0].MTU = &api.MTU{Size: "1500"}
			expectedDomain.Spec.Devices.Interfaces[0].MAC = &api.MAC{MAC: "aa:bb:cc:dd:ee:ff"}
			expectedDomain.Spec.Devices.Interfaces[0].Target = &api.InterfaceTarget{Device: podIfaceName, Managed: "no"}
			Expect(domain).To(Equal(expectedDomain))
		},
		Entry("bridge", libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(v1.DefaultPodNetwork().Name)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		), stubDHCPFactory(&stubDHCPConfigurator{})),
		Entry("masquerade", libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		), stubDHCPFactory(&stubDHCPConfigurator{})),
		Entry("macvtap", libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceWithMacvtapBindingPlugin(v1.DefaultPodNetwork().Name)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		), nil),
	)

	DescribeTable("should panic when DHCP server fails to start",
		func(vmi *v1.VirtualMachineInstance) {
			configurator := launcher.NewVMNetworkConfigurator(vmi, nil,
				launcher.WithNetworkHandler(newStubNetworkHandler(podIfaceName)),
				launcher.WithDomainAttachments(map[string]string{v1.DefaultPodNetwork().Name: string(v1.Tap)}),
				launcher.WithDHCPConfiguratorFactory(stubDHCPFactory(&stubDHCPConfigurator{
					ensureErr: fmt.Errorf("DHCP start failure"),
				})),
			)

			Expect(func() {
				_ = configurator.SetupPodNetworkPhase2(domainWithDefaultInterface(), vmi.Spec.Networks)
			}).To(Panic())
		},
		Entry("bridge", libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(v1.DefaultPodNetwork().Name)),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)),
		Entry("masquerade", libvmi.New(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)),
	)
})

func stubDHCPFactory(stub *stubDHCPConfigurator) launcher.DHCPConfiguratorFactory {
	return func(_ *v1.Interface, _ *v1.Network, _ string) dhcpconfigurator.Configurator {
		return stub
	}
}

type stubDHCPConfigurator struct {
	ensureErr error
}

func (s stubDHCPConfigurator) Generate() (*cache.DHCPConfig, error) {
	return &cache.DHCPConfig{}, nil
}

func (s stubDHCPConfigurator) EnsureDHCPServerStarted(_ string, _ cache.DHCPConfig, _ *v1.DHCPOptions) error {
	return s.ensureErr
}

func domainWithSRIOVHostDevice() *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.HostDevices = []api.HostDevice{{
		Type:    "pci",
		Managed: "no",
		Alias:   api.NewUserDefinedAlias("sriov"),
	}}
	return domain
}

func domainWithPasstInterface() *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Model: &api.Model{Type: v1.VirtIO},
		Type:  "vhostuser",
		Alias: api.NewUserDefinedAlias(v1.DefaultPodNetwork().Name),
	}}
	return domain
}

func domainWithDefaultInterface() *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Model: &api.Model{Type: v1.VirtIO},
		Type:  "ethernet",
		Alias: api.NewUserDefinedAlias(v1.DefaultPodNetwork().Name),
	}}
	return domain
}

func newStubNetworkHandler(podIfaceName string) stubNetworkHandler {
	return stubNetworkHandler{
		links: map[string]netlink.Link{
			podIfaceName: &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{
				Name:         podIfaceName,
				MTU:          1500,
				HardwareAddr: net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
			}},
		},
	}
}

type stubNetworkHandler struct {
	links map[string]netlink.Link
}

func (s stubNetworkHandler) LinkByName(name string) (netlink.Link, error) {
	if l, ok := s.links[name]; ok {
		return l, nil
	}
	return nil, netlink.LinkNotFoundError{}
}

func (s stubNetworkHandler) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return []netlink.Addr{}, nil
}

func (s stubNetworkHandler) HasIPv4GlobalUnicastAddress(_ string) (bool, error) {
	return false, nil
}

func (s stubNetworkHandler) HasIPv6GlobalUnicastAddress(_ string) (bool, error) {
	return false, nil
}
