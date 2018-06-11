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
	vmiPresetInformer cache.SharedIndexInformer
	vmiInitInformer   cache.SharedIndexInformer
	clientset         kubecli.KubevirtClient
	queue             workqueue.RateLimitingInterface
	recorder          record.EventRecorder
	store             cache.Store
}

const initializerMarking = "presets.virtualmachines." + kubev1.GroupName + "/presets-applied"
const exclusionMarking = "virtualmachineinstancepresets.admission.kubevirt.io/exclude"

func NewVirtualMachinePresetController(vmiPresetInformer cache.SharedIndexInformer, vmiInitInformer cache.SharedIndexInformer, queue workqueue.RateLimitingInterface, vmiInitCache cache.Store, clientset kubecli.KubevirtClient, recorder record.EventRecorder) *VirtualMachinePresetController {
	vmii := VirtualMachinePresetController{
		vmiPresetInformer: vmiPresetInformer,
		vmiInitInformer:   vmiInitInformer,
		clientset:         clientset,
		queue:             queue,
		recorder:          recorder,
		store:             vmiInitCache,
	}
	return &vmii
}

func (c *VirtualMachinePresetController) Run(threadiness int, stopCh chan struct{}) {
	defer controller.HandlePanic()
	defer c.queue.ShutDown()
	log.Log.Info("Starting Virtual Machine Initializer.")

	// Wait for cache sync before we start the pod controller
	cache.WaitForCacheSync(stopCh, c.vmiPresetInformer.HasSynced, c.vmiInitInformer.HasSynced)

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
		log.Log.Reason(err).Infof("reenqueuing VirtualMachineInstance %v", key)
		c.queue.AddRateLimited(key)
	} else {
		log.Log.V(4).Infof("processed VirtualMachineInstance %v", key)
		c.queue.Forget(key)
	}
	return true
}

func (c *VirtualMachinePresetController) execute(key string) error {

	// Fetch the latest VirtualMachineInstance state from cache
	obj, exists, err := c.store.GetByKey(key)

	if err != nil {
		return err
	}

	// If the VirtualMachineInstance isn't in the cache, it was just deleted, so shouldn't
	// be initialized
	if exists {
		vmi := &kubev1.VirtualMachineInstance{}
		obj.(*kubev1.VirtualMachineInstance).DeepCopyInto(vmi)
		// only process VirtualMachineInstance's that aren't initialized by this controller yet
		if !isVirtualMachineInitialized(vmi) {
			return c.initializeVirtualMachine(vmi)
		}
	}

	return nil
}

func (c *VirtualMachinePresetController) initializeVirtualMachine(vmi *kubev1.VirtualMachineInstance) error {
	// All VirtualMachineInstance's must be marked as initialized or they are held in limbo forever
	// Collect all errors and defer returning until after the update
	logger := log.Log
	var err error
	success := true

	if !isVmExcluded(vmi) {
		logger.Object(vmi).Info("Initializing VirtualMachineInstance")

		allPresets, err := listPresets(c.vmiPresetInformer, vmi.GetNamespace())
		if err != nil {
			logger.Object(vmi).Errorf("Listing VirtualMachinePresets failed: %v", err)
			return err
		}

		matchingPresets := filterPresets(allPresets, vmi, c.recorder)

		if len(matchingPresets) != 0 {
			success = applyPresets(vmi, matchingPresets, c.recorder)
		}

		if !success {
			logger.Object(vmi).Warning("Marking VirtualMachineInstance as failed")
			vmi.Status.Phase = kubev1.Failed
		} else {
			logger.Object(vmi).V(4).Info("Setting default values on VirtualMachine")
			kubev1.SetObjectDefaults_VirtualMachineInstance(vmi)
		}
	} else {
		logger.Object(vmi).Infof("VirtualMachineInstance is excluded from VirtualMachinePresets")
	}
	// Even failed VirtualMachineInstance's need to be marked as initialized so they're
	// not re-processed by this controller
	logger.Object(vmi).Info("Marking VirtualMachineInstance as initialized")
	addInitializedAnnotation(vmi)
	_, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Update(vmi)
	if err != nil {
		logger.Object(vmi).Errorf("Could not update VirtualMachineInstance: %v", err)
		return err
	}
	return nil
}

// listPresets returns all VirtualMachinePresets by namespace
func listPresets(vmiPresetInformer cache.SharedIndexInformer, namespace string) ([]kubev1.VirtualMachineInstancePreset, error) {
	indexer := vmiPresetInformer.GetIndexer()
	selector := labels.NewSelector()
	result := []kubev1.VirtualMachineInstancePreset{}
	err := cache.ListAllByNamespace(indexer, namespace, selector, func(obj interface{}) {
		vmi := obj.(*kubev1.VirtualMachineInstancePreset)
		result = append(result, *vmi)
	})

	return result, err
}

