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

package snapshotoverlay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/storage/utilityvolume"
)

// ErrNotRecoverable refuses a recovery that could put overlays on the wrong disks
var ErrNotRecoverable = errors.New("the snapshot overlays cannot be recovered")

// RecoveryScratch returns the protected scratch PVC of an interrupted External
// snapshot of the VM and its snapshot content, or nil. It fails with
// ErrNotRecoverable when the overlays can no longer go back on the volumes of
// the VM.
func RecoveryScratch(client kubecli.KubevirtClient, pvcStore cache.Indexer, vm *v1.VirtualMachine) (*k8sv1.PersistentVolumeClaim, *snapshotv1.VirtualMachineSnapshotContent, error) {
	scratch, content, err := findRecoveryScratch(client, pvcStore, vm)
	if err != nil || scratch == nil {
		return nil, nil, err
	}
	if err := validateRecovery(client, pvcStore, vm, content); err != nil {
		return nil, nil, err
	}
	return scratch, content, nil
}

// findRecoveryScratch confirms each protected scratch PVC against its snapshot content
func findRecoveryScratch(client kubecli.KubevirtClient, pvcStore cache.Indexer, vm *v1.VirtualMachine) (*k8sv1.PersistentVolumeClaim, *snapshotv1.VirtualMachineSnapshotContent, error) {
	objs, err := pvcStore.ByIndex(cache.NamespaceIndex, vm.Namespace)
	if err != nil {
		return nil, nil, err
	}
	var scratch *k8sv1.PersistentVolumeClaim
	var content *snapshotv1.VirtualMachineSnapshotContent
	for _, obj := range objs {
		pvc := obj.(*k8sv1.PersistentVolumeClaim)
		candidate, err := scratchContent(client, pvc, vm)
		if err != nil {
			return nil, nil, err
		}
		if candidate == nil {
			continue
		}
		if scratch != nil {
			return nil, nil, fmt.Errorf("%w: both %s and %s hold overlays of the VM", ErrNotRecoverable, scratch.Name, pvc.Name)
		}
		if pvc.DeletionTimestamp != nil {
			return nil, nil, fmt.Errorf("%w: scratch PVC %s is being deleted", ErrNotRecoverable, pvc.Name)
		}
		scratch, content = pvc.DeepCopy(), candidate
	}
	return scratch, content, nil
}

// scratchContent returns the snapshot content of a protected scratch PVC that
// holds overlays of the VM, or nil
func scratchContent(client kubecli.KubevirtClient, pvc *k8sv1.PersistentVolumeClaim, vm *v1.VirtualMachine) (*snapshotv1.VirtualMachineSnapshotContent, error) {
	owner := metav1.GetControllerOf(pvc)
	if owner == nil || owner.Kind != "VirtualMachineSnapshotContent" || pvc.Name != storagetypes.ScratchPVCName(owner.UID) ||
		!controller.HasFinalizer(pvc, storagetypes.OverlayProtectionFinalizer) {
		return nil, nil
	}
	content, err := client.VirtualMachineSnapshotContent(pvc.Namespace).Get(context.Background(), owner.Name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if source := content.Spec.Source.VirtualMachine; content.UID != owner.UID || source == nil || source.UID != vm.UID {
		return nil, nil
	}
	return content, nil
}

// validateRecovery requires every volume to still be backed by the PVC its
// overlay was taken of. Overlays are matched to disks by volume name.
func validateRecovery(client kubecli.KubevirtClient, pvcStore cache.Store, vm *v1.VirtualMachine, content *snapshotv1.VirtualMachineSnapshotContent) error {
	source := content.Spec.Source.VirtualMachine
	if source.Spec.Template == nil || vm.Spec.Template == nil {
		return fmt.Errorf("%w: snapshot content %s has no VM template", ErrNotRecoverable, content.Name)
	}
	volumes := storagetypes.GetVolumesByName(&vm.Spec.Template.Spec)
	for _, volume := range source.Spec.Template.Spec.Volumes {
		claim := storagetypes.PVCNameFromVirtVolume(&volume)
		if claim == "" {
			continue
		}
		if current := volumes[volume.Name]; current == nil || storagetypes.PVCNameFromVirtVolume(current) != claim {
			return fmt.Errorf("%w: volume %s no longer uses PVC %s", ErrNotRecoverable, volume.Name, claim)
		}
	}
	for _, backup := range content.Spec.VolumeBackups {
		pvc, err := getPVC(client, pvcStore, vm.Namespace, backup.PersistentVolumeClaim.Name)
		if err != nil {
			return err
		}
		if pvc == nil || pvc.UID != backup.PersistentVolumeClaim.UID || pvc.DeletionTimestamp != nil {
			return fmt.Errorf("%w: PVC %s was replaced or is being deleted", ErrNotRecoverable, backup.PersistentVolumeClaim.Name)
		}
	}
	return nil
}

// AnnotateRecovery marks a new VMI as the one recovering the overlays on
// scratch. Without a scratch volume it drops the recovery annotations a VM
// template may carry, since only this controller sets them.
func AnnotateRecovery(vmi *v1.VirtualMachineInstance, scratch *k8sv1.PersistentVolumeClaim, content *snapshotv1.VirtualMachineSnapshotContent) error {
	if scratch == nil {
		delete(vmi.Annotations, storagetypes.PendingOverlayRecoveryAnnotation)
		delete(vmi.Annotations, storagetypes.OverlaySnapshotReferenceAnnotation)
		return nil
	}
	reference, err := json.Marshal(storagetypes.SnapshotOverlayReference{ContentUID: content.UID, ScratchPVCUID: scratch.UID})
	if err != nil {
		return err
	}
	if vmi.Annotations == nil {
		vmi.Annotations = make(map[string]string)
	}
	vmi.Annotations[storagetypes.PendingOverlayRecoveryAnnotation] = scratch.Name
	vmi.Annotations[storagetypes.OverlaySnapshotReferenceAnnotation] = string(reference)
	return nil
}

// HotplugRecoveryScratch adds the scratch volume to the recovering VMI, since
// the VMI create admitter rejects utility volumes, then hands its overlays to
// that VMI so the snapshot controller follows what its launcher reports.
func HotplugRecoveryScratch(client kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, scratch *k8sv1.PersistentVolumeClaim) error {
	found, err := utilityvolume.Find(vmi, scratch.Name, scratch.Name, v1.SnapshotOverlay)
	if err != nil {
		return err
	}
	if !found {
		payload, err := utilityvolume.AddPatch(vmi, scratch.Name, scratch.Name, v1.SnapshotOverlay)
		if err != nil {
			return err
		}
		if _, err := client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, payload, metav1.PatchOptions{}); err != nil {
			return err
		}
	}

	if scratch.Annotations[storagetypes.OverlayOwnerVMIUIDAnnotation] == string(vmi.UID) {
		return nil
	}
	patchSet := patch.New(
		patch.WithTest("/metadata/uid", scratch.UID),
		patch.WithTest("/metadata/finalizers", scratch.Finalizers),
		patch.WithReplace("/metadata/annotations/"+patch.EscapeJSONPointer(storagetypes.OverlayOwnerVMIUIDAnnotation), string(vmi.UID)),
	)
	if scratch.ResourceVersion != "" {
		patchSet.AddOption(patch.WithTest("/metadata/resourceVersion", scratch.ResourceVersion))
	}
	payload, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}
	if _, err := client.CoreV1().PersistentVolumeClaims(scratch.Namespace).Patch(context.Background(), scratch.Name, types.JSONPatchType, payload, metav1.PatchOptions{}); err != nil {
		return err
	}
	log.Log.Object(vmi).Infof("Recovering the snapshot overlays from scratch volume %s", scratch.Name)
	return nil
}

