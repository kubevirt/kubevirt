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

package types

import (
	"fmt"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
)

const (
	// pendingPVCTimeoutThreshold is the amount of time after which
	// a pending PVC is considered to fail binding/dynamic provisioning
	pendingPVCTimeoutThreshold = 5 * time.Minute
)

// IsPVCFailedProvisioning detects whether a failure has occurred while provisioning a PersistentVolumeClaim.
// If such failure is detected, 'true' is returned alongside the failure message, and 'false' otherwise.
// If an error occurs during detection, a non-nil err will be returned.
func IsPVCFailedProvisioning(pvcStore cache.Store, storageClassStore cache.Store,
	client kubecli.KubevirtClient, namespace, claimName string) (failed bool, message string, err error) {

	obj, exists, err := pvcStore.GetByKey(namespace + "/" + claimName)
	if err != nil {
		return false, "", err
	}
	if !exists {
		return false, "", fmt.Errorf("PVC %s/%s does not exists", namespace, claimName)
	}

	pvc, ok := obj.(*k8sv1.PersistentVolumeClaim)
	if !ok {
		return false, "", fmt.Errorf("failed converting %s/%s to a PVC: object is of type %T", namespace, claimName, obj)
	}

	switch pvc.Status.Phase {
	case k8sv1.ClaimBound:
		return false, "", nil
	case k8sv1.ClaimLost:
		return true, fmt.Sprintf("PVC %s/%s has lost its underlying PersistentVolume (%s)",
			namespace, claimName, pvc.Spec.VolumeName), nil
	case k8sv1.ClaimPending:
		// PVCs which are pending for >5 minutes are considered as failed.
		if hasPVCProvisioningTimeout(pvc) {
			isWFFC, err := isWaitForFirstConsumer(pvc, storageClassStore)
			if err != nil {
				return false, "", err
			}
			if isWFFC {
				// For WaitForFirstConsumer PVCs, it's perfectly normal to be pending for long time
				// until a consumer pod is scheduled.
				return false, "", nil
			}

			return true,
				fmt.Sprintf("PVC %s/%s is pending for over %s", namespace, claimName, pendingPVCTimeoutThreshold),
				nil
		}

		// If no timeout is detected, attempt to detect Kubernetes events
		// that conventionally indicate of some PVC provisioning failure.
		event, err := hasPVCProvisioningFailureEvent(pvc, client)
		if err != nil {
			return false, "", err
		}
		if event != nil {
			return true, fmt.Sprintf("'%s' event detected while provisioning PVC %s/%s: %s",
				event.Reason, namespace, claimName, event.Message), nil
		}
	}

	// No failure is detected, provisioning is still in progress
	return false, "", nil
}

func hasPVCProvisioningFailureEvent(pvc *k8sv1.PersistentVolumeClaim, client kubecli.KubevirtClient) (*k8sv1.Event, error) {
	// Kubernetes doesn't have a formal API to determine whether the PVC is pending because
	// the binding (or provisioning for a dynamically provisioned volume) is still ongoing,
	// or some failure has occurred.
	//
	// There are, however, events that can be used to detect such errors:
	// When a statically provisioned volume fails to bind, a "FailedBinding" event is fired by the volume controller.
	// When a dynamically provisioned volume fails to provision, it's up to the external provisioner to report the
	// error by its own means. By convention, a "ProvisioningFailed" event is fired - and particularly by the
	// sigs.k8s.io/sig-storage-lib-external-provisioner library which is a common base library for implementing
	// external provisioners.

	events, err := client.CoreV1().Events(pvc.Namespace).Search(scheme.Scheme, pvc)
	if err != nil {
		return nil, err
	}

	var latestEvent *k8sv1.Event
	for _, event := range events.Items {
		switch event.Reason {
		case "FailedBinding", "ProvisioningFailed":
			if latestEvent == nil || latestEvent.LastTimestamp.Time.After(event.LastTimestamp.Time) {
				latestEvent = &event
			}
		}
	}

	return latestEvent, nil

}

func hasPVCProvisioningTimeout(pvc *k8sv1.PersistentVolumeClaim) bool {
	pendingDuration := time.Now().Sub(pvc.CreationTimestamp.Time)
	return pendingDuration >= pendingPVCTimeoutThreshold
}

