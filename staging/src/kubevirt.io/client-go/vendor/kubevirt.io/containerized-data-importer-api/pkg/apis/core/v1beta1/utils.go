/*
Copyright 2020 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsPopulated indicates if the persistent volume passed in has been fully populated. It follow the following logic
// 1. If the PVC is not owned by a DataVolume, return true, we assume someone else has properly populated the image
// 2. If the PVC is owned by a DataVolume, look up the DV and check the phase, if phase succeeded return true
// 3. If the PVC is owned by a DataVolume, look up the DV and check the phase, if phase !succeeded return false
func IsPopulated(pvc *corev1.PersistentVolumeClaim, getDvFunc func(name, namespace string) (*DataVolume, error)) (bool, error) {
	pvcOwner := metav1.GetControllerOf(pvc)
	if pvcOwner != nil && pvcOwner.Kind == "DataVolume" {
		// Find the data volume:
		dv, err := getDvFunc(pvcOwner.Name, pvc.Namespace)
		if err != nil {
			return false, err
		}
		if dv.Status.Phase != Succeeded {
			return false, nil
		}
	}
	return true, nil
}

// IsSucceededOrPendingPopulation indicates if the persistent volume passed in has been fully populated or is waiting for a consumer.
// It follow the following logic
// 1. If the PVC is not owned by a DataVolume, return true, we assume someone else has properly populated the image
// 2. If the PVC is owned by a DataVolume, look up the DV and check the phase, if phase succeeded or pending population return true
// 3. If the PVC is owned by a DataVolume, look up the DV and check the phase, if phase !succeeded return false
func IsSucceededOrPendingPopulation(pvc *corev1.PersistentVolumeClaim, getDvFunc func(name, namespace string) (*DataVolume, error)) (bool, error) {
	pvcOwner := metav1.GetControllerOf(pvc)
	if pvcOwner != nil && pvcOwner.Kind == "DataVolume" {
		// Find the data volume:
		dv, err := getDvFunc(pvcOwner.Name, pvc.Namespace)
		if err != nil {
			return false, err
		}
		return dv.Status.Phase == Succeeded || dv.Status.Phase == PendingPopulation, nil
	}
	return true, nil
}

// IsWaitForFirstConsumerBeforePopulating indicates if the persistent volume passed in is in ClaimPending state and waiting for first consumer.
// It follow the following logic
// 1. If the PVC is not owned by a DataVolume, return false, we can not assume it will be populated
// 2. If the PVC is owned by a DataVolume, look up the DV and check the phase, if phase WaitForFirstConsumer return true
// 3. If the PVC is owned by a DataVolume, look up the DV and check the phase, if phase !WaitForFirstConsumer return false
func IsWaitForFirstConsumerBeforePopulating(pvc *corev1.PersistentVolumeClaim, getDvFunc func(name, namespace string) (*DataVolume, error)) (bool, error) {
	pvcOwner := metav1.GetControllerOf(pvc)
	if pvcOwner != nil && pvcOwner.Kind == "DataVolume" {
		// Find the data volume:
		dv, err := getDvFunc(pvcOwner.Name, pvc.Namespace)
		if err != nil {
			return false, err
		}
		if dv.Status.Phase == WaitForFirstConsumer {
			return true, nil
		}
	}
	return false, nil
}
