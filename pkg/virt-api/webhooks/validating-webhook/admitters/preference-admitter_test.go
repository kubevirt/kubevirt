package admitters

import (
	"encoding/json"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
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

	It("should reject unsupported PreferredCPUTopolgy value", func() {
		unsupportedTopology := instancetypev1beta1.PreferredCPUTopology("foo")
		preferenceObj = &instancetypev1beta1.VirtualMachinePreference{
			Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
				CPU: &instancetypev1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(unsupportedTopology),
				},
			},
		}
		ar := createPreferenceAdmissionReview(preferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf(preferredCPUTopologyUnknownErrFmt, unsupportedTopology)))
		Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "preferredCPUTopology").String()))
	})

	It("should reject unsupported SpreadOptions Across value", func() {
		var unsupportedAcrossValue instancetypev1beta1.SpreadAcross = "foobar"
		preferenceObj = &instancetypev1beta1.VirtualMachinePreference{
			Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
				PreferSpreadSocketToCoreRatio: uint32(3),
				CPU: &instancetypev1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
					SpreadOptions: &instancetypev1beta1.SpreadOptions{
						Across: pointer.P(unsupportedAcrossValue),
					},
				},
			},
		}
		ar := createPreferenceAdmissionReview(preferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf(spreadAcrossUnsupportedErrFmt, unsupportedAcrossValue)))
		Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "spreadOptions", "across").String()))
	})

	DescribeTable("should reject when spreading vCPUs across CoresThreads with a ratio higher than 2 set through", func(preferenceObj instancetypev1beta1.VirtualMachinePreference) {
		ar := createPreferenceAdmissionReview(&preferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
		response := admitter.Admit(ar)
		Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal(spreadAcrossCoresThreadsRatioErr))
		Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "spreadOptions", "ratio").String()))
	},
		Entry("PreferSpreadSocketToCoreRatio",
			instancetypev1beta1.VirtualMachinePreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
			},
		),
		Entry("SpreadOptions",
			instancetypev1beta1.VirtualMachinePreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
							Ratio:  pointer.P(uint32(3)),
						},
					},
				},
			},
		),
	)

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

	It("should reject unsupported SpreadOptions Across value", func() {
		var unsupportedAcrossValue instancetypev1beta1.SpreadAcross = "foobar"
		clusterPreferenceObj = &instancetypev1beta1.VirtualMachineClusterPreference{
			Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
				PreferSpreadSocketToCoreRatio: uint32(3),
				CPU: &instancetypev1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
					SpreadOptions: &instancetypev1beta1.SpreadOptions{
						Across: pointer.P(unsupportedAcrossValue),
					},
				},
			},
		}
		ar := createClusterPreferenceAdmissionReview(clusterPreferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
		response := admitter.Admit(ar)

		Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf(spreadAcrossUnsupportedErrFmt, unsupportedAcrossValue)))
		Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "spreadOptions", "across").String()))
	})

	DescribeTable("should reject when spreading vCPUs across CoresThreads with a ratio higher than 2 set through", func(clusterPreferenceObj instancetypev1beta1.VirtualMachineClusterPreference) {
		ar := createClusterPreferenceAdmissionReview(&clusterPreferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
		response := admitter.Admit(ar)
		Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal(spreadAcrossCoresThreadsRatioErr))
		Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "spreadOptions", "ratio").String()))
	},
		Entry("PreferSpreadSocketToCoreRatio",
			instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
			},
		),
		Entry("SpreadOptions",
			instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
							Ratio:  pointer.P(uint32(3)),
						},
					},
				},
			},
		),
	)
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
