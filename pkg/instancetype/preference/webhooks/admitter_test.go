//nolint:dupl
package webhooks_test

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/preference/webhooks"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Validating Preference Admitter", func() {
	var (
		admitter      *webhooks.PreferenceAdmitter
		preferenceObj *instancetypev1beta1.VirtualMachinePreference
	)

	BeforeEach(func() {
		admitter = &webhooks.PreferenceAdmitter{}

		preferenceObj = &instancetypev1beta1.VirtualMachinePreference{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		}
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
		response := admitter.Admit(context.Background(), ar)

		Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf("unknown preferredCPUTopology %s", unsupportedTopology)))
		Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "preferredCPUTopology").String()))
	})

	DescribeTable("should reject unsupported SpreadOptions Across value", func(preferredCPUTopology instancetypev1beta1.PreferredCPUTopology) {
		var unsupportedAcrossValue instancetypev1beta1.SpreadAcross = "foobar"
		preferenceObj = &instancetypev1beta1.VirtualMachinePreference{
			Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
				PreferSpreadSocketToCoreRatio: uint32(3),
				CPU: &instancetypev1beta1.CPUPreferences{
					PreferredCPUTopology: &preferredCPUTopology,
					SpreadOptions: &instancetypev1beta1.SpreadOptions{
						Across: pointer.P(unsupportedAcrossValue),
					},
				},
			},
		}
		ar := createPreferenceAdmissionReview(preferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
		response := admitter.Admit(context.Background(), ar)

		Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf("across %s is not supported", unsupportedAcrossValue)))
		Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "spreadOptions", "across").String()))
	},
		Entry("with spread", instancetypev1beta1.Spread),
		Entry("with preferSpread", instancetypev1beta1.DeprecatedPreferSpread),
	)

	DescribeTable("should reject when spreading vCPUs across CoresThreads with a ratio higher than 2 set through",
		func(preferenceObj instancetypev1beta1.VirtualMachinePreference) {
			ar := createPreferenceAdmissionReview(&preferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
			response := admitter.Admit(context.Background(), ar)
			Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
			Expect(response.Result.Details.Causes).To(HaveLen(1))
			Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(response.Result.Details.Causes[0].Message).To(Equal(
				"only a ratio of 2 (1 core 2 threads) is allowed when spreading vCPUs over cores and threads"))
			Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "spreadOptions", "ratio").String()))
		},
		Entry("PreferSpreadSocketToCoreRatio with spread",
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
		Entry("PreferSpreadSocketToCoreRatio with preferSpread",
			instancetypev1beta1.VirtualMachinePreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
			},
		),
		Entry("SpreadOptions with spread",
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
		Entry("SpreadOptions with preferSpread",
			instancetypev1beta1.VirtualMachinePreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
							Ratio:  pointer.P(uint32(3)),
						},
					},
				},
			},
		),
	)

	DescribeTable("should raise warning for", func(deprecatedTopology, expectedAlternativeTopology instancetypev1beta1.PreferredCPUTopology) {
		preferenceObj := &instancetypev1beta1.VirtualMachinePreference{
			Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
				CPU: &instancetypev1beta1.CPUPreferences{
					PreferredCPUTopology: pointer.P(deprecatedTopology),
				},
			},
		}
		ar := createPreferenceAdmissionReview(preferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
		response := admitter.Admit(context.Background(), ar)
		Expect(response.Allowed).To(BeTrue())
		Expect(response.Warnings).To(HaveLen(1))
		Expect(response.Warnings[0]).To(ContainSubstring(
			fmt.Sprintf("PreferredCPUTopology %s is deprecated for removal in a future release, please use %s instead",
				deprecatedTopology, expectedAlternativeTopology)))
	},
		Entry("DeprecatedPreferSockets and provide Sockets as an alternative",
			instancetypev1beta1.DeprecatedPreferSockets,
			instancetypev1beta1.Sockets,
		),
		Entry("DeprecatedPreferCores and provide Cores as an alternative",
			instancetypev1beta1.DeprecatedPreferCores,
			instancetypev1beta1.Cores,
		),
		Entry("DeprecatedPreferThreads and provide Threads as an alternative",
			instancetypev1beta1.DeprecatedPreferThreads,
			instancetypev1beta1.Threads,
		),
		Entry("DeprecatedPreferSpread and provide Spread as an alternative",
			instancetypev1beta1.DeprecatedPreferSpread,
			instancetypev1beta1.Spread,
		),
		Entry("DeprecatedPreferAny and provide Any as an alternative",
			instancetypev1beta1.DeprecatedPreferAny,
			instancetypev1beta1.Any,
		),
	)
})

