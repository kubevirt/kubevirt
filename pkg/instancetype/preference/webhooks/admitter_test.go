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

type preferenceTestHelper interface {
	createObject(preferredCPUTopology *instancetypev1beta1.PreferredCPUTopology, spreadAcross *instancetypev1beta1.SpreadAcross, ratio *uint32, socketToCoreRatio uint32) interface{}
	createObjectForWarning(deprecatedTopology instancetypev1beta1.PreferredCPUTopology) interface{}
	createAdmissionReview(obj interface{}, version string) *admissionv1.AdmissionReview
	getAdmitter() interface{}
	getSpecPath() *k8sfield.Path
}

type virtualMachinePreferenceHelper struct {
	admitter *webhooks.PreferenceAdmitter
}

func (h *virtualMachinePreferenceHelper) createObject(preferredCPUTopology *instancetypev1beta1.PreferredCPUTopology, spreadAcross *instancetypev1beta1.SpreadAcross, ratio *uint32, socketToCoreRatio uint32) interface{} {
	obj := &instancetypev1beta1.VirtualMachinePreference{
		Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
			PreferSpreadSocketToCoreRatio: socketToCoreRatio,
			CPU: &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: preferredCPUTopology,
				SpreadOptions: &instancetypev1beta1.SpreadOptions{
					Across: spreadAcross,
					Ratio:  ratio,
				},
			},
		},
	}
	return obj
}

func (h *virtualMachinePreferenceHelper) createObjectForWarning(deprecatedTopology instancetypev1beta1.PreferredCPUTopology) interface{} {
	return &instancetypev1beta1.VirtualMachinePreference{
		Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
			CPU: &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: pointer.P(deprecatedTopology),
			},
		},
	}
}

func (h *virtualMachinePreferenceHelper) createAdmissionReview(obj interface{}, version string) *admissionv1.AdmissionReview {
	return createPreferenceAdmissionReview(obj.(*instancetypev1beta1.VirtualMachinePreference), version)
}

func (h *virtualMachinePreferenceHelper) getAdmitter() interface{} {
	return h.admitter
}

func (h *virtualMachinePreferenceHelper) getSpecPath() *k8sfield.Path {
	return k8sfield.NewPath("spec")
}

type virtualMachineClusterPreferenceHelper struct {
	admitter *webhooks.ClusterPreferenceAdmitter
}

func (h *virtualMachineClusterPreferenceHelper) createObject(preferredCPUTopology *instancetypev1beta1.PreferredCPUTopology, spreadAcross *instancetypev1beta1.SpreadAcross, ratio *uint32, socketToCoreRatio uint32) interface{} {
	obj := &instancetypev1beta1.VirtualMachineClusterPreference{
		Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
			PreferSpreadSocketToCoreRatio: socketToCoreRatio,
			CPU: &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: preferredCPUTopology,
				SpreadOptions: &instancetypev1beta1.SpreadOptions{
					Across: spreadAcross,
					Ratio:  ratio,
				},
			},
		},
	}
	return obj
}

func (h *virtualMachineClusterPreferenceHelper) createObjectForWarning(deprecatedTopology instancetypev1beta1.PreferredCPUTopology) interface{} {
	return &instancetypev1beta1.VirtualMachineClusterPreference{
		Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
			CPU: &instancetypev1beta1.CPUPreferences{
				PreferredCPUTopology: pointer.P(deprecatedTopology),
			},
		},
	}
}

func (h *virtualMachineClusterPreferenceHelper) createAdmissionReview(obj interface{}, version string) *admissionv1.AdmissionReview {
	return createClusterPreferenceAdmissionReview(obj.(*instancetypev1beta1.VirtualMachineClusterPreference), version)
}

func (h *virtualMachineClusterPreferenceHelper) getAdmitter() interface{} {
	return h.admitter
}

func (h *virtualMachineClusterPreferenceHelper) getSpecPath() *k8sfield.Path {
	return k8sfield.NewPath("spec")
}

