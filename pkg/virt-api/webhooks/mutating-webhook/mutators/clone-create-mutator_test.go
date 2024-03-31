package mutators_test

import (
	"encoding/json"

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
	const testSourceVirtualMachineName = "test-source-vm"

	DescribeTable("should mutate the spec", func(vmClone *clonev1alpha1.VirtualMachineClone) {
		cloneSpec := mutate(vmClone)
		Expect(cloneSpec.Target).ShouldNot(BeNil())
		Expect(cloneSpec.Target.Name).ShouldNot(BeEmpty())
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
	)
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

func mutate(vmClone *clonev1alpha1.VirtualMachineClone) *clonev1alpha1.VirtualMachineCloneSpec {
	admissionReview, err := newAdmissionReviewForVMCloneCreation(vmClone)
	Expect(err).ToNot(HaveOccurred())

	mutator := mutators.CloneCreateMutator{}

	resp := mutator.Mutate(admissionReview)
	Expect(resp.Allowed).Should(BeTrue())

	cloneSpec := &clonev1alpha1.VirtualMachineCloneSpec{}
	patch := []patch.PatchOperation{
		{Value: cloneSpec},
	}

	err = json.Unmarshal(resp.Patch, &patch)
	Expect(err).ToNot(HaveOccurred())
	Expect(patch).NotTo(BeEmpty())

	return cloneSpec
}
