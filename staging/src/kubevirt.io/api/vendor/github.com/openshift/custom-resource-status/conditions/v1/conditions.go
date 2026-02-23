package v1

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SetStatusCondition sets the corresponding condition in conditions to newCondition.
func SetStatusCondition(conditions *[]Condition, newCondition Condition) {
	if conditions == nil {
		conditions = &[]Condition{}
	}
	existingCondition := FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition == nil {
		newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		newCondition.LastHeartbeatTime = metav1.NewTime(time.Now())
		*conditions = append(*conditions, newCondition)
		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		existingCondition.LastTransitionTime = metav1.NewTime(time.Now())
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
	existingCondition.LastHeartbeatTime = metav1.NewTime(time.Now())
}

// SetStatusConditionNoHearbeat sets the corresponding condition in conditions to newCondition
// without setting lastHeartbeatTime.
func SetStatusConditionNoHeartbeat(conditions *[]Condition, newCondition Condition) {
	if conditions == nil {
		conditions = &[]Condition{}
	}
	existingCondition := FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition == nil {
		newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		*conditions = append(*conditions, newCondition)
		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		existingCondition.LastTransitionTime = metav1.NewTime(time.Now())
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
}

// RemoveStatusCondition removes the corresponding conditionType from conditions.
func RemoveStatusCondition(conditions *[]Condition, conditionType ConditionType) {
	if conditions == nil {
		return
	}
	newConditions := []Condition{}
	for _, condition := range *conditions {
		if condition.Type != conditionType {
			newConditions = append(newConditions, condition)
		}
	}

	*conditions = newConditions
}

// FindStatusCondition finds the conditionType in conditions.
func FindStatusCondition(conditions []Condition, conditionType ConditionType) *Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}

	return nil
}

// IsStatusConditionTrue returns true when the conditionType is present and set to `corev1.ConditionTrue`
func IsStatusConditionTrue(conditions []Condition, conditionType ConditionType) bool {
	return IsStatusConditionPresentAndEqual(conditions, conditionType, corev1.ConditionTrue)
}

// IsStatusConditionFalse returns true when the conditionType is present and set to `corev1.ConditionFalse`
func IsStatusConditionFalse(conditions []Condition, conditionType ConditionType) bool {
	return IsStatusConditionPresentAndEqual(conditions, conditionType, corev1.ConditionFalse)
}

// IsStatusConditionUnknown returns true when the conditionType is present and set to `corev1.ConditionUnknown`
func IsStatusConditionUnknown(conditions []Condition, conditionType ConditionType) bool {
	return IsStatusConditionPresentAndEqual(conditions, conditionType, corev1.ConditionUnknown)
}

// IsStatusConditionPresentAndEqual returns true when conditionType is present and equal to status.
func IsStatusConditionPresentAndEqual(conditions []Condition, conditionType ConditionType, status corev1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == status
		}
	}
	return false
}