func runUnsupportedSpreadOptionsTest(helper preferenceTestHelper, preferredCPUTopology instancetypev1beta1.PreferredCPUTopology) {
	var unsupportedAcrossValue instancetypev1beta1.SpreadAcross = "foobar"
	obj := helper.createObject(&preferredCPUTopology, pointer.P(unsupportedAcrossValue), nil, 3)
	ar := helper.createAdmissionReview(obj, instancetypev1beta1.SchemeGroupVersion.Version)

	var response *admissionv1.AdmissionResponse
	switch admitter := helper.getAdmitter().(type) {
	case *webhooks.PreferenceAdmitter:
		response = admitter.Admit(context.Background(), ar)
	case *webhooks.ClusterPreferenceAdmitter:
		response = admitter.Admit(context.Background(), ar)
	}

	Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
	Expect(response.Result.Details.Causes).To(HaveLen(1))
	Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
	Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf("across %s is not supported", unsupportedAcrossValue)))
	Expect(response.Result.Details.Causes[0].Field).To(Equal(helper.getSpecPath().Child("cpu", "spreadOptions", "across").String()))
}

func runSpreadingVCPUsTest(helper preferenceTestHelper, testCase string, preferredTopology instancetypev1beta1.PreferredCPUTopology, useRatio bool) {
	var socketToCoreRatio uint32 = 3
	var obj interface{}

	if useRatio {
		obj = helper.createObject(
			pointer.P(preferredTopology),
			pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
			pointer.P(uint32(3)),
			0,
		)
	} else {
		obj = helper.createObject(
			pointer.P(preferredTopology),
			pointer.P(instancetypev1beta1.SpreadAcrossCoresThreads),
			nil,
			socketToCoreRatio,
		)
	}

	ar := helper.createAdmissionReview(obj, instancetypev1beta1.SchemeGroupVersion.Version)

	var response *admissionv1.AdmissionResponse
	switch admitter := helper.getAdmitter().(type) {
	case *webhooks.PreferenceAdmitter:
		response = admitter.Admit(context.Background(), ar)
	case *webhooks.ClusterPreferenceAdmitter:
		response = admitter.Admit(context.Background(), ar)
	}

	Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
	Expect(response.Result.Details.Causes).To(HaveLen(1))
	Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
	Expect(response.Result.Details.Causes[0].Message).To(Equal(
		"only a ratio of 2 (1 core 2 threads) is allowed when spreading vCPUs over cores and threads"))
	Expect(response.Result.Details.Causes[0].Field).To(Equal(helper.getSpecPath().Child("cpu", "spreadOptions", "ratio").String()))
}

func runWarningTest(helper preferenceTestHelper, deprecatedTopology, expectedAlternativeTopology instancetypev1beta1.PreferredCPUTopology) {
	obj := helper.createObjectForWarning(deprecatedTopology)
	ar := helper.createAdmissionReview(obj, instancetypev1beta1.SchemeGroupVersion.Version)

	var response *admissionv1.AdmissionResponse
	switch admitter := helper.getAdmitter().(type) {
	case *webhooks.PreferenceAdmitter:
		response = admitter.Admit(context.Background(), ar)
	case *webhooks.ClusterPreferenceAdmitter:
		response = admitter.Admit(context.Background(), ar)
	}

	Expect(response.Allowed).To(BeTrue())
	Expect(response.Warnings).To(HaveLen(1))
	Expect(response.Warnings[0]).To(ContainSubstring(
		fmt.Sprintf("PreferredCPUTopology %s is deprecated for removal in a future release, please use %s instead",
			deprecatedTopology, expectedAlternativeTopology)))
}

