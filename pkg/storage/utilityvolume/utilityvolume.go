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
 * Copyright The KubeVirt Authors.
 *
 */

package utilityvolume

import (
	"fmt"
	"slices"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
)

// Find reports whether volumeName is in the VMI spec as a utility volume of
// claimName and volumeType, and errors on another source under that name.
func Find(vmi *v1.VirtualMachineInstance, volumeName, claimName string, volumeType v1.UtilityVolumeType) (bool, error) {
	if vmi == nil {
		return false, nil
	}
	for _, volume := range vmi.Spec.UtilityVolumes {
		if volume.Name != volumeName {
			continue
		}
		if volume.ClaimName != claimName || volume.Type == nil || *volume.Type != volumeType {
			return false, fmt.Errorf("utility volume %s has a different source", volumeName)
		}
		return true, nil
	}
	return false, nil
}

// AddPatch returns the JSON patch adding volumeName as a utility volume of
// claimName and volumeType, guarded against a concurrent change to the VMI.
func AddPatch(vmi *v1.VirtualMachineInstance, volumeName, claimName string, volumeType v1.UtilityVolumeType) ([]byte, error) {
	volumes := append(slices.Clone(vmi.Spec.UtilityVolumes), v1.UtilityVolume{
		Name:                              volumeName,
		PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
		Type:                              pointer.P(volumeType),
	})
	patchSet := patch.New(
		patch.WithTest("/metadata/uid", vmi.UID),
		patch.WithTest("/spec/utilityVolumes", vmi.Spec.UtilityVolumes),
		patch.WithAdd("/spec/utilityVolumes", volumes),
	)
	if vmi.ResourceVersion != "" {
		patchSet.AddOption(patch.WithTest("/metadata/resourceVersion", vmi.ResourceVersion))
	}
	return patchSet.GeneratePayload()
}

// Attached reports whether volumeName is mounted in the VMI's pod and usable.
func Attached(vmi *v1.VirtualMachineInstance, volumeName string) bool {
	if vmi == nil {
		return false
	}

	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.Name == volumeName {
			return volumeStatus.HotplugVolume != nil && volumeStatus.Phase == v1.HotplugVolumeMounted
		}
	}
	return false
}

// Detached reports whether volumeName is gone from both the spec and the volume
// status. Not the negation of Attached: a volume removed from the spec is still
// on the way out until virt-handler stops reporting it, and deleting the PVC
// early is what leaves it hanging on kubernetes.io/pvc-protection.
func Detached(vmi *v1.VirtualMachineInstance, volumeName string) bool {
	if vmi == nil {
		return true
	}

	for _, volume := range vmi.Spec.UtilityVolumes {
		if volume.Name == volumeName {
			return false
		}
	}

	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.Name == volumeName {
			return false
		}
	}

	return true
}
