package operatorstatus

import (
	"reflect"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
)

// NewBuilder returns a builder for ClusterOperatorStatus.
func NewBuilder(clock clock.Clock) *Builder {
	return &Builder{
		clock: clock,
	}
}

// Builder helps build ClusterOperatorStatus with appropriate
// ClusterOperatorStatusCondition and OperandVersion.
type Builder struct {
	clock  clock.Clock
	status *configv1.ClusterOperatorStatus
}

// GetStatus returns the ClusterOperatorStatus built.
func (b *Builder) GetStatus() *configv1.ClusterOperatorStatus {
	return b.status
}

// WithProgressing sets an OperatorProgressing type condition.
func (b *Builder) WithProgressing(status configv1.ConditionStatus, message string) *Builder {
	b.init()
	condition := &configv1.ClusterOperatorStatusCondition{
		Type:               configv1.OperatorProgressing,
		Status:             status,
		Message:            message,
		LastTransitionTime: metav1.NewTime(b.clock.Now()),
	}

	b.setCondition(condition)

	return b
}

// WithDegraded sets an OperatorDegraded type condition.
func (b *Builder) WithDegraded(status configv1.ConditionStatus) *Builder {
	b.init()
	condition := &configv1.ClusterOperatorStatusCondition{
		Type:               configv1.OperatorDegraded,
		Status:             status,
		LastTransitionTime: metav1.NewTime(b.clock.Now()),
	}

	b.setCondition(condition)

	return b
}

// WithAvailable sets an OperatorAvailable type condition.
func (b *Builder) WithAvailable(status configv1.ConditionStatus, message string) *Builder {
	b.init()
	condition := &configv1.ClusterOperatorStatusCondition{
		Type:               configv1.OperatorAvailable,
		Status:             status,
		Message:            message,
		LastTransitionTime: metav1.NewTime(b.clock.Now()),
	}

	b.setCondition(condition)

	return b
}

// WithVersion adds the specific version into the status.
func (b *Builder) WithVersion(name, version string) *Builder {
	b.init()

	existing := b.findVersion(name)
	if existing != nil {
		existing.Version = version
		return b
	}

	ov := configv1.OperandVersion{
		Name:    name,
		Version: version,
	}
	b.status.Versions = append(b.status.Versions, ov)

	return b
}

// WithoutVersion removes the specified version from the existing status.
func (b *Builder) WithoutVersion(name, version string) *Builder {
	b.init()

	versions := make([]configv1.OperandVersion, 0)

	for _, v := range b.status.Versions {
		if v.Name == name {
			continue
		}

		versions = append(versions, v)
	}

	b.status.Versions = versions
	return b
}

// WithRelatedObject adds the reference specified to the RelatedObjects list.
func (b *Builder) WithRelatedObject(group, resource, namespace, name string) *Builder {
	b.init()

	reference := configv1.ObjectReference{
		Group:     group,
		Resource:  resource,
		Namespace: namespace,
		Name:      name,
	}

	b.setRelatedObject(reference)

	return b
}

// WithoutRelatedObject removes the reference specified from the RelatedObjects list.
func (b *Builder) WithoutRelatedObject(group, resource, namespace, name string) *Builder {
	b.init()

	reference := configv1.ObjectReference{
		Group:     group,
		Resource:  resource,
		Namespace: namespace,
		Name:      name,
	}

	related := make([]configv1.ObjectReference, 0)
	for i := range b.status.RelatedObjects {
		if reflect.DeepEqual(b.status.RelatedObjects[i], reference) {
			continue
		}

		related = append(related, b.status.RelatedObjects[i])
	}

	b.status.RelatedObjects = related
	return b
}

func (b *Builder) init() {
	if b.status == nil {
		b.status = &configv1.ClusterOperatorStatus{
			Conditions:     []configv1.ClusterOperatorStatusCondition{},
			Versions:       []configv1.OperandVersion{},
			RelatedObjects: []configv1.ObjectReference{},
		}
	}
}

func (b *Builder) findCondition(conditionType configv1.ClusterStatusConditionType) *configv1.ClusterOperatorStatusCondition {
	for i := range b.status.Conditions {
		if b.status.Conditions[i].Type == conditionType {
			return &b.status.Conditions[i]
		}
	}

	return nil
}

func (b *Builder) setCondition(condition *configv1.ClusterOperatorStatusCondition) {
	existing := b.findCondition(condition.Type)
	if existing == nil {
		b.status.Conditions = append(b.status.Conditions, *condition)
		return
	}

	existing.Reason = condition.Reason
	existing.Message = condition.Message

	if existing.Status != condition.Status {
		existing.Status = condition.Status
		existing.LastTransitionTime = condition.LastTransitionTime
	}
}

func (b *Builder) findVersion(name string) *configv1.OperandVersion {
	for i := range b.status.Versions {
		if b.status.Versions[i].Name == name {
			return &b.status.Versions[i]
		}
	}

	return nil
}

func (b *Builder) setRelatedObject(reference configv1.ObjectReference) {
	for i := range b.status.RelatedObjects {
		if reflect.DeepEqual(b.status.RelatedObjects[i], reference) {
			return
		}
	}

	b.status.RelatedObjects = append(b.status.RelatedObjects, reference)
}
