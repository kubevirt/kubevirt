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

package subresourcesv1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
)

// +kubebuilder:object:root=true

// VirtualMachineTemplate is a dummy object to satisfy the k8s.io/apiserver conventions.
// A subresource cannot be served without a storage for its parent resource.
type VirtualMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero" protobuf:"bytes,1,opt,name=metadata"`
}

// +kubebuilder:object:root=true

// ProcessedVirtualMachineTemplate is the object served by the /process and /create subresources.
// It's not a standalone resource but represents a process or create action on the parent VirtualMachineTemplate resource.
type ProcessedVirtualMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero" protobuf:"bytes,1,opt,name=metadata"`

	// TemplateRef contains a reference to the template that was processed. Optional.
	TemplateRef *corev1.ObjectReference `json:"templateRef,omitempty,omitzero" protobuf:"bytes,2,opt,name=templateRef"`

	// VirtualMachine is a VirtualMachine that was created from processing a template. Required.
	VirtualMachine *virtv1.VirtualMachine `json:"virtualMachine" protobuf:"bytes,3,name=virtualMachine"`

	// Message is an optional instructional message that should inform the user how to
	// utilize the newly created VirtualMachine. Optional.
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`
}

// +kubebuilder:object:root=true

// ProcessOptions are the options used when processing a VirtualMachineTemplate.
type ProcessOptions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero" protobuf:"bytes,1,opt,name=metadata"`

	// Parameters is an optional map of key value pairs used during processing of the template. Optional.
	Parameters map[string]string `json:"parameters,omitempty" protobuf:"bytes,2,opt,name=parameters"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachineTemplate{}, &ProcessOptions{}, &ProcessedVirtualMachineTemplate{})
}
