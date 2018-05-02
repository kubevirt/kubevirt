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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package watch

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

type Event struct {
	object    runtime.Object
	eventtype string
	reason    string
	message   string
}

type Events struct {
	eventList []Event
}

type FakeRecorder struct {
	events *Events
}

func NewFakeRecorder() FakeRecorder {
	return FakeRecorder{events: &Events{}}
}

func (recorder FakeRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	event := Event{
		object:    object,
		eventtype: eventtype,
		reason:    reason,
		message:   message,
	}
	Expect(recorder.events).ToNot(BeNil())
	recorder.events.eventList = append(recorder.events.eventList, event)
}

func (recorder FakeRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	msg := fmt.Sprintf(messageFmt, args...)
	recorder.Event(object, eventtype, reason, msg)
}

func (recorder FakeRecorder) PastEventf(object runtime.Object, timestamp k8smetav1.Time, eventtype, reason, messageFmt string, args ...interface{}) {
	recorder.Eventf(object, eventtype, reason, messageFmt, args...)
}

var _ = Describe("VM Initializer", func() {
	Context("Annotate Presets", func() {
		It("should properly annotate a VM", func() {
			vm := v1.VirtualMachine{}
			preset := v1.VirtualMachinePreset{}
			preset.ObjectMeta.Name = "test-preset"

			annotateVM(&vm, preset)
			Expect(len(vm.Annotations)).To(Equal(1))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal(v1.GroupVersion.String()))
		})

		It("should allow multiple annotations", func() {
			vm := v1.VirtualMachine{}
			preset := v1.VirtualMachinePreset{}
			preset.ObjectMeta.Name = "preset-foo"
			annotateVM(&vm, preset)
			preset = v1.VirtualMachinePreset{}
			preset.ObjectMeta.Name = "preset-bar"
			annotateVM(&vm, preset)

			Expect(len(vm.Annotations)).To(Equal(2))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/preset-foo"]).To(Equal(v1.GroupVersion.String()))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/preset-bar"]).To(Equal(v1.GroupVersion.String()))
		})
	})

	Context("Initializer Marking", func() {
		It("Should handle nil Annotations", func() {
			vm := &v1.VirtualMachine{}
			vm.Annotations = nil
			Expect(isVirtualMachineInitialized(vm)).To(BeFalse())
			addInitializedAnnotation(vm)
			Expect(isVirtualMachineInitialized(vm)).To(BeTrue())
		})

		It("Should handle empty Annotations", func() {
			vm := &v1.VirtualMachine{}
			vm.Annotations = map[string]string{}
			Expect(isVirtualMachineInitialized(vm)).To(BeFalse())
			addInitializedAnnotation(vm)
			Expect(isVirtualMachineInitialized(vm)).To(BeTrue())
		})

		It("Should not modify already initialized VM's", func() {
			vm := &v1.VirtualMachine{}
			vm.Annotations = map[string]string{}
			vm.Annotations[initializerMarking] = v1.GroupVersion.String()
			Expect(isVirtualMachineInitialized(vm)).To(BeTrue())
			// call addInitializedAnnotation
			addInitializedAnnotation(vm)
			Expect(isVirtualMachineInitialized(vm)).To(BeTrue())
		})
	})

	Context("Preset Exclusions", func() {
		It("Should not fail if Annotations are nil", func() {
			vm := &v1.VirtualMachine{}
			vm.Annotations = nil
			Expect(arePresetsExcluded(vm)).To(BeFalse())
		})

		It("Should should not fail if Annotations are empty", func() {
			vm := &v1.VirtualMachine{}
			vm.Annotations = map[string]string{}
			Expect(arePresetsExcluded(vm)).To(BeFalse())
		})

		It("Should identify incorrect exclusion marking", func() {
			vm := &v1.VirtualMachine{}
			vm.Annotations = map[string]string{}
			vm.Annotations[exclusionMarking] = "something random"
			Expect(arePresetsExcluded(vm)).To(BeFalse())
		})

		It("Should identify exclusion marking", func() {
			vm := &v1.VirtualMachine{}
			vm.Annotations = map[string]string{}
			vm.Annotations[exclusionMarking] = "true"
			Expect(arePresetsExcluded(vm)).To(BeTrue())
		})
	})

	Context("Override detection", func() {
		var vm v1.VirtualMachine
		var preset v1.VirtualMachinePreset
		truthy := true
		falsy := false
		var recorder FakeRecorder

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
					APIC:   &v1.FeatureAPIC{Enabled: &falsy},
					Hyperv: &v1.FeatureHyperv{},
				},
				Devices: v1.Devices{
					Watchdog: &v1.Watchdog{Name: "testcase",
						WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionReset}}},
					Disks: []v1.Disk{v1.Disk{Name: "testdisk",
						VolumeName: "testvolume",
						DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: "virtio", ReadOnly: true}}}}},
			}}}
			preset = v1.VirtualMachinePreset{Spec: v1.VirtualMachinePresetSpec{Domain: &v1.DomainSpec{}}}
			recorder = NewFakeRecorder()
		})

		It("Should detect CPU overrides", func() {
			// Check without and then with a CPU conflict
			err := checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing no merge conflict occurs for matching preset")
			preset.Spec.Domain.CPU = &v1.CPU{Cores: 4}
			err = checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("applying matching preset")
			vm.Annotations = map[string]string{}
			applyPresets(&vm, []v1.VirtualMachinePreset{preset}, recorder)

			By("checking annotations were applied and CPU count remains the same")
			Expect(len(vm.Annotations)).To(Equal(1))
			Expect(int(vm.Spec.Domain.CPU.Cores)).To(Equal(4))
			Expect(len(recorder.events.eventList)).To(Equal(0))

			By("showing an override occured")
			preset.Spec.Domain.CPU = &v1.CPU{Cores: 6}
			err = checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("applying overriden preset")
			vm.Annotations = map[string]string{}
			applyPresets(&vm, []v1.VirtualMachinePreset{preset}, recorder)

			By("checking annotations were not applied and CPU count remains the same")
			Expect(len(vm.Annotations)).To(Equal(0))
			Expect(int(vm.Spec.Domain.CPU.Cores)).To(Equal(4))

			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeNormal))
		})

		It("Should detect Resource overrides", func() {
			memory128, _ := resource.ParseQuantity("128M")
			preset.Spec.Domain.Resources = v1.ResourceRequirements{Requests: k8sv1.ResourceList{
				"memory": memory128,
			}}

			By("demonstrating that override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("applying mismatch preset")
			vm.Annotations = map[string]string{}
			applyPresets(&vm, []v1.VirtualMachinePreset{preset}, recorder)

			By("checking preset was not applied")
			memory, ok := vm.Spec.Domain.Resources.Requests["memory"]
			Expect(ok).To(BeTrue())
			Expect(int(memory.Value())).To(Equal(64000000))
			Expect(len(vm.Annotations)).To(Equal(0))
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeNormal))

			preset.Spec.Domain.Resources = v1.ResourceRequirements{Requests: k8sv1.ResourceList{
				"memory": memory,
			}}

			By("demonstrating that no override occurs")
			err = checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("applying matching preset")
			recorder = NewFakeRecorder()
			vm.Annotations = map[string]string{}
			applyPresets(&vm, []v1.VirtualMachinePreset{preset}, recorder)

			By("checking vm settings remain the same")
			memory, ok = vm.Spec.Domain.Resources.Requests["memory"]
			Expect(ok).To(BeTrue())
			Expect(int(memory.Value())).To(Equal(64000000))
			Expect(len(vm.Annotations)).To(Equal(1))
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should detect Firmware overrides", func() {
			mismatchUuid := types.UID("88887777-6666-5555-4444-333322221111")
			matchUuid := types.UID("11112222-3333-4444-5555-666677778888")

			preset.Spec.Domain.Firmware = &v1.Firmware{UUID: mismatchUuid}

			By("showing that an override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("showing that presets are not applied")
			vm.Annotations = map[string]string{}
			applyPresets(&vm, []v1.VirtualMachinePreset{preset}, recorder)

			Expect(len(vm.Annotations)).To(Equal(0))
			Expect(vm.Spec.Domain.Firmware.UUID).To(Equal(matchUuid))
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeNormal))

			preset.Spec.Domain.Firmware = &v1.Firmware{UUID: matchUuid}

			By("showing that an override does not occur")
			err = checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing settings did not change when preset is applied")
			recorder = NewFakeRecorder()
			vm.Annotations = map[string]string{}
			applyPresets(&vm, []v1.VirtualMachinePreset{preset}, recorder)

			Expect(len(vm.Annotations)).To(Equal(1))
			Expect(vm.Spec.Domain.Firmware.UUID).To(Equal(matchUuid))
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should detect Clock overrides", func() {
			preset.Spec.Domain.Clock = &v1.Clock{ClockOffset: v1.ClockOffset{},
				Timer: &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyCatchup}},
			}

			By("showing that an override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("showing presets are not applied")
			vm.Annotations = map[string]string{}
			applyPresets(&vm, []v1.VirtualMachinePreset{preset}, recorder)

			Expect(len(vm.Annotations)).To(Equal(0))
			Expect(vm.Spec.Domain.Clock.Timer.HPET.TickPolicy).To(Equal(v1.HPETTickPolicyDelay))
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeNormal))

			preset.Spec.Domain.Clock = &v1.Clock{ClockOffset: v1.ClockOffset{},
				Timer: &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyDelay}},
			}

			By("showing that an ovveride does not occur")
			err = checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing settings were not changed")
			recorder = NewFakeRecorder()
			vm.Annotations = map[string]string{}
			applyPresets(&vm, []v1.VirtualMachinePreset{preset}, recorder)

			Expect(len(vm.Annotations)).To(Equal(1))
			Expect(vm.Spec.Domain.Clock.Timer.HPET.TickPolicy).To(Equal(v1.HPETTickPolicyDelay))
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should detect Feature overrides", func() {
			preset.Spec.Domain.Features = &v1.Features{ACPI: v1.FeatureState{Enabled: &falsy}}

			By("showing that an override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("showing presets are not applied")
			vm.Annotations = map[string]string{}
			applyPresets(&vm, []v1.VirtualMachinePreset{preset}, recorder)

			Expect(len(vm.Annotations)).To(Equal(0))
			Expect(*vm.Spec.Domain.Features.ACPI.Enabled).To(BeTrue())
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeNormal))

			preset.Spec.Domain.Features = &v1.Features{ACPI: v1.FeatureState{Enabled: &truthy},
				APIC:   &v1.FeatureAPIC{Enabled: &falsy},
				Hyperv: &v1.FeatureHyperv{},
			}

			By("showing that an ovveride does not occur")
			err = checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing settings were not changed")
			recorder = NewFakeRecorder()
			vm.Annotations = map[string]string{}
			applyPresets(&vm, []v1.VirtualMachinePreset{preset}, recorder)

			Expect(len(vm.Annotations)).To(Equal(1))
			Expect(*vm.Spec.Domain.Features.ACPI.Enabled).To(BeTrue())
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should detect Watchdog overrides", func() {
			preset.Spec.Domain.Devices.Watchdog = &v1.Watchdog{Name: "foo", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionPoweroff}}}

			By("showing that an override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("showing presets are not applied")
			vm.Annotations = map[string]string{}
			applyPresets(&vm, []v1.VirtualMachinePreset{preset}, recorder)

			Expect(len(vm.Annotations)).To(Equal(0))
			Expect(vm.Spec.Domain.Devices.Watchdog.Name).To(Equal("testcase"))
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeNormal))

			preset.Spec.Domain.Devices.Watchdog = &v1.Watchdog{Name: "testcase", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionReset}}}

			By("showing that an ovveride does not occur")
			err = checkMergeConflicts(preset.Spec.Domain, &vm.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing settings were not changed")
			recorder = NewFakeRecorder()
			vm.Annotations = map[string]string{}
			applyPresets(&vm, []v1.VirtualMachinePreset{preset}, recorder)

			Expect(len(vm.Annotations)).To(Equal(1))
			Expect(vm.Spec.Domain.Devices.Watchdog.Name).To(Equal("testcase"))
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})
	})

	Context("Conflict detection", func() {
		var vm v1.VirtualMachine
		var preset1 v1.VirtualMachinePreset
		var preset2 v1.VirtualMachinePreset
		var preset3 v1.VirtualMachinePreset
		var preset4 v1.VirtualMachinePreset

		m64, _ := resource.ParseQuantity("64M")
		m128, _ := resource.ParseQuantity("128M")

		var recorder FakeRecorder

		BeforeEach(func() {
			vm = v1.VirtualMachine{ObjectMeta: k8smetav1.ObjectMeta{Name: "test-vm"}}

			preset1 = v1.VirtualMachinePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: "memory-64"},
				Spec: v1.VirtualMachinePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"kubevirt.io/m64": "memory-64"}},
					Domain: &v1.DomainSpec{
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{"memory": m64},
						},
					},
				},
			}
			preset2 = v1.VirtualMachinePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: "memory-128"},
				Spec: v1.VirtualMachinePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"kubevirt.io/m128": "memory-128"}},
					Domain: &v1.DomainSpec{
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{"memory": m128},
						},
					},
				},
			}
			preset3 = v1.VirtualMachinePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: "cpu-4"},
				Spec: v1.VirtualMachinePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"kubevirt.io/cpu": "cpu-4"}},
					Domain: &v1.DomainSpec{
						CPU: &v1.CPU{Cores: 4},
					},
				},
			}
			preset4 = v1.VirtualMachinePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: "duplicate-mem"},
				Spec: v1.VirtualMachinePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"kubevirt.io/m64": "memory-64"}},
					Domain: &v1.DomainSpec{
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{"memory": m64},
						},
					},
				},
			}
			recorder = NewFakeRecorder()
		})

		It("should detect conflicts between presets", func() {
			presets := []v1.VirtualMachinePreset{preset1, preset2}
			err := checkPresetConflicts(presets)
			Expect(err).To(HaveOccurred())
		})

		It("should not return an error for no conflict", func() {
			presets := []v1.VirtualMachinePreset{preset1, preset3}
			err := checkPresetConflicts(presets)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not consider presets with same settings to conflict", func() {
			presets := []v1.VirtualMachinePreset{preset1, preset4}
			err := checkPresetConflicts(presets)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not apply presets that conflict", func() {
			presets := []v1.VirtualMachinePreset{preset1, preset2, preset3, preset4}
			vm.Labels = map[string]string{
				"kubevirt.io/m64":  "memory-64",
				"kubevirt.io/m128": "memory-128",
			}

			By("applying presets")
			res := applyPresets(&vm, presets, recorder)
			Expect(res).To(BeFalse())

			By("checking annotations were not applied")
			annotation, ok := vm.Annotations["virtualmachinepreset.kubevirt.io/memory-64"]
			Expect(annotation).To(Equal(""))
			Expect(ok).To(BeFalse())
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeWarning))

			By("checking settings were not applied to VM")
			Expect(vm.Spec.Domain.Resources.Requests).To(BeNil())
		})

		It("should not apply any presets if any conflict", func() {
			presets := []v1.VirtualMachinePreset{preset1, preset2, preset3, preset4}
			vm.Labels = map[string]string{
				"kubevirt.io/m64":  "memory-64",
				"kubevirt.io/m128": "memory-128",
				"kubevirt.io/cpu":  "cpu-4",
			}

			By("applying presets")
			res := applyPresets(&vm, presets, recorder)
			Expect(res).To(BeFalse())

			By("checking annotations were not applied")
			annotation, ok := vm.Annotations["virtualmachinepreset.kubevirt.io/cpu-4"]
			Expect(annotation).To(Equal(""))
			Expect(ok).To(BeFalse())

			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeWarning))

			By("checking settings were not applied to VM")
			Expect(vm.Spec.Domain.Resources.Requests).To(BeNil())
			Expect(vm.Spec.Domain.CPU).To(BeNil())
		})

		It("should apply presets that don't conflict", func() {
			presets := []v1.VirtualMachinePreset{preset1, preset3, preset4}
			vm.Labels = map[string]string{
				"kubevirt.io/m64": "memory-64",
				"kubevirt.io/cpu": "cpu-4",
			}
			By("applying presets")
			res := applyPresets(&vm, presets, recorder)
			Expect(res).To(BeTrue())

			By("checking annotations were applied")
			annotation, ok := vm.Annotations["virtualmachinepreset.kubevirt.io/memory-64"]
			Expect(annotation).To(Equal("kubevirt.io/v1alpha1"))
			Expect(ok).To(BeTrue())

			annotation, ok = vm.Annotations["virtualmachinepreset.kubevirt.io/cpu-4"]
			Expect(annotation).To(Equal("kubevirt.io/v1alpha1"))
			Expect(ok).To(BeTrue())

			annotation, ok = vm.Annotations["virtualmachinepreset.kubevirt.io/duplicate-mem"]
			Expect(annotation).To(Equal("kubevirt.io/v1alpha1"))
			Expect(ok).To(BeTrue())

			By("checking settings were applied")
			Expect(len(vm.Spec.Domain.Resources.Requests)).To(Equal(1))
			memory := vm.Spec.Domain.Resources.Requests["memory"]
			Expect(int(memory.Value())).To(Equal(64000000))

			Expect(vm.Spec.Domain.CPU).ToNot(BeNil())
			Expect(int(vm.Spec.Domain.CPU.Cores)).To(Equal(4))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})
	})

	Context("Apply Presets", func() {
		var vm v1.VirtualMachine
		var preset v1.VirtualMachinePreset
		truthy := true
		falsy := false
		var recorder FakeRecorder

		BeforeEach(func() {
			vm = v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Domain: v1.DomainSpec{}}}
			vm.ObjectMeta.Name = "testvm"
			preset = v1.VirtualMachinePreset{Spec: v1.VirtualMachinePresetSpec{Domain: &v1.DomainSpec{}}}
			preset.ObjectMeta.Name = "test-preset"
			recorder = NewFakeRecorder()
		})

		It("Should apply CPU settings", func() {
			preset.Spec.Domain.CPU = &v1.CPU{Cores: 4}
			presets := []v1.VirtualMachinePreset{preset}
			applyPresets(&vm, presets, recorder)

			Expect(vm.Spec.Domain.CPU.Cores).To(Equal(uint32(4)))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should apply Resources", func() {
			memory, _ := resource.ParseQuantity("64M")
			preset.Spec.Domain.Resources = v1.ResourceRequirements{Requests: k8sv1.ResourceList{
				"memory": memory,
			}}
			presets := []v1.VirtualMachinePreset{preset}
			applyPresets(&vm, presets, recorder)

			Expect(vm.Spec.Domain.Resources.Requests["memory"]).To(Equal(memory))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should apply Firmware settings", func() {
			uuid := types.UID("11112222-3333-4444-5555-666677778888")
			preset.Spec.Domain.Firmware = &v1.Firmware{UUID: uuid}

			presets := []v1.VirtualMachinePreset{preset}
			applyPresets(&vm, presets, recorder)

			Expect(vm.Spec.Domain.Firmware.UUID).To(Equal(uuid))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should apply Clock settings", func() {
			clock := &v1.Clock{ClockOffset: v1.ClockOffset{},
				Timer: &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyDelay}},
			}
			preset.Spec.Domain.Clock = clock

			presets := []v1.VirtualMachinePreset{preset}
			applyPresets(&vm, presets, recorder)

			Expect(vm.Spec.Domain.Clock).To(Equal(clock))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should apply Feature settings", func() {
			features := &v1.Features{ACPI: v1.FeatureState{Enabled: &truthy},
				APIC:   &v1.FeatureAPIC{Enabled: &falsy},
				Hyperv: &v1.FeatureHyperv{},
			}

			preset.Spec.Domain.Features = features

			presets := []v1.VirtualMachinePreset{preset}
			applyPresets(&vm, presets, recorder)

			Expect(vm.Spec.Domain.Features).To(Equal(features))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should apply Watchdog settings", func() {
			watchdog := &v1.Watchdog{Name: "testcase", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionReset}}}

			preset.Spec.Domain.Devices.Watchdog = watchdog

			presets := []v1.VirtualMachinePreset{preset}
			applyPresets(&vm, presets, recorder)

			Expect(vm.Spec.Domain.Devices.Watchdog).To(Equal(watchdog))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})
	})

	Context("Filter Matching", func() {
		var vm v1.VirtualMachine
		var matchingPreset v1.VirtualMachinePreset
		var nonmatchingPreset v1.VirtualMachinePreset
		var errorPreset v1.VirtualMachinePreset
		matchingPresetName := "test-preset"
		flavorKey := fmt.Sprintf("%s/flavor", v1.GroupName)
		matchingLabel := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: "matching"}}
		mismatchLabel := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: "unrelated"}}
		errorLabel := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: "!"}}

		var recorder FakeRecorder

		BeforeEach(func() {
			vm = v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Domain: v1.DomainSpec{}}}
			vm.ObjectMeta.Name = "testvm"
			vm.ObjectMeta.Labels = map[string]string{flavorKey: "matching"}

			matchingPreset = v1.VirtualMachinePreset{Spec: v1.VirtualMachinePresetSpec{Domain: &v1.DomainSpec{}}}
			matchingPreset.ObjectMeta.Name = matchingPresetName
			matchingPreset.Spec.Selector = matchingLabel

			nonmatchingPreset = v1.VirtualMachinePreset{Spec: v1.VirtualMachinePresetSpec{Domain: &v1.DomainSpec{}}}
			nonmatchingPreset.ObjectMeta.Name = "unrelated-preset"
			nonmatchingPreset.Spec.Selector = mismatchLabel

			errorPreset = v1.VirtualMachinePreset{Spec: v1.VirtualMachinePresetSpec{Domain: &v1.DomainSpec{}}}
			errorPreset.ObjectMeta.Name = "broken-preset"
			errorPreset.Spec.Selector = errorLabel
			recorder = NewFakeRecorder()
		})

		It("Should match preset with the correct selector", func() {
			allPresets := []v1.VirtualMachinePreset{matchingPreset, nonmatchingPreset}
			matchingPresets := filterPresets(allPresets, &vm, recorder)
			Expect(len(matchingPresets)).To(Equal(1))
			Expect(matchingPresets[0].Name).To(Equal(matchingPresetName))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should not match preset with the incorrect selector", func() {
			allPresets := []v1.VirtualMachinePreset{nonmatchingPreset}
			matchingPresets := filterPresets(allPresets, &vm, recorder)
			Expect(len(matchingPresets)).To(Equal(0))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should ignore bogus selectors", func() {
			allPresets := []v1.VirtualMachinePreset{matchingPreset, nonmatchingPreset, errorPreset}
			matchingPresets := filterPresets(allPresets, &vm, recorder)
			Expect(len(matchingPresets)).To(Equal(1))
			Expect(matchingPresets[0].Name).To(Equal(matchingPresetName))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeWarning))
		})

	})

	Context("VM Init Watcher", func() {
		var server *ghttp.Server

		log.Log.SetIOWriter(GinkgoWriter)
		var app = VirtControllerApp{}
		app.launcherImage = "kubevirt/virt-launcher"

		var vmPreset *v1.VirtualMachinePreset
		var stopChan chan struct{}

		flavorKey := fmt.Sprintf("%s/flavor", v1.GroupName)
		presetFlavor := "test-case"

		BeforeEach(func() {
			stopChan = make(chan struct{})

			server = ghttp.NewServer()
			app.clientSet, _ = kubecli.GetKubevirtClientFromFlags(server.URL(), "")
			app.restClient = app.clientSet.RestClient()

			// create a reference preset
			selector := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: presetFlavor}}
			vmPreset = v1.NewVirtualMachinePreset("test-preset", selector)
			vmPreset.Spec.Domain.CPU = &v1.CPU{Cores: 4}
			vmPreset.Spec.Domain.Firmware = &v1.Firmware{UUID: "12345678-1234-1234-1234-123456781234"}

			// create a stock VM

			// Synthesize a fake, but fully functional, vmPresetInformer
			presetListWatch := &cache.ListWatch{
				ListFunc: func(options k8smetav1.ListOptions) (runtime.Object, error) {
					return &v1.VirtualMachinePresetList{Items: []v1.VirtualMachinePreset{*vmPreset}}, nil
				},
				WatchFunc: func(options k8smetav1.ListOptions) (watch.Interface, error) {
					fakeWatch := watch.NewFake()
					fakeWatch.Add(vmPreset)
					return fakeWatch, nil
				},
			}
			app.vmPresetInformer = cache.NewSharedIndexInformer(presetListWatch, &v1.VirtualMachinePreset{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			go app.vmPresetInformer.Run(stopChan)

			// Synthesize a fake vmInformer
			vmListWatch := &cache.ListWatch{
				ListFunc: func(options k8smetav1.ListOptions) (runtime.Object, error) {
					return &v1.VirtualMachineList{}, nil
				},
				WatchFunc: func(options k8smetav1.ListOptions) (watch.Interface, error) {
					return watch.NewFake(), nil
				},
			}
			app.vmInformer = cache.NewSharedIndexInformer(vmListWatch, &v1.VirtualMachine{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			app.podInformer = cache.NewSharedIndexInformer(vmListWatch, &v1.VirtualMachine{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			app.nodeInformer = cache.NewSharedIndexInformer(vmListWatch, &k8sv1.Node{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			app.vmPresetCache = app.vmInformer.GetStore()
			app.vmPresetQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			app.vmInformer.AddEventHandler(controller.NewResourceEventHandlerFuncsForWorkqueue(app.vmPresetQueue))
			go app.vmInformer.Run(stopChan)

			recorder := NewFakeRecorder()
			app.vmPresetRecorder = &recorder

			app.initCommon()
			// Make sure the informers are synced before continuing -- avoid race conditions
			cache.WaitForCacheSync(stopChan, app.vmPresetInformer.HasSynced, app.vmPresetInformer.HasSynced)
		})

		AfterEach(func() {
			close(stopChan)
		})

		// This is a meta-test to ensure the preset cache in this test suite works
		It("should have a result in the fake VM Preset cache", func() {
			presets := app.vmPresetInformer.GetStore().List()
			Expect(len(presets)).To(Equal(1))
			for _, obj := range presets {
				var preset *v1.VirtualMachinePreset
				preset = obj.(*v1.VirtualMachinePreset)
				Expect(preset.Name).To(Equal("test-preset"))
			}

		})

		It("should not process an initialized VM", func() {
			vm := v1.NewMinimalVM("testvm")
			addInitializedAnnotation(vm)
			Expect(isVirtualMachineInitialized(vm)).To(BeTrue())

			key, _ := cache.MetaNamespaceKeyFunc(vm)
			app.vmPresetCache.Add(vm)
			app.vmPresetQueue.Add(key)

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.vmPresetController.Execute()
			// the initializer should inspect the VM and decide nothing needs to be done
			// (and skip the update entirely). So zero requests are expected.
			Expect(len(server.ReceivedRequests())).To(Equal(0))
			Expect(isVirtualMachineInitialized(vm)).To(BeTrue())
			Expect(controller.HasFinalizer(vm, v1.VirtualMachineFinalizer)).To(BeTrue())
		})

		It("should initialize a VM if needed", func() {
			vm := v1.NewMinimalVM("testvm")

			key, _ := cache.MetaNamespaceKeyFunc(vm)
			app.vmPresetCache.Add(vm)
			app.vmPresetQueue.Add(key)

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.vmPresetController.Execute()
			Expect(len(server.ReceivedRequests())).To(Equal(1))

			// We should expect no changes to this VM object--because that would mean
			// there were side effects in the cache.
			Expect(isVirtualMachineInitialized(vm)).To(BeFalse())
		})

		It("should apply presets", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Labels = map[string]string{flavorKey: presetFlavor}

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			err := app.vmPresetController.initializeVirtualMachine(vm)

			Expect(err).ToNot(HaveOccurred())

			Expect(len(server.ReceivedRequests())).To(Equal(1))
			// Prove that the VM was annotated (indicates successful application of preset)
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
		})

		It("should annotate partially applied presets", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Labels = map[string]string{flavorKey: presetFlavor}
			vm.Spec.Domain = v1.DomainSpec{CPU: &v1.CPU{Cores: 6}}

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			err := app.vmPresetController.initializeVirtualMachine(vm)

			Expect(err).ToNot(HaveOccurred())

			Expect(len(server.ReceivedRequests())).To(Equal(1))
			// Prove that the VM was annotated (indicates successful application of preset)
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
		})

		It("should should not annotate presets with no settings successfully applied", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Labels = map[string]string{flavorKey: presetFlavor}
			vm.Spec.Domain = v1.DomainSpec{
				CPU:      &v1.CPU{Cores: 6},
				Firmware: &v1.Firmware{UUID: "11111111-2222-3333-4444-123456781234"}}

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			err := app.vmPresetController.initializeVirtualMachine(vm)

			Expect(err).ToNot(HaveOccurred())

			Expect(len(server.ReceivedRequests())).To(Equal(1))

			_, found := vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]
			Expect(found).To(BeFalse())

			Expect(vm.Status.Phase).ToNot(Equal(v1.Failed))
		})

		It("should not mark a VM without presets as failed", func() {
			vm := v1.NewMinimalVM("testvm")
			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)
			err := app.vmPresetController.initializeVirtualMachine(vm)

			Expect(err).ToNot(HaveOccurred())
			Expect(len(server.ReceivedRequests())).To(Equal(1))

			_, found := vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]
			Expect(found).To(BeFalse())

			Expect(vm.Status.Phase).ToNot(Equal(v1.Failed))
		})

		It("should check if exclusion annotation is \"true\"", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Labels = map[string]string{flavorKey: presetFlavor}
			vm.Annotations = map[string]string{}
			vm.Annotations[exclusionMarking] = "anything"

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			err := app.vmPresetController.initializeVirtualMachine(vm)

			Expect(err).ToNot(HaveOccurred())

			Expect(len(server.ReceivedRequests())).To(Equal(1))
			_, ok := vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]
			Expect(ok).To(BeTrue(), "Preset should applied")

			// Prove that the VM was initialized
			Expect(isVirtualMachineInitialized(vm)).To(BeTrue())
		})

		It("should not add annotations to VM's with exclusion marking", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Labels = map[string]string{flavorKey: presetFlavor}
			vm.Annotations = map[string]string{}
			vm.Annotations[exclusionMarking] = "true"

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			err := app.vmPresetController.initializeVirtualMachine(vm)

			Expect(err).ToNot(HaveOccurred())

			Expect(len(server.ReceivedRequests())).To(Equal(1))
			_, ok := vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]
			Expect(ok).To(BeFalse(), "Preset should not have been applied due to exclusion")

			// Prove that the VM was initialized
			Expect(isVirtualMachineInitialized(vm)).To(BeTrue())
		})
	})
})

func TestLogging(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VM Initializer")
}
