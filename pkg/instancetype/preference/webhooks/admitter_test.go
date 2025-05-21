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

const (
	vmPreference        = "VirtualMachinePreference"
	vmClusterPreference = "VirtualMachineClusterPreference"
)

var (
	admitterPreference       *webhooks.PreferenceAdmitter
	preferenceObj            *instancetypev1beta1.VirtualMachinePreference
	clusterPreferenceObj     *instancetypev1beta1.VirtualMachineClusterPreference
	admitterclusterPreferenc *webhooks.ClusterPreferenceAdmitter
)

func validateUnsupportedSpreadAcross(preferredCPUTopology instancetypev1beta1.PreferredCPUTopology, prefType string) {
	var unsupportedAcrossValue instancetypev1beta1.SpreadAcross = "foobar"

	spec := instancetypev1beta1.VirtualMachinePreferenceSpec{
		PreferSpreadSocketToCoreRatio: uint32(3),
		CPU: &instancetypev1beta1.CPUPreferences{
			PreferredCPUTopology: &preferredCPUTopology,
			SpreadOptions: &instancetypev1beta1.SpreadOptions{
				Across: pointer.P(unsupportedAcrossValue),
			},
		},
	}

	var ar *admissionv1.AdmissionReview
	var response *admissionv1.AdmissionResponse

	if prefType == vmPreference {
		preferenceObj = &instancetypev1beta1.VirtualMachinePreference{
			Spec: spec,
		}
		ar = createPreferenceAdmissionReview(preferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
		response = admitterPreference.Admit(context.Background(), ar)
	} else if prefType == vmClusterPreference {
		clusterPreferenceObj = &instancetypev1beta1.VirtualMachineClusterPreference{
			Spec: spec,
		}
		ar = createClusterPreferenceAdmissionReview(clusterPreferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
		response = admitterclusterPreferenc.Admit(context.Background(), ar)
	}

	Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
	Expect(response.Result.Details.Causes).To(HaveLen(1))
	Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
	Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf("across %s is not supported", unsupportedAcrossValue)))
	Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "spreadOptions", "across").String()))
}

func testInvalidSpreadRatio(ar *admissionv1.AdmissionReview, prefType string) {
	var response *admissionv1.AdmissionResponse
	if prefType == vmPreference {
		response = admitterPreference.Admit(context.Background(), ar)
	} else if prefType == vmClusterPreference {
		response = admitterclusterPreferenc.Admit(context.Background(), ar)
	}

	Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
	Expect(response.Result.Details.Causes).To(HaveLen(1))
	Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
	Expect(response.Result.Details.Causes[0].Message).To(Equal(
		"only a ratio of 2 (1 core 2 threads) is allowed when spreading vCPUs over cores and threads"))
	Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "spreadOptions", "ratio").String()))
}
func createPreferenceSpec(isRatioUsed bool, isSpreadUsed bool) instancetypev1beta1.VirtualMachinePreferenceSpec {
	spreadOptions := &instancetypev1beta1.SpreadOptions{
		Across: pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
	}
	if isRatioUsed {
		spreadOptions.Ratio = pointer.P(uint32(3))
	}
	var spec instancetypev1beta1.VirtualMachinePreferenceSpec

	if isSpreadUsed {
		spec = instancetypev1beta1.VirtualMachinePreferenceSpec{
			PreferSpreadSocketToCoreRatio: uint32(3),
			CPU: &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: pointer.P(instancetypev1beta1.Spread),
				SpreadOptions:        spreadOptions,
			},
		}
	} else {
		spec = instancetypev1beta1.VirtualMachinePreferenceSpec{
			PreferSpreadSocketToCoreRatio: uint32(3),
			CPU: &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: pointer.P(instancetypev1beta1.DeprecatedPreferSpread),
				SpreadOptions:        spreadOptions,
			},
		}
	}

	return spec
}

