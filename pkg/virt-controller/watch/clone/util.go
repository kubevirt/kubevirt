package clone

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/rand"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
)

// variable so can be overridden in tests
var currentTime = func() *metav1.Time {
	t := metav1.Now()
	return &t
}

func getKey(name, namespace string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func isNonNilAndTrue(boolptr *bool) bool {
	return boolptr != nil && *boolptr
}

func generateVolumeName(vmName, volumeName string) string {
	const randomStringLength = 5
	return fmt.Sprintf("%s-%s-%s", vmName, volumeName, rand.String(randomStringLength))
}

// If the provided object is owned by a clone object, the first return parameter would be true
// and the second one would be the key of the clone. Otherwise, the first return parameter would
// be false and the second parameter is to be ignored.
func isOwnedByClone(obj metav1.Object) (isOwned bool, key string) {
	cloneKind := clonev1alpha1.VirtualMachineCloneKind.Kind
	cloneApiVersion := clonev1alpha1.VirtualMachineCloneKind.GroupVersion().String()

	ownerRefs := obj.GetOwnerReferences()
	for _, ownerRef := range ownerRefs {
		if ownerRef.Kind != cloneKind || ownerRef.APIVersion != cloneApiVersion {
			continue
		}

		key = getKey(ownerRef.Name, obj.GetNamespace())
		return true, key
	}

	return false, ""
	// TODO: Unit test this?
}

func updateCondition(conditions []clonev1alpha1.Condition, c clonev1alpha1.Condition, includeReason bool) []clonev1alpha1.Condition {
	found := false
	for i := range conditions {
		if conditions[i].Type == c.Type {
			if conditions[i].Status != c.Status || (includeReason && conditions[i].Reason != c.Reason) {
				conditions[i] = c
			}
			found = true
			break
		}
	}

	if !found {
		conditions = append(conditions, c)
	}

	return conditions
}

func updateCloneConditions(vmClone *clonev1alpha1.VirtualMachineClone, conditions ...clonev1alpha1.Condition) {
	for _, cond := range conditions {
		vmClone.Status.Conditions = updateCondition(vmClone.Status.Conditions, cond, false)
	}
}

func newReadyCondition(status corev1.ConditionStatus, reason string) clonev1alpha1.Condition {
	return clonev1alpha1.Condition{
		Type:               clonev1alpha1.ConditionReady,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}

func newProgressingCondition(status corev1.ConditionStatus, reason string) clonev1alpha1.Condition {
	return clonev1alpha1.Condition{
		Type:               clonev1alpha1.ConditionProgressing,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}