func isWaitForFirstConsumer(pvc *k8sv1.PersistentVolumeClaim, storageClassStore cache.Store) (bool, error) {
	sc, err := getStorageClass(pvc, storageClassStore)
	if err != nil {
		return false, err
	}

	if sc == nil {
		// Static provisioning
		return false, nil
	}

	isWFFC := sc.VolumeBindingMode != nil && *sc.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer
	return isWFFC, nil
}

func getStorageClass(pvc *k8sv1.PersistentVolumeClaim, storageClassStore cache.Store) (*storagev1.StorageClass, error) {
	if pvc.Spec.StorageClassName == nil {
		return getDefaultStorageClass(storageClassStore)
	}

	if *pvc.Spec.StorageClassName == "" {
		// Statically provisioned volume
		return nil, nil
	}

	obj, exists, err := storageClassStore.GetByKey(*pvc.Spec.StorageClassName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("StorageClass %s does not exists", *pvc.Spec.StorageClassName)
	}

	sc, ok := obj.(*storagev1.StorageClass)
	if !ok {
		return nil, fmt.Errorf("failed converting %s to a StorageClass: object is of type %T", *pvc.Spec.StorageClassName, obj)
	}

	return sc, nil
}

func getDefaultStorageClass(storageClassStore cache.Store) (*storagev1.StorageClass, error) {
	for _, obj := range storageClassStore.List() {
		sc, ok := obj.(*storagev1.StorageClass)
		if !ok {
			return nil, fmt.Errorf("failed converting object to a StorageClass: object is of type %T", obj)
		}

		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			return sc, nil
		}
	}

	return nil, nil
}

func IsPVCBlockFromStore(store cache.Store, namespace string, claimName string) (pvc *k8sv1.PersistentVolumeClaim, exists bool, isBlockDevice bool, err error) {
	obj, exists, err := store.GetByKey(namespace + "/" + claimName)
	if err != nil || !exists {
		return nil, exists, false, err
	}
	if pvc, ok := obj.(*k8sv1.PersistentVolumeClaim); ok {
		return obj.(*k8sv1.PersistentVolumeClaim), true, isPVCBlock(pvc), nil
	}
	return nil, false, false, fmt.Errorf("this is not a PVC! %v", obj)
}

func isPVCBlock(pvc *k8sv1.PersistentVolumeClaim) bool {
	// We do not need to consider the data in a PersistentVolume (as of Kubernetes 1.9)
	// If a PVC does not specify VolumeMode and the PV specifies VolumeMode = Block
	// the claim will not be bound. So for the sake of a boolean answer, if the PVC's
	// VolumeMode is Block, that unambiguously answers the question
	return pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == k8sv1.PersistentVolumeBlock
}

func HasSharedAccessMode(accessModes []k8sv1.PersistentVolumeAccessMode) bool {
	for _, accessMode := range accessModes {
		if accessMode == k8sv1.ReadWriteMany {
			return true
		}
	}
	return false
}

func IsPreallocated(annotations map[string]string) bool {
	for a, value := range annotations {
		if strings.Contains(a, "/storage.preallocation") && value == "true" {
			return true
		}
		if strings.Contains(a, "/storage.thick-provisioned") && value == "true" {
			return true
		}
	}
	return false
}

func PVCNameFromVirtVolume(volume *virtv1.Volume) string {
	if volume.DataVolume != nil {
		// TODO, look up the correct PVC name based on the datavolume, right now they match, but that will not always be true.
		return volume.DataVolume.Name
	} else if volume.PersistentVolumeClaim != nil {
		return volume.PersistentVolumeClaim.ClaimName
	}
	return ""
}

func VirtVolumesToPVCMap(volumes []*virtv1.Volume, pvcStore cache.Store, namespace string) (map[string]*k8sv1.PersistentVolumeClaim, error) {
	volumeNamesPVCMap := make(map[string]*k8sv1.PersistentVolumeClaim)
	for _, volume := range volumes {
		claimName := PVCNameFromVirtVolume(volume)
		if claimName == "" {
			return nil, fmt.Errorf("volume %s is not a PVC or Datavolume", volume.Name)
		}
		pvc, exists, _, err := IsPVCBlockFromStore(pvcStore, namespace, claimName)
		if err != nil {
			return nil, fmt.Errorf("failed to get PVC: %v", err)
		}
		if !exists {
			return nil, fmt.Errorf("claim %s not found", claimName)
		}
		volumeNamesPVCMap[volume.Name] = pvc
	}
	return volumeNamesPVCMap, nil
}