func warnDeprecated(deprecatedTopo, preferredTopo instancetypev1beta1.PreferredCPUTopology, prefType string) {
	spec := instancetypev1beta1.VirtualMachinePreferenceSpec{
		CPU: &instancetypev1beta1.CPUPreferences{
			PreferredCPUTopology: pointer.P(deprecatedTopo),
		},
	}

	var ar *admissionv1.AdmissionReview
	var response *admissionv1.AdmissionResponse

	if prefType == vmPreference {
		preferenceObj = &instancetypev1beta1.VirtualMachinePreference{
			Spec: spec,
		}
		ar = createPreferenceAdmissionReview(preferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
		response = admitterPreference.Admit(context.Background(), ar)
	} else if prefType == vmClusterPreference {
		clusterPreferenceObj = &instancetypev1beta1.VirtualMachineClusterPreference{
			Spec: spec,
		}
		ar = createClusterPreferenceAdmissionReview(clusterPreferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
		response = admitterclusterPreferenc.Admit(context.Background(), ar)
	}

	Expect(response.Allowed).To(BeTrue())
	Expect(response.Warnings).To(HaveLen(1))
	Expect(response.Warnings[0]).To(
		ContainSubstring(
			fmt.Sprintf("PreferredCPUTopology %s is deprecated for removal in a future release, please use %s instead",
				deprecatedTopo,
				preferredTopo,
			),
		),
	)
}

var _ = Describe("Validating Preference Admitter", func() {
	BeforeEach(func() {
		admitterPreference = &webhooks.PreferenceAdmitter{}

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
		response := admitterPreference.Admit(context.Background(), ar)

		Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf("unknown preferredCPUTopology %s", unsupportedTopology)))
		Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "preferredCPUTopology").String()))
	})

	DescribeTable("should reject unsupported SpreadOptions Across value",
		func(preferredCPUTopology instancetypev1beta1.PreferredCPUTopology) {
			validateUnsupportedSpreadAcross(preferredCPUTopology, vmPreference)
		},
		Entry("with spread", instancetypev1beta1.Spread),
		Entry("with preferSpread", instancetypev1beta1.DeprecatedPreferSpread),
	)

	DescribeTable("should reject when spreading vCPUs across CoresThreads with a ratio higher than 2 set through",
		func(preferenceObj instancetypev1beta1.VirtualMachinePreference) {
			ar := createPreferenceAdmissionReview(&preferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
			testInvalidSpreadRatio(ar, vmPreference)
		},
		Entry("PreferSpreadSocketToCoreRatio with spread",
			instancetypev1beta1.VirtualMachinePreference{
				Spec: createPreferenceSpec(false, true),
			},
		),
		Entry("PreferSpreadSocketToCoreRatio with preferSpread",
			instancetypev1beta1.VirtualMachinePreference{
				Spec: createPreferenceSpec(false, false),
			},
		),
		Entry("SpreadOptions with spread",
			instancetypev1beta1.VirtualMachinePreference{
				Spec: createPreferenceSpec(true, true),
			},
		),
		Entry("SpreadOptions with preferSpread",
			instancetypev1beta1.VirtualMachinePreference{
				Spec: createPreferenceSpec(true, false),
			},
		),
	)

	DescribeTable("should raise warning for", func(deprecatedTopology, expectedAlternativeTopology instancetypev1beta1.PreferredCPUTopology) {
		warnDeprecated(deprecatedTopology, expectedAlternativeTopology, vmPreference)
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
	BeforeEach(func() {
		admitterclusterPreferenc = &webhooks.ClusterPreferenceAdmitter{}

		clusterPreferenceObj = &instancetypev1beta1.VirtualMachineClusterPreference{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		}
	})

	DescribeTable("should reject unsupported SpreadOptions Across value",
		func(preferredCPUTopology instancetypev1beta1.PreferredCPUTopology) {
			validateUnsupportedSpreadAcross(preferredCPUTopology, vmClusterPreference)
		},
		Entry("with spread", instancetypev1beta1.Spread),
		Entry("with preferSpread", instancetypev1beta1.DeprecatedPreferSpread),
	)

	DescribeTable("should reject when spreading vCPUs across CoresThreads with a ratio higher than 2 set through",
		func(clusterPreferenceObj instancetypev1beta1.VirtualMachineClusterPreference) {
			ar := createClusterPreferenceAdmissionReview(&clusterPreferenceObj, instancetypev1beta1.SchemeGroupVersion.Version)
			testInvalidSpreadRatio(ar, vmClusterPreference)
		},
		Entry("PreferSpreadSocketToCoreRatio with spread",
			instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: createPreferenceSpec(false, true),
			},
		),
		Entry("PreferSpreadSocketToCoreRatio with preferSpread",
			instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: createPreferenceSpec(false, false),
			},
		),
		Entry("SpreadOptions with spread",
			instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: createPreferenceSpec(true, true),
			},
		),
		Entry("SpreadOptions with preferSpread",
			instancetypev1beta1.VirtualMachineClusterPreference{
				Spec: createPreferenceSpec(true, false),
			},
		),
	)

	DescribeTable("should raise warning for",
		func(deprecatedTopology, expectedAlternativeTopology instancetypev1beta1.PreferredCPUTopology) {
			warnDeprecated(deprecatedTopology, expectedAlternativeTopology, vmClusterPreference)
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
