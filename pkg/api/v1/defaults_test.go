package v1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Defaults", func() {

	It("should add ACPI feature if it is unspecified", func() {
		vmi := &VirtualMachineInstance{}
		SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(*vmi.Spec.Domain.Features.ACPI.Enabled).To(BeTrue())
	})

	It("should not add non-ACPI feature by default", func() {
		vmi := &VirtualMachineInstance{}
		SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(vmi.Spec.Domain.Features.APIC).To(BeNil())
		Expect(vmi.Spec.Domain.Features.Hyperv).To(BeNil())
	})

	It("should add interface and pod network by default", func() {
		vmi := &VirtualMachineInstance{}
		SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(len(vmi.Spec.Domain.Devices.Interfaces)).NotTo(BeZero())
		Expect(len(vmi.Spec.Networks)).NotTo(BeZero())
	})

	It("should default to true to all defined features", func() {
		vmi := &VirtualMachineInstance{
			Spec: VirtualMachineInstanceSpec{
				Domain: DomainSpec{},
			},
		}
		vmi.Spec.Domain.Features = &Features{
			ACPI: FeatureState{},
			APIC: &FeatureAPIC{},
			Hyperv: &FeatureHyperv{
				Relaxed:    &FeatureState{},
				VAPIC:      &FeatureState{},
				Spinlocks:  &FeatureSpinlocks{},
				VPIndex:    &FeatureState{},
				Runtime:    &FeatureState{},
				SyNIC:      &FeatureState{},
				SyNICTimer: &FeatureState{},
				Reset:      &FeatureState{},
				VendorID:   &FeatureVendorID{},
			},
		}
		SetObjectDefaults_VirtualMachineInstance(vmi)

		features := vmi.Spec.Domain.Features
		hyperv := features.Hyperv

		Expect(*features.ACPI.Enabled).To(BeTrue())
		Expect(*features.APIC.Enabled).To(BeTrue())
		Expect(*hyperv.Relaxed.Enabled).To(BeTrue())
		Expect(*hyperv.VAPIC.Enabled).To(BeTrue())
		Expect(*hyperv.Spinlocks.Enabled).To(BeTrue())
		Expect(*hyperv.Spinlocks.Retries).To(Equal(uint32(4096)))
		Expect(*hyperv.VPIndex.Enabled).To(BeTrue())
		Expect(*hyperv.Runtime.Enabled).To(BeTrue())
		Expect(*hyperv.SyNIC.Enabled).To(BeTrue())
		Expect(*hyperv.SyNICTimer.Enabled).To(BeTrue())
		Expect(*hyperv.Reset.Enabled).To(BeTrue())
		Expect(*hyperv.VendorID.Enabled).To(BeTrue())
	})

	It("should not override defined feature states", func() {
		vmi := &VirtualMachineInstance{
			Spec: VirtualMachineInstanceSpec{
				Domain: DomainSpec{},
			},
		}
		vmi.Spec.Domain.Features = &Features{
			ACPI: FeatureState{Enabled: _true},
			APIC: &FeatureAPIC{Enabled: _false},
			Hyperv: &FeatureHyperv{
				Relaxed:    &FeatureState{Enabled: _true},
				VAPIC:      &FeatureState{Enabled: _false},
				Spinlocks:  &FeatureSpinlocks{Enabled: _true},
				VPIndex:    &FeatureState{Enabled: _false},
				Runtime:    &FeatureState{Enabled: _true},
				SyNIC:      &FeatureState{Enabled: _false},
				SyNICTimer: &FeatureState{Enabled: _true},
				Reset:      &FeatureState{Enabled: _false},
				VendorID:   &FeatureVendorID{Enabled: _true},
			},
		}
		SetObjectDefaults_VirtualMachineInstance(vmi)

		features := vmi.Spec.Domain.Features
		hyperv := features.Hyperv

		Expect(*features.ACPI.Enabled).To(BeTrue())
		Expect(*features.APIC.Enabled).To(BeFalse())
		Expect(*hyperv.Relaxed.Enabled).To(BeTrue())
		Expect(*hyperv.VAPIC.Enabled).To(BeFalse())
		Expect(*hyperv.Spinlocks.Enabled).To(BeTrue())
		Expect(*hyperv.Spinlocks.Retries).To(Equal(uint32(4096)))
		Expect(*hyperv.VPIndex.Enabled).To(BeFalse())
		Expect(*hyperv.Runtime.Enabled).To(BeTrue())
		Expect(*hyperv.SyNIC.Enabled).To(BeFalse())
		Expect(*hyperv.SyNICTimer.Enabled).To(BeTrue())
		Expect(*hyperv.Reset.Enabled).To(BeFalse())
		Expect(*hyperv.VendorID.Enabled).To(BeTrue())

		vmi.Spec.Domain.Features = &Features{
			ACPI: FeatureState{Enabled: _false},
			APIC: &FeatureAPIC{Enabled: _true},
			Hyperv: &FeatureHyperv{
				Relaxed:    &FeatureState{Enabled: _false},
				VAPIC:      &FeatureState{Enabled: _true},
				Spinlocks:  &FeatureSpinlocks{Enabled: _false},
				VPIndex:    &FeatureState{Enabled: _true},
				Runtime:    &FeatureState{Enabled: _false},
				SyNIC:      &FeatureState{Enabled: _true},
				SyNICTimer: &FeatureState{Enabled: _false},
				Reset:      &FeatureState{Enabled: _true},
				VendorID:   &FeatureVendorID{Enabled: _false},
			},
		}
		SetObjectDefaults_VirtualMachineInstance(vmi)

		features = vmi.Spec.Domain.Features
		hyperv = features.Hyperv

		Expect(*features.ACPI.Enabled).To(BeFalse())
		Expect(*features.APIC.Enabled).To(BeTrue())
		Expect(*hyperv.Relaxed.Enabled).To(BeFalse())
		Expect(*hyperv.VAPIC.Enabled).To(BeTrue())
		Expect(*hyperv.Spinlocks.Enabled).To(BeFalse())
		Expect(hyperv.Spinlocks.Retries).To(BeNil())
		Expect(*hyperv.VPIndex.Enabled).To(BeTrue())
		Expect(*hyperv.Runtime.Enabled).To(BeFalse())
		Expect(*hyperv.SyNIC.Enabled).To(BeTrue())
		Expect(*hyperv.SyNICTimer.Enabled).To(BeFalse())
		Expect(*hyperv.Reset.Enabled).To(BeTrue())
		Expect(*hyperv.VendorID.Enabled).To(BeFalse())
	})

	It("should set dis defaults", func() {
		vmi := &VirtualMachineInstance{
			Spec: VirtualMachineInstanceSpec{
				Domain: DomainSpec{
					Devices: Devices{
						Disks: []Disk{
							{
								Name: "cdrom_tray_unspecified",
								DiskDevice: DiskDevice{
									CDRom: &CDRomTarget{},
								},
							},
							{
								Name: "cdrom_tray_open",
								DiskDevice: DiskDevice{
									CDRom: &CDRomTarget{
										Tray:     TrayStateOpen,
										ReadOnly: _false,
									},
								},
							},
							{
								Name: "floppy_tray_unspecified",
								DiskDevice: DiskDevice{
									Floppy: &FloppyTarget{},
								},
							},
							{
								Name: "floppy_tray_open",
								DiskDevice: DiskDevice{
									Floppy: &FloppyTarget{
										Tray:     TrayStateOpen,
										ReadOnly: true,
									},
								},
							},
							{
								Name: "should_default_to_disk",
							},
						},
					},
				},
			},
		}
		SetObjectDefaults_VirtualMachineInstance(vmi)
		disks := vmi.Spec.Domain.Devices.Disks

		Expect(disks[0].CDRom.Tray).To(Equal(TrayStateClosed))
		Expect(*disks[0].CDRom.ReadOnly).To(Equal(true))
		Expect(disks[1].CDRom.Tray).To(Equal(TrayStateOpen))
		Expect(*disks[1].CDRom.ReadOnly).To(Equal(false))
		Expect(disks[2].Floppy.Tray).To(Equal(TrayStateClosed))
		Expect(disks[2].Floppy.ReadOnly).To(Equal(false))
		Expect(disks[3].Floppy.Tray).To(Equal(TrayStateOpen))
		Expect(disks[3].Floppy.Tray).To(Equal(TrayStateOpen))
		Expect(disks[4].Disk).ToNot(BeNil())
	})

	It("should set the default watchdog and the default watchdog action", func() {
		vmi := &VirtualMachineInstance{
			Spec: VirtualMachineInstanceSpec{
				Domain: DomainSpec{
					Devices: Devices{
						Watchdog: &Watchdog{
							WatchdogDevice: WatchdogDevice{
								I6300ESB: &I6300ESBWatchdog{},
							},
						},
					},
				},
			},
		}
		SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(vmi.Spec.Domain.Devices.Watchdog.I6300ESB.Action).To(Equal(WatchdogActionReset))
		vmi.Spec.Domain.Devices.Watchdog.I6300ESB = nil
		SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(vmi.Spec.Domain.Devices.Watchdog.I6300ESB).ToNot(BeNil())
		Expect(vmi.Spec.Domain.Devices.Watchdog.I6300ESB.Action).To(Equal(WatchdogActionReset))
	})

	It("should set timer defaults", func() {
		vmi := &VirtualMachineInstance{
			Spec: VirtualMachineInstanceSpec{
				Domain: DomainSpec{
					Clock: &Clock{
						Timer: &Timer{
							HPET:   &HPETTimer{},
							KVM:    &KVMTimer{},
							PIT:    &PITTimer{},
							RTC:    &RTCTimer{},
							Hyperv: &HypervTimer{},
						},
					},
				},
			},
		}
		SetObjectDefaults_VirtualMachineInstance(vmi)
		timer := vmi.Spec.Domain.Clock.Timer
		Expect(*timer.HPET.Enabled).To(BeTrue())
		Expect(*timer.KVM.Enabled).To(BeTrue())
		Expect(*timer.PIT.Enabled).To(BeTrue())
		Expect(*timer.RTC.Enabled).To(BeTrue())
		Expect(*timer.Hyperv.Enabled).To(BeTrue())
	})
})