// filterPresets returns list of VirtualMachinePresets which match given VirtualMachineInstance.
func filterPresets(list []kubev1.VirtualMachineInstancePreset, vmi *kubev1.VirtualMachineInstance, recorder record.EventRecorder) []kubev1.VirtualMachineInstancePreset {
	matchingPresets := []kubev1.VirtualMachineInstancePreset{}
	logger := log.Log

	for _, preset := range list {
		selector, err := k8smetav1.LabelSelectorAsSelector(&preset.Spec.Selector)

		if err != nil {
			// Do not return an error from this function--or the VirtualMachineInstance will be
			// re-enqueued for processing again.
			recorder.Event(vmi, k8sv1.EventTypeWarning, kubev1.PresetFailed.String(), fmt.Sprintf("Invalid Preset '%s': %v", preset.Name, err))
			logger.Object(&preset).Reason(err).Errorf("label selector conversion failed: %v", err)
		} else if selector.Matches(labels.Set(vmi.GetLabels())) {
			logger.Object(vmi).Infof("VirtualMachineInstancePreset %s matches VirtualMachineInstance", preset.GetName())
			matchingPresets = append(matchingPresets, preset)
		}
	}
	return matchingPresets
}

func checkMergeConflicts(presetSpec *kubev1.DomainSpec, vmiSpec *kubev1.DomainSpec) error {
	errors := []error{}
	if len(presetSpec.Resources.Requests) > 0 {
		for key, presetReq := range presetSpec.Resources.Requests {
			if vmiReq, ok := vmiSpec.Resources.Requests[key]; ok {
				if presetReq != vmiReq {
					errors = append(errors, fmt.Errorf("spec.resources.requests[%s]: %v != %v", key, presetReq, vmiReq))
				}
			}
		}
	}
	if presetSpec.CPU != nil && vmiSpec.CPU != nil {
		if !reflect.DeepEqual(presetSpec.CPU, vmiSpec.CPU) {
			errors = append(errors, fmt.Errorf("spec.cpu: %v != %v", presetSpec.CPU, vmiSpec.CPU))
		}
	}
	if presetSpec.Firmware != nil && vmiSpec.Firmware != nil {
		if !reflect.DeepEqual(presetSpec.Firmware, vmiSpec.Firmware) {
			errors = append(errors, fmt.Errorf("spec.firmware: %v != %v", presetSpec.Firmware, vmiSpec.Firmware))
		}
	}
	if presetSpec.Clock != nil && vmiSpec.Clock != nil {
		if !reflect.DeepEqual(presetSpec.Clock.ClockOffset, vmiSpec.Clock.ClockOffset) {
			errors = append(errors, fmt.Errorf("spec.clock.clockoffset: %v != %v", presetSpec.Clock.ClockOffset, vmiSpec.Clock.ClockOffset))
		}
		if presetSpec.Clock.Timer != nil && vmiSpec.Clock.Timer != nil {
			if !reflect.DeepEqual(presetSpec.Clock.Timer, vmiSpec.Clock.Timer) {
				errors = append(errors, fmt.Errorf("spec.clock.timer: %v != %v", presetSpec.Clock.Timer, vmiSpec.Clock.Timer))
			}
		}
	}
	if presetSpec.Features != nil && vmiSpec.Features != nil {
		if !reflect.DeepEqual(presetSpec.Features, vmiSpec.Features) {
			errors = append(errors, fmt.Errorf("spec.features: %v != %v", presetSpec.Features, vmiSpec.Features))
		}
	}
	if presetSpec.Devices.Watchdog != nil && vmiSpec.Devices.Watchdog != nil {
		if !reflect.DeepEqual(presetSpec.Devices.Watchdog, vmiSpec.Devices.Watchdog) {
			errors = append(errors, fmt.Errorf("spec.devices.watchdog: %v != %v", presetSpec.Devices.Watchdog, vmiSpec.Devices.Watchdog))
		}
	}

	if len(errors) > 0 {
		return utilerrors.NewAggregate(errors)
	}
	return nil
}

