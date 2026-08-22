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

package sriov_test

import (
	"fmt"
	"time"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"

	netsriov "kubevirt.io/kubevirt/pkg/network/deviceinfo"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/sriov"
)

const (
	netname1 = "net1"
	netname2 = "net2"
)

var _ = Describe("SRIOV HostDevice", func() {
	Context("creation", func() {
		It("creates no device given no interfaces", func() {
			vmi := &v1.VirtualMachineInstance{}

			Expect(sriov.CreateHostDevices(vmi)).To(BeEmpty())
		})

		It("creates no device given no SRIOV interfaces", func() {
			iface := v1.Interface{}
			iface.Masquerade = &v1.InterfaceMasquerade{}
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}

			Expect(sriov.CreateHostDevices(vmi)).To(BeEmpty())
		})

		It("creates no device given SRIOV interface that has no status", func() {
			iface := newSRIOVInterface("test")
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}

			Expect(sriov.CreateHostDevices(vmi)).To(BeEmpty())
		})

		It("creates no device given SRIOV interface without multus info source", func() {
			iface := newSRIOVInterface("test")
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}
			vmi.Status = v1.VirtualMachineInstanceStatus{
				Interfaces: []v1.VirtualMachineInstanceNetworkInterface{{
					Name: "test",
				}},
			}

			Expect(sriov.CreateHostDevices(vmi)).To(BeEmpty())
		})

		It("fails to create device given no available host PCI", func() {
			iface := newSRIOVInterface("test")
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}
			vmi.Status = v1.VirtualMachineInstanceStatus{
				Interfaces: []v1.VirtualMachineInstanceNetworkInterface{{
					Name:       "test",
					InfoSource: vmispec.InfoSourceMultusStatus,
				}},
			}

			_, err := sriov.CreateHostDevices(vmi)

			Expect(err).To(HaveOccurred())
		})

		It("fails to create a device given bad host PCI address", func() {
			ifaces := []v1.Interface{newSRIOVInterface("net1")}
			pool := newPCIAddressPoolStub("0bad0pci0address0")

			_, err := sriov.CreateHostDevicesFromIfacesAndPool(ifaces, pool)

			Expect(err).To(HaveOccurred())
		})

		It("fails to create a device given bad guest PCI address", func() {
			iface := newSRIOVInterface("net1")
			iface.PciAddress = "0bad0pci0address0"
			pool := newPCIAddressPoolStub("0000:81:01.0")

			_, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface}, pool)

			Expect(err).To(HaveOccurred())
		})

		It("fails to create a device given two interfaces but only one host PCI", func() {
			iface1 := newSRIOVInterface(netname1)
			iface2 := newSRIOVInterface(netname1)
			pool := newPCIAddressPoolStub("0000:81:01.0")

			_, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)

			Expect(err).To(HaveOccurred())
		})

		It("creates 2 devices that are connected to the same network", func() {
			iface1 := newSRIOVInterface(netname1)
			iface2 := newSRIOVInterface(netname1)
			pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:01.1")

			devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)

			hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			expectHostDevice1 := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
				Type:    api.HostDevicePCI,
				Managed: "no",
			}
			hostPCIAddress2 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x1"}
			expectHostDevice2 := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress2},
				Type:    api.HostDevicePCI,
				Managed: "no",
			}
			Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
		})

		It("creates 2 devices that are connected to different networks", func() {
			iface1 := newSRIOVInterface(netname1)
			iface2 := newSRIOVInterface(netname2)
			pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:02.0")

			devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)

			hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			expectHostDevice1 := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
				Type:    api.HostDevicePCI,
				Managed: "no",
			}
			hostPCIAddress2 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x02", Function: "0x0"}
			expectHostDevice2 := api.HostDevice{
				Alias:   newSRIOVAlias(netname2),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress2},
				Type:    api.HostDevicePCI,
				Managed: "no",
			}
			Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
		})

		It("creates 1 device that includes guest PCI addresses", func() {
			iface := newSRIOVInterface(netname1)
			iface.PciAddress = "0000:01:01.0"
			pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:02.0")

			devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface}, pool)

			hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			guestPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x01", Slot: "0x01", Function: "0x0"}
			expectHostDevice1 := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
				Type:    api.HostDevicePCI,
				Managed: "no",
				Address: &guestPCIAddress1,
			}
			Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1}))
		})

		DescribeTable("create two devices with custom guest PCI address",
			func(iface1, iface2 v1.Interface) {
				var expectedGuestPCIAddress1 *api.Address
				var expectedGuestPCIAddress2 *api.Address

				var err error
				if iface1.PciAddress != "" {
					expectedGuestPCIAddress1, err = device.NewPciAddressField(iface1.PciAddress)
					Expect(err).NotTo(HaveOccurred())
				}

				if iface2.PciAddress != "" {
					expectedGuestPCIAddress2, err = device.NewPciAddressField(iface2.PciAddress)
					Expect(err).NotTo(HaveOccurred())
				}

				pool := newPCIAddressPoolStub("0000:81:00.0", "0000:81:01.0")
				hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x00", Function: "0x0"}
				hostPCIAddress2 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}

				devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)
				Expect(err).NotTo(HaveOccurred())

				expectHostDevice1 := api.HostDevice{
					Alias:   newSRIOVAlias(netname1),
					Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
					Address: expectedGuestPCIAddress1,
					Type:    api.HostDevicePCI,
					Managed: "no",
				}

				expectHostDevice2 := api.HostDevice{
					Alias:   newSRIOVAlias(netname2),
					Source:  api.HostDeviceSource{Address: &hostPCIAddress2},
					Address: expectedGuestPCIAddress2,
					Type:    api.HostDevicePCI,
					Managed: "no",
				}

				Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
			},
			Entry("both interfaces have a custom guest PCI address",
				newSRIOVInterfaceWithPCIAddress(netname1, "0000:20:00.0"),
				newSRIOVInterfaceWithPCIAddress(netname2, "0000:20:01.0"),
			),
			Entry("only the first interface has a custom guest PCI address",
				newSRIOVInterfaceWithPCIAddress(netname1, "0000:20:00.0"),
				newSRIOVInterface(netname2),
			),
			Entry("only the second interface has a custom guest PCI address",
				newSRIOVInterface(netname1),
				newSRIOVInterfaceWithPCIAddress(netname2, "0000:20:01.0"),
			),
		)

		It("creates 1 device that includes boot-order", func() {
			iface := newSRIOVInterface(netname1)
			val := uint(1)
			iface.BootOrder = &val
			pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:02.0")

			devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface}, pool)

			hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			expectHostDevice1 := api.HostDevice{
				Alias:     newSRIOVAlias(netname1),
				Source:    api.HostDeviceSource{Address: &hostPCIAddress1},
				Type:      api.HostDevicePCI,
				Managed:   "no",
				BootOrder: &api.BootOrder{Order: *iface.BootOrder},
			}
			Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1}))
		})

		DescribeTable("create two devices with custom boot-order",
			func(iface1, iface2 v1.Interface) {
				var expectedBootOrder1 *api.BootOrder
				var expectedBootOrder2 *api.BootOrder

				if iface1.BootOrder != nil {
					expectedBootOrder1 = &api.BootOrder{Order: *iface1.BootOrder}
				}

				if iface2.BootOrder != nil {
					expectedBootOrder2 = &api.BootOrder{Order: *iface2.BootOrder}
				}

				pool := newPCIAddressPoolStub("0000:81:00.0", "0000:81:01.0")
				hostPCIAddress1 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x00", Function: "0x0"}
				hostPCIAddress2 := api.Address{Type: api.AddressPCI, Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}

				devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)
				Expect(err).NotTo(HaveOccurred())

				expectHostDevice1 := api.HostDevice{
					Alias:     newSRIOVAlias(netname1),
					Source:    api.HostDeviceSource{Address: &hostPCIAddress1},
					Type:      api.HostDevicePCI,
					Managed:   "no",
					BootOrder: expectedBootOrder1,
				}

				expectHostDevice2 := api.HostDevice{
					Alias:     newSRIOVAlias(netname2),
					Source:    api.HostDeviceSource{Address: &hostPCIAddress2},
					Type:      api.HostDevicePCI,
					Managed:   "no",
					BootOrder: expectedBootOrder2,
				}

				Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
			},
			Entry("both interfaces have a custom bootOrder",
				newSRIOVInterfaceWithBootOrder(netname1, 1),
				newSRIOVInterfaceWithBootOrder(netname2, 2),
			),
			Entry("only the first interface has a custom bootOrder",
				newSRIOVInterfaceWithBootOrder(netname1, 1),
				newSRIOVInterface(netname2),
			),
			Entry("only the second interface has a custom bootOrder",
				newSRIOVInterface(netname1),
				newSRIOVInterfaceWithBootOrder(netname2, 2),
			),
		)
	})

	Context("safe detachment", func() {
		hostDevice := api.HostDevice{Alias: api.NewUserDefinedAlias(netsriov.SRIOVAliasPrefix + "net1")}

		It("ignores an empty list of devices", func() {
			domainSpec := newDomainSpec()

			c := newCallbackerStub(false, false)
			c.sendEvent("foo")
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 0)).To(Succeed())
			Expect(c.EventChannel()).To(HaveLen(1))
		})

		It("fails to register a callback", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(true, false)
			c.sendEvent("foo")
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 0)).To(HaveOccurred())
			Expect(c.EventChannel()).To(HaveLen(1))
		})

		It("fails to detach device", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, false)
			c.sendEvent("foo")
			d := deviceDetacherStub{fail: true}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 0)).To(HaveOccurred())
			Expect(c.EventChannel()).To(HaveLen(1))
		})

		It("fails on timeout due to no detach event", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, false)
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 0)).To(HaveOccurred())
		})

		It("fails due to a missing event from a sriov device", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, false)
			c.sendEvent("non-sriov")
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 10*time.Millisecond)).To(HaveOccurred())
			Expect(c.EventChannel()).To(BeEmpty())
		})

		// Failure to deregister the callback only emits a logging error.
		It("succeeds to wait for a detached device and fails to deregister a callback", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, true)
			c.sendEvent(api.UserAliasPrefix + hostDevice.Alias.GetName())
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 10*time.Millisecond)).To(Succeed())
		})

		It("succeeds detaching 2 sriov devices", func() {
			hostDevice2 := api.HostDevice{Alias: api.NewUserDefinedAlias(netsriov.SRIOVAliasPrefix + "net2")}
			domainSpec := newDomainSpec(hostDevice, hostDevice2)

			c := newCallbackerStub(false, false)
			c.sendEvent(api.UserAliasPrefix + hostDevice.Alias.GetName())
			c.sendEvent(api.UserAliasPrefix + hostDevice2.Alias.GetName())
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 10*time.Millisecond)).To(Succeed())
		})
	})
})