var _ = Describe("Function SetDefaults_NetworkInterface()", func() {

	It("should append pod interface if interface is not defined", func() {
		vmi := &VirtualMachineInstance{}
		SetDefaults_NetworkInterface(vmi)
		Expect(len(vmi.Spec.Domain.Devices.Interfaces)).To(Equal(1))
		Expect(vmi.Spec.Domain.Devices.Interfaces[0].Name).To(Equal("default"))
		Expect(vmi.Spec.Networks[0].Name).To(Equal("default"))
		Expect(vmi.Spec.Networks[0].Pod).ToNot(BeNil())
	})

	It("should not append pod interface if interface is defined", func() {
		vmi := &VirtualMachineInstance{}
		net := Network{
			Name: "testnet",
		}
		iface := Interface{Name: net.Name}
		vmi.Spec.Networks = []Network{net}
		vmi.Spec.Domain.Devices.Interfaces = []Interface{iface}

		SetDefaults_NetworkInterface(vmi)
		Expect(len(vmi.Spec.Domain.Devices.Interfaces)).To(Equal(1))
		Expect(vmi.Spec.Domain.Devices.Interfaces[0].Name).To(Equal("testnet"))
		Expect(vmi.Spec.Networks[0].Name).To(Equal("testnet"))
		Expect(vmi.Spec.Networks[0].Pod).To(BeNil())
	})

	It("should not append pod interface if it's explicitly disabled", func() {
		autoAttach := false
		vmi := &VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.AutoattachPodInterface = &autoAttach

		SetDefaults_NetworkInterface(vmi)
		Expect(len(vmi.Spec.Domain.Devices.Interfaces)).To(Equal(0))
		Expect(len(vmi.Spec.Networks)).To(Equal(0))
	})

	It("should append pod interface if auto attach is true", func() {
		autoAttach := true
		vmi := &VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.AutoattachPodInterface = &autoAttach
		SetDefaults_NetworkInterface(vmi)
		Expect(len(vmi.Spec.Domain.Devices.Interfaces)).To(Equal(1))
		Expect(vmi.Spec.Domain.Devices.Interfaces[0].Name).To(Equal("default"))
		Expect(vmi.Spec.Networks[0].Name).To(Equal("default"))
		Expect(vmi.Spec.Networks[0].Pod).ToNot(BeNil())
	})

})
