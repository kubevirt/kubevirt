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
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
)

// Attach hotplugs pvcName into the VMI as a utility volume called volumeName,
// and is a no-op when the volume is already in the spec. The patch tests the
// whole list, so a VMI whose utility volumes changed underneath us fails with a
// conflict rather than dropping the other writer's volume.
func Attach(client kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, volumeName, pvcName string, volumeType v1.UtilityVolumeType) error {
	for _, vol := range vmi.Spec.UtilityVolumes {
		if vol.Name == volumeName {
			return nil
		}
	}

	utilityVolume := v1.UtilityVolume{
		Name: volumeName,
		PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: pvcName,
		},
		Type: pointer.P(volumeType),
	}

	patchSet := patch.New(
		patch.WithTest("/spec/utilityVolumes", vmi.Spec.UtilityVolumes),
	)

	newUtilityVolumes := append(vmi.Spec.UtilityVolumes, utilityVolume)
	if len(vmi.Spec.UtilityVolumes) > 0 {
		patchSet.AddOption(patch.WithReplace("/spec/utilityVolumes", newUtilityVolumes))
	} else {
		patchSet.AddOption(patch.WithAdd("/spec/utilityVolumes", newUtilityVolumes))
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return fmt.Errorf("failed to generate the patch attaching utility volume %s: %w", volumeName, err)
	}

	_, err = client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to attach utility volume %s to vmi %s: %w", volumeName, vmi.Name, err)
	}

	log.Log.Object(vmi).Infof("Attaching utility volume %s (%s) to vmi %s", volumeName, pvcName, vmi.Name)
	return nil
}

// Detach removes volumeName from the VMI's utility volumes, and is a no-op when
// there are none.
func Detach(client kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, volumeName string) error {
	if len(vmi.Spec.UtilityVolumes) == 0 {
		return nil
	}

	newUtilityVolumes := make([]v1.UtilityVolume, 0, len(vmi.Spec.UtilityVolumes))
	for _, vol := range vmi.Spec.UtilityVolumes {
		if vol.Name != volumeName {
			newUtilityVolumes = append(newUtilityVolumes, vol)
		}
	}

	patchSet := patch.New(
		patch.WithTest("/spec/utilityVolumes", vmi.Spec.UtilityVolumes),
	)
	if len(newUtilityVolumes) == 0 {
		patchSet.AddOption(patch.WithRemove("/spec/utilityVolumes"))
	} else {
		patchSet.AddOption(patch.WithReplace("/spec/utilityVolumes", newUtilityVolumes))
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return fmt.Errorf("failed to generate the patch detaching utility volume %s: %w", volumeName, err)
	}

	_, err = client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to detach utility volume %s from vmi %s: %w", volumeName, vmi.Name, err)
	}

	log.Log.Object(vmi).Infof("Detaching utility volume %s from vmi %s", volumeName, vmi.Name)
	return nil
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

	for _, vol := range vmi.Spec.UtilityVolumes {
		if vol.Name == volumeName {
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
