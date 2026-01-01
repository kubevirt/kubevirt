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

package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

const (
	ifaceTypeVhostUser = "vhostuser"
	//nolint:gosec //linter is confusing passt for password
	passtLogFilePath = "/var/run/kubevirt/passt.log"
	virtioModel      = "virtio-non-transitional"
)

var _ = Describe("pod network configurator", func() {
	Context("generate domain spec interface", func() {
		DescribeTable("should fail to create configurator given",
			func(vmi *v1.VirtualMachineInstance) {
				configurator := network.NewDomainConfigurator()
				var domain api.Domain
				err := configurator.Configure(vmi, &domain)
				Expect(err).To(HaveOccurred())
			},
			Entry("no pod network",
				libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default")),
					libvmi.WithNetwork(libvmi.MultusNetwork("default", "test-nad")),
				),
			),
			Entry("no corresponding iface",
				libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("not-default")),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				),
			),
		)

		It("should fail given interface with invalid PCI address", func() {
			vmi := libvmi.New(
				libvmi.WithInterface(interfaceWithPasstBindingAndPCI("default", "invalid-pci-address")),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			configurator := network.NewDomainConfigurator()
			var domain api.Domain
			err := configurator.Configure(vmi, &domain)
			Expect(err).To(HaveOccurred())
		})

		DescribeTable("should add interface to domain spec given iface with",
			func(iface v1.Interface, expectedDomainIface api.Interface) {
				vmi := libvmi.New(
					libvmi.WithInterface(iface),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)

				configurator := network.NewDomainConfigurator(network.WithVirtioModel(virtioModel))

				var domain api.Domain
				Expect(configurator.Configure(vmi, &domain)).To(Succeed())
				Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1))
				Expect(domain.Spec.Devices.Interfaces[0]).To(Equal(expectedDomainIface))
			},
			Entry("passt binding",
				libvmi.InterfaceDeviceWithPasstBinding("default"),
				newPasstDomainInterface("default", virtioModel,
					withPasstBackend(),
					withPasstPortForwardAll(),
				),
			),
			Entry("PCI address",
				interfaceWithPasstBindingAndPCI("default", "0000:02:02.0"),
				newPasstDomainInterface("default", virtioModel,
					withPasstBackend(),
					withPasstPortForwardAll(),
					withPCIAddress("0000:02:02.0"),
				),
			),
			Entry("MAC address",
				interfaceWithPasstBindingAndMAC("default", "02:02:02:02:02:02"),
				newPasstDomainInterface("default", virtioModel,
					withPasstBackend(),
					withPasstPortForwardAll(),
					withMAC("02:02:02:02:02:02"),
				),
			),
			Entry("ACPI address",
				interfaceWithPasstBindingAndACPI("default", 2),
				newPasstDomainInterface("default", virtioModel,
					withPasstBackend(),
					withPasstPortForwardAll(),
					withACPI(2),
				),
			),
			Entry("non virtio model",
				interfaceWithPasstBindingAndModel("default", "e1000"),
				newPasstDomainInterface("default", "e1000",
					withPasstBackend(),
					withPasstPortForwardAll(),
				),
			),
			Entry("tcp ports (should forward tcp ports only)",
				interfaceWithPasstBindingAndPorts("default", []v1.Port{
					{Protocol: "TCP", Port: 1},
					{Protocol: "TCP", Port: 4},
				}),
				newPasstDomainInterface("default", virtioModel,
					withPasstBackend(),
					withPasstPortForwardTCP([]uint{1, 4}),
				),
			),
			Entry("udp ports (should forward udp ports only)",
				interfaceWithPasstBindingAndPorts("default", []v1.Port{
					{Protocol: "UDP", Port: 2},
					{Protocol: "UDP", Port: 3},
				}),
				newPasstDomainInterface("default", virtioModel,
					withPasstBackend(),
					withPasstPortForwardUDP([]uint{2, 3}),
				),
			),
			Entry("both tcp and udp ports",
				interfaceWithPasstBindingAndPorts("default", []v1.Port{
					{Port: 1},
					{Protocol: "UDP", Port: 2},
					{Protocol: "UDP", Port: 3},
					{Protocol: "TCP", Port: 4},
				}),
				newPasstDomainInterface("default", virtioModel,
					withPasstBackend(),
					withPasstPortForwardTCP([]uint{1, 4}),
					withPasstPortForwardUDP([]uint{2, 3}),
				),
			),
		)

		DescribeTable("should add interface to domain spec given iface given the option",
			func(vmiModifier func(*v1.VirtualMachineInstance), virtioModelType string, expectedDomainIface api.Interface) {
				vmi := libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default")),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)
				if vmiModifier != nil {
					vmiModifier(vmi)
				}

				configurator := network.NewDomainConfigurator(network.WithVirtioModel(virtioModelType))

				var domain api.Domain
				Expect(configurator.Configure(vmi, &domain)).To(Succeed())
				Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1))
				Expect(domain.Spec.Devices.Interfaces[0]).To(Equal(expectedDomainIface))
			},
			Entry("virtio transitional enabled",
				nil,
				"virtio-transitional",
				newPasstDomainInterface("default", "virtio-transitional",
					withPasstBackend(),
					withPasstPortForwardAll(),
				),
			),
			Entry("isitio proxy injection enabled",
				func(vmi *v1.VirtualMachineInstance) {
					vmi.Annotations = map[string]string{
						"sidecar.istio.io/inject": "true",
					}
				},
				virtioModel,
				newPasstDomainInterface("default", virtioModel,
					withPasstBackend(),
					withPasstPortForwardIstio(),
				),
			),
		)

		It("should not override other interfaces", func() {
			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default")),
				libvmi.WithInterface(v1.Interface{
					Name: "secondary",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Bridge: &v1.InterfaceBridge{},
					},
				}),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork("secondary", "test-nad")),
			)

			configurator := network.NewDomainConfigurator(
				network.WithDomainAttachmentByInterfaceName(map[string]string{"secondary": string(v1.Tap)}),
				network.WithVirtioModel(virtioModel),
			)

			var domain api.Domain
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedPasstIface := newPasstDomainInterface("default", virtioModel,
				withPasstBackend(),
				withPasstPortForwardAll(),
			)
			expectedBridgeIface := newDomainInterface("secondary", virtioModel, withTypeEthernet())

			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(2))
			Expect(domain.Spec.Devices.Interfaces).To(ContainElement(expectedPasstIface))
			Expect(domain.Spec.Devices.Interfaces).To(ContainElement(expectedBridgeIface))
		})

		It("should set domain interface correctly when executed more than once", func() {
			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default")),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			configurator := network.NewDomainConfigurator(
				network.WithVirtioModel(virtioModel),
			)

			expectedDomainIface := newPasstDomainInterface("default", virtioModel,
				withPasstBackend(),
				withPasstPortForwardAll(),
			)

			var domain api.Domain
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1))
			Expect(domain.Spec.Devices.Interfaces[0]).To(Equal(expectedDomainIface))

			var domain2 api.Domain
			Expect(configurator.Configure(vmi, &domain2)).To(Succeed())
			Expect(domain2.Spec.Devices.Interfaces).To(Equal(domain.Spec.Devices.Interfaces))
		})

		It("should set domain interface source link to the optional one if exists", func() {
			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding("default")),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				{Name: "default", PodInterfaceName: "ovn-udn1"},
			}

			configurator := network.NewDomainConfigurator(network.WithVirtioModel(virtioModel))

			var domain api.Domain
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomainIface := newPasstDomainInterface("default", virtioModel,
				withPasstBackend(),
				withPasstPortForwardAll(),
				withSourceDevice("ovn-udn1"),
			)
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1))
			Expect(domain.Spec.Devices.Interfaces[0]).To(Equal(expectedDomainIface))
		})
	})
})

