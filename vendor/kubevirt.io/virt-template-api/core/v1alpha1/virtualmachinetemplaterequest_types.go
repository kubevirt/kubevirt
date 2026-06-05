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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualMachineTemplateRequestSpec defines the desired state of VirtualMachineTemplateRequest
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec is immutable"
type VirtualMachineTemplateRequestSpec struct {
	// VirtualMachineReference holds a reference to a VirtualMachine.kubevirt.io
	// +kubebuilder:validation:Required
	// +required
	VirtualMachineRef VirtualMachineReference `json:"virtualMachineRef" protobuf:"bytes,1,name=virtualMachineRef"`

	// TemplateName holds the optional name for the new VirtualMachineTemplate.
	// If not specified the template will have the same name as the VirtualMachineTemplateRequest.
	// +kubebuilder:validation:Optional
	// +optional
	TemplateName string `json:"templateName,omitempty" protobuf:"bytes,2,name=templateName"`
}

// VirtualMachineReference holds a reference to a VirtualMachine.kubevirt.io
type VirtualMachineReference struct {
	// Namespace is the namespace of the VirtualMachine.
	// +kubebuilder:validation:Required
	// +required
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,1,opt,name=namespace"`

	// Name is the name of the VirtualMachine.
	// +kubebuilder:validation:Required
	// +required
	Name string `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`
}

// VirtualMachineTemplateRequestStatus defines the observed state of VirtualMachineTemplateRequest.
type VirtualMachineTemplateRequestStatus struct {
	// Conditions represent the current state of the template request.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Condition types include:
	// - "Ready": the template was created successfully
	//
	// The status of each condition is one of True, False, or Unknown.
	//
	// +kubebuilder:validation:Optional
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" protobuf:"bytes,1,rep,name=conditions"`

	// TemplateRef is a reference to the created VirtualMachineTemplate.
	// +kubebuilder:validation:Optional
	// +optional
	TemplateRef *corev1.LocalObjectReference `json:"templateRef,omitempty" protobuf:"bytes,2,opt,name=templateRef"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Template",type=string,JSONPath=`.status.templateRef.name`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Progressing",type=string,JSONPath=`.status.conditions[?(@.type=="Progressing")].status`
// +kubebuilder:resource:shortName=vmtr;vmtrs
// +kubebuilder:subresource:status
// +genclient

// VirtualMachineTemplateRequest is the Schema for the virtualmachinetemplaterequests API
type VirtualMachineTemplateRequest struct {
	metav1.TypeMeta `json:",inline"`

	// +kubebuilder:validation:Optional
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the desired state of the template requests
	//
	// +kubebuilder:validation:Required
	// +required
	Spec VirtualMachineTemplateRequestSpec `json:"spec" protobuf:"bytes,2,name=spec"`

	// Status defines the observed state of the template request
	//
	// +kubebuilder:validation:Optional
	// +optional
	Status VirtualMachineTemplateRequestStatus `json:"status,omitempty,omitzero" protobuf:"bytes,3,opt,name=status"`
}

// +kubebuilder:object:root=true

// VirtualMachineTemplateRequestList contains a list of VirtualMachineTemplateRequest
type VirtualMachineTemplateRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineTemplateRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachineTemplateRequest{}, &VirtualMachineTemplateRequestList{})
}
