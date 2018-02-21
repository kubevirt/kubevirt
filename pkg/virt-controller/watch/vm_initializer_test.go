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

type FakeRecorder struct {
}

func (recorder *FakeRecorder) Event(object runtime.Object, eventtype, reason, message string) {
}

func (recorder *FakeRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
}

func (recorder *FakeRecorder) PastEventf(object runtime.Object, timestamp k8smetav1.Time, eventtype, reason, messageFmt string, args ...interface{}) {
}

var _ = Describe("VM Initializer", func() {
	Context("Annotate Presets", func() {
		It("should properly annotate a VM", func() {
			vm := v1.VirtualMachine{}
			preset := v1.VirtualMachinePreset{}
			preset.ObjectMeta.Name = "test-preset"
			presets := append([]v1.VirtualMachinePreset{}, preset)

			annotateVM(&vm, presets)
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

			annotateVM(&vm, presets)
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
		recorder := &FakeRecorder{}

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
						DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: "virtio", ReadOnly: true}}}}},
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
			err = applyPresets(&vm, presets, recorder)
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
	})

	Context("Apply Presets", func() {
		var vm v1.VirtualMachine
		var preset v1.VirtualMachinePreset
		truthy := true
		falsy := false
		recorder := &FakeRecorder{}

		BeforeEach(func() {
			vm = v1.VirtualMachine{Spec: v1.VirtualMachineSpec{Domain: v1.DomainSpec{}}}
			vm.ObjectMeta.Name = "testvm"
			preset = v1.VirtualMachinePreset{Spec: v1.VirtualMachinePresetSpec{Domain: &v1.DomainSpec{}}}
			preset.ObjectMeta.Name = "test-preset"
		})

		It("Should apply CPU settings", func() {
			preset.Spec.Domain.CPU = &v1.CPU{Cores: 4}
			presets := []v1.VirtualMachinePreset{preset}
			err := applyPresets(&vm, presets, recorder)
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
			err := applyPresets(&vm, presets, recorder)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Domain.Resources.Requests["memory"]).To(Equal(memory))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
		})

		It("Should apply Firmware settings", func() {
			uuid := types.UID("11112222-3333-4444-5555-666677778888")
			preset.Spec.Domain.Firmware = &v1.Firmware{UUID: uuid}

			presets := []v1.VirtualMachinePreset{preset}
			err := applyPresets(&vm, presets, recorder)
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
			err := applyPresets(&vm, presets, recorder)
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
			err := applyPresets(&vm, presets, recorder)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Domain.Features).To(Equal(features))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
		})

		It("Should apply Watchdog settings", func() {
			watchdog := &v1.Watchdog{Name: "testcase", WatchdogDevice: v1.WatchdogDevice{I6300ESB: &v1.I6300ESBWatchdog{Action: v1.WatchdogActionReset}}}

			preset.Spec.Domain.Devices.Watchdog = watchdog

			presets := []v1.VirtualMachinePreset{preset}
			err := applyPresets(&vm, presets, recorder)
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Domain.Devices.Watchdog).To(Equal(watchdog))
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
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

		recorder := &FakeRecorder{}

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
		})

		It("Should match preset with the correct selector", func() {
			allPresets := []v1.VirtualMachinePreset{matchingPreset, nonmatchingPreset}
			matchingPresets := filterPresets(allPresets, &vm, recorder)
			Expect(len(matchingPresets)).To(Equal(1))
			Expect(matchingPresets[0].Name).To(Equal(matchingPresetName))
		})

		It("Should not match preset with the incorrect selector", func() {
			allPresets := []v1.VirtualMachinePreset{nonmatchingPreset}
			matchingPresets := filterPresets(allPresets, &vm, recorder)
			Expect(len(matchingPresets)).To(Equal(0))
		})

		It("Should ignore bogus selectors", func() {
			allPresets := []v1.VirtualMachinePreset{matchingPreset, nonmatchingPreset, errorPreset}
			matchingPresets := filterPresets(allPresets, &vm, recorder)
			Expect(len(matchingPresets)).To(Equal(1))
			Expect(matchingPresets[0].Name).To(Equal(matchingPresetName))
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

			// Synthesize a fake vmInitInformer
			vmListWatch := &cache.ListWatch{
				ListFunc: func(options k8smetav1.ListOptions) (runtime.Object, error) {
					return &v1.VirtualMachineList{}, nil
				},
				WatchFunc: func(options k8smetav1.ListOptions) (watch.Interface, error) {
					return watch.NewFake(), nil
				},
			}
			app.vmInitInformer = cache.NewSharedIndexInformer(vmListWatch, &v1.VirtualMachine{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
			app.vmInitCache = app.vmInitInformer.GetStore()
			app.vmInitQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			app.vmInitInformer.AddEventHandler(controller.NewResourceEventHandlerFuncsForWorkqueue(app.vmInitQueue))
			go app.vmInitInformer.Run(stopChan)

			app.initCommon()
			// Make sure the informers are synced before continuing -- avoid race conditions
			cache.WaitForCacheSync(stopChan, app.vmPresetInformer.HasSynced, app.vmInitInformer.HasSynced)
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

			key, _ := cache.MetaNamespaceKeyFunc(vm)
			app.vmInitCache.Add(vm)
			app.vmInitQueue.Add(key)

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.vmInitController.Execute()
			// the initializer should inspect the VM and decide nothing needs to be done
			// (and skip the update entirely). So zero requests are expected.
			Expect(len(server.ReceivedRequests())).To(Equal(0))

		})

		It("should initialized a VM if needed", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Initializers = &k8smetav1.Initializers{Pending: []k8smetav1.Initializer{k8smetav1.Initializer{Name: initializerMarking}}}

			key, _ := cache.MetaNamespaceKeyFunc(vm)
			app.vmInitCache.Add(vm)
			app.vmInitQueue.Add(key)

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			app.vmInitController.Execute()
			Expect(len(server.ReceivedRequests())).To(Equal(1))

		})

		It("should apply presets", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Initializers = &k8smetav1.Initializers{Pending: []k8smetav1.Initializer{k8smetav1.Initializer{Name: initializerMarking}}}
			vm.Labels = map[string]string{flavorKey: presetFlavor}

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			err := app.vmInitController.initializeVirtualMachine(vm)

			Expect(err).ToNot(HaveOccurred())

			Expect(len(server.ReceivedRequests())).To(Equal(1))
			// Prove that the VM was annotated (indicates successful application of preset)
			Expect(vm.Annotations["virtualmachinepreset.kubevirt.io/test-preset"]).To(Equal("kubevirt.io/v1alpha1"))
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
