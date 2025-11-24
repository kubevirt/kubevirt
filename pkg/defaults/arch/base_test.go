package arch_defaults

import (
	v1 "kubevirt.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Defaults", func() {
	It("should set the default watchdog and the default watchdog action for amd64", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Watchdog: &v1.Watchdog{
							WatchdogDevice: v1.WatchdogDevice{
								I6300ESB: &v1.I6300ESBWatchdog{},
							},
						},
					},
				},
			},
		}

		SetAmd64Watchdog(&vmi.Spec)
		Expect(vmi.Spec.Domain.Devices.Watchdog.I6300ESB.Action).To(Equal(v1.WatchdogActionReset))

		vmi.Spec.Domain.Devices.Watchdog.I6300ESB = nil
		SetAmd64Watchdog(&vmi.Spec)
		Expect(vmi.Spec.Domain.Devices.Watchdog.I6300ESB).ToNot(BeNil())
		Expect(vmi.Spec.Domain.Devices.Watchdog.I6300ESB.Action).To(Equal(v1.WatchdogActionReset))
	})

	It("should not set a watchdog if none is defined on amd64", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{},
				},
			},
		}

		SetAmd64Watchdog(&vmi.Spec)
		Expect(vmi.Spec.Domain.Devices.Watchdog).To(BeNil())
	})

	It("should set the default watchdog and the default watchdog action for s390x", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Watchdog: &v1.Watchdog{
							WatchdogDevice: v1.WatchdogDevice{
								Diag288: &v1.Diag288Watchdog{},
							},
						},
					},
				},
			},
		}

		SetS390xWatchdog(&vmi.Spec)
		Expect(vmi.Spec.Domain.Devices.Watchdog.Diag288.Action).To(Equal(v1.WatchdogActionReset))

		vmi.Spec.Domain.Devices.Watchdog.Diag288 = nil
		SetS390xWatchdog(&vmi.Spec)
		Expect(vmi.Spec.Domain.Devices.Watchdog.Diag288).ToNot(BeNil())
		Expect(vmi.Spec.Domain.Devices.Watchdog.Diag288.Action).To(Equal(v1.WatchdogActionReset))
	})

	It("should not set a watchdog if none is defined on s390x", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{},
				},
			},
		}

		SetS390xWatchdog(&vmi.Spec)
		Expect(vmi.Spec.Domain.Devices.Watchdog).To(BeNil())
	})
})
