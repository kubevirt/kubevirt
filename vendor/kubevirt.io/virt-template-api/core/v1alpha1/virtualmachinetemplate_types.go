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
	"k8s.io/apimachinery/pkg/runtime"
)

// VirtualMachineTemplateSpec defines the desired state of VirtualMachineTemplate
type VirtualMachineTemplateSpec struct {
	// VirtualMachine is the template VirtualMachine to include in this template.
	// If a namespace value is hardcoded, it will be removed during processing of the
	// template. If the namespace value however contains a ${PARAMETER_REFERENCE},
	// the resolved value after parameter substitution will be respected and the
	// VirtualMachine will be created in that namespace.
	//
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Required
	// +required
	VirtualMachine *runtime.RawExtension `json:"virtualMachine" protobuf:"bytes,1,name=virtualMachine"`

	// Parameters is an optional list of Parameters used during processing of the template.
	//
	// +kubebuilder:validation:Optional
	// +optional
	Parameters []Parameter `json:"parameters,omitempty" protobuf:"bytes,2,rep,name=parameters"`

	// Message is an optional instructional message for this template.
	// This field should inform the user how to utilize the newly created VirtualMachine.
	//
	// +kubebuilder:validation:Optional
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
}

// Parameter defines a name/value combination that is to be substituted during
// processing of the template.
type Parameter struct {
	// Name is the name of the parameter. It can be referenced in
	// the template VirtualMachine using ${PARAMETER_NAME}. Required.
	//
	// +kubebuilder:validation:Required
	// +required
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// DisplayName is an alternative name that can be shown in a UI
	// instead of the parameter's name. Optional.
	//
	// +kubebuilder:validation:Optional
	// +optional
	DisplayName string `json:"displayName,omitempty" protobuf:"bytes,2,opt,name=displayName"`

	// Description is the description of the parameter. Optional.
	//
	// +kubebuilder:validation:Optional
	// +optional
	Description string `json:"description,omitempty" protobuf:"bytes,3,opt,name=description"`

	// Value holds the value of the Parameter. If specified, a generator will be
	// ignored. The value replaces all occurrences of the ${PARAMETER_NAME}
	// expression during processing of the template. Optional.
	//
	// +kubebuilder:validation:Optional
	// +optional
	Value string `json:"value,omitempty" protobuf:"bytes,4,opt,name=value"`

	// Generate specifies the generator to be used to generate a Value for this
	// parameter. The From field can be used to provide input to this generator
	// If empty, no generator is being used, leaving the result Value untouched. Optional.
	//
	// The only supported generator is "expression", which accepts a From
	// value with a regex-like syntax, which should follow the form of "[a-zA-Z0-9]{length}".
	// The expression defines the range and length of the resulting random characters.
	//
	// The following character classes are supported in the range:
	//
	// range | characters
	// -------------------------------------------------------------
	// "\w"  | abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_
	// "\d"  | 0123456789
	// "\a"  | abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ
	// "\A"  | !"#$%&'()*+,-./:;<=>?@[\]^_`{|}~
	//
	// Generated examples:
	//
	// expression       | generated value
	// ----------------------------------
	// "test[0-9]{1}x"  | "test7x"
	// "[0-1]{8}"       | "01001100"
	// "0x[A-F0-9]{4}"  | "0xB3AF"
	// "[a-zA-Z0-9]{8}" | "hW4yQU5i"
	//
	// +kubebuilder:validation:Enum=expression
	// +kubebuilder:validation:Optional
	// +optional
	Generate string `json:"generate,omitempty" protobuf:"bytes,5,opt,name=generate"`

	// From is used as input for the generator specified in Generate. Optional.
	//
	// +kubebuilder:validation:Pattern=`\[([a-zA-Z0-9\-\\]+)\](\{(\w+)\})`
	// +kubebuilder:validation:Optional
	// +optional
	From string `json:"from,omitempty" protobuf:"bytes,6,opt,name=from"`

	// Indicates that the parameter must have a Value or valid Generate and From values.
	// Defaults to false. Optional.
	//
	// +kubebuilder:validation:Optional
	// +optional
	Required bool `json:"required,omitempty" protobuf:"varint,7,opt,name=required"`
}

// VirtualMachineTemplateStatus defines the observed state of VirtualMachineTemplate.
type VirtualMachineTemplateStatus struct {
	// Conditions represent the current state of the template.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Condition types include:
	// - "Ready": the template is ready to be processed
	//
	// The status of each condition is one of True, False, or Unknown.
	//
	// +kubebuilder:validation:Optional
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:resource:shortName=vmt;vmts
// +kubebuilder:subresource:status
// +genclient

// VirtualMachineTemplate is the Schema for the virtualmachinetemplates API
type VirtualMachineTemplate struct {
	metav1.TypeMeta `json:",inline"`

	// +kubebuilder:validation:Optional
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the desired state of the template
	//
	// +kubebuilder:validation:Required
	// +required
	Spec VirtualMachineTemplateSpec `json:"spec" protobuf:"bytes,2,name=spec"`

	// Status defines the observed state of the template
	//
	// +kubebuilder:validation:Optional
	// +optional
	Status VirtualMachineTemplateStatus `json:"status,omitempty,omitzero" protobuf:"bytes,3,opt,name=status"`
}

// +kubebuilder:object:root=true

// VirtualMachineTemplateList contains a list of VirtualMachineTemplate
type VirtualMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty,omitzero" protobuf:"bytes,1,opt,name=metadata"`
	Items           []VirtualMachineTemplate `json:"items" protobuf:"bytes,2,rep,name=items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachineTemplate{}, &VirtualMachineTemplateList{})
}