func mergeDomainSpec(presetSpec *kubev1.DomainSpec, vmiSpec *kubev1.DomainSpec) (bool, error) {
	presetConflicts := checkMergeConflicts(presetSpec, vmiSpec)
	applied := false

	if len(presetSpec.Resources.Requests) > 0 {
		if vmiSpec.Resources.Requests == nil {
			vmiSpec.Resources.Requests = k8sv1.ResourceList{}
			for key, val := range presetSpec.Resources.Requests {
				vmiSpec.Resources.Requests[key] = val
			}
		}
		if reflect.DeepEqual(vmiSpec.Resources.Requests, presetSpec.Resources.Requests) {
			applied = true
		}
	}
	if presetSpec.CPU != nil {
		if vmiSpec.CPU == nil {
			vmiSpec.CPU = &kubev1.CPU{}
			presetSpec.CPU.DeepCopyInto(vmiSpec.CPU)
		}
		if reflect.DeepEqual(vmiSpec.CPU, presetSpec.CPU) {
			applied = true
		}
	}
	if presetSpec.Firmware != nil {
		if vmiSpec.Firmware == nil {
			vmiSpec.Firmware = &kubev1.Firmware{}
			presetSpec.Firmware.DeepCopyInto(vmiSpec.Firmware)
		}
		if reflect.DeepEqual(vmiSpec.Firmware, presetSpec.Firmware) {
			applied = true
		}
	}
	if presetSpec.Clock != nil {
		if vmiSpec.Clock == nil {
			vmiSpec.Clock = &kubev1.Clock{}
			vmiSpec.Clock.ClockOffset = presetSpec.Clock.ClockOffset
		}
		if reflect.DeepEqual(vmiSpec.Clock, presetSpec.Clock) {
			applied = true
		}

		if presetSpec.Clock.Timer != nil {
			if vmiSpec.Clock.Timer == nil {
				vmiSpec.Clock.Timer = &kubev1.Timer{}
				presetSpec.Clock.Timer.DeepCopyInto(vmiSpec.Clock.Timer)
			}
			if reflect.DeepEqual(vmiSpec.Clock.Timer, presetSpec.Clock.Timer) {
				applied = true
			}
		}
	}
	if presetSpec.Features != nil {
		if vmiSpec.Features == nil {
			vmiSpec.Features = &kubev1.Features{}
			presetSpec.Features.DeepCopyInto(vmiSpec.Features)
		}
		if reflect.DeepEqual(vmiSpec.Features, presetSpec.Features) {
			applied = true
		}
	}
	if presetSpec.Devices.Watchdog != nil {
		if vmiSpec.Devices.Watchdog == nil {
			vmiSpec.Devices.Watchdog = &kubev1.Watchdog{}
			presetSpec.Devices.Watchdog.DeepCopyInto(vmiSpec.Devices.Watchdog)
		}
		if reflect.DeepEqual(vmiSpec.Devices.Watchdog, presetSpec.Devices.Watchdog) {
			applied = true
		}
	}
	return applied, presetConflicts
}

// Compare the domain of every preset to ensure they can all be applied cleanly
func checkPresetConflicts(presets []kubev1.VirtualMachineInstancePreset) error {
	errors := []error{}
	visitedPresets := []kubev1.VirtualMachineInstancePreset{}
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

func applyPresets(vmi *kubev1.VirtualMachineInstance, presets []kubev1.VirtualMachineInstancePreset, recorder record.EventRecorder) bool {
	logger := log.Log
	err := checkPresetConflicts(presets)
	if err != nil {
		msg := fmt.Sprintf("VirtualMachinePresets cannot be applied due to conflicts: %v", err)
		recorder.Event(vmi, k8sv1.EventTypeWarning, kubev1.PresetFailed.String(), msg)
		logger.Object(vmi).Error(msg)
		return false
	}

	for _, preset := range presets {
		applied, err := mergeDomainSpec(preset.Spec.Domain, &vmi.Spec.Domain)
		if err != nil {
			msg := fmt.Sprintf("Unable to apply VirtualMachineInstancePreset '%s': %v", preset.Name, err)
			if applied {
				msg = fmt.Sprintf("Some settings were not applied for VirtualMachineInstancePreset '%s': %v", preset.Name, err)
			}

			recorder.Event(vmi, k8sv1.EventTypeNormal, kubev1.Override.String(), msg)
			logger.Object(vmi).Info(msg)
		}
		if applied {
			annotateVMI(vmi, preset)
		}
	}
	return true
}

// isVirtualMachineInitialized checks if this module has applied presets
func isVirtualMachineInitialized(vmi *kubev1.VirtualMachineInstance) bool {
	if vmi.Annotations != nil {
		_, found := vmi.Annotations[initializerMarking]
		return found
	}
	return false
}

func isVmExcluded(vmi *kubev1.VirtualMachineInstance) bool {
	if vmi.Annotations != nil {
		excluded, ok := vmi.Annotations[exclusionMarking]
		return ok && (excluded == "true")
	}
	return false
}

func addInitializedAnnotation(vmi *kubev1.VirtualMachineInstance) {
	if vmi.Annotations == nil {
		vmi.Annotations = map[string]string{}
	}
	vmi.Annotations[initializerMarking] = kubev1.GroupVersion.String()
	if !controller.HasFinalizer(vmi, kubev1.VirtualMachineInstanceFinalizer) {
		vmi.Finalizers = append(vmi.Finalizers, kubev1.VirtualMachineInstanceFinalizer)
	}
}

func annotateVMI(vmi *kubev1.VirtualMachineInstance, preset kubev1.VirtualMachineInstancePreset) {
	if vmi.Annotations == nil {
		vmi.Annotations = map[string]string{}
	}
	annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", kubev1.GroupName, preset.Name)
	vmi.Annotations[annotationKey] = kubev1.GroupVersion.String()
}
