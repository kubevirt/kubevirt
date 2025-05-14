package clone

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/pointer"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"

	"k8s.io/apimachinery/pkg/util/rand"

	clone "kubevirt.io/api/clone/v1beta1"
	v1 "kubevirt.io/api/core/v1"
)

const (
	vmKind           = "VirtualMachine"
	kubevirtApiGroup = "kubevirt.io"
)

// variable so can be overridden in tests
var currentTime = func() *metav1.Time {
	t := metav1.Now()
	return &t
}

func getKey(name, namespace string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func generateNameWithRandomSuffix(names ...string) string {
	const randomStringLength = 5

	if len(names) == 0 {
		return ""
	}

	generatedName := names[0]
	for _, name := range names[1:] {
		generatedName = fmt.Sprintf("%s-%s", generatedName, name)
	}

	// Kubernetes' object names have limit of 252 characters.
	// For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
	if len(generatedName) > 252 {
		generatedName = "clone-object"
	}

	generatedName = fmt.Sprintf("%s-%s", generatedName, rand.String(randomStringLength))
	return generatedName
}

func generateSnapshotName(vmCloneUID types.UID) string {
	return fmt.Sprintf("tmp-snapshot-%s", string(vmCloneUID))
}

func generateRestoreName(vmCloneUID types.UID) string {
	return fmt.Sprintf("tmp-restore-%s", string(vmCloneUID))
}

func generateVMName(oldVMName string) string {
	return generateNameWithRandomSuffix(oldVMName, "clone")
}

func isInPhase(vmClone *clone.VirtualMachineClone, phase clone.VirtualMachineClonePhase) bool {
	return vmClone.Status.Phase == phase
}

func generateSnapshot(vmClone *clone.VirtualMachineClone, sourceVM *v1.VirtualMachine) *snapshotv1.VirtualMachineSnapshot {
	return &snapshotv1.VirtualMachineSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateSnapshotName(vmClone.UID),
			Namespace: sourceVM.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				getCloneOwnerReference(vmClone.Name, vmClone.UID),
			},
		},
		Spec: snapshotv1.VirtualMachineSnapshotSpec{
			Source: corev1.TypedLocalObjectReference{
				Kind:     vmKind,
				Name:     sourceVM.Name,
				APIGroup: pointer.P(kubevirtApiGroup),
			},
		},
	}
}

func generateRestore(targetInfo *corev1.TypedObjectReference, sourceVMName, namespace, cloneName, snapshotName string, cloneUID types.UID, patches []string) *snapshotv1.VirtualMachineRestore {
	targetInfo = targetInfo.DeepCopy()
	if targetInfo.Name == "" {
		targetInfo.Name = generateVMName(sourceVMName)
	}

	return &snapshotv1.VirtualMachineRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateRestoreName(cloneUID),
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				getCloneOwnerReference(cloneName, cloneUID),
			},
		},
		Spec: snapshotv1.VirtualMachineRestoreSpec{
			Target: corev1.TypedLocalObjectReference{
				APIGroup: targetInfo.APIGroup,
				Name:     targetInfo.Name,
				Kind:     targetInfo.Kind,
			},
			VirtualMachineSnapshotName: snapshotName,
			Patches:                    patches,
		},
	}
}

func getCloneOwnerReference(cloneName string, cloneUID types.UID) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         clone.VirtualMachineCloneKind.GroupVersion().String(),
		Kind:               clone.VirtualMachineCloneKind.Kind,
		Name:               cloneName,
		UID:                cloneUID,
		Controller:         pointer.P(true),
		BlockOwnerDeletion: pointer.P(true),
	}
}

// If the provided object is owned by a clone object, the first return parameter would be true
// and the second one would be the key of the clone. Otherwise, the first return parameter would
// be false and the second parameter is to be ignored.
func isOwnedByClone(obj metav1.Object) (isOwned bool, key string) {
	cloneKind := clone.VirtualMachineCloneKind.Kind
	cloneApiVersion := clone.VirtualMachineCloneKind.GroupVersion().String()

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

func updateCondition(conditions []clone.Condition, c clone.Condition, includeReason bool) []clone.Condition {
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

func updateCloneConditions(vmClone *clone.VirtualMachineClone, conditions ...clone.Condition) {
	for _, cond := range conditions {
		vmClone.Status.Conditions = updateCondition(vmClone.Status.Conditions, cond, true)
	}
}

func newReadyCondition(status corev1.ConditionStatus, reason string) clone.Condition {
	return clone.Condition{
		Type:               clone.ConditionReady,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}

func newProgressingCondition(status corev1.ConditionStatus, reason string) clone.Condition {
	return clone.Condition{
		Type:               clone.ConditionProgressing,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}
