/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package clone

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	clone "kubevirt.io/api/clone/v1beta1"
	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
)

var _ = Describe("Clone utils", func() {

	Context("getKey", func() {
		It("should return namespace/name format", func() {
			Expect(getKey("myname", "mynamespace")).To(Equal("mynamespace/myname"))
		})
	})

	Context("generateNameWithRandomSuffix", func() {
		It("should return empty string for no names", func() {
			Expect(generateNameWithRandomSuffix()).To(Equal(""))
		})

		It("should append a random suffix to a single name", func() {
			result := generateNameWithRandomSuffix("myvm")
			Expect(result).To(HavePrefix("myvm-"))
			Expect(strings.TrimPrefix(result, "myvm-")).NotTo(BeEmpty())
		})

		It("should join multiple names with dashes before adding suffix", func() {
			result := generateNameWithRandomSuffix("myvm", "clone")
			Expect(result).To(HavePrefix("myvm-clone-"))
		})

		It("should fall back to clone-object base when joined name exceeds 252 chars", func() {
			longName := strings.Repeat("a", 253)
			result := generateNameWithRandomSuffix(longName)
			Expect(result).To(HavePrefix("clone-object-"))
		})

		It("should always produce a name within the Kubernetes 253-char limit", func() {
			// A name of 248 chars + "-clone" = 254 chars — triggers the > 252 guard
			longName := strings.Repeat("a", 248)
			result := generateNameWithRandomSuffix(longName, "clone")
			Expect(result).To(HavePrefix("clone-object-"))
			Expect(len(result)).To(BeNumerically("<=", 253))
		})
	})

	Context("generateSnapshotName", func() {
		It("should return the expected snapshot name format", func() {
			uid := types.UID("test-uid-123")
			Expect(generateSnapshotName(uid)).To(Equal("tmp-snapshot-test-uid-123"))
		})
	})

	Context("generateRestoreName", func() {
		It("should return the expected restore name format", func() {
			uid := types.UID("test-uid-456")
			Expect(generateRestoreName(uid)).To(Equal("tmp-restore-test-uid-456"))
		})
	})

	Context("generateVMName", func() {
		It("should append clone suffix with random string", func() {
			result := generateVMName("myvm")
			Expect(result).To(HavePrefix("myvm-clone-"))
		})
	})

	DescribeTable("isInPhase", func(currentPhase, checkedPhase clone.VirtualMachineClonePhase, expected bool) {
		vmClone := &clone.VirtualMachineClone{}
		vmClone.Status.Phase = currentPhase
		Expect(isInPhase(vmClone, checkedPhase)).To(Equal(expected))
	},
		Entry("should return true when phase matches", clone.Succeeded, clone.Succeeded, true),
		Entry("should return false when phase does not match", clone.Failed, clone.Succeeded, false),
		Entry("should return true for unset phase when checking PhaseUnset", clone.PhaseUnset, clone.PhaseUnset, true),
	)

	Context("isOwnedByClone", func() {
		cloneKind := clone.VirtualMachineCloneKind.Kind
		cloneAPIVersion := clone.VirtualMachineCloneKind.GroupVersion().String()

		makeSnapshot := func(ownerRefs []metav1.OwnerReference) *snapshotv1.VirtualMachineSnapshot {
			return &snapshotv1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-snapshot",
					Namespace:       "test-ns",
					OwnerReferences: ownerRefs,
				},
			}
		}

		DescribeTable("should detect clone ownership", func(ownerRefs []metav1.OwnerReference, expectedOwned bool, expectedKey string) {
			snapshot := makeSnapshot(ownerRefs)
			owned, key := isOwnedByClone(snapshot)
			Expect(owned).To(Equal(expectedOwned))
			Expect(key).To(Equal(expectedKey))
		},
			Entry("should return false when there are no owner references",
				[]metav1.OwnerReference(nil), false, ""),
			Entry("should return false when owner is a different kind",
				[]metav1.OwnerReference{
					{Kind: "VirtualMachine", APIVersion: cloneAPIVersion, Name: "some-vm"},
				}, false, ""),
			Entry("should return false when owner kind matches but APIVersion does not",
				[]metav1.OwnerReference{
					{Kind: cloneKind, APIVersion: "wrong.group/v1", Name: "some-clone"},
				}, false, ""),
			Entry("should return true with the correct key when owned by a clone",
				[]metav1.OwnerReference{
					{Kind: cloneKind, APIVersion: cloneAPIVersion, Name: "my-clone"},
				}, true, "test-ns/my-clone"),
			Entry("should detect clone ownership among multiple owner references",
				[]metav1.OwnerReference{
					{Kind: "VirtualMachine", APIVersion: "kubevirt.io/v1", Name: "some-vm"},
					{Kind: cloneKind, APIVersion: cloneAPIVersion, Name: "my-clone"},
				}, true, "test-ns/my-clone"),
		)
	})

	Context("getCloneOwnerReference", func() {
		It("should return an owner reference pointing at the clone", func() {
			uid := types.UID("clone-uid-1")
			ref := getCloneOwnerReference("my-clone", uid)

			Expect(ref.Kind).To(Equal(clone.VirtualMachineCloneKind.Kind))
			Expect(ref.APIVersion).To(Equal(clone.VirtualMachineCloneKind.GroupVersion().String()))
			Expect(ref.Name).To(Equal("my-clone"))
			Expect(ref.UID).To(Equal(uid))
			Expect(ref.Controller).NotTo(BeNil())
			Expect(*ref.Controller).To(BeTrue())
			Expect(ref.BlockOwnerDeletion).NotTo(BeNil())
			Expect(*ref.BlockOwnerDeletion).To(BeTrue())
		})
	})

	Context("generateSnapshot", func() {
		It("should generate a snapshot owned by the clone and sourced from the VM", func() {
			vmClone := &clone.VirtualMachineClone{
				ObjectMeta: metav1.ObjectMeta{Name: "my-clone", UID: types.UID("clone-uid")},
			}
			sourceVM := &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "source-vm", Namespace: "test-ns"},
			}

			snapshot := generateSnapshot(vmClone, sourceVM)

			Expect(snapshot.Name).To(Equal(generateSnapshotName(vmClone.UID)))
			Expect(snapshot.Namespace).To(Equal(sourceVM.Namespace))
			Expect(snapshot.OwnerReferences).To(HaveLen(1))
			Expect(snapshot.OwnerReferences[0].Name).To(Equal(vmClone.Name))
			Expect(snapshot.OwnerReferences[0].UID).To(Equal(vmClone.UID))
			Expect(snapshot.Spec.Source.Kind).To(Equal(vmKind))
			Expect(snapshot.Spec.Source.Name).To(Equal(sourceVM.Name))
			Expect(snapshot.Spec.Source.APIGroup).NotTo(BeNil())
			Expect(*snapshot.Spec.Source.APIGroup).To(Equal(kubevirtApiGroup))
		})
	})

	Context("generateRestore", func() {
		cloneUID := types.UID("clone-uid")

		It("should generate a restore owned by the clone with a generated target name when none is provided", func() {
			targetInfo := &corev1.TypedLocalObjectReference{Kind: vmKind}

			restore := generateRestore(targetInfo, "source-vm", "test-ns", "my-clone", "my-snapshot", cloneUID, nil, nil)

			Expect(restore.Name).To(Equal(generateRestoreName(cloneUID)))
			Expect(restore.Namespace).To(Equal("test-ns"))
			Expect(restore.OwnerReferences).To(HaveLen(1))
			Expect(restore.OwnerReferences[0].Name).To(Equal("my-clone"))
			Expect(restore.OwnerReferences[0].UID).To(Equal(cloneUID))
			Expect(restore.Spec.Target.Name).To(HavePrefix("source-vm-clone-"))
			Expect(restore.Spec.VirtualMachineSnapshotName).To(Equal("my-snapshot"))
			Expect(restore.Spec.VolumeRestorePolicy).To(BeNil())
		})

		It("should preserve a provided target name instead of generating one", func() {
			targetInfo := &corev1.TypedLocalObjectReference{Kind: vmKind, Name: "explicit-target"}

			restore := generateRestore(targetInfo, "source-vm", "test-ns", "my-clone", "my-snapshot", cloneUID, nil, nil)

			Expect(restore.Spec.Target.Name).To(Equal("explicit-target"))
		})

		It("should not mutate the caller's target info", func() {
			targetInfo := &corev1.TypedLocalObjectReference{Kind: vmKind}

			generateRestore(targetInfo, "source-vm", "test-ns", "my-clone", "my-snapshot", cloneUID, nil, nil)

			Expect(targetInfo.Name).To(BeEmpty())
		})

		It("should carry over the given patches", func() {
			targetInfo := &corev1.TypedLocalObjectReference{Kind: vmKind, Name: "explicit-target"}
			patches := []string{`{"op": "add", "path": "/spec/foo", "value": "bar"}`}

			restore := generateRestore(targetInfo, "source-vm", "test-ns", "my-clone", "my-snapshot", cloneUID, patches, nil)

			Expect(restore.Spec.Patches).To(Equal(patches))
		})

		It("should set the volume restore policy when a volume name policy is provided", func() {
			targetInfo := &corev1.TypedLocalObjectReference{Kind: vmKind, Name: "explicit-target"}
			policy := clone.VolumeNamePolicyPrefixTargetName

			restore := generateRestore(targetInfo, "source-vm", "test-ns", "my-clone", "my-snapshot", cloneUID, nil, &policy)

			Expect(restore.Spec.VolumeRestorePolicy).NotTo(BeNil())
			Expect(*restore.Spec.VolumeRestorePolicy).To(Equal(policy.ToVolumeRestorePolicy()))
		})
	})

	Context("updateCondition", func() {
		makeCondition := func(condType clone.ConditionType, status corev1.ConditionStatus, reason string) clone.Condition {
			return clone.Condition{Type: condType, Status: status, Reason: reason}
		}

		It("should add a condition to an empty slice", func() {
			cond := makeCondition(clone.ConditionReady, corev1.ConditionTrue, "AllGood")
			result := updateCondition(nil, cond, false)
			Expect(result).To(HaveLen(1))
			Expect(result[0].Type).To(Equal(clone.ConditionReady))
			Expect(result[0].Status).To(Equal(corev1.ConditionTrue))
		})

		It("should update an existing condition when status changes", func() {
			existing := makeCondition(clone.ConditionReady, corev1.ConditionFalse, "NotReady")
			updated := makeCondition(clone.ConditionReady, corev1.ConditionTrue, "Ready")
			result := updateCondition([]clone.Condition{existing}, updated, false)
			Expect(result).To(HaveLen(1))
			Expect(result[0].Status).To(Equal(corev1.ConditionTrue))
		})

		It("should not duplicate when condition with same type and status already exists", func() {
			existing := makeCondition(clone.ConditionReady, corev1.ConditionTrue, "OldReason")
			same := makeCondition(clone.ConditionReady, corev1.ConditionTrue, "NewReason")
			result := updateCondition([]clone.Condition{existing}, same, false)
			Expect(result).To(HaveLen(1))
			Expect(result[0].Reason).To(Equal("OldReason"))
		})

		It("should update reason when includeReason is true and reason differs", func() {
			existing := makeCondition(clone.ConditionReady, corev1.ConditionTrue, "OldReason")
			updated := makeCondition(clone.ConditionReady, corev1.ConditionTrue, "NewReason")
			result := updateCondition([]clone.Condition{existing}, updated, true)
			Expect(result).To(HaveLen(1))
			Expect(result[0].Reason).To(Equal("NewReason"))
		})

		It("should preserve other conditions when updating one", func() {
			progressing := makeCondition(clone.ConditionProgressing, corev1.ConditionTrue, "InProgress")
			conditions := []clone.Condition{
				makeCondition(clone.ConditionReady, corev1.ConditionFalse, "NotReady"),
				progressing,
			}
			updated := makeCondition(clone.ConditionReady, corev1.ConditionTrue, "Ready")
			result := updateCondition(conditions, updated, false)
			Expect(result).To(HaveLen(2))
			Expect(result[1]).To(Equal(progressing))
		})

		It("should not update anything when status and reason are unchanged, even with includeReason true", func() {
			existing := makeCondition(clone.ConditionReady, corev1.ConditionTrue, "AllGood")
			same := makeCondition(clone.ConditionReady, corev1.ConditionTrue, "AllGood")
			result := updateCondition([]clone.Condition{existing}, same, true)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(Equal(existing))
		})
	})

	Context("newReadyCondition", func() {
		It("should create a Ready condition with the given status and reason", func() {
			cond := newReadyCondition(corev1.ConditionTrue, "AllGood")
			Expect(cond.Type).To(Equal(clone.ConditionReady))
			Expect(cond.Status).To(Equal(corev1.ConditionTrue))
			Expect(cond.Reason).To(Equal("AllGood"))
		})
	})

	Context("newProgressingCondition", func() {
		It("should create a Progressing condition with the given status and reason", func() {
			cond := newProgressingCondition(corev1.ConditionTrue, "StillWorking")
			Expect(cond.Type).To(Equal(clone.ConditionProgressing))
			Expect(cond.Status).To(Equal(corev1.ConditionTrue))
			Expect(cond.Reason).To(Equal("StillWorking"))
		})
	})
})
