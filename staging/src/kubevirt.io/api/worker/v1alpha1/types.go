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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// WorkerPoolLabel identifies which worker pool a resource belongs to.
	WorkerPoolLabel string = "kubevirt.io/worker-pool"
)

// WorkerPool defines configuration for an additional virt-handler
// DaemonSet that targets specific nodes with custom images and automatically
// matches VMIs via device and label selectors.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
// +genclient:nonNamespaced
type WorkerPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WorkerPoolSpec `json:"spec" valid:"required"`
	// +nullable
	Status WorkerPoolStatus `json:"status,omitempty"`
}

// +k8s:openapi-gen=true
type WorkerPoolSpec struct {
	// virtHandlerImage overrides the virt-handler container image for this
	// pool's DaemonSet. If not specified, the default virt-handler image is used.
	// +optional
	VirtHandlerImage string `json:"virtHandlerImage,omitempty"`

	// virtLauncherImage overrides the virt-launcher image used by virt-launcher
	// pods on nodes served by this pool's handler. If not specified, the default
	// virt-launcher image is used.
	// +optional
	VirtLauncherImage string `json:"virtLauncherImage,omitempty"`

	// nodeSelector specifies labels that must match a node's labels for this
	// pool's DaemonSet pods to be scheduled on that node. When a VMI matches
	// this pool's selector, the nodeSelector is also merged into the
	// virt-launcher pod's node affinity.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinProperties=1
	NodeSelector map[string]string `json:"nodeSelector"`

	// selector defines the criteria for matching VMIs to this pool. A VMI
	// matches if any of the selector's criteria are met (OR semantics).
	// +kubebuilder:validation:Required
	Selector WorkerPoolSelector `json:"selector"`
}

// WorkerPoolSelector defines the criteria for matching VMIs to a pool.
// DeviceNames and VMLabels are OR'd: if either matches, the pool applies.
// +k8s:openapi-gen=true
type WorkerPoolSelector struct {
	// deviceNames matches VMIs that request any of the listed device names
	// via spec.domain.devices.gpus[].deviceName or
	// spec.domain.devices.hostDevices[].deviceName.
	// +listType=atomic
	// +optional
	DeviceNames []string `json:"deviceNames,omitempty"`

	// vmLabels matches VMIs whose labels contain all of the specified
	// key-value pairs.
	// +optional
	VMLabels *WorkerPoolVMLabels `json:"vmLabels,omitempty"`
}

// WorkerPoolVMLabels matches VMIs by label selectors.
// +k8s:openapi-gen=true
type WorkerPoolVMLabels struct {
	// matchLabels is a map of key-value pairs. A VMI matches if all
	// entries are present in the VMI's labels.
	// +kubebuilder:validation:MinProperties=1
	MatchLabels map[string]string `json:"matchLabels"`
}

// +k8s:openapi-gen=true
type WorkerPoolStatus struct {
}

// WorkerPoolList is a list of WorkerPool resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type WorkerPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +listType=atomic
	Items []WorkerPool `json:"items"`
}
