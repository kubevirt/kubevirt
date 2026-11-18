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

package memorydump

import (
	"context"
	"fmt"
	"slices"

	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

const (
	ErrorReason = "MemoryDumpError"
	failed      = "Memory dump failed"

	stoppedBeforeCompletion = "The VMI stopped before the memory dump completed"
)

// RemoveDeprecatedVolumes removes the MemoryDump volumes that memory dumps were attached as before
// they moved to utility volumes.
// TODO: remove once upgrades from releases that attached MemoryDump volumes are no longer supported.
func RemoveDeprecatedVolumes(spec *v1.VirtualMachineInstanceSpec) {
	spec.Volumes = slices.DeleteFunc(spec.Volumes, func(volume v1.Volume) bool {
		return volume.MemoryDump != nil
	})
}

func HandleRequest(client kubecli.KubevirtClient, vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, pvcStore cache.Store) error {
	request := vm.Status.MemoryDumpRequest
	if request == nil {
		return nil
	}

	switch request.Phase {
	case v1.MemoryDumpAssociating:
		if vmi == nil || vmi.DeletionTimestamp != nil || !vmi.IsRunning() {
			return nil
		}
		return attachMemoryDumpVolume(client, vmi, request.ClaimName)
	case v1.MemoryDumpUnmounting, v1.MemoryDumpFailed:
		if err := patchMemoryDumpPVCAnnotation(client, vm, pvcStore); err != nil {
			return err
		}
		return detachMemoryDumpVolume(client, vmi, request.ClaimName)
	case v1.MemoryDumpDissociating:
		return detachMemoryDumpVolume(client, vmi, request.ClaimName)
	}

	return nil
}

func UpdateRequest(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	if vm.Status.MemoryDumpRequest == nil {
		return
	}

	updatedMemoryDumpReq := vm.Status.MemoryDumpRequest.DeepCopy()

	if vm.Status.MemoryDumpRequest.Remove {
		updatedMemoryDumpReq.Phase = v1.MemoryDumpDissociating
	}

	switch vm.Status.MemoryDumpRequest.Phase {
	case v1.MemoryDumpCompleted:
		return
	case v1.MemoryDumpAssociating:
		if vmi != nil && hasMemoryDumpVolume(vmi, vm.Status.MemoryDumpRequest.ClaimName) {
			updatedMemoryDumpReq.Phase = v1.MemoryDumpInProgress
		}
	case v1.MemoryDumpInProgress:
		if !hasRunningMemoryDumpVolume(vmi, vm.Status.MemoryDumpRequest.ClaimName) {
			if !vm.Status.MemoryDumpRequest.Remove {
				updatedMemoryDumpReq.Phase = v1.MemoryDumpFailed
				updatedMemoryDumpReq.Message = stoppedBeforeCompletion
			}
			break
		}
		for _, volumeStatus := range vmi.Status.VolumeStatus {
			if volumeStatus.Name == vm.Status.MemoryDumpRequest.ClaimName &&
				volumeStatus.MemoryDumpVolume != nil {
				if volumeStatus.MemoryDumpVolume.StartTimestamp != nil {
					updatedMemoryDumpReq.StartTimestamp = volumeStatus.MemoryDumpVolume.StartTimestamp
				}
				if volumeStatus.Phase == v1.MemoryDumpVolumeCompleted {
					updatedMemoryDumpReq.Phase = v1.MemoryDumpUnmounting
					updatedMemoryDumpReq.EndTimestamp = volumeStatus.MemoryDumpVolume.EndTimestamp
					updatedMemoryDumpReq.FileName = &volumeStatus.MemoryDumpVolume.TargetFileName
				} else if volumeStatus.Phase == v1.MemoryDumpVolumeFailed {
					updatedMemoryDumpReq.Phase = v1.MemoryDumpFailed
					updatedMemoryDumpReq.Message = volumeStatus.Message
					updatedMemoryDumpReq.EndTimestamp = volumeStatus.MemoryDumpVolume.EndTimestamp
				}
			}
		}
	case v1.MemoryDumpUnmounting:
		if vmi != nil {
			for _, volumeStatus := range vmi.Status.VolumeStatus {
				if volumeStatus.Name == vm.Status.MemoryDumpRequest.ClaimName {
					return
				}
			}
		}
		updatedMemoryDumpReq.Phase = v1.MemoryDumpCompleted
	case v1.MemoryDumpDissociating:
		if vmi != nil {
			for _, volumeStatus := range vmi.Status.VolumeStatus {
				if volumeStatus.Name == vm.Status.MemoryDumpRequest.ClaimName {
					return
				}
			}
		}
		updatedMemoryDumpReq = nil
	}

	vm.Status.MemoryDumpRequest = updatedMemoryDumpReq
}

func hasRunningMemoryDumpVolume(vmi *v1.VirtualMachineInstance, claimName string) bool {
	return vmi != nil && vmi.IsRunning() && hasMemoryDumpVolume(vmi, claimName)
}

func hasMemoryDumpVolume(vmi *v1.VirtualMachineInstance, claimName string) bool {
	return hasUtilityVolume(vmi, claimName) || hasDeprecatedVolume(vmi, claimName)
}

func hasUtilityVolume(vmi *v1.VirtualMachineInstance, claimName string) bool {
	return slices.ContainsFunc(vmi.Spec.UtilityVolumes, func(volume v1.UtilityVolume) bool {
		return volume.Name == claimName
	})
}

// TODO: remove together with RemoveDeprecatedVolumes.
func hasDeprecatedVolume(vmi *v1.VirtualMachineInstance, claimName string) bool {
	return slices.ContainsFunc(vmi.Spec.Volumes, func(volume v1.Volume) bool {
		return volume.Name == claimName && volume.MemoryDump != nil
	})
}

func attachMemoryDumpVolume(client kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, claimName string) error {
	if hasMemoryDumpVolume(vmi, claimName) {
		return nil
	}
	memoryDumpVolume := v1.UtilityVolume{
		Name: claimName,
		PersistentVolumeClaimVolumeSource: k8score.PersistentVolumeClaimVolumeSource{
			ClaimName: claimName,
		},
		Type: new(v1.MemoryDump),
	}
	if err := patchUtilityVolumes(client, vmi, append(slices.Clone(vmi.Spec.UtilityVolumes), memoryDumpVolume)); err != nil {
		return fmt.Errorf("unable to attach memory dump volume: %w", err)
	}
	return nil
}

func detachMemoryDumpVolume(client kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, claimName string) error {
	if vmi == nil {
		return nil
	}
	if hasDeprecatedVolume(vmi, claimName) {
		return detachDeprecatedVolume(client, vmi, claimName)
	}
	if !hasUtilityVolume(vmi, claimName) {
		return nil
	}
	err := patchUtilityVolumes(client, vmi, slices.DeleteFunc(slices.Clone(vmi.Spec.UtilityVolumes), func(volume v1.UtilityVolume) bool {
		return volume.Name == claimName
	}))
	if err != nil {
		return fmt.Errorf("unable to detach memory dump volume: %w", err)
	}
	return nil
}

// TODO: remove together with RemoveDeprecatedVolumes.
func detachDeprecatedVolume(client kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, claimName string) error {
	volumes := slices.DeleteFunc(slices.Clone(vmi.Spec.Volumes), func(volume v1.Volume) bool {
		return volume.Name == claimName
	})
	patchBytes, err := patch.New(
		patch.WithTest("/spec/volumes", vmi.Spec.Volumes),
		patch.WithReplace("/spec/volumes", volumes),
	).GeneratePayload()
	if err != nil {
		return err
	}
	if _, err = client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("unable to detach memory dump volume: %w", err)
	}
	return nil
}

