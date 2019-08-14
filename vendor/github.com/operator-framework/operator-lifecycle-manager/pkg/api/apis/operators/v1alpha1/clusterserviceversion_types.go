// Package v1alpha1 implements all the required types and methods for parsing
// resources for v1alpha1 versioned ClusterServiceVersions.
package v1alpha1

import (
	"encoding/json"
	"sort"

	"github.com/coreos/go-semver/semver"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ClusterServiceVersionAPIVersion = operators.GroupName + "/" + GroupVersion
	ClusterServiceVersionKind       = "ClusterServiceVersion"
)

// NamedInstallStrategy represents the block of an ClusterServiceVersion resource
// where the install strategy is specified.
type NamedInstallStrategy struct {
	StrategyName    string          `json:"strategy"`
	StrategySpecRaw json.RawMessage `json:"spec,omitempty"`
}

// StatusDescriptor describes a field in a status block of a CRD so that ALM can consume it
type StatusDescriptor struct {
	Path         string           `json:"path"`
	DisplayName  string           `json:"displayName,omitempty"`
	Description  string           `json:"description,omitempty"`
	XDescriptors []string         `json:"x-descriptors,omitempty"`
	Value        *json.RawMessage `json:"value,omitempty"`
}

// SpecDescriptor describes a field in a spec block of a CRD so that ALM can consume it
type SpecDescriptor struct {
	Path         string           `json:"path"`
	DisplayName  string           `json:"displayName,omitempty"`
	Description  string           `json:"description,omitempty"`
	XDescriptors []string         `json:"x-descriptors,omitempty"`
	Value        *json.RawMessage `json:"value,omitempty"`
}

// ActionDescriptor describes a declarative action that can be performed on a custom resource instance
type ActionDescriptor struct {
	Path         string           `json:"path"`
	DisplayName  string           `json:"displayName,omitempty"`
	Description  string           `json:"description,omitempty"`
	XDescriptors []string         `json:"x-descriptors,omitempty"`
	Value        *json.RawMessage `json:"value,omitempty"`
}

// CRDDescription provides details to ALM about the CRDs
type CRDDescription struct {
	Name              string                 `json:"name"`
	Version           string                 `json:"version"`
	Kind              string                 `json:"kind"`
	DisplayName       string                 `json:"displayName,omitempty"`
	Description       string                 `json:"description,omitempty"`
	Resources         []CRDResourceReference `json:"resources,omitempty"`
	StatusDescriptors []StatusDescriptor     `json:"statusDescriptors,omitempty"`
	SpecDescriptors   []SpecDescriptor       `json:"specDescriptors,omitempty"`
	ActionDescriptor  []ActionDescriptor     `json:"actionDescriptors,omitempty"`
}

// CRDResourceReference is a Kubernetes resource type used by a custom resource
type CRDResourceReference struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
}

// CustomResourceDefinitions declares all of the CRDs managed or required by
// an operator being ran by ClusterServiceVersion.
//
// If the CRD is present in the Owned list, it is implicitly required.
type CustomResourceDefinitions struct {
	Owned    []CRDDescription `json:"owned,omitempty"`
	Required []CRDDescription `json:"required,omitempty"`
}

// ClusterServiceVersionSpec declarations tell the ALM how to install an operator
// that can manage apps for given version and AppType.
type ClusterServiceVersionSpec struct {
	InstallStrategy           NamedInstallStrategy      `json:"install"`
	Version                   semver.Version            `json:"version,omitempty"`
	Maturity                  string                    `json:"maturity,omitempty"`
	CustomResourceDefinitions CustomResourceDefinitions `json:"customresourcedefinitions,omitempty"`
	DisplayName               string                    `json:"displayName"`
	Description               string                    `json:"description,omitempty"`
	Keywords                  []string                  `json:"keywords,omitempty"`
	Maintainers               []Maintainer              `json:"maintainers,omitempty"`
	Provider                  AppLink                   `json:"provider,omitempty"`
	Links                     []AppLink                 `json:"links,omitempty"`
	Icon                      []Icon                    `json:"icon,omitempty"`

	// The name of a CSV this one replaces. Should match the `metadata.Name` field of the old CSV.
	// +optional
	Replaces string `json:"replaces,omitempty"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects.
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,12,rep,name=annotations"`

	// Label selector for related resources.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,2,opt,name=selector"`
}

type Maintainer struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type AppLink struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type Icon struct {
	Data      string `json:"base64data"`
	MediaType string `json:"mediatype"`
}

// ClusterServiceVersionPhase is a label for the condition of a ClusterServiceVersion at the current time.
type ClusterServiceVersionPhase string

