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

var _ = Describe("Validating Preference Admitter", func() {
	var (
		admitter      *PreferenceAdmitter
		preferenceObj *instancetypev1beta1.VirtualMachinePreference
	)

	BeforeEach(func() {
		admitter = &PreferenceAdmitter{}

		preferenceObj = &instancetypev1beta1.VirtualMachinePreference{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		}
	})

	DescribeTable("should accept valid preference", func(version string) {
		ar := createPreferenceAdmissionReview(preferenceObj, version)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeTrue(), "Expected preference to be allowed.")
	},
		Entry("with v1alpha1 version", instancetypev1beta1.SchemeGroupVersion.Version),
		Entry("with v1alpha2 version", instancetypev1beta1.SchemeGroupVersion.Version),
		Entry("with v1beta1 version", instancetypev1beta1.SchemeGroupVersion.Version),
	)

	It("should reject unsupported version", func() {
		ar := createPreferenceAdmissionReview(preferenceObj, "unsupportedversion")
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
		Expect(response.Result.Code).To(Equal(int32(http.StatusBadRequest)), "Expected error 400: BadRequest")
	})
})

var _ = Describe("Validating ClusterPreference Admitter", func() {
	var (
		admitter             *ClusterPreferenceAdmitter
		clusterPreferenceObj *instancetypev1beta1.VirtualMachineClusterPreference
	)

	BeforeEach(func() {
		admitter = &ClusterPreferenceAdmitter{}

		clusterPreferenceObj = &instancetypev1beta1.VirtualMachineClusterPreference{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		}
	})

	DescribeTable("should accept valid preference", func(version string) {
		ar := createClusterPreferenceAdmissionReview(clusterPreferenceObj, version)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeTrue(), "Expected preference to be allowed.")
	},
		Entry("with v1alpha1 version", instancetypev1beta1.SchemeGroupVersion.Version),
		Entry("with v1alpha2 version", instancetypev1beta1.SchemeGroupVersion.Version),
		Entry("with v1beta1 version", instancetypev1beta1.SchemeGroupVersion.Version),
	)

	It("should reject unsupported version", func() {
		ar := createClusterPreferenceAdmissionReview(clusterPreferenceObj, "unsupportedversion")
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
		Expect(response.Result.Code).To(Equal(int32(http.StatusBadRequest)), "Expected error 400: BadRequest")
	})
})

func createPreferenceAdmissionReview(preference *instancetypev1beta1.VirtualMachinePreference, version string) *admissionv1.AdmissionReview {
	bytes, err := json.Marshal(preference)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Could not JSON encode preference: %v", preference)

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    instancetypev1beta1.SchemeGroupVersion.Group,
				Version:  version,
				Resource: apiinstancetype.PluralPreferenceResourceName,
			},
			Object: runtime.RawExtension{
				Raw: bytes,
			},
		},
	}
}

func createClusterPreferenceAdmissionReview(clusterPreference *instancetypev1beta1.VirtualMachineClusterPreference, version string) *admissionv1.AdmissionReview {
	bytes, err := json.Marshal(clusterPreference)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Could not JSON encode preference: %v", clusterPreference)

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    instancetypev1beta1.SchemeGroupVersion.Group,
				Version:  version,
				Resource: apiinstancetype.ClusterPluralPreferenceResourceName,
			},
			Object: runtime.RawExtension{
				Raw: bytes,
			},
		},
	}
}
