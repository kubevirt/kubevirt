/*
Copyright 2020 The vm import Authors.

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
	conditions "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VMImportConfigSpec defines the desired state of VMImportConfig
type VMImportConfigSpec struct {
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// VMImportPhase is the current phase of the VMImport deployment
type VMImportPhase string

// VMImportConfigStatus defines the observed state of VMImportConfig
type VMImportConfigStatus struct {
	// +optional
	Conditions []conditions.Condition `json:"conditions,omitempty" optional:"true"`
	// +optional
	OperatorVersion string `json:"operatorVersion,omitempty" optional:"true"`
	// +optional
	TargetVersion string `json:"targetVersion,omitempty" optional:"true"`
	// +optional
	ObservedVersion string `json:"observedVersion,omitempty" optional:"true"`
	// +optional
	Phase VMImportPhase `json:"phase,omitempty"`
}

const (
	// PhaseDeploying signals that the resources are being deployed
	PhaseDeploying VMImportPhase = "Deploying"

	// PhaseDeployed signals that the resources are successfully deployed
	PhaseDeployed VMImportPhase = "Deployed"

	// PhaseDeleting signals that the resources are being removed
	PhaseDeleting VMImportPhase = "Deleting"

	// PhaseDeleted signals that the resources are deleted
	PhaseDeleted VMImportPhase = "Deleted"

	// PhaseError signals that the deployment is in an error state
	PhaseError VMImportPhase = "Error"

	// PhaseUpgrading signals that the resources are being deployed
	PhaseUpgrading VMImportPhase = "Upgrading"

	// PhaseEmpty is an uninitialized phase
	PhaseEmpty VMImportPhase = ""

	// UpgradeStartedReason signals that upgrdade started
	UpgradeStartedReason = "UpgradeStarted"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VMImportConfig is the Schema for the vmimportconfigs API
// +kubebuilder:resource:path=vmimportconfigs,scope=Namespaced
type VMImportConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VMImportConfigSpec   `json:"spec,omitempty"`
	Status VMImportConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VMImportConfigList contains a list of VMImportConfig
type VMImportConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VMImportConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VMImportConfig{}, &VMImportConfigList{})
}
