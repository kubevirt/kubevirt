package mutators_test

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/api/clone"
	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook/mutators"
)

var _ = Describe("Clone mutating webhook", func() {
	const (
		testSourceVirtualMachineName         = "test-source-vm"
		testSourceVirtualMachineSnapshotName = "test-snapshot"
	)

	DescribeTable("should mutate the spec", func(vmClone *clonev1alpha1.VirtualMachineClone) {
		admissionReview, err := newAdmissionReviewForVMCloneCreation(vmClone)
		Expect(err).ToNot(HaveOccurred())

		const expectedTargetSuffix = "12345"
		mutator := mutators.NewCloneCreateMutatorWithTargetSuffix(expectedTargetSuffix)

		expectedVirtualMachineCloneSpec := vmClone.Spec.DeepCopy()
		expectedVirtualMachineCloneSpec.Target = &k8sv1.TypedLocalObjectReference{
			APIGroup: pointer.P(virtualMachineAPIGroup),
			Kind:     virtualMachineKind,
			Name:     fmt.Sprintf("clone-%s-%s", expectedVirtualMachineCloneSpec.Source.Name, expectedTargetSuffix),
		}

		expectedJSONPatch, err := expectedJSONPatchForVMCloneCreation(expectedVirtualMachineCloneSpec)
		Expect(err).NotTo(HaveOccurred())

		Expect(mutator.Mutate(admissionReview)).To(Equal(
			&admissionv1.AdmissionResponse{
				Allowed:   true,
				PatchType: pointer.P(admissionv1.PatchTypeJSONPatch),
				Patch:     expectedJSONPatch,
			},
		))
	},
		Entry("When the source is a VirtualMachine and the target is nil",
			newVirtualMachineClone(
				withVirtualMachineSource(testSourceVirtualMachineName),
			),
		),
		Entry("When the source is a VirtualMachine and the target name is empty",
			newVirtualMachineClone(
				withVirtualMachineSource(testSourceVirtualMachineName),
				withVirtualMachineTarget(""),
			),
		),
		Entry("when source is a VirtualMachineSnapshot and target is nil",
			newVirtualMachineClone(
				withVirtualMachineSnapshotSource(testSourceVirtualMachineSnapshotName),
			),
		),
		Entry("when source is a VirtualMachineSnapshot and target name is missing",
			newVirtualMachineClone(
				withVirtualMachineSnapshotSource(testSourceVirtualMachineSnapshotName),
				withVirtualMachineTarget(""),
			),
		),
	)

	It("should not mutate the spec when the target is fully set", func() {
		const testTargetName = "my-vm"

		vmClone := newVirtualMachineClone(
			withVirtualMachineSource(testSourceVirtualMachineName),
			withVirtualMachineTarget(testTargetName),
		)

		admissionReview, err := newAdmissionReviewForVMCloneCreation(vmClone)
		Expect(err).ToNot(HaveOccurred())

		mutator := mutators.NewCloneCreateMutator()

		Expect(mutator.Mutate(admissionReview)).To(Equal(
			&admissionv1.AdmissionResponse{
				Allowed: true,
			},
		))
	})
})

type option func(vmClone *clonev1alpha1.VirtualMachineClone)

func newVirtualMachineClone(options ...option) *clonev1alpha1.VirtualMachineClone {
	newVMClone := &clonev1alpha1.VirtualMachineClone{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kubevirt-test-default",
			Name:      "testclone",
		},
	}

	for _, optionFunc := range options {
		optionFunc(newVMClone)
	}

	return newVMClone
}

const (
	virtualMachineAPIGroup = "kubevirt.io"
	virtualMachineKind     = "VirtualMachine"

	virtualMachineSnapshotAPIGroup = "snapshot.kubevirt.io"
	virtualMachineSnapshotKind     = "VirtualMachineSnapshot"
)

func withVirtualMachineSource(virtualMachineName string) option {
	return func(vmClone *clonev1alpha1.VirtualMachineClone) {
		vmClone.Spec.Source = &k8sv1.TypedLocalObjectReference{
			APIGroup: pointer.P(virtualMachineAPIGroup),
			Kind:     virtualMachineKind,
			Name:     virtualMachineName,
		}
	}
}

func withVirtualMachineSnapshotSource(virtualMachineSnapshotName string) option {
	return func(vmClone *clonev1alpha1.VirtualMachineClone) {
		vmClone.Spec.Source = &k8sv1.TypedLocalObjectReference{
			APIGroup: pointer.P(virtualMachineSnapshotAPIGroup),
			Kind:     virtualMachineSnapshotKind,
			Name:     virtualMachineSnapshotName,
		}
	}
}

func withVirtualMachineTarget(virtualMachineName string) option {
	return func(vmClone *clonev1alpha1.VirtualMachineClone) {
		vmClone.Spec.Target = &k8sv1.TypedLocalObjectReference{
			APIGroup: pointer.P(virtualMachineAPIGroup),
			Kind:     virtualMachineKind,
			Name:     virtualMachineName,
		}
	}
}

func newAdmissionReviewForVMCloneCreation(vmClone *clonev1alpha1.VirtualMachineClone) (*admissionv1.AdmissionReview, error) {
	cloneBytes, err := json.Marshal(vmClone)
	if err != nil {
		return nil, err
	}

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    clone.GroupName,
				Resource: clone.ResourceVMClonePlural,
			},
			Object: runtime.RawExtension{
				Raw: cloneBytes,
			},
		},
	}

	return ar, nil
}

func expectedJSONPatchForVMCloneCreation(vmCloneSpec *clonev1alpha1.VirtualMachineCloneSpec) ([]byte, error) {
	return patch.GeneratePatchPayload(
		patch.PatchOperation{
			Op:    patch.PatchReplaceOp,
			Path:  "/spec",
			Value: vmCloneSpec,
		},
	)
}