func interfaceWithPasstBindingAndPCI(name, pciAddress string) v1.Interface {
	iface := libvmi.InterfaceDeviceWithPasstBinding(name)
	iface.PciAddress = pciAddress
	return iface
}

func interfaceWithPasstBindingAndMAC(name, macAddress string) v1.Interface {
	iface := libvmi.InterfaceDeviceWithPasstBinding(name)
	iface.MacAddress = macAddress
	return iface
}

func interfaceWithPasstBindingAndACPI(name string, acpiIndex int) v1.Interface {
	iface := libvmi.InterfaceDeviceWithPasstBinding(name)
	iface.ACPIIndex = acpiIndex
	return iface
}

func interfaceWithPasstBindingAndModel(name, model string) v1.Interface {
	iface := libvmi.InterfaceDeviceWithPasstBinding(name)
	iface.Model = model
	return iface
}

func interfaceWithPasstBindingAndPorts(name string, ports []v1.Port) v1.Interface {
	iface := libvmi.InterfaceDeviceWithPasstBinding(name)
	iface.Ports = ports
	return iface
}

type passtOption func(iface *api.Interface)

func newPasstDomainInterface(networkName, modelType string, options ...passtOption) api.Interface {
	newIface := api.Interface{
		Alias: api.NewUserDefinedAlias(networkName),
		Model: &api.Model{Type: modelType},
		Type:  ifaceTypeVhostUser,
		Source: api.InterfaceSource{
			Device: "eth0",
		},
	}

	for _, f := range options {
		f(&newIface)
	}

	return newIface
}

