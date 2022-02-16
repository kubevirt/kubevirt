/*
Copyright 2016 The Kubernetes Authors.
Copyright 2017 The KubeVirt Authors.

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

package controller

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"
)

// GetControllerOf returns the controllerRef if controllee has a controller,
// otherwise returns nil.
func GetControllerOf(pod *k8sv1.Pod) *metav1.OwnerReference {
	controllerRef := metav1.GetControllerOf(pod)
	if controllerRef != nil {
		return controllerRef
	}
	// We may find pods that are only using CreatedByLabel and not set with an OwnerReference
	if createdBy := pod.Labels[virtv1.CreatedByLabel]; len(createdBy) > 0 {
		name := pod.Annotations[virtv1.DomainAnnotation]
		uid := types.UID(createdBy)
		vmi := virtv1.NewVMI(name, uid)
		return metav1.NewControllerRef(vmi, virtv1.VirtualMachineInstanceGroupVersionKind)
	}
	return nil
}

func IsControlledBy(pod *k8sv1.Pod, vmi *virtv1.VirtualMachineInstance) bool {
	if controllerRef := GetControllerOf(pod); controllerRef != nil {
		return controllerRef.UID == vmi.UID
	}
	return false
}