// HandlePendingRecovery finishes a recovery that failed after its VMI was created
func HandlePendingRecovery(client kubecli.KubevirtClient, pvcStore cache.Store, vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) error {
	if vmi == nil || vmi.DeletionTimestamp != nil || vmi.IsFinal() {
		return nil
	}
	name := vmi.Annotations[storagetypes.PendingOverlayRecoveryAnnotation]
	if name == "" {
		return nil
	}
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil || reference == nil || name != storagetypes.ScratchPVCName(reference.ContentUID) {
		return fmt.Errorf("the recovery reference of scratch volume %s is missing or invalid", name)
	}
	scratch, err := getPVC(client, pvcStore, vmi.Namespace, name)
	if err != nil {
		return err
	}
	if scratch == nil || !controller.HasFinalizer(scratch, storagetypes.OverlayProtectionFinalizer) {
		// Released: the snapshot controller clears the recovery context
		return nil
	}
	if scratch.UID != reference.ScratchPVCUID {
		return fmt.Errorf("scratch volume %s was replaced", name)
	}
	// The annotations alone do not prove the overlays are this VM's
	if scratch.Annotations[storagetypes.OverlayOwnerVMIUIDAnnotation] != string(vmi.UID) {
		content, err := scratchContent(client, scratch, vm)
		if err != nil {
			return err
		}
		if content == nil {
			return fmt.Errorf("scratch volume %s does not hold overlays of VM %s", name, vm.Name)
		}
	}
	return HotplugRecoveryScratch(client, vmi, scratch)
}

// AwaitingRecoveryScratch reports a VMI still waiting for the scratch volume it recovers
func AwaitingRecoveryScratch(vmi *v1.VirtualMachineInstance) bool {
	expected := vmi.Annotations[storagetypes.PendingOverlayRecoveryAnnotation]
	if expected == "" {
		return false
	}
	found, err := utilityvolume.Find(vmi, expected, expected, v1.SnapshotOverlay)
	return err != nil || !found
}

// getPVC confirms a cache miss against the API
func getPVC(client kubecli.KubevirtClient, pvcStore cache.Store, namespace, name string) (*k8sv1.PersistentVolumeClaim, error) {
	pvc, err := storagetypes.GetPersistentVolumeClaimFromCache(namespace, name, pvcStore)
	if err != nil || pvc != nil {
		return pvc, err
	}
	pvc, err = client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	return pvc, err
}
