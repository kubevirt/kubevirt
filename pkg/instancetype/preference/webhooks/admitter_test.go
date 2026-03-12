/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

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
	instancetypev1 "kubevirt.io/api/instancetype/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/preference/webhooks"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Validating Preference Admitter", func() {
	var (
		admitter      *webhooks.PreferenceAdmitter
		preferenceObj *instancetypev1.VirtualMachinePreference
	)

	BeforeEach(func() {
		admitter = &webhooks.PreferenceAdmitter{}

		preferenceObj = &instancetypev1.VirtualMachinePreference{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		}
	})

	It("should reject unsupported PreferredCPUTopolgy value", func() {
		unsupportedTopology := instancetypev1.PreferredCPUTopology("foo")
		preferenceObj = &instancetypev1.VirtualMachinePreference{
			Spec: instancetypev1.VirtualMachinePreferenceSpec{
				CPU: &instancetypev1.CPUPreferences{
					PreferredCPUTopology: pointer.P(unsupportedTopology),
				},
			},
		}
		ar := createPreferenceAdmissionReview(preferenceObj, instancetypev1.SchemeGroupVersion.Version)
		response := admitter.Admit(context.Background(), ar)

		Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf("unknown preferredCPUTopology %s", unsupportedTopology)))
		Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "preferredCPUTopology").String()))
	})

	DescribeTable("should reject unsupported SpreadOptions Across value", func(preferredCPUTopology instancetypev1.PreferredCPUTopology) {
		var unsupportedAcrossValue instancetypev1.SpreadAcross = "foobar"
		preferenceObj = &instancetypev1.VirtualMachinePreference{
			Spec: instancetypev1.VirtualMachinePreferenceSpec{
				PreferSpreadSocketToCoreRatio: uint32(3),
				CPU: &instancetypev1.CPUPreferences{
					PreferredCPUTopology: &preferredCPUTopology,
					SpreadOptions: &instancetypev1.SpreadOptions{
						Across: pointer.P(unsupportedAcrossValue),
					},
				},
			},
		}
		ar := createPreferenceAdmissionReview(preferenceObj, instancetypev1.SchemeGroupVersion.Version)
		response := admitter.Admit(context.Background(), ar)

		Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf("across %s is not supported", unsupportedAcrossValue)))
		Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "spreadOptions", "across").String()))
	},
		Entry("with spread", instancetypev1.Spread),
	)

	DescribeTable("should reject deprecated PreferredCPUTopology values", func(deprecatedTopology instancetypev1.PreferredCPUTopology) {
		preferenceObj := &instancetypev1.VirtualMachinePreference{
			Spec: instancetypev1.VirtualMachinePreferenceSpec{
				CPU: &instancetypev1.CPUPreferences{
					PreferredCPUTopology: pointer.P(deprecatedTopology),
				},
			},
		}
		ar := createPreferenceAdmissionReview(preferenceObj, instancetypev1.SchemeGroupVersion.Version)
		response := admitter.Admit(context.Background(), ar)
		Expect(response.Allowed).To(BeFalse())
		Expect(response.Result.Details.Causes).To(HaveLen(1))
		Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
		Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf("unknown preferredCPUTopology %s", deprecatedTopology)))
		Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "preferredCPUTopology").String()))
	},
		Entry("DeprecatedPreferSockets",
			instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferSockets),
		),
		Entry("DeprecatedPreferCores",
			instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferCores),
		),
		Entry("DeprecatedPreferThreads",
			instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferThreads),
		),
		Entry("DeprecatedPreferSpread",
			instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferSpread),
		),
		Entry("DeprecatedPreferAny",
			instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferAny),
		),
	)
})

var _ = Describe("Validating ClusterPreference Admitter", func() {
	var (
		admitter             *webhooks.ClusterPreferenceAdmitter
		clusterPreferenceObj *instancetypev1.VirtualMachineClusterPreference
	)

	BeforeEach(func() {
		admitter = &webhooks.ClusterPreferenceAdmitter{}

		clusterPreferenceObj = &instancetypev1.VirtualMachineClusterPreference{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-name",
				Namespace: "test-namespace",
			},
		}
	})

	DescribeTable("should reject unsupported SpreadOptions Across value",
		func(preferredCPUTopology instancetypev1.PreferredCPUTopology) {
			var unsupportedAcrossValue instancetypev1.SpreadAcross = "foobar"
			clusterPreferenceObj = &instancetypev1.VirtualMachineClusterPreference{
				Spec: instancetypev1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &instancetypev1.CPUPreferences{
						PreferredCPUTopology: &preferredCPUTopology,
						SpreadOptions: &instancetypev1.SpreadOptions{
							Across: pointer.P(unsupportedAcrossValue),
						},
					},
				},
			}
			ar := createClusterPreferenceAdmissionReview(clusterPreferenceObj, instancetypev1.SchemeGroupVersion.Version)
			response := admitter.Admit(context.Background(), ar)

			Expect(response.Allowed).To(BeFalse(), "Expected preference to not be allowed")
			Expect(response.Result.Details.Causes).To(HaveLen(1))
			Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf("across %s is not supported", unsupportedAcrossValue)))
			Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "spreadOptions", "across").String()))
		},
		Entry("with spread", instancetypev1.Spread),
	)

	DescribeTable("should reject deprecated PreferredCPUTopology values",
		func(deprecatedTopology instancetypev1.PreferredCPUTopology) {
			clusterPreferenceObj := &instancetypev1.VirtualMachineClusterPreference{
				Spec: instancetypev1.VirtualMachinePreferenceSpec{
					CPU: &instancetypev1.CPUPreferences{
						PreferredCPUTopology: pointer.P(deprecatedTopology),
					},
				},
			}
			ar := createClusterPreferenceAdmissionReview(clusterPreferenceObj, instancetypev1.SchemeGroupVersion.Version)
			response := admitter.Admit(context.Background(), ar)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result.Details.Causes).To(HaveLen(1))
			Expect(response.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(response.Result.Details.Causes[0].Message).To(Equal(fmt.Sprintf("unknown preferredCPUTopology %s", deprecatedTopology)))
			Expect(response.Result.Details.Causes[0].Field).To(Equal(k8sfield.NewPath("spec", "cpu", "preferredCPUTopology").String()))
		},
		Entry("DeprecatedPreferSockets",
			instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferSockets),
		),
		Entry("DeprecatedPreferCores",
			instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferCores),
		),
		Entry("DeprecatedPreferThreads",
			instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferThreads),
		),
		Entry("DeprecatedPreferSpread",
			instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferSpread),
		),
		Entry("DeprecatedPreferAny",
			instancetypev1.PreferredCPUTopology(instancetypev1beta1.DeprecatedPreferAny),
		),
	)
})

func createPreferenceAdmissionReview(
	preference *instancetypev1.VirtualMachinePreference,
	version string,
) *admissionv1.AdmissionReview {
	bytes, err := json.Marshal(preference)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Could not JSON encode preference: %v", preference)

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    instancetypev1.SchemeGroupVersion.Group,
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
	clusterPreference *instancetypev1.VirtualMachineClusterPreference,
	version string,
) *admissionv1.AdmissionReview {
	bytes, err := json.Marshal(clusterPreference)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Could not JSON encode preference: %v", clusterPreference)

	return &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    instancetypev1.SchemeGroupVersion.Group,
				Version:  version,
				Resource: apiinstancetype.ClusterPluralPreferenceResourceName,
			},
			Object: runtime.RawExtension{
				Raw: bytes,
			},
		},
	}
}
