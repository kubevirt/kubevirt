package v1alpha1

import (
	"bytes"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators"
)

const (
	InstallPlanKind       = "InstallPlan"
	InstallPlanAPIVersion = operators.GroupName + "/" + GroupVersion
)

// Approval is the user approval policy for an InstallPlan.
type Approval string

const (
	ApprovalAutomatic Approval = "Automatic"
	ApprovalManual    Approval = "Manual"
)

// InstallPlanSpec defines a set of Application resources to be installed
type InstallPlanSpec struct {
	CatalogSource              string   `json:"source"`
	CatalogSourceNamespace     string   `json:"sourceNamespace"`
	ClusterServiceVersionNames []string `json:"clusterServiceVersionNames"`
	Approval                   Approval `json:"approval"`
	Approved                   bool     `json:"approved"`
}

// InstallPlanPhase is the current status of a InstallPlan as a whole.
type InstallPlanPhase string

const (
	InstallPlanPhaseNone             InstallPlanPhase = ""
	InstallPlanPhasePlanning         InstallPlanPhase = "Planning"
	InstallPlanPhaseRequiresApproval InstallPlanPhase = "RequiresApproval"
	InstallPlanPhaseInstalling       InstallPlanPhase = "Installing"
	InstallPlanPhaseComplete         InstallPlanPhase = "Complete"
	InstallPlanPhaseFailed           InstallPlanPhase = "Failed"
)

// InstallPlanConditionType describes the state of an InstallPlan at a certain point as a whole.
type InstallPlanConditionType string

const (
	InstallPlanResolved  InstallPlanConditionType = "Resolved"
	InstallPlanInstalled InstallPlanConditionType = "Installed"
)

// ConditionReason is a camelcased reason for the state transition.
type InstallPlanConditionReason string

const (
	InstallPlanReasonPlanUnknown        InstallPlanConditionReason = "PlanUnknown"
	InstallPlanReasonInstallCheckFailed InstallPlanConditionReason = "InstallCheckFailed"
	InstallPlanReasonDependencyConflict InstallPlanConditionReason = "DependenciesConflict"
	InstallPlanReasonComponentFailed    InstallPlanConditionReason = "InstallComponentFailed"
)

// StepStatus is the current status of a particular resource an in
// InstallPlan
type StepStatus string

const (
	StepStatusUnknown    StepStatus = "Unknown"
	StepStatusNotPresent StepStatus = "NotPresent"
	StepStatusPresent    StepStatus = "Present"
	StepStatusCreated    StepStatus = "Created"
)

// ErrInvalidInstallPlan is the error returned by functions that operate on
// InstallPlans when the InstallPlan does not contain totally valid data.
var ErrInvalidInstallPlan = errors.New("the InstallPlan contains invalid data")

// InstallPlanStatus represents the information about the status of
// steps required to complete installation.
//
// Status may trail the actual state of a system.
type InstallPlanStatus struct {
	Phase          InstallPlanPhase       `json:"phase"`
	Conditions     []InstallPlanCondition `json:"conditions,omitempty"`
	CatalogSources []string               `json:"catalogSources"`
	Plan           []Step                 `json:"plan,omitempty"`
}

// InstallPlanCondition represents the overall status of the execution of
// an InstallPlan.
type InstallPlanCondition struct {
	Type               InstallPlanConditionType   `json:"type,omitempty"`
	Status             corev1.ConditionStatus     `json:"status,omitempty"` // True, False, or Unknown
	LastUpdateTime     metav1.Time                `json:"lastUpdateTime,omitempty"`
	LastTransitionTime metav1.Time                `json:"lastTransitionTime,omitempty"`
	Reason             InstallPlanConditionReason `json:"reason,omitempty"`
	Message            string                     `json:"message,omitempty"`
}

// allow overwriting `now` function for deterministic tests
var now = metav1.Now

// SetCondition adds or updates a condition, using `Type` as merge key
func (s *InstallPlanStatus) SetCondition(cond InstallPlanCondition) InstallPlanCondition {
	updated := now()
	cond.LastUpdateTime = updated
	cond.LastTransitionTime = updated

	for i, existing := range s.Conditions {
		if existing.Type != cond.Type {
			continue
		}
		if existing.Status == cond.Status {
			cond.LastTransitionTime = existing.LastTransitionTime
		}
		s.Conditions[i] = cond
		return cond
	}
	s.Conditions = append(s.Conditions, cond)
	return cond
}

func ConditionFailed(cond InstallPlanConditionType, reason InstallPlanConditionReason, err error) InstallPlanCondition {
	return InstallPlanCondition{
		Type:    cond,
		Status:  corev1.ConditionFalse,
		Reason:  reason,
		Message: err.Error(),
	}
}

