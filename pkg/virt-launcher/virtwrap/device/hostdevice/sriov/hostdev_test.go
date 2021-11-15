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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package sriov_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
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

		It("fails to create device given no available host PCI", func() {
			iface := newSRIOVInterface("test")
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}

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

			hostPCIAddress1 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			expectHostDevice1 := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
				Type:    "pci",
				Managed: "no",
			}
			hostPCIAddress2 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x1"}
			expectHostDevice2 := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress2},
				Type:    "pci",
				Managed: "no",
			}
			Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
		})

		It("creates 2 devices that are connected to different networks", func() {
			iface1 := newSRIOVInterface(netname1)
			iface2 := newSRIOVInterface(netname2)
			pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:02.0")

			devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface1, iface2}, pool)

			hostPCIAddress1 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			expectHostDevice1 := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
				Type:    "pci",
				Managed: "no",
			}
			hostPCIAddress2 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x81", Slot: "0x02", Function: "0x0"}
			expectHostDevice2 := api.HostDevice{
				Alias:   newSRIOVAlias(netname2),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress2},
				Type:    "pci",
				Managed: "no",
			}
			Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1, expectHostDevice2}))
		})

		It("creates 1 device that includes guest PCI addresses", func() {
			iface := newSRIOVInterface(netname1)
			iface.PciAddress = "0000:01:01.0"
			pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:02.0")

			devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface}, pool)

			hostPCIAddress1 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			guestPCIAddress1 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x01", Slot: "0x01", Function: "0x0"}
			expectHostDevice1 := api.HostDevice{
				Alias:   newSRIOVAlias(netname1),
				Source:  api.HostDeviceSource{Address: &hostPCIAddress1},
				Type:    "pci",
				Managed: "no",
				Address: &guestPCIAddress1,
			}
			Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1}))
		})

		It("creates 1 device that includes boot-order", func() {
			iface := newSRIOVInterface(netname1)
			val := uint(1)
			iface.BootOrder = &val
			pool := newPCIAddressPoolStub("0000:81:01.0", "0000:81:02.0")

			devices, err := sriov.CreateHostDevicesFromIfacesAndPool([]v1.Interface{iface}, pool)

			hostPCIAddress1 := api.Address{Type: "pci", Domain: "0x0000", Bus: "0x81", Slot: "0x01", Function: "0x0"}
			expectHostDevice1 := api.HostDevice{
				Alias:     newSRIOVAlias(netname1),
				Source:    api.HostDeviceSource{Address: &hostPCIAddress1},
				Type:      "pci",
				Managed:   "no",
				BootOrder: &api.BootOrder{Order: *iface.BootOrder},
			}
			Expect(devices, err).To(Equal([]api.HostDevice{expectHostDevice1}))
		})
	})

	Context("filter", func() {
		It("filters 0 SRIOV devices, given non-SRIOV devices", func() {
			var domainSpec api.DomainSpec

			domainSpec.Devices.HostDevices = append(
				domainSpec.Devices.HostDevices,
				api.HostDevice{Alias: api.NewUserDefinedAlias("non-sriov1")},
				api.HostDevice{Alias: api.NewUserDefinedAlias("non-sriov2")},
			)
			Expect(sriov.FilterHostDevices(&domainSpec)).To(BeEmpty())
		})

		It("filters 2 SRIOV devices, given 2 SRIOV devices and 2 non-SRIOV devices", func() {
			var domainSpec api.DomainSpec

			hostDevice1 := api.HostDevice{Alias: api.NewUserDefinedAlias(sriov.AliasPrefix + "is-sriov1")}
			hostDevice2 := api.HostDevice{Alias: api.NewUserDefinedAlias(sriov.AliasPrefix + "is-sriov2")}
			domainSpec.Devices.HostDevices = append(
				domainSpec.Devices.HostDevices,
				hostDevice1,
				api.HostDevice{Alias: api.NewUserDefinedAlias("non-sriov1")},
				hostDevice2,
				api.HostDevice{Alias: api.NewUserDefinedAlias("non-sriov2")},
			)
			Expect(sriov.FilterHostDevices(&domainSpec)).To(Equal([]api.HostDevice{hostDevice1, hostDevice2}))
		})
	})

	Context("safe detachment", func() {
		hostDevice := api.HostDevice{Alias: api.NewUserDefinedAlias(sriov.AliasPrefix + "net1")}

		It("ignores an empty list of devices", func() {
			domainSpec := newDomainSpec()

			c := newCallbackerStub(false, false)
			c.sendEvent("foo")
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 0)).To(Succeed())
			Expect(len(c.EventChannel())).To(Equal(1))
		})

		It("fails to register a callback", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(true, false)
			c.sendEvent("foo")
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 0)).To(HaveOccurred())
			Expect(len(c.EventChannel())).To(Equal(1))
		})

		It("fails to detach device", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, false)
			c.sendEvent("foo")
			d := deviceDetacherStub{fail: true}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 0)).To(HaveOccurred())
			Expect(len(c.EventChannel())).To(Equal(1))
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
			Expect(len(c.EventChannel())).To(Equal(0))
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
			hostDevice2 := api.HostDevice{Alias: api.NewUserDefinedAlias(sriov.AliasPrefix + "net2")}
			domainSpec := newDomainSpec(hostDevice, hostDevice2)

			c := newCallbackerStub(false, false)
			c.sendEvent(api.UserAliasPrefix + hostDevice.Alias.GetName())
			c.sendEvent(api.UserAliasPrefix + hostDevice2.Alias.GetName())
			d := deviceDetacherStub{}
			Expect(sriov.SafelyDetachHostDevices(domainSpec, c, d, 10*time.Millisecond)).To(Succeed())
		})
	})

	Context("attachment", func() {
		hostDevice := api.HostDevice{Alias: api.NewUserDefinedAlias("net1")}

		It("ignores nil list of devices", func() {
			Expect(sriov.AttachHostDevices(deviceAttacherStub{}, nil)).Should(Succeed())
		})

		It("ignores an empty list of devices", func() {
			Expect(sriov.AttachHostDevices(deviceAttacherStub{}, []api.HostDevice{})).Should(Succeed())
		})

		It("succeeds to attach device", func() {
			Expect(sriov.AttachHostDevices(deviceAttacherStub{}, []api.HostDevice{hostDevice})).Should(Succeed())
		})

		It("succeeds to attach more than one device", func() {
			hostDevice2 := api.HostDevice{Alias: api.NewUserDefinedAlias("net2")}

			Expect(sriov.AttachHostDevices(deviceAttacherStub{}, []api.HostDevice{hostDevice, hostDevice2})).Should(Succeed())
		})

		It("fails to attach device", func() {
			obj := deviceAttacherStub{fail: true}
			Expect(sriov.AttachHostDevices(obj, []api.HostDevice{hostDevice})).ShouldNot(Succeed())
		})

		It("error should contain at least the Alias of each device that failed to attach", func() {
			obj := deviceAttacherStub{fail: true}
			hostDevice2 := api.HostDevice{Alias: api.NewUserDefinedAlias("net2")}
			err := sriov.AttachHostDevices(obj, []api.HostDevice{hostDevice, hostDevice2})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(And(
				ContainSubstring(hostDevice.Alias.GetName()),
				ContainSubstring(hostDevice2.Alias.GetName())))
		})
	})

	Context("difference", func() {
		table.DescribeTable("should return the correct host-devices set comparing by host-devices's Alias.Name",
			func(hostDevices, removeHostDevices, expectedHostDevices []api.HostDevice) {
				Expect(sriov.DifferenceHostDevicesByAlias(hostDevices, removeHostDevices)).To(ConsistOf(expectedHostDevices))
			},
			table.Entry("empty set and zero elements to filter",
				// slice A
				[]api.HostDevice{},
				// slice B
				[]api.HostDevice{},
				// expected
				[]api.HostDevice{},
			),
			table.Entry("empty set and at least one element to filter",
				// slice A
				[]api.HostDevice{},
				// slice B
				[]api.HostDevice{
					{Alias: api.NewUserDefinedAlias("hostdev2")},
					{Alias: api.NewUserDefinedAlias("hostdev1")},
				},
				// expected
				[]api.HostDevice{},
			),
			table.Entry("valid set and zero elements to filter",
				// slice A
				[]api.HostDevice{
					{Alias: api.NewUserDefinedAlias("hostdev1")},
					{Alias: api.NewUserDefinedAlias("hostdev2")},
					{Alias: api.NewUserDefinedAlias("hostdev3")},
				},
				// slice B
				[]api.HostDevice{},
				// expected
				[]api.HostDevice{
					{Alias: api.NewUserDefinedAlias("hostdev1")},
					{Alias: api.NewUserDefinedAlias("hostdev2")},
					{Alias: api.NewUserDefinedAlias("hostdev3")},
				},
			),
			table.Entry("valid set and at least one element to filter",
				// slice A
				[]api.HostDevice{
					{Alias: api.NewUserDefinedAlias("hostdev4")},
					{Alias: api.NewUserDefinedAlias("hostdev2")},
					{Alias: api.NewUserDefinedAlias("hostdev3")},
					{Alias: api.NewUserDefinedAlias("hostdev1")},
				},
				// slice B
				[]api.HostDevice{
					{Alias: api.NewUserDefinedAlias("hostdev4")},
					{Alias: api.NewUserDefinedAlias("hostdev2")},
				},
				// expected
				[]api.HostDevice{
					{Alias: api.NewUserDefinedAlias("hostdev1")},
					{Alias: api.NewUserDefinedAlias("hostdev3")},
				},
			),

			table.Entry("valid set and a set that includes all elements from the first set",
				// slice A
				[]api.HostDevice{
					{Alias: api.NewUserDefinedAlias("hostdev4")},
					{Alias: api.NewUserDefinedAlias("hostdev2")},
				},
				// slice B
				[]api.HostDevice{
					{Alias: api.NewUserDefinedAlias("hostdev4")},
					{Alias: api.NewUserDefinedAlias("hostdev1")},
					{Alias: api.NewUserDefinedAlias("hostdev2")},
					{Alias: api.NewUserDefinedAlias("hostdev3")},
				},
				// expected
				[]api.HostDevice{},
			),
			table.Entry("valid set and larger set to to filter",
				// slice A
				[]api.HostDevice{
					{Alias: api.NewUserDefinedAlias("hostdev4")},
					{Alias: api.NewUserDefinedAlias("hostdev2")},
				},
				// slice B
				[]api.HostDevice{
					{Alias: api.NewUserDefinedAlias("hostdev4")},
					{Alias: api.NewUserDefinedAlias("hostdev1")},
					{Alias: api.NewUserDefinedAlias("hostdev7")},
					{Alias: api.NewUserDefinedAlias("hostdev3")},
				},
				// expected
				[]api.HostDevice{
					{Alias: api.NewUserDefinedAlias("hostdev2")},
				},
			),
		)
	})
})

func newDomainSpec(hostDevices ...api.HostDevice) *api.DomainSpec {
	domainSpec := &api.DomainSpec{}
	domainSpec.Devices.HostDevices = append(domainSpec.Devices.HostDevices, hostDevices...)
	return domainSpec
}

func newSRIOVAlias(netName string) *api.Alias {
	return api.NewUserDefinedAlias(sriov.AliasPrefix + netName)
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

type deviceAttacherStub struct {
	fail bool
}

func (d deviceAttacherStub) AttachDeviceFlags(data string, flags libvirt.DomainDeviceModifyFlags) error {
	if d.fail {
		return fmt.Errorf("attach device error")
	}
	return nil
}

func newCallbackerStub(failRegister, failDeregister bool) *callbackerStub {
	return &callbackerStub{
		failRegister:   failRegister,
		failDeregister: failDeregister,
		eventChan:      make(chan interface{}, sriov.MaxConcurrentHotPlugDevicesEvents),
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
