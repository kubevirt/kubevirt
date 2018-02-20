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

	"k8s.io/api/core/v1"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

type VirtualMachineInitializer struct {
	vmPresetInformer cache.SharedIndexInformer
	vmInitInformer   cache.SharedIndexInformer
	clientset        kubecli.KubevirtClient
	queue            workqueue.RateLimitingInterface
	store            cache.Store
}

const initializerMarking = "presets.virtualmachines.kubevirt.io"

func NewVirtualMachineInitializer(vmPresetInformer cache.SharedIndexInformer, vmInitInformer cache.SharedIndexInformer, queue workqueue.RateLimitingInterface, vmInitCache cache.Store, clientset kubecli.KubevirtClient) *VirtualMachineInitializer {
	vmi := VirtualMachineInitializer{
		vmPresetInformer: vmPresetInformer,
		vmInitInformer:   vmInitInformer,
		clientset:        clientset,
		queue:            queue,
		store:            vmInitCache,
	}
	return &vmi
}

func (c *VirtualMachineInitializer) Run(threadiness int, stopCh chan struct{}) {
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

func (c *VirtualMachineInitializer) runWorker() {
	for c.Execute() {
	}
}

func (c *VirtualMachineInitializer) Execute() bool {
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

func (c *VirtualMachineInitializer) execute(key string) error {

	// Fetch the latest VM state from cache
	obj, exists, err := c.store.GetByKey(key)

	if err != nil {
		return err
	}

	// If the VM isn't in the cache, it was just deleted, so shouldn't
	// be initialized
	if exists {
		var vm *kubev1.VirtualMachine
		vm = obj.(*kubev1.VirtualMachine)
		// only process VM's that aren't initialized by this controller yet
		if !isInitialized(vm) {
			return c.initializeVirtualMachine(vm)
		}
	}

	return nil
}

func (c *VirtualMachineInitializer) initializeVirtualMachine(vm *kubev1.VirtualMachine) error {
	// All VM's must be marked as initialized or they are held in limbo forever
	// Collect all errors and defer returning until after the update
	logger := log.Log
	errors := []error{}
	var err error

	logger.Object(vm).Info("Initializing VirtualMachine")

	allPresets := listPresets(c.vmPresetInformer, vm.GetNamespace())

	matchingPresets := filterPresets(allPresets, vm)

	if len(matchingPresets) != 0 {
		err = applyPresets(vm, matchingPresets)
		if err != nil {
			// A more specific error should have been logged during the applyPresets call.
			// We don't know *which* preset in the list was problematic at this level.
			logger.Object(vm).Errorf("Unable to apply presets to virtualmachine: %v", err)
			errors = append(errors, err)
		}
	}

	logger.Object(vm).Info("Marking VM as initialized and updating")
	removeInitializer(vm)
	_, err = c.clientset.VM(vm.Namespace).Update(vm)
	if err != nil {
		logger.Object(vm).Errorf("Could not update VirtualMachine: %v", err)
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return utilerrors.NewAggregate(errors)
	}
	return nil
}

// FIXME: There is probably a way to set up the vmPresetInformer such that
// items are already partitioned into namespaces (and can just be listed)
func listPresets(vmPresetInformer cache.SharedIndexInformer, namespace string) []kubev1.VirtualMachinePreset {
	result := []kubev1.VirtualMachinePreset{}
	for _, obj := range vmPresetInformer.GetStore().List() {
		var preset *kubev1.VirtualMachinePreset
		preset = obj.(*kubev1.VirtualMachinePreset)
		if preset.Namespace == namespace {
			result = append(result, *preset)
		}
	}
	return result
}

// filterPresets returns list of VirtualMachinePresets which match given VirtualMachine.
func filterPresets(list []kubev1.VirtualMachinePreset, vm *kubev1.VirtualMachine) []kubev1.VirtualMachinePreset {
	matchingPresets := []kubev1.VirtualMachinePreset{}

	logger := log.Log

	for _, preset := range list {
		selector, err := k8smetav1.LabelSelectorAsSelector(&preset.Spec.Selector)
		if err != nil {
			// FIXME: create an event here
			// Do not return an error from this function--or the VM will be
			// re-enqueued for processing again.
			logger.Object(&preset).Reason(err).Errorf("label selector conversion failed: %v", err)
		} else if selector.Matches(labels.Set(vm.GetLabels())) {
			logger.Object(vm).Infof("VirtualMachinePreset %s matches VirtualMachine", preset.GetName())
			matchingPresets = append(matchingPresets, preset)
		}
	}
	return matchingPresets
}

func checkPresetMergeConflicts(presetSpec *kubev1.DomainSpec, vmSpec *kubev1.DomainSpec) error {
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

func mergeDomainSpec(presetSpec *kubev1.DomainSpec, vmSpec *kubev1.DomainSpec) error {
	err := checkPresetMergeConflicts(presetSpec, vmSpec)
	if err != nil {
		return err
	}
	if len(presetSpec.Resources.Requests) > 0 {
		if vmSpec.Resources.Requests == nil {
			vmSpec.Resources.Requests = v1.ResourceList{}
		}
		for key, val := range presetSpec.Resources.Requests {
			vmSpec.Resources.Requests[key] = val
		}
	}
	if presetSpec.CPU != nil {
		if vmSpec.CPU == nil {
			vmSpec.CPU = &kubev1.CPU{}
		}
		presetSpec.CPU.DeepCopyInto(vmSpec.CPU)
	}
	if presetSpec.Firmware != nil {
		if vmSpec.Firmware == nil {
			vmSpec.Firmware = &kubev1.Firmware{}
		}
		presetSpec.Firmware.DeepCopyInto(vmSpec.Firmware)
	}
	if presetSpec.Clock != nil {
		if vmSpec.Clock == nil {
			vmSpec.Clock = &kubev1.Clock{}
		}
		vmSpec.Clock.ClockOffset = presetSpec.Clock.ClockOffset
		if presetSpec.Clock.Timer != nil {
			if vmSpec.Clock.Timer == nil {
				vmSpec.Clock.Timer = &kubev1.Timer{}
			}
			presetSpec.Clock.Timer.DeepCopyInto(vmSpec.Clock.Timer)
		}
	}
	if presetSpec.Features != nil {
		if vmSpec.Features == nil {
			vmSpec.Features = &kubev1.Features{}
		}
		presetSpec.Features.DeepCopyInto(vmSpec.Features)
	}
	if presetSpec.Devices.Watchdog != nil {
		if vmSpec.Devices.Watchdog == nil {
			vmSpec.Devices.Watchdog = &kubev1.Watchdog{}
		}
		presetSpec.Devices.Watchdog.DeepCopyInto(vmSpec.Devices.Watchdog)
	}
	return nil
}

func applyPresets(vm *kubev1.VirtualMachine, presets []kubev1.VirtualMachinePreset) error {
	logger := log.Log
	for _, preset := range presets {
		err := mergeDomainSpec(preset.Spec.Domain, &vm.Spec.Domain)
		if err != nil {
			// FIXME: a Kubernetes event should be reported here
			logger.Object(vm).Errorf("Unable to apply Preset '%s': %v", preset.Name, err)
			return nil
		}
	}

	annotateVM(vm, presets)
	return nil
}

// isInitialized checks if *this* module has initialized the VM,
// which is distinct from "has the VM been initialized by all controllers?"
func isInitialized(vm *kubev1.VirtualMachine) bool {
	// if initializers is nil/empty then consider this resource as initialized
	if vm.Initializers != nil && len(vm.Initializers.Pending) > 0 {
		for _, i := range vm.Initializers.Pending {
			if i.Name == initializerMarking {
				return false
			}
		}
	}
	return true
}

func removeInitializer(vm *kubev1.VirtualMachine) {
	if vm.Initializers == nil {
		// If Initializers is nil, there's nothing to remove.
		return
	}
	newInitilizers := []k8smetav1.Initializer{}
	for _, i := range vm.Initializers.Pending {
		if i.Name != initializerMarking {
			newInitilizers = append(newInitilizers, i)
		}
	}
	vm.Initializers.Pending = newInitilizers
}

func annotateVM(vm *kubev1.VirtualMachine, presets []kubev1.VirtualMachinePreset) {
	if vm.Annotations == nil {
		vm.Annotations = map[string]string{}
	}
	for _, preset := range presets {
		annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", kubev1.GroupName, preset.Name)
		vm.Annotations[annotationKey] = kubev1.GroupVersion.String()
	}
}
