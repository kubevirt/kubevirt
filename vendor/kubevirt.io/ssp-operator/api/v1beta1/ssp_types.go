/*


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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	lifecycleapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
)

const (
	OperatorPausedAnnotation = "kubevirt.io/operator.paused"
)

type TemplateValidator struct {
	// Replicas is the number of replicas of the template validator pod
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:default=2
	Replicas *int32 `json:"replicas,omitempty"`

	// Placement describes the node scheduling configuration
	Placement *lifecycleapi.NodePlacement `json:"placement,omitempty"`
}

type CommonTemplates struct {
	// Namespace is the k8s namespace where CommonTemplates should be installed
	//+kubebuilder:validation:MaxLength=63
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	Namespace string `json:"namespace"`

	// DataImportCronTemplates defines a list of DataImportCrons managed by the SSP
	// Operator. This is intended for images used by CommonTemplates.
	DataImportCronTemplates []DataImportCronTemplate `json:"dataImportCronTemplates,omitempty"`
}

type NodeLabeller struct {
	// Placement describes the node scheduling configuration
	Placement *lifecycleapi.NodePlacement `json:"placement,omitempty"`
}

// SSPSpec defines the desired state of SSP
type SSPSpec struct {
	// TemplateValidator is configuration of the template validator operand
	TemplateValidator TemplateValidator `json:"templateValidator,omitempty"`

	// CommonTemplates is the configuration of the common templates operand
	CommonTemplates CommonTemplates `json:"commonTemplates"`

	// NodeLabeller is configuration of the node-labeller operand
	NodeLabeller NodeLabeller `json:"nodeLabeller,omitempty"`
}

// DataImportCronTemplate defines the template type for DataImportCrons.
// It requires metadata.name to be specified while leaving namespace as optional.
type DataImportCronTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec cdiv1beta1.DataImportCronSpec `json:"spec"`
}

// AsDataImportCron converts the DataImportCronTemplate to a cdiv1beta1.DataImportCron
func (t *DataImportCronTemplate) AsDataImportCron() cdiv1beta1.DataImportCron {
	return cdiv1beta1.DataImportCron{
		ObjectMeta: t.ObjectMeta,
		Spec:       t.Spec,
	}
}

// SSPStatus defines the observed state of SSP
type SSPStatus struct {
	lifecycleapi.Status `json:",inline"`

	// Paused is true when the operator notices paused annotation.
	Paused bool `json:"paused,omitempty"`

	// ObservedGeneration is the latest generation observed by the operator.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SSP is the Schema for the ssps API
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
type SSP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SSPSpec   `json:"spec,omitempty"`
	Status SSPStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SSPList contains a list of SSP
type SSPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SSP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SSP{}, &SSPList{})
}