// These are the valid phases of ClusterServiceVersion
const (
	CSVPhaseNone = ""
	// CSVPhasePending means the csv has been accepted by the system, but the install strategy has not been attempted.
	// This is likely because there are unmet requirements.
	CSVPhasePending ClusterServiceVersionPhase = "Pending"
	// CSVPhaseInstallReady means that the requirements are met but the install strategy has not been run.
	CSVPhaseInstallReady ClusterServiceVersionPhase = "InstallReady"
	// CSVPhaseInstalling means that the install strategy has been initiated but not completed.
	CSVPhaseInstalling ClusterServiceVersionPhase = "Installing"
	// CSVPhaseSucceeded means that the resources in the CSV were created successfully.
	CSVPhaseSucceeded ClusterServiceVersionPhase = "Succeeded"
	// CSVPhaseFailed means that the install strategy could not be successfully completed.
	CSVPhaseFailed ClusterServiceVersionPhase = "Failed"
	// CSVPhaseUnknown means that for some reason the state of the csv could not be obtained.
	CSVPhaseUnknown ClusterServiceVersionPhase = "Unknown"
	// CSVPhaseReplacing means that a newer CSV has been created and the csv's resources will be transitioned to a new owner.
	CSVPhaseReplacing ClusterServiceVersionPhase = "Replacing"
	// CSVPhaseDeleting means that a CSV has been replaced by a new one and will be checked for safety before being deleted
	CSVPhaseDeleting ClusterServiceVersionPhase = "Deleting"
)

// ConditionReason is a camelcased reason for the state transition
type ConditionReason string

const (
	CSVReasonRequirementsUnknown ConditionReason = "RequirementsUnknown"
	CSVReasonRequirementsNotMet  ConditionReason = "RequirementsNotMet"
	CSVReasonRequirementsMet     ConditionReason = "AllRequirementsMet"
	CSVReasonOwnerConflict       ConditionReason = "OwnerConflict"
	CSVReasonComponentFailed     ConditionReason = "InstallComponentFailed"
	CSVReasonInvalidStrategy     ConditionReason = "InvalidInstallStrategy"
	CSVReasonWaiting             ConditionReason = "InstallWaiting"
	CSVReasonInstallSuccessful   ConditionReason = "InstallSucceeded"
	CSVReasonInstallCheckFailed  ConditionReason = "InstallCheckFailed"
	CSVReasonComponentUnhealthy  ConditionReason = "ComponentUnhealthy"
	CSVReasonBeingReplaced       ConditionReason = "BeingReplaced"
	CSVReasonReplaced            ConditionReason = "Replaced"
)

// Conditions appear in the status as a record of state transitions on the ClusterServiceVersion
type ClusterServiceVersionCondition struct {
	// Condition of the ClusterServiceVersion
	Phase ClusterServiceVersionPhase `json:"phase,omitempty"`
	// A human readable message indicating details about why the ClusterServiceVersion is in this condition.
	// +optional
	Message string `json:"message,omitempty"`
	// A brief CamelCase message indicating details about why the ClusterServiceVersion is in this state.
	// e.g. 'RequirementsNotMet'
	// +optional
	Reason ConditionReason `json:"reason,omitempty"`
	// Last time we updated the status
	// +optional
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the status transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// OwnsCRD determines whether the current CSV owns a paritcular CRD.
func (csv ClusterServiceVersion) OwnsCRD(name string) bool {
	for _, crdDescription := range csv.Spec.CustomResourceDefinitions.Owned {
		if crdDescription.Name == name {
			return true
		}
	}

	return false
}

type RequirementStatus struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	UUID    string `json:"uuid,omitempty"`
}

// ClusterServiceVersionStatus represents information about the status of a pod. Status may trail the actual
// state of a system.
type ClusterServiceVersionStatus struct {
	// Current condition of the ClusterServiceVersion
	Phase ClusterServiceVersionPhase `json:"phase,omitempty"`
	// A human readable message indicating details about why the ClusterServiceVersion is in this condition.
	// +optional
	Message string `json:"message,omitempty"`
	// A brief CamelCase message indicating details about why the ClusterServiceVersion is in this state.
	// e.g. 'RequirementsNotMet'
	// +optional
	Reason ConditionReason `json:"reason,omitempty"`
	// Last time we updated the status
	// +optional
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the status transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// List of conditions, a history of state transitions
	Conditions []ClusterServiceVersionCondition `json:"conditions,omitempty"`
	// The status of each requirement for this CSV
	RequirementStatus []RequirementStatus `json:"requirementStatus,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
// ClusterServiceVersion is a Custom Resource of type `ClusterServiceVersionSpec`.
type ClusterServiceVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ClusterServiceVersionSpec   `json:"spec"`
	Status ClusterServiceVersionStatus `json:"status"`
}

// ClusterServiceVersionList represents a list of ClusterServiceVersions.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterServiceVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ClusterServiceVersion `json:"items"`
}

// GetAllCRDDescriptions returns a deduplicated set of CRDDescriptions that is
// the union of the owned and required CRDDescriptions.
//
// Descriptions with the same name prefer the value in Owned.
// Descriptions are returned in alphabetical order.
func (csv ClusterServiceVersion) GetAllCRDDescriptions() []CRDDescription {
	set := make(map[string]CRDDescription)
	for _, required := range csv.Spec.CustomResourceDefinitions.Required {
		set[required.Name] = required
	}

	for _, owned := range csv.Spec.CustomResourceDefinitions.Owned {
		set[owned.Name] = owned
	}

	keys := make([]string, 0)
	for key := range set {
		keys = append(keys, key)
	}
	sort.StringSlice(keys).Sort()

	descs := make([]CRDDescription, 0)
	for _, key := range keys {
		descs = append(descs, set[key])
	}

	return descs
}
