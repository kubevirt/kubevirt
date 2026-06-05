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

	k8score "k8s.io/api/core/v1"
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
)

func HasCompleted(vm *v1.VirtualMachine) bool {
	return vm.Status.MemoryDumpRequest != nil && vm.Status.MemoryDumpRequest.Phase != v1.MemoryDumpAssociating && vm.Status.MemoryDumpRequest.Phase != v1.MemoryDumpInProgress
}

func RemoveMemoryDumpVolumeFromVMISpec(vmiSpec *v1.VirtualMachineInstanceSpec, claimName string) *v1.VirtualMachineInstanceSpec {
	newVolumesList := []v1.Volume{}
	for _, volume := range vmiSpec.Volumes {
		if volume.Name != claimName {
			newVolumesList = append(newVolumesList, volume)
		}
	}
	vmiSpec.Volumes = newVolumesList
	return vmiSpec
}

func HandleRequest(client kubecli.KubevirtClient, vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, pvcStore cache.Store) error {
	if vm.Status.MemoryDumpRequest == nil {
		return nil
	}

	vmiVolumeMap := make(map[string]v1.Volume)
	if vmi != nil {
		for _, volume := range vmi.Spec.Volumes {
			vmiVolumeMap[volume.Name] = volume
		}
	}
	switch vm.Status.MemoryDumpRequest.Phase {
	case v1.MemoryDumpAssociating:
		if vmi == nil || vmi.DeletionTimestamp != nil || !vmi.IsRunning() {
			return nil
		}
		// When in state associating we want to add the memory dump pvc
		// as a volume in the vm and in the vmi to trigger the mount
		// to virt launcher and the memory dump
		vm.Spec.Template.Spec = *applyMemoryDumpVolumeRequestOnVMISpec(&vm.Spec.Template.Spec, vm.Status.MemoryDumpRequest.ClaimName)
		if _, exists := vmiVolumeMap[vm.Status.MemoryDumpRequest.ClaimName]; exists {
			return nil
		}
		if err := generateVMIMemoryDumpVolumePatch(client, vmi, vm.Status.MemoryDumpRequest, true); err != nil {
			log.Log.Object(vmi).Errorf("unable to patch vmi to add memory dump volume: %v", err)
			return err
		}
	case v1.MemoryDumpUnmounting, v1.MemoryDumpFailed:
		if err := patchMemoryDumpPVCAnnotation(client, vm, pvcStore); err != nil {
			return err
		}
		// Check if the memory dump is in the vmi list of volumes,
		// if it still there remove it to make it unmount from virt launcher
		if _, exists := vmiVolumeMap[vm.Status.MemoryDumpRequest.ClaimName]; !exists {
			return nil
		}

		if err := generateVMIMemoryDumpVolumePatch(client, vmi, vm.Status.MemoryDumpRequest, false); err != nil {
			log.Log.Object(vmi).Errorf("unable to patch vmi to remove memory dump volume: %v", err)
			return err
		}
	case v1.MemoryDumpDissociating:
		// Check if the memory dump is in the vmi list of volumes,
		// if it still there remove it to make it unmount from virt launcher
		if _, exists := vmiVolumeMap[vm.Status.MemoryDumpRequest.ClaimName]; exists {
			if err := generateVMIMemoryDumpVolumePatch(client, vmi, vm.Status.MemoryDumpRequest, false); err != nil {
				log.Log.Object(vmi).Errorf("unable to patch vmi to remove memory dump volume: %v", err)
				return err
			}
		}

		vm.Spec.Template.Spec = *RemoveMemoryDumpVolumeFromVMISpec(&vm.Spec.Template.Spec, vm.Status.MemoryDumpRequest.ClaimName)
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
		// Once memory dump completed, there is no update neeeded,
		// A new update will come from the subresource API once
		// a new request will be issued
		return
	case v1.MemoryDumpAssociating:
		// Update Phase to InProgrees once the memory dump
		// is in the list of vm volumes
		for _, volume := range vm.Spec.Template.Spec.Volumes {
			if vm.Status.MemoryDumpRequest.ClaimName == volume.Name {
				updatedMemoryDumpReq.Phase = v1.MemoryDumpInProgress
				break
			}
		}
	case v1.MemoryDumpInProgress:
		// Update to unmounting once getting update in the vmi volume status
		// that the dump timestamp is updated
		if vmi != nil && len(vmi.Status.VolumeStatus) > 0 {
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
		}
	case v1.MemoryDumpUnmounting:
		// Update memory dump as completed once the memory dump has been
		// unmounted - not a part of the vmi volume status
		if vmi != nil {
			for _, volumeStatus := range vmi.Status.VolumeStatus {
				// If we found the claim name in the vmi volume status
				// then the pvc is still mounted
				if volumeStatus.Name == vm.Status.MemoryDumpRequest.ClaimName {
					return
				}
			}
		}
		updatedMemoryDumpReq.Phase = v1.MemoryDumpCompleted
	case v1.MemoryDumpDissociating:
		// Make sure the memory dump is not in the vmi list of volumes
		if vmi != nil {
			for _, volumeStatus := range vmi.Status.VolumeStatus {
				if volumeStatus.Name == vm.Status.MemoryDumpRequest.ClaimName {
					return
				}
			}
		}
		// Make sure the memory dump is not in the list of vm volumes
		for _, volume := range vm.Spec.Template.Spec.Volumes {
			if vm.Status.MemoryDumpRequest.ClaimName == volume.Name {
				return
			}
		}
		// Remove the memory dump request
		updatedMemoryDumpReq = nil
	}

	vm.Status.MemoryDumpRequest = updatedMemoryDumpReq
}