func newDomainSpec(hostDevices ...api.HostDevice) *api.DomainSpec {
	domainSpec := &api.DomainSpec{}
	domainSpec.Devices.HostDevices = append(domainSpec.Devices.HostDevices, hostDevices...)
	return domainSpec
}

func newSRIOVAlias(netName string) *api.Alias {
	return api.NewUserDefinedAlias(netsriov.SRIOVAliasPrefix + netName)
}

func newSRIOVInterfaceWithPCIAddress(name, customPCIAddress string) v1.Interface {
	iface := newSRIOVInterface(name)
	iface.PciAddress = customPCIAddress

	return iface
}

func newSRIOVInterfaceWithBootOrder(name string, bootOrder uint) v1.Interface {
	iface := newSRIOVInterface(name)
	iface.BootOrder = &bootOrder

	return iface
}

type stubPCIAddressPool struct {
	pciAddresses []string
}

func newPCIAddressPoolStub(PCIAddresses ...string) *stubPCIAddressPool {
	return &stubPCIAddressPool{PCIAddresses}
}

func (p *stubPCIAddressPool) Pop(_ string) (string, error) {
	if len(p.pciAddresses) == 0 {
		return "", fmt.Errorf("pool is empty")
	}

	address := p.pciAddresses[0]
	p.pciAddresses = p.pciAddresses[1:]

	return address, nil
}

type deviceDetacherStub struct {
	fail bool
}

func (d deviceDetacherStub) DetachDeviceFlags(data string, flags libvirt.DomainDeviceModifyFlags) error {
	if d.fail {
		return fmt.Errorf("detach device error")
	}
	return nil
}

func newCallbackerStub(failRegister, failDeregister bool) *callbackerStub {
	return &callbackerStub{
		failRegister:   failRegister,
		failDeregister: failDeregister,
		eventChan:      make(chan interface{}, hostdevice.MaxConcurrentHotPlugDevicesEvents),
	}
}

type callbackerStub struct {
	failRegister   bool
	failDeregister bool
	eventChan      chan interface{}
}

func (c *callbackerStub) Register() error {
	if c.failRegister {
		return fmt.Errorf("register error")
	}
	return nil
}

func (c *callbackerStub) Deregister() error {
	if c.failDeregister {
		return fmt.Errorf("deregister error")
	}
	return nil
}

func (c *callbackerStub) EventChannel() <-chan interface{} {
	return c.eventChan
}

func (c *callbackerStub) sendEvent(data string) {
	c.eventChan <- data
}
