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

package mutating_webhook

import (
	"fmt"
	"reflect"

	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/cache"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
)

const exclusionMarking = "virtualmachineinstancepresets.admission.kubevirt.io/exclude"

// listPresets returns all VirtualMachinePresets by namespace
func listPresets(vmiPresetInformer cache.SharedIndexInformer, namespace string) ([]kubev1.VirtualMachineInstancePreset, error) {
	indexer := vmiPresetInformer.GetIndexer()
	selector := labels.NewSelector()
	result := []kubev1.VirtualMachineInstancePreset{}
	err := cache.ListAllByNamespace(indexer, namespace, selector, func(obj interface{}) {
		preset := obj.(*kubev1.VirtualMachineInstancePreset)
		result = append(result, *preset)
	})
	return result, err
}

// filterPresets returns list of VirtualMachinePresets which match given VirtualMachineInstance.
func filterPresets(list []kubev1.VirtualMachineInstancePreset, vmi *kubev1.VirtualMachineInstance) ([]kubev1.VirtualMachineInstancePreset, error) {
	matchingPresets := []kubev1.VirtualMachineInstancePreset{}

	for _, preset := range list {
		selector, err := k8smetav1.LabelSelectorAsSelector(&preset.Spec.Selector)

		if err != nil {
			return matchingPresets, err
		} else if selector.Matches(labels.Set(vmi.GetLabels())) {
			log.Log.Object(vmi).Infof("VirtualMachineInstancePreset %s matches VirtualMachineInstance", preset.GetName())
			matchingPresets = append(matchingPresets, preset)
		}
	}
	return matchingPresets, nil
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
	if presetSpec.IOThreadsPolicy != nil && vmiSpec.IOThreadsPolicy != nil {
		if !reflect.DeepEqual(presetSpec.IOThreadsPolicy, vmiSpec.IOThreadsPolicy) {
			errors = append(errors, fmt.Errorf("spec.ioThreadsPolicy: %v != %v", presetSpec.IOThreadsPolicy, vmiSpec.IOThreadsPolicy))
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
	if presetSpec.IOThreadsPolicy != nil {
		if vmiSpec.IOThreadsPolicy == nil {
			ioThreadsPolicy := *presetSpec.IOThreadsPolicy
			vmiSpec.IOThreadsPolicy = &(ioThreadsPolicy)
		}
		if reflect.DeepEqual(vmiSpec.IOThreadsPolicy, presetSpec.IOThreadsPolicy) {
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

func applyPresets(vmi *kubev1.VirtualMachineInstance, presetInformer cache.SharedIndexInformer) error {
	if isVMIExcluded(vmi) {
		log.Log.Object(vmi).Info("VMI excluded from preset logic")
		return nil
	}

	presets, err := listPresets(presetInformer, vmi.Namespace)
	if err != nil {
		return err
	}

	presets, err = filterPresets(presets, vmi)
	if err != nil {
		return err
	}

	if len(presets) == 0 {
		log.Log.Object(vmi).V(4).Infof("Unable to find any preset that can be accepted to the VMI %s", vmi.Name)
		return nil
	}

	err = checkPresetConflicts(presets)
	if err != nil {
		return fmt.Errorf("VirtualMachinePresets cannot be applied due to conflicts: %v", err)
	}

	for _, preset := range presets {
		applied, err := mergeDomainSpec(preset.Spec.Domain, &vmi.Spec.Domain)
		if applied {
			if err != nil {
				log.Log.Object(vmi).Warningf("Some settings were not applied for VirtualMachineInstancePreset '%s': %v", preset.Name, err)
			}
			annotateVMI(vmi, preset)
			log.Log.Object(vmi).V(4).Infof("Apply preset %s", preset.Name)
		} else {
			log.Log.Object(vmi).Warningf("Unable to apply VirtualMachineInstancePreset '%s': %v", preset.Name, err)
		}
	}
	return nil
}

func annotateVMI(vmi *kubev1.VirtualMachineInstance, preset kubev1.VirtualMachineInstancePreset) {
	if vmi.Annotations == nil {
		vmi.Annotations = map[string]string{}
	}
	annotationKey := fmt.Sprintf("virtualmachinepreset.%s/%s", kubev1.GroupName, preset.Name)
	vmi.Annotations[annotationKey] = kubev1.GroupVersion.String()
}

func isVMIExcluded(vmi *kubev1.VirtualMachineInstance) bool {
	if vmi.Annotations != nil {
		excluded, ok := vmi.Annotations[exclusionMarking]
		return ok && (excluded == "true")
	}
	return false
}
