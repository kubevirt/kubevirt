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
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
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

var _ = Describe("VirtualMachineInstance Initializer", func() {
	Context("Annotate Presets", func() {
		It("should properly annotate a VirtualMachineInstance", func() {
			vmi := v1.VirtualMachineInstance{}
			preset := v1.VirtualMachineInstancePreset{}
			preset.ObjectMeta.Name = "test-preset"

			annotateVMI(&vmi, preset)
			Expect(len(vmi.Annotations)).To(Equal(1))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal(v1.GroupVersion.String()))
		})

		It("should allow multiple annotations", func() {
			vmi := v1.VirtualMachineInstance{}
			preset := v1.VirtualMachineInstancePreset{}
			preset.ObjectMeta.Name = "preset-foo"
			annotateVMI(&vmi, preset)
			preset = v1.VirtualMachineInstancePreset{}
			preset.ObjectMeta.Name = "preset-bar"
			annotateVMI(&vmi, preset)

			Expect(len(vmi.Annotations)).To(Equal(2))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/preset-foo"]).To(Equal(v1.GroupVersion.String()))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/preset-bar"]).To(Equal(v1.GroupVersion.String()))
		})
	})

	Context("Initializer Marking", func() {
		It("Should handle nil Annotations", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Annotations = nil
			Expect(isVirtualMachineInitialized(vmi)).To(BeFalse())
			addInitializedAnnotation(vmi)
			Expect(isVirtualMachineInitialized(vmi)).To(BeTrue())
		})

		It("Should handle empty Annotations", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Annotations = map[string]string{}
			Expect(isVirtualMachineInitialized(vmi)).To(BeFalse())
			addInitializedAnnotation(vmi)
			Expect(isVirtualMachineInitialized(vmi)).To(BeTrue())
		})

		It("Should not modify already initialized VirtualMachineInstance's", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Annotations = map[string]string{}
			vmi.Annotations[initializerMarking] = v1.GroupVersion.String()
			Expect(isVirtualMachineInitialized(vmi)).To(BeTrue())
			// call addInitializedAnnotation
			addInitializedAnnotation(vmi)
			Expect(isVirtualMachineInitialized(vmi)).To(BeTrue())
		})
	})

	Context("Preset Exclusions", func() {
		It("Should not fail if Annotations are nil", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Annotations = nil
			Expect(isVmExcluded(vmi)).To(BeFalse())
		})

		It("Should should not fail if Annotations are empty", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Annotations = map[string]string{}
			Expect(isVmExcluded(vmi)).To(BeFalse())
		})

		It("Should identify incorrect exclusion marking", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Annotations = map[string]string{}
			vmi.Annotations[exclusionMarking] = "something random"
			Expect(isVmExcluded(vmi)).To(BeFalse())
		})

		It("Should identify exclusion marking", func() {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Annotations = map[string]string{}
			vmi.Annotations[exclusionMarking] = "true"
			Expect(isVmExcluded(vmi)).To(BeTrue())
		})
	})

	Context("Override detection", func() {
		var vmi v1.VirtualMachineInstance
		var preset v1.VirtualMachineInstancePreset
		truthy := true
		falsy := false
		var recorder FakeRecorder

		memory, _ := resource.ParseQuantity("64M")

		BeforeEach(func() {
			vmi = v1.VirtualMachineInstance{Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{
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
					Disks: []v1.Disk{{Name: "testdisk",
						VolumeName: "testvolume",
						DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: "virtio", ReadOnly: true}}}}},
			}}}
			preset = v1.VirtualMachineInstancePreset{Spec: v1.VirtualMachineInstancePresetSpec{Domain: &v1.DomainPresetSpec{}}}
			recorder = NewFakeRecorder()
		})

		It("Should detect CPU overrides", func() {
			// Check without and then with a CPU conflict
			err := checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing no merge conflict occurs for matching preset")
			preset.Spec.Domain.CPU = &v1.CPU{Cores: 4}
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("applying matching preset")
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, []v1.VirtualMachineInstancePreset{preset}, recorder)

			By("checking annotations were applied and CPU count remains the same")
			Expect(len(vmi.Annotations)).To(Equal(1))
			Expect(int(vmi.Spec.Domain.CPU.Cores)).To(Equal(4))
			Expect(len(recorder.events.eventList)).To(Equal(0))

			By("showing an override occurred")
			preset.Spec.Domain.CPU = &v1.CPU{Cores: 6}
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("applying overridden preset")
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, []v1.VirtualMachineInstancePreset{preset}, recorder)

			By("checking annotations were not applied and CPU count remains the same")
			Expect(len(vmi.Annotations)).To(Equal(0))
			Expect(int(vmi.Spec.Domain.CPU.Cores)).To(Equal(4))

			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeNormal))
		})

		It("Should detect Resource overrides", func() {
			memory128, _ := resource.ParseQuantity("128M")
			preset.Spec.Domain.Resources = v1.ResourceRequirements{Requests: k8sv1.ResourceList{
				"memory": memory128,
			}}

			By("demonstrating that override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("applying mismatch preset")
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, []v1.VirtualMachineInstancePreset{preset}, recorder)

			By("checking preset was not applied")
			memory, ok := vmi.Spec.Domain.Resources.Requests["memory"]
			Expect(ok).To(BeTrue())
			Expect(int(memory.Value())).To(Equal(64000000))
			Expect(len(vmi.Annotations)).To(Equal(0))
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeNormal))

			preset.Spec.Domain.Resources = v1.ResourceRequirements{Requests: k8sv1.ResourceList{
				"memory": memory,
			}}

			By("demonstrating that no override occurs")
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("applying matching preset")
			recorder = NewFakeRecorder()
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, []v1.VirtualMachineInstancePreset{preset}, recorder)

			By("checking vmi settings remain the same")
			memory, ok = vmi.Spec.Domain.Resources.Requests["memory"]
			Expect(ok).To(BeTrue())
			Expect(int(memory.Value())).To(Equal(64000000))
			Expect(len(vmi.Annotations)).To(Equal(1))
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should detect Firmware overrides", func() {
			mismatchUuid := types.UID("88887777-6666-5555-4444-333322221111")
			matchUuid := types.UID("11112222-3333-4444-5555-666677778888")

			preset.Spec.Domain.Firmware = &v1.Firmware{UUID: mismatchUuid}

			By("showing that an override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("showing that presets are not applied")
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, []v1.VirtualMachineInstancePreset{preset}, recorder)

			Expect(len(vmi.Annotations)).To(Equal(0))
			Expect(vmi.Spec.Domain.Firmware.UUID).To(Equal(matchUuid))
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeNormal))

			preset.Spec.Domain.Firmware = &v1.Firmware{UUID: matchUuid}

			By("showing that an override does not occur")
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing settings did not change when preset is applied")
			recorder = NewFakeRecorder()
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, []v1.VirtualMachineInstancePreset{preset}, recorder)

			Expect(len(vmi.Annotations)).To(Equal(1))
			Expect(vmi.Spec.Domain.Firmware.UUID).To(Equal(matchUuid))
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should detect Clock overrides", func() {
			preset.Spec.Domain.Clock = &v1.Clock{ClockOffset: v1.ClockOffset{},
				Timer: &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyCatchup}},
			}

			By("showing that an override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("showing presets are not applied")
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, []v1.VirtualMachineInstancePreset{preset}, recorder)

			Expect(len(vmi.Annotations)).To(Equal(0))
			Expect(vmi.Spec.Domain.Clock.Timer.HPET.TickPolicy).To(Equal(v1.HPETTickPolicyDelay))
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeNormal))

			preset.Spec.Domain.Clock = &v1.Clock{ClockOffset: v1.ClockOffset{},
				Timer: &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyDelay}},
			}

			By("showing that an ovveride does not occur")
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing settings were not changed")
			recorder = NewFakeRecorder()
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, []v1.VirtualMachineInstancePreset{preset}, recorder)

			Expect(len(vmi.Annotations)).To(Equal(1))
			Expect(vmi.Spec.Domain.Clock.Timer.HPET.TickPolicy).To(Equal(v1.HPETTickPolicyDelay))
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should detect Feature overrides", func() {
			preset.Spec.Domain.Features = &v1.Features{ACPI: v1.FeatureState{Enabled: &falsy}}

			By("showing that an override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("showing presets are not applied")
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, []v1.VirtualMachineInstancePreset{preset}, recorder)

			Expect(len(vmi.Annotations)).To(Equal(0))
			Expect(*vmi.Spec.Domain.Features.ACPI.Enabled).To(BeTrue())
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeNormal))

			preset.Spec.Domain.Features = &v1.Features{ACPI: v1.FeatureState{Enabled: &truthy},
				APIC:   &v1.FeatureAPIC{Enabled: &falsy},
				Hyperv: &v1.FeatureHyperv{},
			}

			By("showing that an ovveride does not occur")
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing settings were not changed")
			recorder = NewFakeRecorder()
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, []v1.VirtualMachineInstancePreset{preset}, recorder)

			Expect(len(vmi.Annotations)).To(Equal(1))
			Expect(*vmi.Spec.Domain.Features.ACPI.Enabled).To(BeTrue())
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should detect Watchdog overrides", func() {
			preset.Spec.Domain.Devices.Watchdog = &v1.Watchdog{Name: "foo", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionPoweroff}}}

			By("showing that an override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("showing presets are not applied")
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, []v1.VirtualMachineInstancePreset{preset}, recorder)

			Expect(len(vmi.Annotations)).To(Equal(0))
			Expect(vmi.Spec.Domain.Devices.Watchdog.Name).To(Equal("testcase"))
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeNormal))

			preset.Spec.Domain.Devices.Watchdog = &v1.Watchdog{Name: "testcase", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionReset}}}

			By("showing that an ovveride does not occur")
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing settings were not changed")
			recorder = NewFakeRecorder()
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, []v1.VirtualMachineInstancePreset{preset}, recorder)

			Expect(len(vmi.Annotations)).To(Equal(1))
			Expect(vmi.Spec.Domain.Devices.Watchdog.Name).To(Equal("testcase"))
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})
	})

	Context("Conflict detection", func() {
		var vmi v1.VirtualMachineInstance
		var preset1 v1.VirtualMachineInstancePreset
		var preset2 v1.VirtualMachineInstancePreset
		var preset3 v1.VirtualMachineInstancePreset
		var preset4 v1.VirtualMachineInstancePreset

		m64, _ := resource.ParseQuantity("64M")
		m128, _ := resource.ParseQuantity("128M")

		var recorder FakeRecorder

		BeforeEach(func() {
			vmi = v1.VirtualMachineInstance{ObjectMeta: k8smetav1.ObjectMeta{Name: "test-vmi"}}

			preset1 = v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: "memory-64"},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"kubevirt.io/m64": "memory-64"}},
					Domain: &v1.DomainPresetSpec{
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{"memory": m64},
						},
					},
				},
			}
			preset2 = v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: "memory-128"},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"kubevirt.io/m128": "memory-128"}},
					Domain: &v1.DomainPresetSpec{
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{"memory": m128},
						},
					},
				},
			}
			preset3 = v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: "cpu-4"},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"kubevirt.io/cpu": "cpu-4"}},
					Domain: &v1.DomainPresetSpec{
						CPU: &v1.CPU{Cores: 4},
					},
				},
			}
			preset4 = v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: "duplicate-mem"},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"kubevirt.io/m64": "memory-64"}},
					Domain: &v1.DomainPresetSpec{
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{"memory": m64},
						},
					},
				},
			}
			recorder = NewFakeRecorder()
		})

		It("should detect conflicts between presets", func() {
			presets := []v1.VirtualMachineInstancePreset{preset1, preset2}
			err := checkPresetConflicts(presets)
			Expect(err).To(HaveOccurred())
		})

		It("should not return an error for no conflict", func() {
			presets := []v1.VirtualMachineInstancePreset{preset1, preset3}
			err := checkPresetConflicts(presets)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not consider presets with same settings to conflict", func() {
			presets := []v1.VirtualMachineInstancePreset{preset1, preset4}
			err := checkPresetConflicts(presets)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not apply presets that conflict", func() {
			presets := []v1.VirtualMachineInstancePreset{preset1, preset2, preset3, preset4}
			vmi.Labels = map[string]string{
				"kubevirt.io/m64":  "memory-64",
				"kubevirt.io/m128": "memory-128",
			}

			By("applying presets")
			res := applyPresets(&vmi, presets, recorder)
			Expect(res).To(BeFalse())

			By("checking annotations were not applied")
			annotation, ok := vmi.Annotations["virtualmachinepreset.kubevirt.io/memory-64"]
			Expect(annotation).To(Equal(""))
			Expect(ok).To(BeFalse())
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeWarning))

			By("checking settings were not applied to VirtualMachineInstance")
			Expect(vmi.Spec.Domain.Resources.Requests).To(BeNil())
		})

		It("should not apply any presets if any conflict", func() {
			presets := []v1.VirtualMachineInstancePreset{preset1, preset2, preset3, preset4}
			vmi.Labels = map[string]string{
				"kubevirt.io/m64":  "memory-64",
				"kubevirt.io/m128": "memory-128",
				"kubevirt.io/cpu":  "cpu-4",
			}

			By("applying presets")
			res := applyPresets(&vmi, presets, recorder)
			Expect(res).To(BeFalse())

			By("checking annotations were not applied")
			annotation, ok := vmi.Annotations["virtualmachinepreset.kubevirt.io/cpu-4"]
			Expect(annotation).To(Equal(""))
			Expect(ok).To(BeFalse())

			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeWarning))

			By("checking settings were not applied to VirtualMachineInstance")
			Expect(vmi.Spec.Domain.Resources.Requests).To(BeNil())
			Expect(vmi.Spec.Domain.CPU).To(BeNil())
		})

		It("should apply presets that don't conflict", func() {
			presets := []v1.VirtualMachineInstancePreset{preset1, preset3, preset4}
			vmi.Labels = map[string]string{
				"kubevirt.io/m64": "memory-64",
				"kubevirt.io/cpu": "cpu-4",
			}
			By("applying presets")
			res := applyPresets(&vmi, presets, recorder)
			Expect(res).To(BeTrue())

			By("checking annotations were applied")
			annotation, ok := vmi.Annotations["virtualmachinepreset.kubevirt.io/memory-64"]
			Expect(annotation).To(Equal("kubevirt.io/v1alpha2"))
			Expect(ok).To(BeTrue())

			annotation, ok = vmi.Annotations["virtualmachinepreset.kubevirt.io/cpu-4"]
			Expect(annotation).To(Equal("kubevirt.io/v1alpha2"))
			Expect(ok).To(BeTrue())

			annotation, ok = vmi.Annotations["virtualmachinepreset.kubevirt.io/duplicate-mem"]
			Expect(annotation).To(Equal("kubevirt.io/v1alpha2"))
			Expect(ok).To(BeTrue())

			By("checking settings were applied")
			Expect(len(vmi.Spec.Domain.Resources.Requests)).To(Equal(1))
			memory := vmi.Spec.Domain.Resources.Requests["memory"]
			Expect(int(memory.Value())).To(Equal(64000000))

			Expect(vmi.Spec.Domain.CPU).ToNot(BeNil())
			Expect(int(vmi.Spec.Domain.CPU.Cores)).To(Equal(4))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})
	})

	Context("Apply Presets", func() {
		var vmi v1.VirtualMachineInstance
		var preset v1.VirtualMachineInstancePreset
		truthy := true
		falsy := false
		var recorder FakeRecorder

		BeforeEach(func() {
			vmi = v1.VirtualMachineInstance{Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}}}
			vmi.ObjectMeta.Name = "testvmi"
			preset = v1.VirtualMachineInstancePreset{Spec: v1.VirtualMachineInstancePresetSpec{Domain: &v1.DomainPresetSpec{}}}
			preset.ObjectMeta.Name = "test-preset"
			recorder = NewFakeRecorder()
		})

		It("Should apply CPU settings", func() {
			preset.Spec.Domain.CPU = &v1.CPU{Cores: 4}
			presets := []v1.VirtualMachineInstancePreset{preset}
			applyPresets(&vmi, presets, recorder)

			Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(4)))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha2"))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should apply Resources", func() {
			memory, _ := resource.ParseQuantity("64M")
			preset.Spec.Domain.Resources = v1.ResourceRequirements{Requests: k8sv1.ResourceList{
				"memory": memory,
			}}
			presets := []v1.VirtualMachineInstancePreset{preset}
			applyPresets(&vmi, presets, recorder)

			Expect(vmi.Spec.Domain.Resources.Requests["memory"]).To(Equal(memory))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha2"))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should apply Firmware settings", func() {
			uuid := types.UID("11112222-3333-4444-5555-666677778888")
			preset.Spec.Domain.Firmware = &v1.Firmware{UUID: uuid}

			presets := []v1.VirtualMachineInstancePreset{preset}
			applyPresets(&vmi, presets, recorder)

			Expect(vmi.Spec.Domain.Firmware.UUID).To(Equal(uuid))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha2"))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should apply Clock settings", func() {
			clock := &v1.Clock{ClockOffset: v1.ClockOffset{},
				Timer: &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyDelay}},
			}
			preset.Spec.Domain.Clock = clock

			presets := []v1.VirtualMachineInstancePreset{preset}
			applyPresets(&vmi, presets, recorder)

			Expect(vmi.Spec.Domain.Clock).To(Equal(clock))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha2"))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should apply Feature settings", func() {
			features := &v1.Features{ACPI: v1.FeatureState{Enabled: &truthy},
				APIC:   &v1.FeatureAPIC{Enabled: &falsy},
				Hyperv: &v1.FeatureHyperv{},
			}

			preset.Spec.Domain.Features = features

			presets := []v1.VirtualMachineInstancePreset{preset}
			applyPresets(&vmi, presets, recorder)

			Expect(vmi.Spec.Domain.Features).To(Equal(features))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha2"))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should apply Watchdog settings", func() {
			watchdog := &v1.Watchdog{Name: "testcase", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionReset}}}

			preset.Spec.Domain.Devices.Watchdog = watchdog

			presets := []v1.VirtualMachineInstancePreset{preset}
			applyPresets(&vmi, presets, recorder)

			Expect(vmi.Spec.Domain.Devices.Watchdog).To(Equal(watchdog))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha2"))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})
	})

	Context("Filter Matching", func() {
		var vmi v1.VirtualMachineInstance
		var matchingPreset v1.VirtualMachineInstancePreset
		var nonmatchingPreset v1.VirtualMachineInstancePreset
		var errorPreset v1.VirtualMachineInstancePreset
		matchingPresetName := "test-preset"
		flavorKey := fmt.Sprintf("%s/flavor", v1.GroupName)
		matchingLabel := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: "matching"}}
		mismatchLabel := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: "unrelated"}}
		errorLabel := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: "!"}}

		var recorder FakeRecorder

		BeforeEach(func() {
			vmi = v1.VirtualMachineInstance{Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}}}
			vmi.ObjectMeta.Name = "testvmi"
			vmi.ObjectMeta.Labels = map[string]string{flavorKey: "matching"}

			matchingPreset = v1.VirtualMachineInstancePreset{Spec: v1.VirtualMachineInstancePresetSpec{Domain: &v1.DomainPresetSpec{}}}
			matchingPreset.ObjectMeta.Name = matchingPresetName
			matchingPreset.Spec.Selector = matchingLabel

			nonmatchingPreset = v1.VirtualMachineInstancePreset{Spec: v1.VirtualMachineInstancePresetSpec{Domain: &v1.DomainPresetSpec{}}}
			nonmatchingPreset.ObjectMeta.Name = "unrelated-preset"
			nonmatchingPreset.Spec.Selector = mismatchLabel

			errorPreset = v1.VirtualMachineInstancePreset{Spec: v1.VirtualMachineInstancePresetSpec{Domain: &v1.DomainPresetSpec{}}}
			errorPreset.ObjectMeta.Name = "broken-preset"
			errorPreset.Spec.Selector = errorLabel
			recorder = NewFakeRecorder()
		})

		It("Should match preset with the correct selector", func() {
			allPresets := []v1.VirtualMachineInstancePreset{matchingPreset, nonmatchingPreset}
			matchingPresets := filterPresets(allPresets, &vmi, recorder)
			Expect(len(matchingPresets)).To(Equal(1))
			Expect(matchingPresets[0].Name).To(Equal(matchingPresetName))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should not match preset with the incorrect selector", func() {
			allPresets := []v1.VirtualMachineInstancePreset{nonmatchingPreset}
			matchingPresets := filterPresets(allPresets, &vmi, recorder)
			Expect(len(matchingPresets)).To(Equal(0))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(0))
		})

		It("Should ignore bogus selectors", func() {
			allPresets := []v1.VirtualMachineInstancePreset{matchingPreset, nonmatchingPreset, errorPreset}
			matchingPresets := filterPresets(allPresets, &vmi, recorder)
			Expect(len(matchingPresets)).To(Equal(1))
			Expect(matchingPresets[0].Name).To(Equal(matchingPresetName))

			By("checking that no events were recorded")
			Expect(len(recorder.events.eventList)).To(Equal(1))
			Expect(recorder.events.eventList[0].eventtype).To(Equal(k8sv1.EventTypeWarning))
		})

	})

	Context("VirtualMachineInstance Init Watcher", func() {
		var vmiPresetController *VirtualMachinePresetController

		var ctrl *gomock.Controller
		var virtClient *kubecli.MockKubevirtClient
		var vmiInterface *kubecli.MockVirtualMachineInstanceInterface

		var vmiPreset *v1.VirtualMachineInstancePreset
		var stopChan chan struct{}

		var vmiPresetInformer cache.SharedIndexInformer
		var vmiPresetCache *framework.FakeControllerSource
		var vmiInformer cache.SharedIndexInformer
		var vmiInitCache cache.Store
		var recorder *record.FakeRecorder

		var vmiPresetQueue *testutils.MockWorkQueue

		flavorKey := fmt.Sprintf("%s/flavor", v1.GroupName)
		presetFlavor := "test-case"

		BeforeEach(func() {
			stopChan = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())

			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

			vmiPresetInformer, vmiPresetCache = testutils.NewFakeInformerFor(&v1.VirtualMachineInstancePreset{})
			vmiInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmiInitCache = vmiInformer.GetStore()

			vmiPresetQueue = testutils.NewMockWorkQueue(workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()))

			recorder = record.NewFakeRecorder(100)

			vmiPresetController = NewVirtualMachinePresetController(vmiPresetInformer, vmiInformer, vmiPresetQueue, vmiInitCache, virtClient, recorder)

			// create a reference preset
			selector := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: presetFlavor}}

			vmiPreset = v1.NewVirtualMachinePreset("test-preset", selector)
			vmiPreset.Spec.Domain.CPU = &v1.CPU{Cores: 4}
			vmiPreset.Spec.Domain.Firmware = &v1.Firmware{UUID: "12345678-1234-1234-1234-123456781234"}

			idx := vmiPresetInformer.GetIndexer()
			idx.Add(vmiPreset)
		})

		AfterEach(func() {
			close(stopChan)
		})

		It("should not process an initialized VirtualMachineInstance", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			addInitializedAnnotation(vmi)
			Expect(isVirtualMachineInitialized(vmi)).To(BeTrue())

			key, _ := cache.MetaNamespaceKeyFunc(vmi)
			vmiPresetCache.Add(vmi)
			vmiPresetQueue.Add(key)

			// the initializer should inspect the VirtualMachineInstance and decide nothing needs to be done
			// (and skip the update entirely). So zero requests are expected.
			virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).MaxTimes(0)

			vmiPresetController.Execute()

			Expect(isVirtualMachineInitialized(vmi)).To(BeTrue())
			Expect(controller.HasFinalizer(vmi, v1.VirtualMachineInstanceFinalizer)).To(BeTrue())
		})

		It("should initialize a VirtualMachineInstance if needed", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			key, _ := cache.MetaNamespaceKeyFunc(vmi)
			vmiPresetCache.Add(vmi)
			vmiPresetQueue.Add(key)

			virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				// The copy of the VMI sent to the server should be initialized
				_, found := (arg.(*v1.VirtualMachineInstance)).Annotations[initializerMarking]
				Expect(found).To(BeTrue())
			}).Return(nil, nil)

			vmiPresetController.Execute()

			// We should expect no changes to this VirtualMachineInstance object--because that would mean
			// there were side effects in the cache.
			Expect(isVirtualMachineInitialized(vmi)).To(BeFalse())
		})

		It("should apply presets", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Labels = map[string]string{flavorKey: presetFlavor}

			virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				vmi := arg.(*v1.VirtualMachineInstance)
				val, found := vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]
				Expect(found).To(BeTrue(), "preset should have been applied")
				Expect(val).To(Equal("kubevirt.io/v1alpha2"))

				// The copy of the VMI sent to the server should be initialized
				_, found = vmi.Annotations[initializerMarking]
				Expect(found).To(BeTrue(), "vmi should have been initialized")
			}).Return(nil, nil)

			err := vmiPresetController.initializeVirtualMachine(vmi)
			Expect(err).ToNot(HaveOccurred())

			// Prove that the VirtualMachineInstance was annotated (indicates successful application of preset)
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha2"))
		})

		It("should annotate partially applied presets", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Labels = map[string]string{flavorKey: presetFlavor}
			vmi.Spec.Domain = v1.DomainSpec{CPU: &v1.CPU{Cores: 6}}

			virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				vmi := arg.(*v1.VirtualMachineInstance)
				val, found := vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]
				Expect(found).To(BeTrue(), "preset should have been applied")
				Expect(val).To(Equal("kubevirt.io/v1alpha2"))

				// The copy of the VMI sent to the server should be initialized
				_, found = vmi.Annotations[initializerMarking]
				Expect(found).To(BeTrue(), "vmi should have been initialized")
			}).Return(nil, nil)

			err := vmiPresetController.initializeVirtualMachine(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should should not annotate presets with no settings successfully applied", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Labels = map[string]string{flavorKey: presetFlavor}
			vmi.Spec.Domain = v1.DomainSpec{
				CPU:      &v1.CPU{Cores: 6},
				Firmware: &v1.Firmware{UUID: "11111111-2222-3333-4444-123456781234"}}

			virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				vmi := arg.(*v1.VirtualMachineInstance)
				_, found := vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]
				Expect(found).To(BeFalse(), "preset should not have been applied")

				// The copy of the VMI sent to the server should be initialized
				_, found = vmi.Annotations[initializerMarking]
				Expect(found).To(BeTrue(), "vmi should have been initialized")
				Expect(vmi.Status.Phase).ToNot(Equal(v1.Failed))
			}).Return(nil, nil)

			err := vmiPresetController.initializeVirtualMachine(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not mark a VirtualMachineInstance without presets as failed", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				vmi := arg.(*v1.VirtualMachineInstance)
				_, found := vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]
				Expect(found).To(BeFalse(), "preset should not have been applied")

				// The copy of the VMI sent to the server should be initialized
				_, found = vmi.Annotations[initializerMarking]
				Expect(found).To(BeTrue(), "vmi should have been initialized")

				Expect(vmi.Status.Phase).ToNot(Equal(v1.Failed))
			}).Return(nil, nil)

			err := vmiPresetController.initializeVirtualMachine(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should check if exclusion annotation is \"true\"", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Labels = map[string]string{flavorKey: presetFlavor}
			vmi.Annotations = map[string]string{}
			// Since the exclusion marking is invalid, it won't take effect
			vmi.Annotations[exclusionMarking] = "anything"

			virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				vmi := arg.(*v1.VirtualMachineInstance)
				val, found := vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]
				Expect(found).To(BeTrue(), "preset should have been applied")
				Expect(val).To(Equal("kubevirt.io/v1alpha2"))

				// The copy of the VMI sent to the server should be initialized
				_, found = vmi.Annotations[initializerMarking]
				Expect(found).To(BeTrue(), "vmi should have been initialized")
			}).Return(nil, nil)

			err := vmiPresetController.initializeVirtualMachine(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not add annotations to VirtualMachineInstance's with exclusion marking", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Labels = map[string]string{flavorKey: presetFlavor}
			vmi.Annotations = map[string]string{}
			vmi.Annotations[exclusionMarking] = "true"

			virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				vmi := arg.(*v1.VirtualMachineInstance)
				_, found := vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]
				Expect(found).To(BeFalse(), "preset should not have been applied")

				// The copy of the VMI sent to the server should be initialized
				_, found = vmi.Annotations[initializerMarking]
				Expect(found).To(BeTrue(), "vmi should have been initialized")

				Expect(vmi.Status.Phase).ToNot(Equal(v1.Failed))
			}).Return(nil, nil)

			err := vmiPresetController.initializeVirtualMachine(vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should set default values to VMI", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec = v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						Disks: []v1.Disk{
							{Name: "testdisk"},
						},
					},
				},
			}

			virtClient.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				vmi := arg.(*v1.VirtualMachineInstance)
				disk := vmi.Spec.Domain.Devices.Disks[0]
				Expect(disk.Disk).ToNot(BeNil(), "DiskTarget should not be nil")
				Expect(disk.Disk.Bus).ToNot(BeEmpty(), "DiskTarget's bus should not be empty")

				// The copy of the VMI sent to the server should be initialized
				_, found := vmi.Annotations[initializerMarking]
				Expect(found).To(BeTrue(), "vmi should have been initialized")

				Expect(vmi.Status.Phase).ToNot(Equal(v1.Failed))
			}).Return(nil, nil)

			err := vmiPresetController.initializeVirtualMachine(vmi)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func TestLogging(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VirtualMachineInstance Initializer")
}
