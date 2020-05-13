package operators

import (
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallPlanKind is the PascalCase name of an InstallPlan's kind.
const InstallPlanKind = "InstallPlan"

// Approval is the user approval policy for an InstallPlan.
type Approval string

const (
	ApprovalAutomatic Approval = "Automatic"
	ApprovalManual    Approval = "Manual"
)

// InstallPlanSpec defines a set of Application resources to be installed
type InstallPlanSpec struct {
	CatalogSource              string
	CatalogSourceNamespace     string
	ClusterServiceVersionNames []string
	Approval                   Approval
	Approved                   bool
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
	StepStatusUnknown             StepStatus = "Unknown"
	StepStatusNotPresent          StepStatus = "NotPresent"
	StepStatusPresent             StepStatus = "Present"
	StepStatusCreated             StepStatus = "Created"
	StepStatusWaitingForAPI       StepStatus = "WaitingForApi"
	StepStatusUnsupportedResource StepStatus = "UnsupportedResource"
)

// ErrInvalidInstallPlan is the error returned by functions that operate on
// InstallPlans when the InstallPlan does not contain totally valid data.
var ErrInvalidInstallPlan = errors.New("the InstallPlan contains invalid data")

// InstallPlanStatus represents the information about the status of
// steps required to complete installation.
//
// Status may trail the actual state of a system.
type InstallPlanStatus struct {
	Phase          InstallPlanPhase
	Conditions     []InstallPlanCondition
	CatalogSources []string
	Plan           []*Step
	// BundleLookups is the set of in-progress requests to pull and unpackage bundle content to the cluster.
	// +optional
	BundleLookups []BundleLookup
	// AttenuatedServiceAccountRef references the service account that is used
	// to do scoped operator install.
	AttenuatedServiceAccountRef *corev1.ObjectReference
}

// InstallPlanCondition represents the overall status of the execution of
// an InstallPlan.
type InstallPlanCondition struct {
	Type               InstallPlanConditionType
	Status             corev1.ConditionStatus // True, False, or Unknown
	LastUpdateTime     *metav1.Time
	LastTransitionTime *metav1.Time
	Reason             InstallPlanConditionReason
	Message            string
}

// allow overwriting `now` function for deterministic tests
var now = metav1.Now

// GetCondition returns the InstallPlanCondition of the given type if it exists in the InstallPlanStatus' Conditions.
// Returns a condition of the given type with a ConditionStatus of "Unknown" if not found.
func (s InstallPlanStatus) GetCondition(conditionType InstallPlanConditionType) InstallPlanCondition {
	for _, cond := range s.Conditions {
		if cond.Type == conditionType {
			return cond
		}
	}

	return InstallPlanCondition{
		Type:   conditionType,
		Status: corev1.ConditionUnknown,
	}
}

// SetCondition adds or updates a condition, using `Type` as merge key.
func (s *InstallPlanStatus) SetCondition(cond InstallPlanCondition) InstallPlanCondition {
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

func ConditionFailed(cond InstallPlanConditionType, reason InstallPlanConditionReason, message string, now *metav1.Time) InstallPlanCondition {
	return InstallPlanCondition{
		Type:               cond,
		Status:             corev1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		LastUpdateTime:     now,
		LastTransitionTime: now,
	}
}

func ConditionMet(cond InstallPlanConditionType, now *metav1.Time) InstallPlanCondition {
	return InstallPlanCondition{
		Type:               cond,
		Status:             corev1.ConditionTrue,
		LastUpdateTime:     now,
		LastTransitionTime: now,
	}
}

// Step represents the status of an individual step in an InstallPlan.
type Step struct {
	Resolving string
	Resource  StepResource
	Status    StepStatus
}

// BundleLookupConditionType is a category of the overall state of a BundleLookup.
type BundleLookupConditionType string

const (
	// BundleLookupPending describes BundleLookups that are not complete.
	BundleLookupPending BundleLookupConditionType = "BundleLookupPending"

	crdKind = "CustomResourceDefinition"
)

type BundleLookupCondition struct {
	// Type of condition.
	Type BundleLookupConditionType
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus
	// The reason for the condition's last transition.
	// +optional
	Reason string
	// A human readable message indicating details about the transition.
	// +optional
	Message string
	// Last time the condition was probed
	// +optional
	LastUpdateTime *metav1.Time
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime *metav1.Time
}

// BundleLookup is a request to pull and unpackage the content of a bundle to the cluster.
type BundleLookup struct {
	// Path refers to the location of a bundle to pull.
	// It's typically an image reference.
	Path string
	// Replaces is the name of the bundle to replace with the one found at Path.
	Replaces string
	// CatalogSourceRef is a reference to the CatalogSource the bundle path was resolved from.
	CatalogSourceRef *corev1.ObjectReference
	// Conditions represents the overall state of a BundleLookup.
	// +optional
	Conditions []BundleLookupCondition
}

// GetCondition returns the BundleLookupCondition of the given type if it exists in the BundleLookup's Conditions.
// Returns a condition of the given type with a ConditionStatus of "Unknown" if not found.
func (b BundleLookup) GetCondition(conditionType BundleLookupConditionType) BundleLookupCondition {
	for _, cond := range b.Conditions {
		if cond.Type == conditionType {
			return cond
		}
	}

	return BundleLookupCondition{
		Type:   conditionType,
		Status: corev1.ConditionUnknown,
	}
}

// RemoveCondition removes the BundleLookupCondition of the given type from the BundleLookup's Conditions if it exists.
func (b *BundleLookup) RemoveCondition(conditionType BundleLookupConditionType) {
	for i, cond := range b.Conditions {
		if cond.Type == conditionType {
			b.Conditions = append(b.Conditions[:i], b.Conditions[i+1:]...)
			if len(b.Conditions) == 0 {
				b.Conditions = nil
			}
			return
		}
	}
}

// SetCondition replaces the existing BundleLookupCondition of the same type, or adds it if it was not found.
func (b *BundleLookup) SetCondition(cond BundleLookupCondition) BundleLookupCondition {
	for i, existing := range b.Conditions {
		if existing.Type != cond.Type {
			continue
		}
		if existing.Status == cond.Status {
			cond.LastTransitionTime = existing.LastTransitionTime
		}
		b.Conditions[i] = cond
		return cond
	}
	b.Conditions = append(b.Conditions, cond)

	return cond
}

func OrderSteps(steps []*Step) []*Step {
	// CSVs must be applied first
	csvList := []*Step{}

	// CRDs must be applied second
	crdList := []*Step{}

	// Other resources may be applied in any order
	remainingResources := []*Step{}
	for _, step := range steps {
		switch step.Resource.Kind {
		case crdKind:
			crdList = append(crdList, step)
		case ClusterServiceVersionKind:
			csvList = append(csvList, step)
		default:
			remainingResources = append(remainingResources, step)
		}
	}

	result := make([]*Step, len(steps))
	i := 0

	for j := range csvList {
		result[i] = csvList[j]
		i++
	}

	for j := range crdList {
		result[i] = crdList[j]
		i++
	}

	for j := range remainingResources {
		result[i] = remainingResources[j]
		i++
	}

	return result
}

func (s InstallPlanStatus) NeedsRequeue() bool {
	for _, step := range s.Plan {
		switch step.Status {
		case StepStatusWaitingForAPI:
			return true
		}
	}

	return false
}

// ManifestsMatch returns true if the CSV manifests in the StepResources of the given list of steps
// matches those in the InstallPlanStatus.
func (s *InstallPlanStatus) CSVManifestsMatch(steps []*Step) bool {
	if s.Plan == nil && steps == nil {
		return true
	}
	if s.Plan == nil || steps == nil {
		return false
	}

	manifests := make(map[string]struct{})
	for _, step := range s.Plan {
		resource := step.Resource
		if resource.Kind != ClusterServiceVersionKind {
			continue
		}
		manifests[resource.Manifest] = struct{}{}
	}

	for _, step := range steps {
		resource := step.Resource
		if resource.Kind != ClusterServiceVersionKind {
			continue
		}
		if _, ok := manifests[resource.Manifest]; !ok {
			return false
		}
		delete(manifests, resource.Manifest)
	}

	return len(manifests) == 0
}

func (s *Step) String() string {
	return fmt.Sprintf("%s: %s (%s)", s.Resolving, s.Resource, s.Status)
}

// StepResource represents the status of a resource to be tracked by an
// InstallPlan.
type StepResource struct {
	CatalogSource          string
	CatalogSourceNamespace string
	Group                  string
	Version                string
	Kind                   string
	Name                   string
	Manifest               string
}

func (r StepResource) String() string {
	return fmt.Sprintf("%s[%s/%s/%s (%s/%s)]", r.Name, r.Group, r.Version, r.Kind, r.CatalogSource, r.CatalogSourceNamespace)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// InstallPlan defines the installation of a set of operators.
type InstallPlan struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   InstallPlanSpec
	Status InstallPlanStatus
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

// InstallPlanList is a list of InstallPlan resources.
type InstallPlanList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []InstallPlan
}
