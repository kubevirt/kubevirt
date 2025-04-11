package api

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Defaults", func() {

	It("should leave the scheduler name unset by default", func() {
		vmi := &v1.VirtualMachineInstance{}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(vmi.Spec.SchedulerName).To(BeEmpty())
	})

	It("should take a custom scheduler if specified", func() {
		vmi := &v1.VirtualMachineInstance{Spec: v1.VirtualMachineInstanceSpec{SchedulerName: "custom-one"}}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(vmi.Spec.SchedulerName).To(Equal("custom-one"))
	})

	It("should add ACPI feature if it is unspecified", func() {
		vmi := &v1.VirtualMachineInstance{}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(*vmi.Spec.Domain.Features.ACPI.Enabled).To(BeTrue())
	})

	It("should not add non-ACPI feature by default", func() {
		vmi := &v1.VirtualMachineInstance{}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(vmi.Spec.Domain.Features.APIC).To(BeNil())
		Expect(vmi.Spec.Domain.Features.Hyperv).To(BeNil())
	})

	It("should not add SMM feature if it is unspecified", func() {
		vmi := &v1.VirtualMachineInstance{}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(vmi.Spec.Domain.Features.SMM).To(BeNil())
	})

	It("should default to true to all defined features", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{},
			},
		}
		vmi.Spec.Domain.Features = &v1.Features{
			ACPI: v1.FeatureState{},
			SMM:  &v1.FeatureState{},
			APIC: &v1.FeatureAPIC{},
			Hyperv: &v1.FeatureHyperv{
				Relaxed:         &v1.FeatureState{},
				VAPIC:           &v1.FeatureState{},
				Spinlocks:       &v1.FeatureSpinlocks{},
				VPIndex:         &v1.FeatureState{},
				Runtime:         &v1.FeatureState{},
				SyNIC:           &v1.FeatureState{},
				SyNICTimer:      &v1.SyNICTimer{},
				Reset:           &v1.FeatureState{},
				VendorID:        &v1.FeatureVendorID{},
				Frequencies:     &v1.FeatureState{},
				Reenlightenment: &v1.FeatureState{},
				TLBFlush:        &v1.FeatureState{},
			},
		}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)

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
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{},
			},
		}
		vmi.Spec.Domain.Features = &v1.Features{
			ACPI: v1.FeatureState{Enabled: pointer.BoolPtr(true)},
			APIC: &v1.FeatureAPIC{Enabled: pointer.BoolPtr(false)},
			Hyperv: &v1.FeatureHyperv{
				Relaxed:         &v1.FeatureState{Enabled: pointer.BoolPtr(true)},
				VAPIC:           &v1.FeatureState{Enabled: pointer.BoolPtr(false)},
				Spinlocks:       &v1.FeatureSpinlocks{Enabled: pointer.BoolPtr(true)},
				VPIndex:         &v1.FeatureState{Enabled: pointer.BoolPtr(false)},
				Runtime:         &v1.FeatureState{Enabled: pointer.BoolPtr(true)},
				SyNIC:           &v1.FeatureState{Enabled: pointer.BoolPtr(false)},
				SyNICTimer:      &v1.SyNICTimer{Enabled: pointer.BoolPtr(true), Direct: &v1.FeatureState{Enabled: pointer.BoolPtr(true)}},
				Reset:           &v1.FeatureState{Enabled: pointer.BoolPtr(false)},
				VendorID:        &v1.FeatureVendorID{Enabled: pointer.BoolPtr(true)},
				Frequencies:     &v1.FeatureState{Enabled: pointer.BoolPtr(false)},
				Reenlightenment: &v1.FeatureState{Enabled: pointer.BoolPtr(false)},
				TLBFlush:        &v1.FeatureState{Enabled: pointer.BoolPtr(true)},
			},
		}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)

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

		vmi.Spec.Domain.Features = &v1.Features{
			ACPI: v1.FeatureState{Enabled: pointer.BoolPtr(false)},
			APIC: &v1.FeatureAPIC{Enabled: pointer.BoolPtr(true)},
			Hyperv: &v1.FeatureHyperv{
				Relaxed:         &v1.FeatureState{Enabled: pointer.BoolPtr(false)},
				VAPIC:           &v1.FeatureState{Enabled: pointer.BoolPtr(true)},
				Spinlocks:       &v1.FeatureSpinlocks{Enabled: pointer.BoolPtr(false)},
				VPIndex:         &v1.FeatureState{Enabled: pointer.BoolPtr(true)},
				Runtime:         &v1.FeatureState{Enabled: pointer.BoolPtr(false)},
				SyNIC:           &v1.FeatureState{Enabled: pointer.BoolPtr(true)},
				SyNICTimer:      &v1.SyNICTimer{Enabled: pointer.BoolPtr(false)},
				Reset:           &v1.FeatureState{Enabled: pointer.BoolPtr(true)},
				VendorID:        &v1.FeatureVendorID{Enabled: pointer.BoolPtr(false)},
				Frequencies:     &v1.FeatureState{Enabled: pointer.BoolPtr(false)},
				Reenlightenment: &v1.FeatureState{Enabled: pointer.BoolPtr(false)},
				TLBFlush:        &v1.FeatureState{Enabled: pointer.BoolPtr(true)},
			},
		}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)

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
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Disks: []v1.Disk{
							{
								Name: "cdrom_tray_unspecified",
								DiskDevice: v1.DiskDevice{
									CDRom: &v1.CDRomTarget{},
								},
							},
							{
								Name: "cdrom_tray_open",
								DiskDevice: v1.DiskDevice{
									CDRom: &v1.CDRomTarget{
										Tray:     v1.TrayStateOpen,
										ReadOnly: pointer.BoolPtr(false),
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
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		disks := vmi.Spec.Domain.Devices.Disks

		Expect(disks[0].CDRom.Tray).To(Equal(v1.TrayStateClosed), "Default tray state for CDROM should be closed")
		Expect(*disks[0].CDRom.ReadOnly).To(BeTrue(), "Default ReadOnly state for CDROM should be true")
		Expect(disks[0].DedicatedIOThread).To(BeNil(), "Default DedicatedIOThread state should be nil")
		Expect(disks[1].CDRom.Tray).To(Equal(v1.TrayStateOpen), "Tray state was explicitly set to open")
		Expect(*disks[1].CDRom.ReadOnly).To(BeFalse(), "ReadOnly state was explicitly set to true")
		Expect(disks[1].DedicatedIOThread).To(BeNil(), "Default DedicatedIOThread state should be nil")
		Expect(disks[2].Disk).ToNot(BeNil(), "Default type should be Disk")
		Expect(disks[2].DedicatedIOThread).To(BeNil(), "Default DedicatedIOThread state should be nil")
	})

	It("should set timer defaults", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Clock: &v1.Clock{
						Timer: &v1.Timer{
							HPET:   &v1.HPETTimer{},
							KVM:    &v1.KVMTimer{},
							PIT:    &v1.PITTimer{},
							RTC:    &v1.RTCTimer{},
							Hyperv: &v1.HypervTimer{},
						},
					},
				},
			},
		}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		timer := vmi.Spec.Domain.Clock.Timer
		Expect(*timer.HPET.Enabled).To(BeTrue())
		Expect(*timer.KVM.Enabled).To(BeTrue())
		Expect(*timer.PIT.Enabled).To(BeTrue())
		Expect(*timer.RTC.Enabled).To(BeTrue())
		Expect(*timer.Hyperv.Enabled).To(BeTrue())
	})

	It("should omit IOThreads by default", func() {
		vmi := &v1.VirtualMachineInstance{}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		Expect(vmi.Spec.Domain.IOThreadsPolicy).To(BeNil(), "Default IOThreadsPolicy should be nil")
	})

	It("should default probes", func() {
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				ReadinessProbe: &v1.Probe{},
				LivenessProbe:  &v1.Probe{},
			},
		}
		v1.SetDefaults_VirtualMachineInstance(vmi)

		validateProbe := func(probe *v1.Probe) {
			Expect(probe.TimeoutSeconds).To(BeEquivalentTo(1))
			Expect(probe.PeriodSeconds).To(BeEquivalentTo(10))
			Expect(probe.SuccessThreshold).To(BeEquivalentTo(1))
			Expect(probe.FailureThreshold).To(BeEquivalentTo(3))
		}
		validateProbe(vmi.Spec.ReadinessProbe)
		validateProbe(vmi.Spec.LivenessProbe)
	})
})
