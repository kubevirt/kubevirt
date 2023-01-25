package admitters

import (
	"encoding/json"
	"net/http"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
)

var _ = Describe("Validating Instancetype Admitter", func() {
	var (
		admitter        *InstancetypeAdmitter
		instancetypeObj *instancetypev1alpha2.VirtualMachineInstancetype
	)

	BeforeEach(func() {
		admitter = &InstancetypeAdmitter{}

		instancetypeObj = &instancetypev1alpha2.VirtualMachineInstancetype{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		}
	})

	It("should accept valid instancetype", func() {
		ar := createInstancetypeAdmissionReview(instancetypeObj)
		response := admitter.Admit(ar)
		Expect(response.Allowed).To(BeTrue(), "Expected instancetype to be allowed.")
	})

	It("should reject unsupported version", func() {
		ar := createInstancetypeAdmissionReview(instancetypeObj)
		ar.Request.Resource.Version = "unsupportedversion"
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected instancetype to not be allowed")
		Expect(response.Result.Code).To(Equal(int32(http.StatusBadRequest)), "Expected error 400: BadRequest")
	})

	It("should reject unsupported group", func() {
		ar := createInstancetypeAdmissionReview(instancetypeObj)
		ar.Request.Resource.Group = "unsupportedgroup"
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected instancetype to not be allowed")
		Expect(response.Result.Code).To(Equal(int32(http.StatusBadRequest)), "Expected error 400: BadRequest")
	})

	It("should reject instancetype with dedicatedCPUPlacement", func() {
		instancetypeObj.Spec = instancetypev1alpha2.VirtualMachineInstancetypeSpec{
			CPU: instancetypev1alpha2.CPUInstancetype{
				DedicatedCPUPlacement: true,
			},
		}
		ar := createInstancetypeAdmissionReview(instancetypeObj)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expect instancetype to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal("dedicatedCPUPlacement is not currently supported"))
	})

	It("should reject instancetype with guestMappingPassthrough", func() {
		instancetypeObj.Spec = instancetypev1alpha2.VirtualMachineInstancetypeSpec{
			CPU: instancetypev1alpha2.CPUInstancetype{
				NUMA: &v1.NUMA{
					GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{},
				},
			},
		}
		ar := createInstancetypeAdmissionReview(instancetypeObj)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expect instancetype to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal("guestMappingPassthrough is not currently supported"))
	})
})

var _ = Describe("Validating ClusterInstancetype Admitter", func() {
	var (
		admitter               *ClusterInstancetypeAdmitter
		clusterInstancetypeObj *instancetypev1alpha2.VirtualMachineClusterInstancetype
	)

	BeforeEach(func() {
		admitter = &ClusterInstancetypeAdmitter{}

		clusterInstancetypeObj = &instancetypev1alpha2.VirtualMachineClusterInstancetype{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		}
	})

	It("should accept valid instancetype", func() {
		ar := createClusterInstancetypeAdmissionReview(clusterInstancetypeObj)
		response := admitter.Admit(ar)
		Expect(response.Allowed).To(BeTrue(), "Expected instancetype to be allowed.")
	})

	It("should reject unsupported version", func() {
		ar := createClusterInstancetypeAdmissionReview(clusterInstancetypeObj)
		ar.Request.Resource.Version = "unsupportedversion"
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected instancetype to not be allowed")
		Expect(response.Result.Code).To(Equal(int32(http.StatusBadRequest)), "Expected error 400: BadRequest")
	})

	It("should reject unsupported resource", func() {
		ar := createClusterInstancetypeAdmissionReview(clusterInstancetypeObj)
		ar.Request.Resource.Resource = "unsupportedresource"
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected instancetype to not be allowed")
		Expect(response.Result.Code).To(Equal(int32(http.StatusBadRequest)), "Expected error 400: BadRequest")
	})

	It("should reject cluster instancetype with dedicatedCPUPlacement", func() {
		clusterInstancetypeObj.Spec = instancetypev1alpha2.VirtualMachineInstancetypeSpec{
			CPU: instancetypev1alpha2.CPUInstancetype{
				DedicatedCPUPlacement: true,
			},
		}
		ar := createClusterInstancetypeAdmissionReview(clusterInstancetypeObj)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expect instancetype to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal("dedicatedCPUPlacement is not currently supported"))
	})
})

func createInstancetypeAdmissionReview(instancetype *instancetypev1alpha2.VirtualMachineInstancetype) *admissionv1.AdmissionReview {
	bytes, err := json.Marshal(instancetype)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Could not JSON encode instancetype: %v", instancetype)

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    instancetypev1alpha2.SchemeGroupVersion.Group,
				Version:  instancetypev1alpha2.SchemeGroupVersion.Version,
				Resource: apiinstancetype.PluralResourceName,
			},
			Object: runtime.RawExtension{
				Raw: bytes,
			},
		},
	}
}

func createClusterInstancetypeAdmissionReview(clusterInstancetype *instancetypev1alpha2.VirtualMachineClusterInstancetype) *admissionv1.AdmissionReview {
	bytes, err := json.Marshal(clusterInstancetype)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Could not JSON encode instancetype: %v", clusterInstancetype)

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    instancetypev1alpha2.SchemeGroupVersion.Group,
				Version:  instancetypev1alpha2.SchemeGroupVersion.Version,
				Resource: apiinstancetype.ClusterPluralResourceName,
			},
			Object: runtime.RawExtension{
				Raw: bytes,
			},
		},
	}
}
