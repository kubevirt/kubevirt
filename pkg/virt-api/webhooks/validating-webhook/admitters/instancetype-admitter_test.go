package admitters

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/api/core/v1"
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

	DescribeTable("should reject negative and over 100% memory overcommit values", func(percent int) {
		version := instancetypev1beta1.SchemeGroupVersion.Version
		instancetypeObj.Spec = instancetypev1beta1.VirtualMachineInstancetypeSpec{
			CPU: instancetypev1beta1.CPUInstancetype{
				Guest: uint32(1),
			},
			Memory: instancetypev1beta1.MemoryInstancetype{
				Guest:             resource.MustParse("128M"),
				OvercommitPercent: percent,
			},
		}
		ar := createInstancetypeAdmissionReview(instancetypeObj, version)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected instancetype to not be allowed")
	},
		Entry("negative overcommit percent", int(-15)),
		Entry("over 100 percent overcommit", int(150)),
	)

	It("should reject specs with memory overcommit and hugepages", func() {
		version := instancetypev1beta1.SchemeGroupVersion.Version
		instancetypeObj.Spec = instancetypev1beta1.VirtualMachineInstancetypeSpec{
			CPU: instancetypev1beta1.CPUInstancetype{
				Guest: uint32(1),
			},
			Memory: instancetypev1beta1.MemoryInstancetype{
				Guest:             resource.MustParse("128M"),
				OvercommitPercent: 15,
				Hugepages: &v1.Hugepages{
					PageSize: "1Gi",
				},
			},
		}
		ar := createInstancetypeAdmissionReview(instancetypeObj, version)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected instancetype to not be allowed")
		Expect(response.Result.Code).To(Equal(int32(http.StatusUnprocessableEntity)), "overCommitPercent and hugepages should not be requested together.")
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
	It("should reject specs with memory overcommit and hugepages", func() {
		version := instancetypev1beta1.SchemeGroupVersion.Version
		clusterInstancetypeObj.Spec = instancetypev1beta1.VirtualMachineInstancetypeSpec{
			CPU: instancetypev1beta1.CPUInstancetype{
				Guest: uint32(1),
			},
			Memory: instancetypev1beta1.MemoryInstancetype{
				Guest:             resource.MustParse("128M"),
				OvercommitPercent: 15,
				Hugepages: &v1.Hugepages{
					PageSize: "1Gi",
				},
			},
		}
		ar := createClusterInstancetypeAdmissionReview(clusterInstancetypeObj, version)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected instancetype to not be allowed")
		Expect(response.Result.Code).To(Equal(int32(http.StatusUnprocessableEntity)), "overCommitPercent and hugepages should not be requested together.")
	})

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
