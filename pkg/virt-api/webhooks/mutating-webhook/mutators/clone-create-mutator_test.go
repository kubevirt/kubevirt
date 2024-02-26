package mutators

import (
	"encoding/json"

	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"

	"kubevirt.io/kubevirt/tests/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"kubevirt.io/api/clone"
	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("Clone mutating webhook", func() {

	var vmClone *clonev1alpha1.VirtualMachineClone

	BeforeEach(func() {
		vmClone = kubecli.NewMinimalCloneWithNS("testclone", util.NamespaceTestDefault)
		vmClone.Spec.Source = &k8sv1.TypedLocalObjectReference{
			APIGroup: pointer.P(clone.GroupName),
			Kind:     "VirtualMachine",
			Name:     "test-source-vm",
		}
	})

	It("Target should be auto generated if missing", func() {
		cloneSpec := mutate(vmClone)
		Expect(cloneSpec.Target).ShouldNot(BeNil())
		Expect(cloneSpec.Target.Name).ShouldNot(BeEmpty())
	})

	It("Target name should be auto generated if missing", func() {
		vmClone.Spec.Target = &k8sv1.TypedLocalObjectReference{
			APIGroup: pointer.P(clone.GroupName),
			Kind:     "VirtualMachine",
			Name:     "",
		}
		cloneSpec := mutate(vmClone)
		Expect(cloneSpec.Target).ShouldNot(BeNil())
		Expect(cloneSpec.Target.Name).ShouldNot(BeEmpty())
	})

})

func createCloneAdmissionReview(vmClone *clonev1alpha1.VirtualMachineClone) *admissionv1.AdmissionReview {
	cloneBytes, _ := json.Marshal(vmClone)

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

	return ar
}

func mutate(vmClone *clonev1alpha1.VirtualMachineClone) *clonev1alpha1.VirtualMachineCloneSpec {
	ar := createCloneAdmissionReview(vmClone)
	mutator := CloneCreateMutator{}

	resp := mutator.Mutate(ar)
	Expect(resp.Allowed).Should(BeTrue())

	cloneSpec := &clonev1alpha1.VirtualMachineCloneSpec{}
	patch := []patch.PatchOperation{
		{Value: cloneSpec},
	}

	err := json.Unmarshal(resp.Patch, &patch)
	Expect(err).ToNot(HaveOccurred())
	Expect(patch).NotTo(BeEmpty())

	return cloneSpec
}