func patchUtilityVolumes(client kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, utilityVolumes []v1.UtilityVolume) error {
	patchSet := patch.New(patch.WithTest("/spec/utilityVolumes", vmi.Spec.UtilityVolumes))
	switch {
	case len(utilityVolumes) == 0:
		patchSet.AddOption(patch.WithRemove("/spec/utilityVolumes"))
	case len(vmi.Spec.UtilityVolumes) == 0:
		patchSet.AddOption(patch.WithAdd("/spec/utilityVolumes", utilityVolumes))
	default:
		patchSet.AddOption(patch.WithReplace("/spec/utilityVolumes", utilityVolumes))
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}
	_, err = client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func patchMemoryDumpPVCAnnotation(client kubecli.KubevirtClient, vm *v1.VirtualMachine, pvcStore cache.Store) error {
	request := vm.Status.MemoryDumpRequest
	pvc, err := storagetypes.GetPersistentVolumeClaimFromCache(vm.Namespace, request.ClaimName, pvcStore)
	if err != nil {
		log.Log.Object(vm).Errorf("Error getting PersistentVolumeClaim to update memory dump annotation: %v", err)
		return err
	}
	if pvc == nil {
		log.Log.Object(vm).Errorf("Error getting PersistentVolumeClaim to update memory dump annotation: %v", err)
		return fmt.Errorf("Error when trying to update memory dump annotation, pvc %s not found", request.ClaimName)
	}

	var patchVal string
	switch request.Phase {
	case v1.MemoryDumpUnmounting:
		// skip patching pvc annotation if file name
		// is empty
		if request.FileName == nil {
			return nil
		}
		patchVal = *request.FileName
	case v1.MemoryDumpFailed:
		patchVal = failed
	default:
		log.Log.Object(vm).Errorf("Unexpected phase when patching memory dump pvc annotation")
		return nil
	}

	annoPatch := patch.New()
	if len(pvc.Annotations) == 0 {
		annoPatch.AddOption(patch.WithAdd("/metadata/annotations", map[string]string{v1.PVCMemoryDumpAnnotation: patchVal}))
	} else if ann, ok := pvc.Annotations[v1.PVCMemoryDumpAnnotation]; ok && ann == patchVal {
		return nil
	} else {
		annoPatch.AddOption(patch.WithReplace("/metadata/annotations/"+patch.EscapeJSONPointer(v1.PVCMemoryDumpAnnotation), patchVal))
	}

	annoPatchPayload, err := annoPatch.GeneratePayload()
	if err != nil {
		return err
	}

	_, err = client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Patch(context.Background(), pvc.Name, types.JSONPatchType, annoPatchPayload, metav1.PatchOptions{})
	if err != nil {
		log.Log.Object(vm).Errorf("failed to annotate memory dump PVC %s/%s, error: %s", pvc.Namespace, pvc.Name, err)
	}

	return nil
}

func CalcExpectedSize(vmi *v1.VirtualMachineInstance) *resource.Quantity {
	const memoryDumpOverhead = 100 * 1024 * 1024
	domain := vmi.Spec.Domain
	vmiMemoryReq := domain.Resources.Requests.Memory()
	expectedPvcSize := resource.NewQuantity(int64(memoryDumpOverhead), vmiMemoryReq.Format)
	expectedPvcSize.Add(*vmiMemoryReq)
	return expectedPvcSize
}
