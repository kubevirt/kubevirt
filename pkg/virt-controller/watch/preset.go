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
 * Copyright 2017-2018 Red Hat, Inc.
 *
 */

package watch

import (
	"fmt"
	"reflect"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

type VirtualMachinePresetController struct {
	vmPresetInformer cache.SharedIndexInformer
	vmInitInformer   cache.SharedIndexInformer
	clientset        kubecli.KubevirtClient
	queue            workqueue.RateLimitingInterface
	recorder         record.EventRecorder
	store            cache.Store
}

const initializerMarking = "presets.virtualmachines." + kubev1.GroupName + "/presets-applied"

func NewVirtualMachinePresetController(vmPresetInformer cache.SharedIndexInformer, vmInitInformer cache.SharedIndexInformer, queue workqueue.RateLimitingInterface, vmInitCache cache.Store, clientset kubecli.KubevirtClient, recorder record.EventRecorder) *VirtualMachinePresetController {
	vmi := VirtualMachinePresetController{
		vmPresetInformer: vmPresetInformer,
		vmInitInformer:   vmInitInformer,
		clientset:        clientset,
		queue:            queue,
		recorder:         recorder,
		store:            vmInitCache,
	}
	return &vmi
}

func (c *VirtualMachinePresetController) Run(threadiness int, stopCh chan struct{}) {
	defer controller.HandlePanic()
	defer c.queue.ShutDown()
	log.Log.Info("Starting Virtual Machine Initializer.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.vmPresetInformer.HasSynced, c.vmInitInformer.HasSynced)

	// Start the actual work
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Log.Info("Stopping controller.")
}

func (c *VirtualMachinePresetController) runWorker() {
	for c.Execute() {
	}
}

func (c *VirtualMachinePresetController) Execute() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	err := c.execute(key.(string))

	if err != nil {
		log.Log.Reason(err).Infof("reenqueuing VM %v", key)
		c.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VM %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *VirtualMachinePresetController) execute(key string) error {

	// Fetch the latest VM state from cache
	obj, exists, err := c.store.GetByKey(key)

	if err != nil {
		return err
	}

	// If the VM isn't in the cache, it was just deleted, so shouldn't
	// be initialized
	if exists {
		vm := &kubev1.VirtualMachine{}
		obj.(*kubev1.VirtualMachine).DeepCopyInto(vm)
		// only process VM's that aren't initialized by this controller yet
		if !isVirtualMachineInitialized(vm) {
			return c.initializeVirtualMachine(vm)
		}
	}

	return nil
}

func (c *VirtualMachinePresetController) initializeVirtualMachine(vm *kubev1.VirtualMachine) error {
	// All VM's must be marked as initialized or they are held in limbo forever
	// Collect all errors and defer returning until after the update
	logger := log.Log
	var err error
	success := true

	logger.Object(vm).Info("Initializing VirtualMachine")

	allPresets, err := listPresets(c.vmPresetInformer, vm.GetNamespace())
	if err != nil {
		logger.Object(vm).Errorf("Listing VirtualMachinePresets failed: %v", err)
		return err
	}

	matchingPresets := filterPresets(allPresets, vm, c.recorder)

	if len(matchingPresets) != 0 {
		success = applyPresets(vm, matchingPresets, c.recorder)
	}

	if !success {
		logger.Object(vm).Warning("Marking VM as failed")
		vm.Status.Phase = kubev1.Failed
	}
	// Even failed VM's need to be marked as initialized so they're
	// not re-processed by this controller
	logger.Object(vm).Info("Marking VM as initialized")
	addInitializedAnnotation(vm)
	_, err = c.clientset.VM(vm.Namespace).Update(vm)
	if err != nil {
		logger.Object(vm).Errorf("Could not update VirtualMachine: %v", err)
		return err
	}
	return nil
}

// listPresets returns all VirtualMachinePresets by namespace
func listPresets(vmPresetInformer cache.SharedIndexInformer, namespace string) ([]kubev1.VirtualMachinePreset, error) {
	indexer := vmPresetInformer.GetIndexer()
	selector := labels.NewSelector()
	result := []kubev1.VirtualMachinePreset{}
	err := cache.ListAllByNamespace(indexer, namespace, selector, func(obj interface{}) {
		vm := obj.(*kubev1.VirtualMachinePreset)
		result = append(result, *vm)
	})

	return result, err
}

// filterPresets returns list of VirtualMachinePresets which match given VirtualMachine.
func filterPresets(list []kubev1.VirtualMachinePreset, vm *kubev1.VirtualMachine, recorder record.EventRecorder) []kubev1.VirtualMachinePreset {
	matchingPresets := []kubev1.VirtualMachinePreset{}
	logger := log.Log

	for _, preset := range list {
		selector, err := k8smetav1.LabelSelectorAsSelector(&preset.Spec.Selector)

		if err != nil {
			// Do not return an error from this function--or the VM will be
			// re-enqueued for processing again.
			recorder.Event(vm, k8sv1.EventTypeWarning, kubev1.PresetFailed.String(), fmt.Sprintf("Invalid Preset '%s': %v", preset.Name, err))
			logger.Object(&preset).Reason(err).Errorf("label selector conversion failed: %v", err)
		} else if selector.Matches(labels.Set(vm.GetLabels())) {
			logger.Object(vm).Infof("VirtualMachinePreset %s matches VirtualMachine", preset.GetName())
			matchingPresets = append(matchingPresets, preset)
		}
	}
	return matchingPresets
}

func checkMergeConflicts(presetSpec *kubev1.DomainSpec, vmSpec *kubev1.DomainSpec) error {
	errors := []error{}
	if len(presetSpec.Resources.Requests) > 0 {
		for key, presetReq := range presetSpec.Resources.Requests {
			if vmReq, ok := vmSpec.Resources.Requests[key]; ok {
				if presetReq != vmReq {
					errors = append(errors, fmt.Errorf("spec.resources.requests[%s]: %v != %v", key, presetReq, vmReq))
				}
			}
		}
	}
	if presetSpec.CPU != nil && vmSpec.CPU != nil {
		if !reflect.DeepEqual(presetSpec.CPU, vmSpec.CPU) {
			errors = append(errors, fmt.Errorf("spec.cpu: %v != %v", presetSpec.CPU, vmSpec.CPU))
		}
	}
	if presetSpec.Firmware != nil && vmSpec.Firmware != nil {
		if !reflect.DeepEqual(presetSpec.Firmware, vmSpec.Firmware) {
			errors = append(errors, fmt.Errorf("spec.firmware: %v != %v", presetSpec.Firmware, vmSpec.Firmware))
		}
	}
	if presetSpec.Clock != nil && vmSpec.Clock != nil {
		if !reflect.DeepEqual(presetSpec.Clock.ClockOffset, vmSpec.Clock.ClockOffset) {
			errors = append(errors, fmt.Errorf("spec.clock.clockoffset: %v != %v", presetSpec.Clock.ClockOffset, vmSpec.Clock.ClockOffset))
		}
		if presetSpec.Clock.Timer != nil && vmSpec.Clock.Timer != nil {
			if !reflect.DeepEqual(presetSpec.Clock.Timer, vmSpec.Clock.Timer) {
				errors = append(errors, fmt.Errorf("spec.clock.timer: %v != %v", presetSpec.Clock.Timer, vmSpec.Clock.Timer))
			}
		}
	}
	if presetSpec.Features != nil && vmSpec.Features != nil {
		if !reflect.DeepEqual(presetSpec.Features, vmSpec.Features) {
			errors = append(errors, fmt.Errorf("spec.features: %v != %v", presetSpec.Features, vmSpec.Features))
		}
	}
	if presetSpec.Devices.Watchdog != nil && vmSpec.Devices.Watchdog != nil {
		if !reflect.DeepEqual(presetSpec.Devices.Watchdog, vmSpec.Devices.Watchdog) {
			errors = append(errors, fmt.Errorf("spec.devices.watchdog: %v != %v", presetSpec.Devices.Watchdog, vmSpec.Devices.Watchdog))
		}
	}

	if len(errors) > 0 {
		return utilerrors.NewAggregate(errors)
	}
	return nil
}

func mergeDomainSpec(presetSpec *kubev1.DomainSpec, vmSpec *kubev1.DomainSpec) (bool, error) {
	presetConflicts := checkMergeConflicts(presetSpec, vmSpec)
	applied := false

	if len(presetSpec.Resources.Requests) > 0 {
		if vmSpec.Resources.Requests == nil {
			vmSpec.Resources.Requests = k8sv1.ResourceList{}
			for key, val := range presetSpec.Resources.Requests {
				vmSpec.Resources.Requests[key] = val
			}
		}
		if reflect.DeepEqual(vmSpec.Resources.Requests, presetSpec.Resources.Requests) {
			applied = true
		}
	}
	if presetSpec.CPU != nil {
		if vmSpec.CPU == nil {
			vmSpec.CPU = &kubev1.CPU{}
			presetSpec.CPU.DeepCopyInto(vmSpec.CPU)
		}
		if reflect.DeepEqual(vmSpec.CPU, presetSpec.CPU) {
			applied = true
		}
	}
	if presetSpec.Firmware != nil {
		if vmSpec.Firmware == nil {
			vmSpec.Firmware = &kubev1.Firmware{}
			presetSpec.Firmware.DeepCopyInto(vmSpec.Firmware)
		}
		if reflect.DeepEqual(vmSpec.Firmware, presetSpec.Firmware) {
			applied = true
		}
	}
	if presetSpec.Clock != nil {
		if vmSpec.Clock == nil {
			vmSpec.Clock = &kubev1.Clock{}
			vmSpec.Clock.ClockOffset = presetSpec.Clock.ClockOffset
		}
		if reflect.DeepEqual(vmSpec.Clock, presetSpec.Clock) {
			applied = true
		}

		if presetSpec.Clock.Timer != nil {
			if vmSpec.Clock.Timer == nil {
				vmSpec.Clock.Timer = &kubev1.Timer{}
				presetSpec.Clock.Timer.DeepCopyInto(vmSpec.Clock.Timer)
			}
			if reflect.DeepEqual(vmSpec.Clock.Timer, presetSpec.Clock.Timer) {
				applied = true
			}
		}
	}
	if presetSpec.Features != nil {
		if vmSpec.Features == nil {
			vmSpec.Features = &kubev1.Features{}
			presetSpec.Features.DeepCopyInto(vmSpec.Features)
		}
		if reflect.DeepEqual(vmSpec.Features, presetSpec.Features) {
			applied = true
		}
	}
	if presetSpec.Devices.Watchdog != nil {
		if vmSpec.Devices.Watchdog == nil {
			vmSpec.Devices.Watchdog = &kubev1.Watchdog{}
			presetSpec.Devices.Watchdog.DeepCopyInto(vmSpec.Devices.Watchdog)
		}
		if reflect.DeepEqual(vmSpec.Devices.Watchdog, presetSpec.Devices.Watchdog) {
			applied = true
		}
	}
	return applied, presetConflicts
}

// Compare the domain of every preset to ensure they can all be applied cleanly
func checkPresetConflicts(presets []kubev1.VirtualMachinePreset) error {
	errors := []error{}
	visitedPresets := []kubev1.VirtualMachinePreset{}
	for _, preset := range presets {
		for _, visited := range visitedPresets {
			err := checkMergeConflicts(preset.Spec.Domain, visited.Spec.Domain)
			if err != nil {
				errors = append(errors, fmt.Errorf("presets '%s' and '%s' conflict: %v", preset.Name, visited.Name, err))
			}
		}
		visitedPresets = append(visitedPresets, preset)
	}
	if len(errors) > 0 {
		return utilerrors.NewAggregate(errors)
	}
	return nil
}

func applyPresets(vm *kubev1.VirtualMachine, presets []kubev1.VirtualMachinePreset, recorder record.EventRecorder) bool {
	logger := log.Log
	err := checkPresetConflicts(presets)
	if err != nil {
		msg := fmt.Sprintf("VirtualMachinePresets cannot be applied due to conflicts: %v", err)
		recorder.Event(vm, k8sv1.EventTypeWarning, kubev1.PresetFailed.String(), msg)
		logger.Object(vm).Error(msg)
		return false
	}

	for _, preset := range presets {
		applied, err := mergeDomainSpec(preset.Spec.Domain, &vm.Spec.Domain)
		if err != nil {
			msg := fmt.Sprintf("Unable to apply VirtualMachinePreset '%s': %v", preset.Name, err)
			if applied {
				msg = fmt.Sprintf("Some settings were not applied for VirtualMachinePreset '%s': %v", preset.Name, err)
			}

			recorder.Event(vm, k8sv1.EventTypeNormal, kubev1.Override.String(), msg)
			logger.Object(vm).Info(msg)
		}
		if applied {
			annotateVM(vm, preset)
		}
	}
	return true
}

// isVirtualMachineInitialized checks if this module has applied presets
func isVirtualMachineInitialized(vm *kubev1.VirtualMachine) bool {
	if vm.Annotations != nil {
		_, found := vm.Annotations[initializerMarking]
		return found
	}
	return false
}

func addInitializedAnnotation(vm *kubev1.VirtualMachine) {
	if vm.Annotations == nil {
		vm.Annotations = map[string]string{}
	}
	vm.Annotations[initializerMarking] = kubev1.GroupVersion.String()
}

func annotateVM(vm *kubev1.VirtualMachine, preset kubev1.VirtualMachinePreset) {
	if vm.Annotations == nil {
		vm.Annotations = map[string]string{}
	}
	annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", kubev1.GroupName, preset.Name)
	vm.Annotations[annotationKey] = kubev1.GroupVersion.String()
}
