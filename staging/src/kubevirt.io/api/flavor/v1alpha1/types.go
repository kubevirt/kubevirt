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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

// VirtualMachineFlavor resource contains common VirtualMachine configuration
// that can be used by multiple VirtualMachine resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
type VirtualMachineFlavor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// VirtualMachineFlavorSpec for the flavor
	Spec VirtualMachineFlavorSpec `json:"spec"`
}

// VirtualMachineFlavorList is a list of VirtualMachineFlavor resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineFlavorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineFlavor `json:"items"`
}

// VirtualMachineClusterFlavor is a cluster scoped version of VirtualMachineFlavor resource.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
// +genclient:nonNamespaced
type VirtualMachineClusterFlavor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// VirtualMachineFlavorSpec for the flavor
	Spec VirtualMachineFlavorSpec `json:"spec"`
}

// VirtualMachineClusterFlavorList is a list of VirtualMachineClusterFlavor resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachineClusterFlavorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineClusterFlavor `json:"items"`
}

// VirtualMachineFlavorSpec
//
// +k8s:openapi-gen=true
type VirtualMachineFlavorSpec struct {

	// +optional
	CPU *v1.CPU `json:"cpu,omitempty"`
}
