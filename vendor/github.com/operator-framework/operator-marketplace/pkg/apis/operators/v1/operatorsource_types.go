package v1

import (
	"strings"

	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// OpSrcFinalizer is the name for the finalizer to allow for deletion
	// reconciliation when an OperatorSource is deleted.
	OpSrcFinalizer = "finalizer.operatorsources.operators.coreos.com"
)

// Only type definitions go into this file.
// All other constructs (constants, variables, receiver functions and such)
// related to OperatorSource type should be added to operatorsource.go file.

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatorSourceList contains a list of OperatorSource
type OperatorSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OperatorSource `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// OperatorSource is the Schema for the operatorsources API
// +k8s:openapi-gen=true
type OperatorSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              OperatorSourceSpec   `json:"spec,omitempty"`
	Status            OperatorSourceStatus `json:"status,omitempty"`
}

// OperatorSourceSpec defines the desired state of OperatorSource
type OperatorSourceSpec struct {
	// Type of operator source.
	Type string `json:"type,omitempty"`

	// Endpoint points to the remote app registry server from
	// where operator manifests can be fetched.
	Endpoint string `json:"endpoint,omitempty"`

	// RegistryNamespace refers to the namespace in app registry. Only operator
	// manifests under this namespace will be visible.
	// Please note that this is not a k8s namespace.
	RegistryNamespace string `json:"registryNamespace,omitempty"`

	// AuthorizationToken is the authorization token used to access private
	// repositories in remote registry associated with the operator source.
	AuthorizationToken OperatorSourceAuthorizationToken `json:"authorizationToken,omitempty"`

	// DisplayName is passed along to the CatalogSourceConfig to be used
	// by the resulting CatalogSource to be used as a pretty name.
	DisplayName string `json:"displayName,omitempty"`

	// Publisher is passed along to the CatalogSourceConfig to be used
	// by the resulting CatalogSource that defines what entity published
	// the artifacts from the OperatorSource.
	Publisher string `json:"publisher,omitempty"`
}

// OperatorSourceAuthentication refers to a kubernetes Secret object that
// contains authorization token required to access private repositories.
type OperatorSourceAuthorizationToken struct {
	// SecretName is the name of the kubernetes Secret object.
	SecretName string `json:"secretName,omitempty"`
}

// OperatorSourceStatus defines the observed state of OperatorSource
type OperatorSourceStatus struct {
	// Current phase of the OperatorSource object
	CurrentPhase shared.ObjectPhase `json:"currentPhase,omitempty"`

	// Packages is a comma separated list of package(s) each of which has been
	// downloaded and processed by Marketplace operator from the specified
	// endpoint.
	Packages string `json:"packages,omitempty"`
}

// Set group, version, and kind strings
// from the internal reference that we defined in the v1 package.
// The object the sdk client returns does not set these
// so we must find the correct values and set them manually.
func (opsrc *OperatorSource) EnsureGVK() {
	gvk := schema.GroupVersionKind{
		Group:   SchemeGroupVersion.Group,
		Version: SchemeGroupVersion.Version,
		Kind:    OperatorSourceKind,
	}
	opsrc.SetGroupVersionKind(gvk)
}

// GetCurrentPhaseName returns the name of the current phase of the
// given OperatorSource object.
func (opsrc *OperatorSource) GetCurrentPhaseName() string {
	return opsrc.Status.CurrentPhase.Name
}

// IsEqual returns true if the Spec specified in this is the same as the other.
// Otherwise, the function returns false.
//
// The function performs a case insensitive comparison of corresponding
// attributes.
//
// If the Spec specified in other is nil then the function returns false.
func (s *OperatorSourceSpec) IsEqual(other *OperatorSourceSpec) bool {
	if other == nil {
		return false
	}
	if strings.EqualFold(s.Endpoint, other.Endpoint) &&
		strings.EqualFold(s.RegistryNamespace, other.RegistryNamespace) &&
		strings.EqualFold(s.Type, other.Type) {
		return true
	}
	return false
}

// RemoveFinalizer removes the operator source finalizer from the
// OperatorSource ObjectMeta.
func (s *OperatorSource) RemoveFinalizer() {
	shared.RemoveFinalizer(&s.ObjectMeta, OpSrcFinalizer)
}

// EnsureFinalizer ensures that the operator source finalizer is included
// in the ObjectMeta Finalizers slice. If it already exists, no state change occurs.
// If it doesn't, the finalizer is appended to the slice.
func (s *OperatorSource) EnsureFinalizer() {
	shared.EnsureFinalizer(&s.ObjectMeta, OpSrcFinalizer)
}

func init() {
	SchemeBuilder.Register(&OperatorSource{}, &OperatorSourceList{})
}
