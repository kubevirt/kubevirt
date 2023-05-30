package admitters

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

var _ = Describe("Validating Instancetype Admitter", func() {
	var (
		admitter        *InstancetypeAdmitter
		instancetypeObj *instancetypev1beta1.VirtualMachineInstancetype
	)

	BeforeEach(func() {
		admitter = &InstancetypeAdmitter{}

		instancetypeObj = &instancetypev1beta1.VirtualMachineInstancetype{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		}
	})

	DescribeTable("should accept valid instancetype", func(version string) {
		ar := createInstancetypeAdmissionReview(instancetypeObj, version)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeTrue(), "Expected instancetype to be allowed.")
	},
		Entry("with v1alpha1 version", instancetypev1beta1.SchemeGroupVersion.Version),
		Entry("with v1alpha2 version", instancetypev1beta1.SchemeGroupVersion.Version),
		Entry("with v1beta1 version", instancetypev1beta1.SchemeGroupVersion.Version),
	)

	It("should reject unsupported version", func() {
		ar := createInstancetypeAdmissionReview(instancetypeObj, "unsupportedversion")
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected instancetype to not be allowed")
		Expect(response.Result.Code).To(Equal(int32(http.StatusBadRequest)), "Expected error 400: BadRequest")
	})
})

var _ = Describe("Validating ClusterInstancetype Admitter", func() {
	var (
		admitter               *ClusterInstancetypeAdmitter
		clusterInstancetypeObj *instancetypev1beta1.VirtualMachineClusterInstancetype
	)

	BeforeEach(func() {
		admitter = &ClusterInstancetypeAdmitter{}

		clusterInstancetypeObj = &instancetypev1beta1.VirtualMachineClusterInstancetype{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		}
	})

	DescribeTable("should accept valid instancetype", func(version string) {
		ar := createClusterInstancetypeAdmissionReview(clusterInstancetypeObj, version)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeTrue(), "Expected instancetype to be allowed.")
	},
		Entry("with v1alpha1 version", instancetypev1beta1.SchemeGroupVersion.Version),
		Entry("with v1alpha2 version", instancetypev1beta1.SchemeGroupVersion.Version),
		Entry("with v1beta1 version", instancetypev1beta1.SchemeGroupVersion.Version),
	)

	It("should reject unsupported version", func() {
		ar := createClusterInstancetypeAdmissionReview(clusterInstancetypeObj, "unsupportedversion")
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected instancetype to not be allowed")
		Expect(response.Result.Code).To(Equal(int32(http.StatusBadRequest)), "Expected error 400: BadRequest")
	})
})

func createInstancetypeAdmissionReview(instancetype *instancetypev1beta1.VirtualMachineInstancetype, version string) *admissionv1.AdmissionReview {
	bytes, err := json.Marshal(instancetype)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Could not JSON encode instancetype: %v", instancetype)

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    instancetypev1beta1.SchemeGroupVersion.Group,
				Version:  version,
				Resource: apiinstancetype.PluralResourceName,
			},
			Object: runtime.RawExtension{
				Raw: bytes,
			},
		},
	}
}

func createClusterInstancetypeAdmissionReview(clusterInstancetype *instancetypev1beta1.VirtualMachineClusterInstancetype, version string) *admissionv1.AdmissionReview {
	bytes, err := json.Marshal(clusterInstancetype)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Could not JSON encode instancetype: %v", clusterInstancetype)

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    instancetypev1beta1.SchemeGroupVersion.Group,
				Version:  version,
				Resource: apiinstancetype.ClusterPluralResourceName,
			},
			Object: runtime.RawExtension{
				Raw: bytes,
			},
		},
	}
}
