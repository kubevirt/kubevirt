/*
Copyright 2022.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	lifecycleapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
)

const (
	OperatorPausedAnnotation = "kubevirt.io/operator.paused"
)

// TektonTasksSpec defines the desired state of TektonTasks
type TektonTasksSpec struct {
	Pipelines    Pipelines    `json:"pipelines,omitempty"`
	FeatureGates FeatureGates `json:"featureGates,omitempty"`
}

// FeatureGates defines feature gate for tto operator
type FeatureGates struct {
	DeployTektonTaskResources bool `json:"deployTektonTaskResources,omitempty"`
}

// Pipelines defines variables for configuration of pipelines
type Pipelines struct {
	Namespace string `json:"namespace,omitempty"`
}

// TektonTasksStatus defines the observed state of TektonTasks
type TektonTasksStatus struct {
	lifecycleapi.Status `json:",inline"`

	// Paused is true when the operator notices paused annotation.
	Paused bool `json:"paused,omitempty"`

	// ObservedGeneration is the latest generation observed by the operator.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TektonTasks is the Schema for the tektontasks API
type TektonTasks struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TektonTasksSpec   `json:"spec,omitempty"`
	Status TektonTasksStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TektonTasksList contains a list of TektonTasks
type TektonTasksList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TektonTasks `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TektonTasks{}, &TektonTasksList{})
}