func generateVMIMemoryDumpVolumePatch(client kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, request *v1.VirtualMachineMemoryDumpRequest, addVolume bool) error {
	foundRemoveVol := false
	for _, volume := range vmi.Spec.Volumes {
		if request.ClaimName == volume.Name {
			if addVolume {
				return fmt.Errorf("Unable to add volume [%s] because it already exists", volume.Name)
			} else {
				foundRemoveVol = true
			}
		}
	}

	if !foundRemoveVol && !addVolume {
		return fmt.Errorf("Unable to remove volume [%s] because it does not exist", request.ClaimName)
	}

	vmiCopy := vmi.DeepCopy()
	if addVolume {
		vmiCopy.Spec = *applyMemoryDumpVolumeRequestOnVMISpec(&vmiCopy.Spec, request.ClaimName)
	} else {
		vmiCopy.Spec = *RemoveMemoryDumpVolumeFromVMISpec(&vmiCopy.Spec, request.ClaimName)
	}
	patchset := patch.New(
		patch.WithTest("/spec/volumes", vmi.Spec.Volumes),
	)
	if len(vmi.Spec.Volumes) > 0 {
		patchset.AddOption(patch.WithReplace("/spec/volumes", vmiCopy.Spec.Volumes))
	} else {
		patchset.AddOption(patch.WithAdd("/spec/volumes", vmiCopy.Spec.Volumes))
	}

	patchBytes, err := patchset.GeneratePayload()
	if err != nil {
		return err
	}
	_, err = client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func applyMemoryDumpVolumeRequestOnVMISpec(vmiSpec *v1.VirtualMachineInstanceSpec, claimName string) *v1.VirtualMachineInstanceSpec {
	for _, volume := range vmiSpec.Volumes {
		if volume.Name == claimName {
			return vmiSpec
		}
	}

	memoryDumpVol := &v1.MemoryDumpVolumeSource{
		PersistentVolumeClaimVolumeSource: v1.PersistentVolumeClaimVolumeSource{
			PersistentVolumeClaimVolumeSource: k8score.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			},
			Hotpluggable: true,
		},
	}

	newVolume := v1.Volume{
		Name: claimName,
	}
	newVolume.VolumeSource.MemoryDump = memoryDumpVol

	vmiSpec.Volumes = append(vmiSpec.Volumes, newVolume)

	return vmiSpec
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