var _ = Describe("Validating ClusterPreference Admitter", func() {
	var (
		admitter             *webhooks.ClusterPreferenceAdmitter
		clusterPreferenceObj *instancetypev1beta1.VirtualMachineClusterPreference
	)

	BeforeEach(func() {
		admitter = &webhooks.ClusterPreferenceAdmitter{}

		clusterPreferenceObj = &instancetypev1beta1.VirtualMachineClusterPreference{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		}
	})

	DescribeTable("should reject unsupported SpreadOptions Across value",
		func(preferredCPUTopology instancetypev1beta1.PreferredCPUTopology) {
			var unsupportedAcrossValue instancetypev1beta1.SpreadAcross = "foobar"
			clusterPreferenceObj = &instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: &preferredCPUTopology,
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(unsupportedAcrossValue),
						},
					},
				},
			}
			ar := createClusterPreferenceAdmissionReview(clusterPreferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
			response := admitter.Admit(context.Background(), ar)

			Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
			Expect(response.Result.Details.Causes).To(HaveLen(1))
			Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf("across %s is not supported", unsupportedAcrossValue)))
			Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "spreadOptions", "across").String()))
		},
		Entry("with spread", instancetypev1beta1.Spread),
		Entry("with preferSpread", instancetypev1beta1.DeprecatedPreferSpread),
	)

	DescribeTable("should reject when spreading vCPUs across CoresThreads with a ratio higher than 2 set through",
		func(clusterPreferenceObj instancetypev1beta1.VirtualMachineClusterPreference) {
			ar := createClusterPreferenceAdmissionReview(&clusterPreferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
			response := admitter.Admit(context.Background(), ar)
			Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
			Expect(response.Result.Details.Causes).To(HaveLen(1))
			Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(response.Result.Details.Causes[0].Message).To(Equal(
				"only a ratio of 2 (1 core 2 threads) is allowed when spreading vCPUs over cores and threads"))
			Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "spreadOptions", "ratio").String()))
		},
		Entry("PreferSpreadSocketToCoreRatio with spread",
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
		Entry("PreferSpreadSocketToCoreRatio with preferSpread",
			instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
			},
		),
		Entry("SpreadOptions with spread",
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
		Entry("SpreadOptions with preferSpread",
			instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
						SpreadOptions: &instancetypev1beta1.SpreadOptions{
							Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
							Ratio:  pointer.P(uint32(3)),
						},
					},
				},
			},
		),
	)

	DescribeTable("should raise warning for",
		func(deprecatedTopology, expectedAlternativeTopology instancetypev1beta1.PreferredCPUTopology) {
			preferenceObj := &instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(deprecatedTopology),
					},
				},
			}
			ar := createClusterPreferenceAdmissionReview(preferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
			response := admitter.Admit(context.Background(), ar)
			Expect(response.Allowed).To(BeTrue())
			Expect(response.Warnings).To(HaveLen(1))
			Expect(response.Warnings[0]).To(
				ContainSubstring(
					fmt.Sprintf("PreferredCPUTopology %s is deprecated for removal in a future release, please use %s instead",
						deprecatedTopology,
						expectedAlternativeTopology,
					),
				),
			)
		},
		Entry("DeprecatedPreferSockets and provide Sockets as an alternative",
			instancetypev1beta1.DeprecatedPreferSockets,
			instancetypev1beta1.Sockets,
		),
		Entry("DeprecatedPreferCores and provide Cores as an alternative",
			instancetypev1beta1.DeprecatedPreferCores,
			instancetypev1beta1.Cores,
		),
		Entry("DeprecatedPreferThreads and provide Threads as an alternative",
			instancetypev1beta1.DeprecatedPreferThreads,
			instancetypev1beta1.Threads,
		),
		Entry("DeprecatedPreferSpread and provide Spread as an alternative",
			instancetypev1beta1.DeprecatedPreferSpread,
			instancetypev1beta1.Spread,
		),
		Entry("DeprecatedPreferAny and provide Any as an alternative",
			instancetypev1beta1.DeprecatedPreferAny,
			instancetypev1beta1.Any,
		),
	)
})

func createPreferenceAdmissionReview(
	preference *instancetypev1beta1.VirtualMachinePreference,
	version string,
) *admissionv1.AdmissionReview {
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

func createClusterPreferenceAdmissionReview(
	clusterPreference *instancetypev1beta1.VirtualMachineClusterPreference,
	version string,
) *admissionv1.AdmissionReview {
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