func withPasstBackend() passtOption {
	return func(iface *api.Interface) {
		iface.Backend = &api.InterfaceBackend{
			Type:    "passt",
			LogFile: passtLogFilePath,
		}
	}
}

func withPasstPortForwardAll() passtOption {
	return func(iface *api.Interface) {
		iface.PortForward = []api.InterfacePortForward{
			{Proto: "tcp"},
			{Proto: "udp"},
		}
	}
}

func withPasstPortForwardTCP(ports []uint) passtOption {
	return func(iface *api.Interface) {
		var ranges []api.InterfacePortForwardRange
		for _, port := range ports {
			ranges = append(ranges, api.InterfacePortForwardRange{Start: port})
		}
		if iface.PortForward == nil {
			iface.PortForward = []api.InterfacePortForward{}
		}
		iface.PortForward = append(iface.PortForward, api.InterfacePortForward{Proto: "tcp", Ranges: ranges})
	}
}

func withPasstPortForwardUDP(ports []uint) passtOption {
	return func(iface *api.Interface) {
		var ranges []api.InterfacePortForwardRange
		for _, port := range ports {
			ranges = append(ranges, api.InterfacePortForwardRange{Start: port})
		}
		if iface.PortForward == nil {
			iface.PortForward = []api.InterfacePortForward{}
		}
		iface.PortForward = append(iface.PortForward, api.InterfacePortForward{Proto: "udp", Ranges: ranges})
	}
}

func withPasstPortForwardIstio() passtOption {
	return func(iface *api.Interface) {
		iface.PortForward = []api.InterfacePortForward{
			{
				Proto: "tcp",
				Ranges: []api.InterfacePortForwardRange{
					{Start: 15000, Exclude: "yes"},
					{Start: 15001, Exclude: "yes"},
					{Start: 15004, Exclude: "yes"},
					{Start: 15006, Exclude: "yes"},
					{Start: 15008, Exclude: "yes"},
					{Start: 15009, Exclude: "yes"},
					{Start: 15020, Exclude: "yes"},
					{Start: 15021, Exclude: "yes"},
					{Start: 15053, Exclude: "yes"},
					{Start: 15090, Exclude: "yes"},
				},
			},
		}
	}
}

func withPCIAddress(pciAddress string) passtOption {
	return func(iface *api.Interface) {
		addr, err := device.NewPciAddressField(pciAddress)
		Expect(err).ToNot(HaveOccurred())
		iface.Address = addr
	}
}

func withMAC(macAddress string) passtOption {
	return func(iface *api.Interface) {
		iface.MAC = &api.MAC{MAC: macAddress}
	}
}

func withACPI(index int) passtOption {
	return func(iface *api.Interface) {
		if index > 0 {
			iface.ACPI = &api.ACPI{Index: uint(index)}
		}
	}
}

func withSourceDevice(sourceDevice string) passtOption {
	return func(iface *api.Interface) {
		iface.Source.Device = sourceDevice
	}
}
