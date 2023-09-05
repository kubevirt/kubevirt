package clone

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/utils/pointer"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/api/snapshot/v1alpha1"

	"k8s.io/apimachinery/pkg/util/rand"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
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

func generateSnapshotName(cloneName, vmName string) string {
	return generateNameWithRandomSuffix("clone", cloneName, "snapshot", vmName)
}

func generateRestoreName(cloneName, vmName string) string {
	return generateNameWithRandomSuffix("clone", cloneName, "restore", vmName)
}

func generateVolumeName(volumeName string) string {
	return generateNameWithRandomSuffix("clone", "volume", volumeName)
}

func generateVMName(oldVMName string) string {
	return generateNameWithRandomSuffix(oldVMName, "clone")
}

func isInPhase(vmClone *clonev1alpha1.VirtualMachineClone, phase clonev1alpha1.VirtualMachineClonePhase) bool {
	return vmClone.Status.Phase == phase
}

func generateSnapshot(vmClone *clonev1alpha1.VirtualMachineClone, sourceVM *v1.VirtualMachine) *v1alpha1.VirtualMachineSnapshot {
	return &v1alpha1.VirtualMachineSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateSnapshotName(vmClone.Name, sourceVM.Name),
			Namespace: sourceVM.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				getCloneOwnerReference(vmClone.Name, vmClone.UID),
			},
		},
		Spec: v1alpha1.VirtualMachineSnapshotSpec{
			Source: corev1.TypedLocalObjectReference{
				Kind:     vmKind,
				Name:     sourceVM.Name,
				APIGroup: pointer.String(kubevirtApiGroup),
			},
		},
	}
}

func generateRestore(targetInfo *corev1.TypedLocalObjectReference, sourceVMName, namespace, cloneName, snapshotName string, cloneUID types.UID, patches []string) *v1alpha1.VirtualMachineRestore {
	targetInfo = targetInfo.DeepCopy()
	if targetInfo.Name == "" {
		targetInfo.Name = generateVMName(sourceVMName)
	}

	return &v1alpha1.VirtualMachineRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateRestoreName(cloneName, sourceVMName),
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				getCloneOwnerReference(cloneName, cloneUID),
			},
		},
		Spec: v1alpha1.VirtualMachineRestoreSpec{
			Target:                     *targetInfo,
			VirtualMachineSnapshotName: snapshotName,
			Patches:                    patches,
		},
	}
}

func getCloneOwnerReference(cloneName string, cloneUID types.UID) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         clonev1alpha1.VirtualMachineCloneKind.GroupVersion().String(),
		Kind:               clonev1alpha1.VirtualMachineCloneKind.Kind,
		Name:               cloneName,
		UID:                cloneUID,
		Controller:         pointer.Bool(true),
		BlockOwnerDeletion: pointer.Bool(true),
	}
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
