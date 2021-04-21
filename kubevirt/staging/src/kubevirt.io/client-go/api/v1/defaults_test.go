package v1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Defaults", func() {

	It("should leave the scheduler name unset by default", func() {
		vmi := &VirtualMachineInstance{}
		SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(vmi.Spec.SchedulerName).To(BeEmpty())
	})

	It("should take a custom scheduler if specified", func() {
		vmi := &VirtualMachineInstance{Spec: VirtualMachineInstanceSpec{SchedulerName: "custom-one"}}
		SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(vmi.Spec.SchedulerName).To(Equal("custom-one"))
	})

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

	It("should not add SMM feature if it is unspecified", func() {
		vmi := &VirtualMachineInstance{}
		SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(vmi.Spec.Domain.Features.SMM).To(BeNil())
	})

	It("should add interface and pod network by default", func() {
		vmi := &VirtualMachineInstance{}
		SetDefaults_NetworkInterface(vmi)
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
			SMM:  &FeatureState{},
			APIC: &FeatureAPIC{},
			Hyperv: &FeatureHyperv{
				Relaxed:         &FeatureState{},
				VAPIC:           &FeatureState{},
				Spinlocks:       &FeatureSpinlocks{},
				VPIndex:         &FeatureState{},
				Runtime:         &FeatureState{},
				SyNIC:           &FeatureState{},
				SyNICTimer:      &SyNICTimer{},
				Reset:           &FeatureState{},
				VendorID:        &FeatureVendorID{},
				Frequencies:     &FeatureState{},
				Reenlightenment: &FeatureState{},
				TLBFlush:        &FeatureState{},
			},
		}
		SetObjectDefaults_VirtualMachineInstance(vmi)

		features := vmi.Spec.Domain.Features
		hyperv := features.Hyperv

		Expect(*features.ACPI.Enabled).To(BeTrue())
		Expect(*features.SMM.Enabled).To(BeTrue())
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
		Expect(*hyperv.Frequencies.Enabled).To(BeTrue())
		Expect(*hyperv.Reenlightenment.Enabled).To(BeTrue())
		Expect(*hyperv.TLBFlush.Enabled).To(BeTrue())
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
				Relaxed:         &FeatureState{Enabled: _true},
				VAPIC:           &FeatureState{Enabled: _false},
				Spinlocks:       &FeatureSpinlocks{Enabled: _true},
				VPIndex:         &FeatureState{Enabled: _false},
				Runtime:         &FeatureState{Enabled: _true},
				SyNIC:           &FeatureState{Enabled: _false},
				SyNICTimer:      &SyNICTimer{Enabled: _true, Direct: &FeatureState{Enabled: _true}},
				Reset:           &FeatureState{Enabled: _false},
				VendorID:        &FeatureVendorID{Enabled: _true},
				Frequencies:     &FeatureState{Enabled: _false},
				Reenlightenment: &FeatureState{Enabled: _false},
				TLBFlush:        &FeatureState{Enabled: _true},
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
		Expect(*hyperv.SyNICTimer.Direct.Enabled).To(BeTrue())
		Expect(*hyperv.Reset.Enabled).To(BeFalse())
		Expect(*hyperv.VendorID.Enabled).To(BeTrue())
		Expect(*hyperv.Frequencies.Enabled).To(BeFalse())
		Expect(*hyperv.Reenlightenment.Enabled).To(BeFalse())
		Expect(*hyperv.TLBFlush.Enabled).To(BeTrue())

		vmi.Spec.Domain.Features = &Features{
			ACPI: FeatureState{Enabled: _false},
			APIC: &FeatureAPIC{Enabled: _true},
			Hyperv: &FeatureHyperv{
				Relaxed:         &FeatureState{Enabled: _false},
				VAPIC:           &FeatureState{Enabled: _true},
				Spinlocks:       &FeatureSpinlocks{Enabled: _false},
				VPIndex:         &FeatureState{Enabled: _true},
				Runtime:         &FeatureState{Enabled: _false},
				SyNIC:           &FeatureState{Enabled: _true},
				SyNICTimer:      &SyNICTimer{Enabled: _false},
				Reset:           &FeatureState{Enabled: _true},
				VendorID:        &FeatureVendorID{Enabled: _false},
				Frequencies:     &FeatureState{Enabled: _false},
				Reenlightenment: &FeatureState{Enabled: _false},
				TLBFlush:        &FeatureState{Enabled: _true},
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
		Expect(*hyperv.Frequencies.Enabled).To(BeFalse())
		Expect(*hyperv.Reenlightenment.Enabled).To(BeFalse())
		Expect(*hyperv.TLBFlush.Enabled).To(BeTrue())
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

		Expect(disks[0].CDRom.Tray).To(Equal(TrayStateClosed), "Default tray state for CDROM should be closed")
		Expect(*disks[0].CDRom.ReadOnly).To(BeTrue(), "Default ReadOnly state for CDROM should be true")
		Expect(disks[0].DedicatedIOThread).To(BeNil(), "Default DedicatedIOThread state should be nil")
		Expect(disks[1].CDRom.Tray).To(Equal(TrayStateOpen), "Tray state was explicitly set to open")
		Expect(*disks[1].CDRom.ReadOnly).To(BeFalse(), "ReadOnly state was explicitly set to true")
		Expect(disks[1].DedicatedIOThread).To(BeNil(), "Default DedicatedIOThread state should be nil")
		Expect(disks[2].Floppy.Tray).To(Equal(TrayStateClosed), "Default tray state for Floppy should be closed")
		Expect(disks[2].Floppy.ReadOnly).To(BeFalse(), "Default ReadOnly state for Floppy should be false")
		Expect(disks[2].DedicatedIOThread).To(BeNil(), "Default DedicatedIOThread state should be nil")
		Expect(disks[3].Floppy.Tray).To(Equal(TrayStateOpen), "TrayState was explicitly set to open")
		Expect(disks[3].Floppy.ReadOnly).To(BeTrue(), "ReadOnly was explicitly set to true")
		Expect(disks[3].DedicatedIOThread).To(BeNil(), "Default DedicatedIOThread state should be nil")
		Expect(disks[4].Disk).ToNot(BeNil(), "Default type should be Disk")
		Expect(disks[4].DedicatedIOThread).To(BeNil(), "Default DedicatedIOThread state should be nil")
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

	It("should omit IOThreads by default", func() {
		vmi := &VirtualMachineInstance{}
		SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(vmi.Spec.Domain.IOThreadsPolicy).To(BeNil(), "Default IOThreadsPolicy should be nil")
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
