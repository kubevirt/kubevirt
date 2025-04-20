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

package hostdevice_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"libvirt.org/go/libvirt"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
)

var _ = Describe("Hot(un)Plug HostDevice", func() {
	Context("filter", func() {
		It("filters 0 SRIOV devices, given non-SRIOV devices", func() {
			var domainSpec api.DomainSpec

			domainSpec.Devices.HostDevices = append(
				domainSpec.Devices.HostDevices,
				api.HostDevice{Alias: api.NewUserDefinedAlias("non-sriov1")},
				api.HostDevice{Alias: api.NewUserDefinedAlias("non-sriov2")},
			)
			Expect(hostdevice.FilterHostDevicesByAlias(domainSpec.Devices.HostDevices, aliasPrefix)).To(BeEmpty())
		})

		It("filters 2 SRIOV devices, given 2 SRIOV devices and 2 non-SRIOV devices", func() {
			var domainSpec api.DomainSpec

			hostDevice1 := api.HostDevice{Alias: api.NewUserDefinedAlias(aliasPrefix + "is-sriov1")}
			hostDevice2 := api.HostDevice{Alias: api.NewUserDefinedAlias(aliasPrefix + "is-sriov2")}
			domainSpec.Devices.HostDevices = append(
				domainSpec.Devices.HostDevices,
				hostDevice1,
				api.HostDevice{Alias: api.NewUserDefinedAlias("non-sriov1")},
				hostDevice2,
				api.HostDevice{Alias: api.NewUserDefinedAlias("non-sriov2")},
			)
			Expect(hostdevice.FilterHostDevicesByAlias(domainSpec.Devices.HostDevices, aliasPrefix)).To(Equal([]api.HostDevice{hostDevice1, hostDevice2}))
		})
	})

	Context("safe detachment", func() {
		hostDevice := api.HostDevice{Alias: api.NewUserDefinedAlias(aliasPrefix + "net1")}

		It("ignores an empty list of devices", func() {
			domainSpec := newDomainSpec()

			c := newCallbackerStub(false, false)
			c.sendEvent("foo")
			d := deviceDetacherStub{}
			Expect(hostdevice.SafelyDetachHostDevices(domainSpec.Devices.HostDevices, c, d, 0)).To(Succeed())
			Expect(c.EventChannel()).To(HaveLen(1))
		})

		It("fails to register a callback", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(true, false)
			c.sendEvent("foo")
			d := deviceDetacherStub{}
			Expect(hostdevice.SafelyDetachHostDevices(domainSpec.Devices.HostDevices, c, d, 0)).ToNot(Succeed())
			Expect(c.EventChannel()).To(HaveLen(1))
		})

		It("fails to detach device", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, false)
			c.sendEvent("foo")
			d := deviceDetacherStub{fail: true}
			Expect(hostdevice.SafelyDetachHostDevices(domainSpec.Devices.HostDevices, c, d, 0)).ToNot(Succeed())
			Expect(c.EventChannel()).To(HaveLen(1))
		})

		It("fails on timeout due to no detach event", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, false)
			d := deviceDetacherStub{}
			Expect(hostdevice.SafelyDetachHostDevices(domainSpec.Devices.HostDevices, c, d, 0)).ToNot(Succeed())
		})

		It("fails due to a missing event from a device", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, false)
			c.sendEvent("unknown-device")
			d := deviceDetacherStub{}
			Expect(hostdevice.SafelyDetachHostDevices(domainSpec.Devices.HostDevices, c, d, 10*time.Millisecond)).ToNot(Succeed())
			Expect(c.EventChannel()).To(BeEmpty())
		})

		// Failure to deregister the callback only emits a logging error.
		It("succeeds to wait for a detached device and fails to deregister a callback", func() {
			domainSpec := newDomainSpec(hostDevice)

			c := newCallbackerStub(false, true)
			c.sendEvent(api.UserAliasPrefix + hostDevice.Alias.GetName())
			d := deviceDetacherStub{}
			Expect(hostdevice.SafelyDetachHostDevices(domainSpec.Devices.HostDevices, c, d, 10*time.Millisecond)).To(Succeed())
		})

		It("succeeds detaching 2 devices", func() {
			hostDevice2 := api.HostDevice{Alias: api.NewUserDefinedAlias(aliasPrefix + "net2")}
			domainSpec := newDomainSpec(hostDevice, hostDevice2)

			c := newCallbackerStub(false, false)
			c.sendEvent(api.UserAliasPrefix + hostDevice.Alias.GetName())
			c.sendEvent(api.UserAliasPrefix + hostDevice2.Alias.GetName())
			d := deviceDetacherStub{}
			Expect(hostdevice.SafelyDetachHostDevices(domainSpec.Devices.HostDevices, c, d, 10*time.Millisecond)).To(Succeed())
		})
	})

	Context("attachment", func() {
		hostDevice := api.HostDevice{Alias: api.NewUserDefinedAlias("net1")}

		It("ignores nil list of devices", func() {
			Expect(hostdevice.AttachHostDevices(deviceAttacherStub{}, nil)).Should(Succeed())
		})

		It("ignores an empty list of devices", func() {
			Expect(hostdevice.AttachHostDevices(deviceAttacherStub{}, []api.HostDevice{})).Should(Succeed())
		})

		It("succeeds to attach device", func() {
			Expect(hostdevice.AttachHostDevices(deviceAttacherStub{}, []api.HostDevice{hostDevice})).Should(Succeed())
		})

		It("succeeds to attach more than one device", func() {
			hostDevice2 := api.HostDevice{Alias: api.NewUserDefinedAlias("net2")}

			Expect(hostdevice.AttachHostDevices(deviceAttacherStub{}, []api.HostDevice{hostDevice, hostDevice2})).Should(Succeed())
		})

		It("fails to attach device", func() {
			obj := deviceAttacherStub{fail: true}
			Expect(hostdevice.AttachHostDevices(obj, []api.HostDevice{hostDevice})).ShouldNot(Succeed())
		})

		It("error should contain at least the Alias of each device that failed to attach", func() {
			obj := deviceAttacherStub{fail: true}
			hostDevice2 := api.HostDevice{Alias: api.NewUserDefinedAlias("net2")}
			err := hostdevice.AttachHostDevices(obj, []api.HostDevice{hostDevice, hostDevice2})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(And(
				ContainSubstring(hostDevice.Alias.GetName()),
				ContainSubstring(hostDevice2.Alias.GetName())))
		})
	})

	Context("difference", func() {
		DescribeTable("should return the correct host-devices set comparing by host-devices's Alias.Name",
			func(hostDevices, removeHostDevices, expectedHostDevices []api.HostDevice) {
				Expect(hostdevice.DifferenceHostDevicesByAlias(hostDevices, removeHostDevices)).To(ConsistOf(expectedHostDevices))
			},
			Entry("empty set and zero elements to filter",
				// slice A
				[]api.HostDevice{},
				// slice B
				[]api.HostDevice{},
				// expected
				[]api.HostDevice{},
			),
			Entry("empty set and at least one element to filter",
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
			Entry("valid set and zero elements to filter",
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
			Entry("valid set and at least one element to filter",
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

			Entry("valid set and a set that includes all elements from the first set",
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
			Entry("valid set and larger set to to filter",
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
