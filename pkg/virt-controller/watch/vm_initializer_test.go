package watch

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("VM Initializer", func() {
	Context("Utility tests", func() {
		It("Should map device names", func() {
			dev := v1.DiskDevice{}
			dev.Disk = &v1.DiskTarget{Device: "diskdevice"}
			result := diskDeviceToDeviceName(dev)
			Expect(result).To(Equal("diskdevice"))

			dev = v1.DiskDevice{}
			dev.LUN = &v1.LunTarget{Device: "lundevice"}
			result = diskDeviceToDeviceName(dev)
			Expect(result).To(Equal("lundevice"))

			dev = v1.DiskDevice{}
			dev.Floppy = &v1.FloppyTarget{Device: "floppydevice"}
			result = diskDeviceToDeviceName(dev)
			Expect(result).To(Equal("floppydevice"))

			dev = v1.DiskDevice{}
			dev.CDRom = &v1.CDRomTarget{Device: "cdromdevice"}
			result = diskDeviceToDeviceName(dev)
			Expect(result).To(Equal("cdromdevice"))

			dev = v1.DiskDevice{}
			result = diskDeviceToDeviceName(dev)
			Expect(result).To(Equal(""))
		})
	})

	Context("Annotate Presets", func() {
		It("should properly annotate a VM", func() {
			vm := v1.VirtualMachine{}
			preset := v1.VirtualMachinePreset{}
			preset.ObjectMeta.Name = "test-preset"
			presets := append([]v1.VirtualMachinePreset{}, preset)

			err := annotateVM(&vm, presets)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(vm.Annotations)).To(Equal(1))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal(v1.GroupVersion.String()))
		})

		It("should allow multiple annotations", func() {
			vm := v1.VirtualMachine{}
			preset := v1.VirtualMachinePreset{}
			preset.ObjectMeta.Name = "preset-foo"
			presets := append([]v1.VirtualMachinePreset{}, preset)
			preset = v1.VirtualMachinePreset{}
			preset.ObjectMeta.Name = "preset-bar"

			presets = append(presets, preset)

			err := annotateVM(&vm, presets)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(vm.Annotations)).To(Equal(2))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/preset-foo"]).To(Equal(v1.GroupVersion.String()))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/preset-bar"]).To(Equal(v1.GroupVersion.String()))
		})
	})

	Context("Initializer Marking", func() {
		var thisInitializer k8smetav1.Initializer
		var initializer1 k8smetav1.Initializer
		var initializer2 k8smetav1.Initializer
		var initName1 string
		var initName2 string

		BeforeEach(func() {
			initName1 = "test.initializer.1"
			initName2 = "test.initializer.2"
			thisInitializer = k8smetav1.Initializer{Name: initializerMarking}
			initializer1 = k8smetav1.Initializer{Name: initName1}
			initializer2 = k8smetav1.Initializer{Name: initName2}
		})

		It("Should handle nil initializers", func() {
			vm := v1.VirtualMachine{}
			// sanity check that the Initializers array is indeed nil (for testing)
			Expect(vm.Initializers).To(BeNil())
			removeInitializer(&vm)
		})

		It("Should not modify an empty array", func() {
			vm := v1.VirtualMachine{}
			vm.Initializers = new(k8smetav1.Initializers)
			Expect(len(vm.Initializers.Pending)).To(Equal(0))
			removeInitializer(&vm)
			Expect(len(vm.Initializers.Pending)).To(Equal(0))
		})

		It("Should not modify an array without the correct initializer marking", func() {
			vm := v1.VirtualMachine{}
			vm.Initializers = new(k8smetav1.Initializers)
			vm.Initializers.Pending = []k8smetav1.Initializer{initializer1}
			removeInitializer(&vm)
			Expect(len(vm.Initializers.Pending)).To(Equal(1))
		})

		It("Should remove the correct initializer marking", func() {
			vm := v1.VirtualMachine{}
			vm.Initializers = new(k8smetav1.Initializers)
			vm.Initializers.Pending = []k8smetav1.Initializer{thisInitializer}
			removeInitializer(&vm)
			Expect(len(vm.Initializers.Pending)).To(Equal(0))
		})

		It("Should preserve the rest of the list", func() {
			vm := v1.VirtualMachine{}
			vm.Initializers = new(k8smetav1.Initializers)
			vm.Initializers.Pending = []k8smetav1.Initializer{
				initializer1,
				thisInitializer,
				initializer2}
			removeInitializer(&vm)
			Expect(len(vm.Initializers.Pending)).To(Equal(2))
			Expect(vm.Initializers.Pending[0].Name).To(Equal(initName1))
			Expect(vm.Initializers.Pending[1].Name).To(Equal(initName2))
		})

		It("Should recognize a nil initializer", func() {
			vm := v1.VirtualMachine{}
			vm.Initializers = nil
			Expect(isInitialized(&vm)).To(Equal(true))
		})

		It("Should recognize an empty initializer", func() {
			vm := v1.VirtualMachine{}
			vm.Initializers = new(k8smetav1.Initializers)
			Expect(isInitialized(&vm)).To(Equal(true))
		})

		It("Should return false if initializer marking is present", func() {
			vm := v1.VirtualMachine{}
			vm.Initializers = new(k8smetav1.Initializers)
			vm.Initializers.Pending = []k8smetav1.Initializer{initializer1, thisInitializer, initializer2}
			Expect(isInitialized(&vm)).To(Equal(false))
		})

		It("Should return true for missing initializer", func() {
			vm := v1.VirtualMachine{}
			vm.Initializers = new(k8smetav1.Initializers)
			vm.Initializers.Pending = []k8smetav1.Initializer{initializer1, initializer2}
			Expect(isInitialized(&vm)).To(Equal(true))
		})
	})

	Context("Conflict detection", func() {
		var vm v1.VirtualMachine
		var preset v1.VirtualMachinePreset
		truthy := true
		falsy := false

		memory, _ := resource.ParseQuantity("64M")

		BeforeEach(func() {
			vm = v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Domain: v1.DomainSpec{
				Resources: v1.ResourceRequirements{Requests: k8sv1.ResourceList{
					"memory": memory,
				}},
				CPU:      &v1.CPU{Cores: 4},
				Firmware: &v1.Firmware{UUID: types.UID("11112222-3333-4444-5555-666677778888")},
				Clock: &v1.Clock{ClockOffset: v1.ClockOffset{},
					Timer: &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyDelay}},
				},
				Features: &v1.Features{ACPI: v1.FeatureState{Enabled: &truthy},
					APIC:   &v1.FeatureState{Enabled: &falsy},
					Hyperv: &v1.FeatureHyperv{},
				},
				Devices: v1.Devices{
					Watchdog: &v1.Watchdog{Name: "testcase",
						WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionReset}}},
					Disks: []v1.Disk{v1.Disk{Name: "testdisk",
						VolumeName: "testvolume",
						DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Device: "/dev/vda", ReadOnly: true}}}}},
			}}}
			preset = v1.VirtualMachinePreset{Spec: v1.VirtualMachinePresetSpec{Domain: &v1.DomainSpec{}}}
		})

		It("Should detect CPU conflicts", func() {
			// Check without and then with a CPU conflict
			err := checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			preset.Spec.Domain.CPU = &v1.CPU{Cores: 8}
			err = checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			preset.Spec.Domain.CPU = &v1.CPU{Cores: 4}
			err = checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			presets := []v1.VirtualMachinePreset{preset}
			err = applyPresets(&vm, presets)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should detect Resource conflicts", func() {
			memory128, _ := resource.ParseQuantity("128M")
			preset.Spec.Domain.Resources = v1.ResourceRequirements{Requests: k8sv1.ResourceList{
				"memory": memory128,
			}}

			err := checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			preset.Spec.Domain.Resources = v1.ResourceRequirements{Requests: k8sv1.ResourceList{
				"memory": memory,
			}}

			err = checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should detect Firmware conflicts", func() {
			preset.Spec.Domain.Firmware = &v1.Firmware{UUID: types.UID("88887777-6666-5555-4444-333322221111")}

			err := checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			preset.Spec.Domain.Firmware = &v1.Firmware{UUID: types.UID("11112222-3333-4444-5555-666677778888")}

			err = checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should detect Clock conflicts", func() {
			preset.Spec.Domain.Clock = &v1.Clock{ClockOffset: v1.ClockOffset{},
				Timer: &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyCatchup}},
			}

			err := checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			preset.Spec.Domain.Clock = &v1.Clock{ClockOffset: v1.ClockOffset{},
				Timer: &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyDelay}},
			}

			err = checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should detect Feature conflicts", func() {
			preset.Spec.Domain.Features = &v1.Features{ACPI: v1.FeatureState{Enabled: &falsy}}
			err := checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			preset.Spec.Domain.Features = &v1.Features{ACPI: v1.FeatureState{Enabled: &truthy},
				APIC:   &v1.FeatureState{Enabled: &falsy},
				Hyperv: &v1.FeatureHyperv{},
			}
			err = checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should detect Watchdog conflicts", func() {
			preset.Spec.Domain.Devices.Watchdog = &v1.Watchdog{Name: "foo", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionPoweroff}}}
			err := checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			preset.Spec.Domain.Devices.Watchdog = &v1.Watchdog{Name: "testcase", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionReset}}}

			err = checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should detect Disk conflicts", func() {
			matchingDisk := v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Device: "/dev/vda", ReadOnly: true}}}
			sameName := v1.Disk{Name: "testdisk", VolumeName: "wrong", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Device: "/dev/vdb", ReadOnly: true}}}
			sameVolume := v1.Disk{Name: "randomname", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Device: "/dev/vdb", ReadOnly: true}}}
			sameMountPoint := v1.Disk{Name: "wrongname", VolumeName: "different", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Device: "/dev/vda", ReadOnly: true}}}
			unrelated := v1.Disk{Name: "wrongname", VolumeName: "different", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Device: "/dev/vdb", ReadOnly: true}}}

			// First check everything works if all fields match
			preset.Spec.Domain.Devices.Disks = []v1.Disk{matchingDisk, unrelated}
			err := checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			// Name field matches, but nothing else
			preset.Spec.Domain.Devices.Disks = []v1.Disk{sameName, unrelated}
			err = checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			// Two devices using the same volume name
			preset.Spec.Domain.Devices.Disks = []v1.Disk{sameVolume, unrelated}
			err = checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			// Two devices using the same mount point
			preset.Spec.Domain.Devices.Disks = []v1.Disk{sameMountPoint, unrelated}
			err = checkPresetMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Apply Presets", func() {
		var vm v1.VirtualMachine
		var preset v1.VirtualMachinePreset
		truthy := true
		falsy := false

		BeforeEach(func() {
			vm = v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Domain: v1.DomainSpec{}}}
			vm.ObjectMeta.Name = "testvm"
			preset = v1.VirtualMachinePreset{Spec: v1.VirtualMachinePresetSpec{Domain: &v1.DomainSpec{}}}
			preset.ObjectMeta.Name = "test-preset"
		})

		It("Should apply CPU settings", func() {
			preset.Spec.Domain.CPU = &v1.CPU{Cores: 4}
			presets := []v1.VirtualMachinePreset{preset}
			err := applyPresets(&vm, presets)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Domain.CPU.Cores).To(Equal(uint32(4)))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
		})

		It("Should apply Resources", func() {
			memory, _ := resource.ParseQuantity("64M")
			preset.Spec.Domain.Resources = v1.ResourceRequirements{Requests: k8sv1.ResourceList{
				"memory": memory,
			}}
			presets := []v1.VirtualMachinePreset{preset}
			err := applyPresets(&vm, presets)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Domain.Resources.Requests["memory"]).To(Equal(memory))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
		})

		It("Should apply Firmware settings", func() {
			uuid := types.UID("11112222-3333-4444-5555-666677778888")
			preset.Spec.Domain.Firmware = &v1.Firmware{UUID: uuid}

			presets := []v1.VirtualMachinePreset{preset}
			err := applyPresets(&vm, presets)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Domain.Firmware.UUID).To(Equal(uuid))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
		})

		It("Should apply Clock settings", func() {
			clock := &v1.Clock{ClockOffset: v1.ClockOffset{},
				Timer: &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyDelay}},
			}
			preset.Spec.Domain.Clock = clock

			presets := []v1.VirtualMachinePreset{preset}
			err := applyPresets(&vm, presets)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Domain.Clock).To(Equal(clock))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
		})

		It("Should apply Feature settings", func() {
			features := &v1.Features{ACPI: v1.FeatureState{Enabled: &truthy},
				APIC:   &v1.FeatureState{Enabled: &falsy},
				Hyperv: &v1.FeatureHyperv{},
			}

			preset.Spec.Domain.Features = features

			presets := []v1.VirtualMachinePreset{preset}
			err := applyPresets(&vm, presets)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Domain.Features).To(Equal(features))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
		})

		It("Should apply Watchdog settings", func() {
			watchdog := &v1.Watchdog{Name: "testcase", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionReset}}}

			preset.Spec.Domain.Devices.Watchdog = watchdog

			presets := []v1.VirtualMachinePreset{preset}
			err := applyPresets(&vm, presets)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Domain.Devices.Watchdog).To(Equal(watchdog))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
		})

		It("Should apply Disk devices", func() {
			referenceDisk := v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Device: "/dev/vda", ReadOnly: true}}}

			preset.Spec.Domain.Devices.Disks = []v1.Disk{referenceDisk}

			presets := []v1.VirtualMachinePreset{preset}
			err := applyPresets(&vm, presets)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(vm.Spec.Domain.Devices.Disks)).To(Equal(1))
			Expect(vm.Spec.Domain.Devices.Disks[0].Name).To(Equal("testdisk"))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
		})

		It("Should not duplicate Disks", func() {
			referenceDisk := v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Device: "/dev/vda", ReadOnly: true}}}

			// Both preset and VM will have the same disk defined
			preset.Spec.Domain.Devices.Disks = []v1.Disk{referenceDisk}
			vm.Spec.Domain.Devices.Disks = []v1.Disk{referenceDisk}

			presets := []v1.VirtualMachinePreset{preset}
			err := applyPresets(&vm, presets)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(vm.Spec.Domain.Devices.Disks)).To(Equal(1))
			Expect(vm.Spec.Domain.Devices.Disks[0].Name).To(Equal("testdisk"))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
		})
	})

	Context("Filter Matching", func() {
		var vm v1.VirtualMachine
		var matchingPreset v1.VirtualMachinePreset
		var nonmatchingPreset v1.VirtualMachinePreset
		var errorPreset v1.VirtualMachinePreset
		matchingLabel := k8smetav1.LabelSelector{MatchLabels: map[string]string{"flavor": "matching"}}
		mismatchLabel := k8smetav1.LabelSelector{MatchLabels: map[string]string{"flavor": "unrelated"}}
		errorLabel := k8smetav1.LabelSelector{MatchLabels: map[string]string{"flavor": "!"}}

		BeforeEach(func() {
			vm = v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Domain: v1.DomainSpec{}}}
			vm.ObjectMeta.Name = "testvm"
			vm.ObjectMeta.Labels = map[string]string{"flavor": "matching"}

			matchingPreset = v1.VirtualMachinePreset{Spec: v1.VirtualMachinePresetSpec{Domain: &v1.DomainSpec{}}}
			matchingPreset.ObjectMeta.Name = "test-preset"
			matchingPreset.Spec.Selector = matchingLabel

			nonmatchingPreset = v1.VirtualMachinePreset{Spec: v1.VirtualMachinePresetSpec{Domain: &v1.DomainSpec{}}}
			nonmatchingPreset.ObjectMeta.Name = "unrelated-preset"
			nonmatchingPreset.Spec.Selector = mismatchLabel

			errorPreset = v1.VirtualMachinePreset{Spec: v1.VirtualMachinePresetSpec{Domain: &v1.DomainSpec{}}}
			errorPreset.ObjectMeta.Name = "broken-preset"
			errorPreset.Spec.Selector = errorLabel
		})

		It("Should match preset with the correct selector", func() {
			allPresets := []v1.VirtualMachinePreset{matchingPreset, nonmatchingPreset}
			matchingPresets, err := filterPresets(allPresets, &vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(matchingPresets)).To(Equal(1))
			Expect(matchingPresets[0].Name).To(Equal("test-preset"))
		})

		It("Should reject bogus selectors", func() {
			allPresets := []v1.VirtualMachinePreset{matchingPreset, nonmatchingPreset, errorPreset}
			matching, err := filterPresets(allPresets, &vm)
			Expect(err).To(HaveOccurred())
			Expect(matching).To(BeNil())
		})
	})
})

func NewInitializer(name string) k8smetav1.Initializer {
	return k8smetav1.Initializer{Name: name}
}

func TestLogging(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VM Initializer")
}