func ConditionMet(cond InstallPlanConditionType) InstallPlanCondition {
	return InstallPlanCondition{
		Type:   cond,
		Status: corev1.ConditionTrue,
	}
}

// Step represents the status of an individual step in an InstallPlan.
type Step struct {
	Resolving string       `json:"resolving"`
	Resource  StepResource `json:"resource"`
	Status    StepStatus   `json:"status"`
}

// StepResource represents the status of a resource to be tracked by an
// InstallPlan.
type StepResource struct {
	CatalogSource          string `json:"sourceName"`
	CatalogSourceNamespace string `json:"sourceNamespace"`
	Group                  string `json:"group"`
	Version                string `json:"version"`
	Kind                   string `json:"kind"`
	Name                   string `json:"name"`
	Manifest               string `json:"manifest,omitempty"`
}

// NewStepResourceFromCSV creates an unresolved Step for the provided CSV.
func NewStepResourceFromCSV(csv *ClusterServiceVersion) (StepResource, error) {
	csvScheme := runtime.NewScheme()
	if err := AddToScheme(csvScheme); err != nil {
		return StepResource{}, err
	}
	csvSerializer := k8sjson.NewSerializer(k8sjson.DefaultMetaFactory, csvScheme, csvScheme, true)

	var manifestCSV bytes.Buffer
	if err := csvSerializer.Encode(csv, &manifestCSV); err != nil {
		return StepResource{}, err
	}

	step := StepResource{
		Name:     csv.Name,
		Kind:     csv.Kind,
		Group:    csv.GroupVersionKind().Group,
		Version:  csv.GroupVersionKind().Version,
		Manifest: manifestCSV.String(),
	}

	return step, nil
}

// NewStepResourceFromCRD creates an unresolved Step for the provided CRD.
func NewStepResourcesFromCRD(crd *v1beta1.CustomResourceDefinition) ([]StepResource, error) {
	serScheme := runtime.NewScheme()
	k8sscheme.AddToScheme(serScheme)
	scheme.AddToScheme(serScheme)
	serializer := k8sjson.NewSerializer(k8sjson.DefaultMetaFactory, serScheme, serScheme, true)

	var manifest bytes.Buffer
	if err := serializer.Encode(crd, &manifest); err != nil {
		return nil, err
	}

	crdStep := StepResource{
		Name:     crd.Name,
		Kind:     crd.Kind,
		Group:    crd.Spec.Group,
		Version:  crd.Spec.Version,
		Manifest: manifest.String(),
	}

	editRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("edit-%s-%s", crd.Name, crd.Spec.Version),
			Labels: map[string]string{
				"rbac.authorization.k8s.io/aggregate-to-admin": "true",
				"rbac.authorization.k8s.io/aggregate-to-edit":  "true",
			},
		},
		Rules: []rbacv1.PolicyRule{{Verbs: []string{"create", "update", "patch", "delete"}, APIGroups: []string{crd.Spec.Group}, Resources: []string{crd.Spec.Names.Plural}}},
	}
	var editRoleManifest bytes.Buffer
	if err := serializer.Encode(&editRole, &editRoleManifest); err != nil {
		return nil, err
	}
	aggregatedEditClusterRoleStep := StepResource{
		Name:     editRole.Name,
		Kind:     "ClusterRole",
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Manifest: editRoleManifest.String(),
	}

	viewRole := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("view-%s-%s", crd.Name, crd.Spec.Version),
			Labels: map[string]string{
				"rbac.authorization.k8s.io/aggregate-to-view": "true",
			},
		},
		Rules: []rbacv1.PolicyRule{{Verbs: []string{"get", "list", "watch"}, APIGroups: []string{crd.Spec.Group}, Resources: []string{crd.Spec.Names.Plural}}},
	}
	var viewRoleManifest bytes.Buffer
	if err := serializer.Encode(&viewRole, &viewRoleManifest); err != nil {
		return nil, err
	}
	aggregatedViewClusterRoleStep := StepResource{
		Name:     viewRole.Name,
		Kind:     "ClusterRole",
		Group:    "rbac.authorization.k8s.io",
		Version:  "v1",
		Manifest: viewRoleManifest.String(),
	}

	return []StepResource{crdStep, aggregatedEditClusterRoleStep, aggregatedViewClusterRoleStep}, nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type InstallPlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   InstallPlanSpec   `json:"spec"`
	Status InstallPlanStatus `json:"status"`
}

// EnsureCatalogSource ensures that a CatalogSource is present in the Status
// block of an InstallPlan.
func (p *InstallPlan) EnsureCatalogSource(sourceName string) {
	for _, srcName := range p.Status.CatalogSources {
		if srcName == sourceName {
			return
		}
	}

	p.Status.CatalogSources = append(p.Status.CatalogSources, sourceName)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type InstallPlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []InstallPlan `json:"items"`
}
