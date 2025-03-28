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

package mutators

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/api/core"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Mutating Webhook Presets", func() {
	Context("Annotate Presets", func() {
		It("should properly annotate a VirtualMachineInstance", func() {
			vmi := v1.VirtualMachineInstance{}
			preset := v1.VirtualMachineInstancePreset{}
			preset.ObjectMeta.Name = "test-preset"

			annotateVMI(&vmi, preset)
			Expect(vmi.Annotations).To(HaveLen(1))
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

			Expect(vmi.Annotations).To(HaveLen(2))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/preset-foo"]).To(Equal(v1.GroupVersion.String()))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/preset-bar"]).To(Equal(v1.GroupVersion.String()))
		})
	})

	Context("Override detection", func() {
		var vmi v1.VirtualMachineInstance
		var preset *v1.VirtualMachineInstancePreset
		var presetInformer cache.SharedIndexInformer

		truthy := true
		falsy := false

		memory, _ := resource.ParseQuantity("64M")

		BeforeEach(func() {
			vmi = v1.VirtualMachineInstance{
				ObjectMeta: k8smetav1.ObjectMeta{
					Labels: map[string]string{"test": "test"},
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Resources: v1.ResourceRequirements{Requests: k8sv1.ResourceList{
							"memory": memory,
						}},
						CPU:      &v1.CPU{Cores: 4},
						Firmware: &v1.Firmware{UUID: types.UID("11112222-3333-4444-5555-666677778888")},
						Clock: &v1.Clock{
							ClockOffset: v1.ClockOffset{},
							Timer:       &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyDelay}},
						},
						Features: &v1.Features{
							ACPI:   v1.FeatureState{Enabled: &truthy},
							APIC:   &v1.FeatureAPIC{Enabled: &falsy},
							Hyperv: &v1.FeatureHyperv{},
						},
						Devices: v1.Devices{
							Watchdog: &v1.Watchdog{
								Name:           "testcase",
								WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionReset}},
							},
							Disks: []v1.Disk{{
								Name: "testdisk",
								DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{
									Bus: v1.DiskBusVirtio, ReadOnly: true,
								}},
							}},
						},
					},
				},
			}

			selector := k8smetav1.LabelSelector{MatchLabels: map[string]string{"test": "test"}}
			preset = &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "test-preset",
				},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Domain:   &v1.DomainSpec{},
					Selector: selector,
				},
			}
			presetInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachineInstancePreset{})
			presetInformer.GetIndexer().Add(preset)
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
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			By("checking annotations were applied and CPU count remains the same")
			Expect(vmi.Annotations).To(HaveLen(1))
			Expect(int(vmi.Spec.Domain.CPU.Cores)).To(Equal(4))

			By("showing an override occurred")
			preset.Spec.Domain.CPU = &v1.CPU{Cores: 6}
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("applying overridden preset")
			vmi.Annotations = map[string]string{}
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			By("checking annotations were not applied and CPU count remains the same")
			Expect(vmi.Annotations).To(BeEmpty())
			Expect(int(vmi.Spec.Domain.CPU.Cores)).To(Equal(4))
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
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			By("checking preset was not applied")
			memory, ok := vmi.Spec.Domain.Resources.Requests["memory"]
			Expect(ok).To(BeTrue())
			Expect(int(memory.Value())).To(Equal(64000000))
			Expect(vmi.Annotations).To(BeEmpty())

			preset.Spec.Domain.Resources = v1.ResourceRequirements{Requests: k8sv1.ResourceList{
				"memory": memory,
			}}

			By("demonstrating that no override occurs")
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("applying matching preset")
			vmi.Annotations = map[string]string{}
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			By("checking vmi settings remain the same")
			memory, ok = vmi.Spec.Domain.Resources.Requests["memory"]
			Expect(ok).To(BeTrue())
			Expect(int(memory.Value())).To(Equal(64000000))
			Expect(vmi.Annotations).To(HaveLen(1))
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
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Annotations).To(BeEmpty())
			Expect(vmi.Spec.Domain.Firmware.UUID).To(Equal(matchUuid))

			preset.Spec.Domain.Firmware = &v1.Firmware{UUID: matchUuid}

			By("showing that an override does not occur")
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing settings did not change when preset is applied")
			vmi.Annotations = map[string]string{}
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Annotations).To(HaveLen(1))
			Expect(vmi.Spec.Domain.Firmware.UUID).To(Equal(matchUuid))
		})

		It("Should detect Clock overrides", func() {
			preset.Spec.Domain.Clock = &v1.Clock{
				ClockOffset: v1.ClockOffset{},
				Timer:       &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyCatchup}},
			}

			By("showing that an override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("showing presets are not applied")
			vmi.Annotations = map[string]string{}
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Annotations).To(BeEmpty())
			Expect(vmi.Spec.Domain.Clock.Timer.HPET.TickPolicy).To(Equal(v1.HPETTickPolicyDelay))

			preset.Spec.Domain.Clock = &v1.Clock{
				ClockOffset: v1.ClockOffset{},
				Timer:       &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyDelay}},
			}

			By("showing that an overide does not occur")
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing settings were not changed")
			presetInformer.GetIndexer().Add(preset)
			vmi.Annotations = map[string]string{}
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Annotations).To(HaveLen(1))
			Expect(vmi.Spec.Domain.Clock.Timer.HPET.TickPolicy).To(Equal(v1.HPETTickPolicyDelay))
		})

		It("Should detect Feature overrides", func() {
			preset.Spec.Domain.Features = &v1.Features{ACPI: v1.FeatureState{Enabled: &falsy}}

			By("showing that an override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("showing presets are not applied")
			vmi.Annotations = map[string]string{}

			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Annotations).To(BeEmpty())
			Expect(*vmi.Spec.Domain.Features.ACPI.Enabled).To(BeTrue())

			preset.Spec.Domain.Features = &v1.Features{
				ACPI:   v1.FeatureState{Enabled: &truthy},
				APIC:   &v1.FeatureAPIC{Enabled: &falsy},
				Hyperv: &v1.FeatureHyperv{},
			}

			By("showing that an overide does not occur")
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing settings were not changed")
			vmi.Annotations = map[string]string{}
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Annotations).To(HaveLen(1))
			Expect(*vmi.Spec.Domain.Features.ACPI.Enabled).To(BeTrue())
		})

		It("Should detect Watchdog overrides", func() {
			preset.Spec.Domain.Devices.Watchdog = &v1.Watchdog{Name: "foo", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionPoweroff}}}

			By("showing that an override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("showing presets are not applied")
			vmi.Annotations = map[string]string{}
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Annotations).To(BeEmpty())
			Expect(vmi.Spec.Domain.Devices.Watchdog.Name).To(Equal("testcase"))

			preset.Spec.Domain.Devices.Watchdog = &v1.Watchdog{Name: "testcase", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionReset}}}

			By("showing that an overide does not occur")
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			By("showing settings were not changed")
			vmi.Annotations = map[string]string{}
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Annotations).To(HaveLen(1))
			Expect(vmi.Spec.Domain.Devices.Watchdog.Name).To(Equal("testcase"))
		})

		It("Should detect ioThreadsPolicy overrides", func() {
			sharedPolicy := v1.IOThreadsPolicyShared
			preset.Spec.Domain.IOThreadsPolicy = &sharedPolicy

			automaticPolicy := v1.IOThreadsPolicyAuto
			vmi.Spec.Domain.IOThreadsPolicy = &automaticPolicy

			By("showing that an override occurs")
			err := checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).To(HaveOccurred())

			By("showing presets are not applied")
			vmi.Annotations = map[string]string{}
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Annotations).To(BeEmpty(), "There should not be annotations if presets weren't applied")
			Expect(*vmi.Spec.Domain.IOThreadsPolicy).To(Equal(automaticPolicy), "IOThreads policy should not have been overridden")

			preset.Spec.Domain.IOThreadsPolicy = &automaticPolicy

			By("showing that settings were not changed")
			err = checkMergeConflicts(preset.Spec.Domain, &vmi.Spec.Domain)
			Expect(err).ToNot(HaveOccurred())

			vmi.Annotations = map[string]string{}
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Annotations).To(HaveLen(1), "There should be an annotation indicating presets were applied")
			Expect(*vmi.Spec.Domain.IOThreadsPolicy).To(Equal(automaticPolicy), "IOThreadsPolicy should not have been changed")
		})
	})

	Context("Conflict detection", func() {
		var vmi v1.VirtualMachineInstance

		var preset1 *v1.VirtualMachineInstancePreset
		var preset2 *v1.VirtualMachineInstancePreset
		var preset3 *v1.VirtualMachineInstancePreset
		var preset4 *v1.VirtualMachineInstancePreset

		m64, _ := resource.ParseQuantity("64M")
		m128, _ := resource.ParseQuantity("128M")

		BeforeEach(func() {
			vmi = v1.VirtualMachineInstance{ObjectMeta: k8smetav1.ObjectMeta{Name: "test-vmi"}}

			preset1 = &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: "memory-64"},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"kubevirt.io/m64": "memory-64"}},
					Domain: &v1.DomainSpec{
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{"memory": m64},
						},
					},
				},
			}
			preset2 = &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: "memory-128"},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"kubevirt.io/m128": "memory-128"}},
					Domain: &v1.DomainSpec{
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{"memory": m128},
						},
					},
				},
			}
			preset3 = &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: "cpu-4"},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"kubevirt.io/cpu": "cpu-4"}},
					Domain: &v1.DomainSpec{
						CPU: &v1.CPU{Cores: 4},
					},
				},
			}
			preset4 = &v1.VirtualMachineInstancePreset{
				ObjectMeta: k8smetav1.ObjectMeta{Name: "duplicate-mem"},
				Spec: v1.VirtualMachineInstancePresetSpec{
					Selector: k8smetav1.LabelSelector{MatchLabels: map[string]string{"kubevirt.io/m64": "memory-64"}},
					Domain: &v1.DomainSpec{
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{"memory": m64},
						},
					},
				},
			}
		})

		It("should detect conflicts between presets", func() {
			presets := []v1.VirtualMachineInstancePreset{*preset1, *preset2}
			err := checkPresetConflicts(presets)
			Expect(err).To(HaveOccurred())
		})

		It("should not return an error for no conflict", func() {
			presets := []v1.VirtualMachineInstancePreset{*preset1, *preset3}
			err := checkPresetConflicts(presets)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not consider presets with same settings to conflict", func() {
			presets := []v1.VirtualMachineInstancePreset{*preset1, *preset4}
			err := checkPresetConflicts(presets)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not apply presets that conflict", func() {
			presetInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstancePreset{})
			presetInformer.GetIndexer().Add(preset1)
			presetInformer.GetIndexer().Add(preset2)
			presetInformer.GetIndexer().Add(preset3)
			presetInformer.GetIndexer().Add(preset4)

			vmi.Labels = map[string]string{
				"kubevirt.io/m64":  "memory-64",
				"kubevirt.io/m128": "memory-128",
			}

			By("applying presets")
			err := applyPresets(&vmi, presetInformer)
			Expect(err).To(HaveOccurred())

			By("checking annotations were not applied")
			annotation, ok := vmi.Annotations["virtualmachinepreset.kubevirt.io/memory-64"]
			Expect(annotation).To(Equal(""))
			Expect(ok).To(BeFalse())

			By("checking settings were not applied to VirtualMachineInstance")
			Expect(vmi.Spec.Domain.Resources.Requests).To(BeNil())
		})

		It("should not apply any presets if any conflict", func() {
			presetInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstancePreset{})
			presetInformer.GetIndexer().Add(preset1)
			presetInformer.GetIndexer().Add(preset2)
			presetInformer.GetIndexer().Add(preset3)
			presetInformer.GetIndexer().Add(preset4)

			vmi.Labels = map[string]string{
				"kubevirt.io/m64":  "memory-64",
				"kubevirt.io/m128": "memory-128",
				"kubevirt.io/cpu":  "cpu-4",
			}

			By("applying presets")
			err := applyPresets(&vmi, presetInformer)
			Expect(err).To(HaveOccurred())

			By("checking annotations were not applied")
			annotation, ok := vmi.Annotations["virtualmachinepreset.kubevirt.io/cpu-4"]
			Expect(annotation).To(Equal(""))
			Expect(ok).To(BeFalse())

			By("checking settings were not applied to VirtualMachineInstance")
			Expect(vmi.Spec.Domain.Resources.Requests).To(BeNil())
			Expect(vmi.Spec.Domain.CPU).To(BeNil())
		})

		It("should apply presets that don't conflict", func() {
			presetInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstancePreset{})
			presetInformer.GetIndexer().Add(preset1)
			presetInformer.GetIndexer().Add(preset3)
			presetInformer.GetIndexer().Add(preset4)
			vmi.Labels = map[string]string{
				"kubevirt.io/m64": "memory-64",
				"kubevirt.io/cpu": "cpu-4",
			}
			By("applying presets")
			err := applyPresets(&vmi, presetInformer)
			Expect(err).ToNot(HaveOccurred())

			By("checking annotations were applied")
			annotation, ok := vmi.Annotations["virtualmachinepreset.kubevirt.io/memory-64"]
			Expect(annotation).To(Equal(fmt.Sprintf("kubevirt.io/%s", v1.ApiLatestVersion)))
			Expect(ok).To(BeTrue())

			annotation, ok = vmi.Annotations["virtualmachinepreset.kubevirt.io/cpu-4"]
			Expect(annotation).To(Equal(fmt.Sprintf("kubevirt.io/%s", v1.ApiLatestVersion)))
			Expect(ok).To(BeTrue())

			annotation, ok = vmi.Annotations["virtualmachinepreset.kubevirt.io/duplicate-mem"]
			Expect(annotation).To(Equal(fmt.Sprintf("kubevirt.io/%s", v1.ApiLatestVersion)))
			Expect(ok).To(BeTrue())

			By("checking settings were applied")
			Expect(vmi.Spec.Domain.Resources.Requests).To(HaveLen(1))
			memory := vmi.Spec.Domain.Resources.Requests["memory"]
			Expect(int(memory.Value())).To(Equal(64000000))

			Expect(vmi.Spec.Domain.CPU).ToNot(BeNil())
			Expect(int(vmi.Spec.Domain.CPU.Cores)).To(Equal(4))
		})
	})

	Context("Apply Presets", func() {
		var vmi v1.VirtualMachineInstance
		var preset *v1.VirtualMachineInstancePreset
		var presetInformer cache.SharedIndexInformer

		truthy := true
		falsy := false

		BeforeEach(func() {
			vmi = v1.VirtualMachineInstance{Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}}}
			vmi.ObjectMeta.Name = "testvmi"
			preset = &v1.VirtualMachineInstancePreset{Spec: v1.VirtualMachineInstancePresetSpec{Domain: &v1.DomainSpec{}}}
			preset.Name = "test-preset"
			presetInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachineInstancePreset{})
		})

		When("VMI has exclusion annotation", func() {
			It("Should be skipped", func() {
				preset.Spec.Domain.CPU = &v1.CPU{Cores: 4}
				presetInformer.GetIndexer().Add(preset)

				vmiCopy := vmi.DeepCopy()
				vmiCopy.Annotations = make(map[string]string)
				vmiCopy.Annotations[exclusionMarking] = "true"
				applyPresets(vmiCopy, presetInformer)
				Expect(vmi.Spec.Domain.CPU).To(BeNil())
			})
		})

		It("Should apply CPU settings", func() {
			preset.Spec.Domain.CPU = &v1.CPU{Cores: 4}
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Spec.Domain.CPU).ToNot(BeNil())
			Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(uint32(4)))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal(fmt.Sprintf("kubevirt.io/%s", v1.ApiLatestVersion)))
		})

		It("Should apply Resources", func() {
			memory, _ := resource.ParseQuantity("64M")
			preset.Spec.Domain.Resources = v1.ResourceRequirements{Requests: k8sv1.ResourceList{
				"memory": memory,
			}}
			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Spec.Domain.Resources.Requests["memory"]).To(Equal(memory))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal(fmt.Sprintf("kubevirt.io/%s", v1.ApiLatestVersion)))
		})

		It("Should apply Firmware settings", func() {
			uuid := types.UID("11112222-3333-4444-5555-666677778888")
			preset.Spec.Domain.Firmware = &v1.Firmware{UUID: uuid}

			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Spec.Domain.Firmware).ToNot(BeNil())
			Expect(vmi.Spec.Domain.Firmware.UUID).To(Equal(uuid))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal(fmt.Sprintf("kubevirt.io/%s", v1.ApiLatestVersion)))
		})

		It("Should apply Clock settings", func() {
			clock := &v1.Clock{
				ClockOffset: v1.ClockOffset{},
				Timer:       &v1.Timer{HPET: &v1.HPETTimer{TickPolicy: v1.HPETTickPolicyDelay}},
			}
			preset.Spec.Domain.Clock = clock

			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Spec.Domain.Clock).To(Equal(clock))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal(fmt.Sprintf("kubevirt.io/%s", v1.ApiLatestVersion)))
		})

		It("Should apply Feature settings", func() {
			features := &v1.Features{
				ACPI:   v1.FeatureState{Enabled: &truthy},
				APIC:   &v1.FeatureAPIC{Enabled: &falsy},
				Hyperv: &v1.FeatureHyperv{},
			}

			preset.Spec.Domain.Features = features

			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Spec.Domain.Features).To(Equal(features))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal(fmt.Sprintf("kubevirt.io/%s", v1.ApiLatestVersion)))
		})

		It("Should apply Watchdog settings", func() {
			watchdog := &v1.Watchdog{Name: "testcase", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionReset}}}

			preset.Spec.Domain.Devices.Watchdog = watchdog

			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Spec.Domain.Devices.Watchdog).To(Equal(watchdog))
			Expect(vmi.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal(fmt.Sprintf("kubevirt.io/%s", v1.ApiLatestVersion)))
		})

		It("Should apply IOThreads settings", func() {
			ioThreads := v1.IOThreadsPolicyShared
			preset.Spec.Domain.IOThreadsPolicy = &ioThreads

			presetInformer.GetIndexer().Add(preset)
			applyPresets(&vmi, presetInformer)

			Expect(vmi.Spec.Domain.IOThreadsPolicy).ToNot(BeNil(), "IOThreads policy should have been applied by preset")
			Expect(*vmi.Spec.Domain.IOThreadsPolicy).To(Equal(ioThreads), "Expected IOThreadsPolicy to be 'shared' (set by preset)")
		})
	})

	Context("Filter Matching", func() {
		var vmi v1.VirtualMachineInstance
		var matchingPreset v1.VirtualMachineInstancePreset
		var nonmatchingPreset v1.VirtualMachineInstancePreset
		var errorPreset v1.VirtualMachineInstancePreset
		matchingPresetName := "test-preset"
		flavorKey := fmt.Sprintf("%s/flavor", core.GroupName)
		matchingLabel := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: "matching"}}
		mismatchLabel := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: "unrelated"}}
		errorLabel := k8smetav1.LabelSelector{MatchLabels: map[string]string{flavorKey: "!"}}

		BeforeEach(func() {
			vmi = v1.VirtualMachineInstance{Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}}}
			vmi.ObjectMeta.Name = "testvmi"
			vmi.ObjectMeta.Labels = map[string]string{flavorKey: "matching"}

			matchingPreset = v1.VirtualMachineInstancePreset{Spec: v1.VirtualMachineInstancePresetSpec{Domain: &v1.DomainSpec{}}}
			matchingPreset.ObjectMeta.Name = matchingPresetName
			matchingPreset.Spec.Selector = matchingLabel

			nonmatchingPreset = v1.VirtualMachineInstancePreset{Spec: v1.VirtualMachineInstancePresetSpec{Domain: &v1.DomainSpec{}}}
			nonmatchingPreset.ObjectMeta.Name = "unrelated-preset"
			nonmatchingPreset.Spec.Selector = mismatchLabel

			errorPreset = v1.VirtualMachineInstancePreset{Spec: v1.VirtualMachineInstancePresetSpec{Domain: &v1.DomainSpec{}}}
			errorPreset.ObjectMeta.Name = "broken-preset"
			errorPreset.Spec.Selector = errorLabel
		})

		It("Should match preset with the correct selector", func() {
			allPresets := []v1.VirtualMachineInstancePreset{matchingPreset, nonmatchingPreset}
			matchingPresets, err := filterPresets(allPresets, &vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(matchingPresets).To(HaveLen(1))
			Expect(matchingPresets[0].Name).To(Equal(matchingPresetName))
		})

		It("Should not match preset with the incorrect selector", func() {
			allPresets := []v1.VirtualMachineInstancePreset{nonmatchingPreset}
			matchingPresets, err := filterPresets(allPresets, &vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(matchingPresets).To(BeEmpty())
		})

		It("Should not ignore bogus selectors", func() {
			allPresets := []v1.VirtualMachineInstancePreset{errorPreset}
			_, err := filterPresets(allPresets, &vmi)
			Expect(err).To(HaveOccurred())
		})
	})
})