var _ = Describe("Validating Preference Admitter", func() {
	var (
		admitter      *webhooks.PreferenceAdmitter
		preferenceObj *instancetypev1beta1.VirtualMachinePreference
		helper        preferenceTestHelper
	)

	BeforeEach(func() {
		admitter = &webhooks.PreferenceAdmitter{}
		helper = &virtualMachinePreferenceHelper{admitter: admitter}

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

	DescribeTable("should reject unsupported SpreadOptions Across value",
		runUnsupportedSpreadOptionsTest,
		Entry("with spread", helper, instancetypev1beta1.Spread),
		Entry("with preferSpread", helper, instancetypev1beta1.DeprecatedPreferSpread),
	)

	DescribeTable("should reject when spreading vCPUs across CoresThreads with a ratio higher than 2",
		func(testCase string, preferredTopology instancetypev1beta1.PreferredCPUTopology, useRatio bool) {
			runSpreadingVCPUsTest(helper, testCase, preferredTopology, useRatio)
		},
		Entry("PreferSpreadSocketToCoreRatio with spread", "case1", instancetypev1beta1.Spread, false),
		Entry("PreferSpreadSocketToCoreRatio with preferSpread", "case2", instancetypev1beta1.DeprecatedPreferSpread, false),
		Entry("SpreadOptions with spread", "case3", instancetypev1beta1.Spread, true),
		Entry("SpreadOptions with preferSpread", "case4", instancetypev1beta1.DeprecatedPreferSpread, true),
	)

	DescribeTable("should raise warning for",
		runWarningTest,
		Entry("DeprecatedPreferSockets and provide Sockets as an alternative",
			helper, instancetypev1beta1.DeprecatedPreferSockets, instancetypev1beta1.Sockets,
		),
		Entry("DeprecatedPreferCores and provide Cores as an alternative",
			helper, instancetypev1beta1.DeprecatedPreferCores, instancetypev1beta1.Cores,
		),
		Entry("DeprecatedPreferThreads and provide Threads as an alternative",
			helper, instancetypev1beta1.DeprecatedPreferThreads, instancetypev1beta1.Threads,
		),
		Entry("DeprecatedPreferSpread and provide Spread as an alternative",
			helper, instancetypev1beta1.DeprecatedPreferSpread, instancetypev1beta1.Spread,
		),
		Entry("DeprecatedPreferAny and provide Any as an alternative",
			helper, instancetypev1beta1.DeprecatedPreferAny, instancetypev1beta1.Any,
		),
	)
})

var _ = Describe("Validating ClusterPreference Admitter", func() {
	var (
		admitter *webhooks.ClusterPreferenceAdmitter
		helper   preferenceTestHelper
	)

	BeforeEach(func() {
		admitter = &webhooks.ClusterPreferenceAdmitter{}
		helper = &virtualMachineClusterPreferenceHelper{admitter: admitter}
	})

	DescribeTable("should reject unsupported SpreadOptions Across value",
		runUnsupportedSpreadOptionsTest,
		Entry("with spread", helper, instancetypev1beta1.Spread),
		Entry("with preferSpread", helper, instancetypev1beta1.DeprecatedPreferSpread),
	)

	DescribeTable("should reject when spreading vCPUs across CoresThreads with a ratio higher than 2",
		func(testCase string, preferredTopology instancetypev1beta1.PreferredCPUTopology, useRatio bool) {
			runSpreadingVCPUsTest(helper, testCase, preferredTopology, useRatio)
		},
		Entry("PreferSpreadSocketToCoreRatio with spread", "case1", instancetypev1beta1.Spread, false),
		Entry("PreferSpreadSocketToCoreRatio with preferSpread", "case2", instancetypev1beta1.DeprecatedPreferSpread, false),
		Entry("SpreadOptions with spread", "case3", instancetypev1beta1.Spread, true),
		Entry("SpreadOptions with preferSpread", "case4", instancetypev1beta1.DeprecatedPreferSpread, true),
	)

	DescribeTable("should raise warning for",
		runWarningTest,
		Entry("DeprecatedPreferSockets and provide Sockets as an alternative",
			helper, instancetypev1beta1.DeprecatedPreferSockets, instancetypev1beta1.Sockets,
		),
		Entry("DeprecatedPreferCores and provide Cores as an alternative",
			helper, instancetypev1beta1.DeprecatedPreferCores, instancetypev1beta1.Cores,
		),
		Entry("DeprecatedPreferThreads and provide Threads as an alternative",
			helper, instancetypev1beta1.DeprecatedPreferThreads, instancetypev1beta1.Threads,
		),
		Entry("DeprecatedPreferSpread and provide Spread as an alternative",
			helper, instancetypev1beta1.DeprecatedPreferSpread, instancetypev1beta1.Spread,
		),
		Entry("DeprecatedPreferAny and provide Any as an alternative",
			helper, instancetypev1beta1.DeprecatedPreferAny, instancetypev1beta1.Any,
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
