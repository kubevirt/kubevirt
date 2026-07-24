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
*
 */

package admitters

import (
	"context"
	"encoding/json"

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/migrations"

	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Validating MigrationPolicy Admitter", func() {
	var admitter *MigrationPolicyAdmitter
	var policyName string

	kv := &v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt",
			Namespace: "kubevirt",
		},
		Spec: v1.KubeVirtSpec{
			Configuration: v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{
					FeatureGates: []string{featuregate.MigrationStallDetection},
				},
			},
		},
		Status: v1.KubeVirtStatus{
			Phase:               v1.KubeVirtPhaseDeploying,
			DefaultArchitecture: "amd64",
		},
	}
	config, _, kvStore := testutils.NewFakeClusterConfigUsingKV(kv)

	enableFeatureGate := func(fg string) {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{fg}
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
	}

	disableFeatureGates := func() {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = nil
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
	}

	BeforeEach(func() {
		admitter = NewMigrationPolicyAdmitter(config)
		policyName = "test-policy"
		enableFeatureGate(featuregate.MigrationStallDetection)
	})

	AfterEach(func() {
		disableFeatureGates()
	})

	DescribeTable("should reject migration policy with", func(policySpec migrationsv1.MigrationPolicySpec) {
		By("Setting up a new policy")
		policy := kubecli.NewMinimalMigrationPolicy(policyName)
		policy.Spec = policySpec

		By("Expecting admitter would not allow it")
		admitter.admitAndExpect(policy, false)
	},
		Entry("negative BandwidthPerMigration",
			migrationsv1.MigrationPolicySpec{BandwidthPerMigration: resource.NewScaledQuantity(-123, 1)},
		),

		Entry("negative CompletionTimeoutPerGiB",
			migrationsv1.MigrationPolicySpec{CompletionTimeoutPerGiB: pointer.P(int64(-1))},
		),

		Entry("negative precopyPossibleFactor",
			migrationsv1.MigrationPolicySpec{
				ExperimentalMigrationOptions: &v1.ExperimentalMigrationOptions{
					StallDetector: &v1.StallDetectorOptions{
						PrecopyPossibleFactor: resource.NewQuantity(-1, resource.DecimalSI),
					},
				},
			},
		),

		Entry("ewmaAlpha zero",
			migrationsv1.MigrationPolicySpec{
				ExperimentalMigrationOptions: &v1.ExperimentalMigrationOptions{
					StallDetector: &v1.StallDetectorOptions{
						EwmaAlpha: pointer.P(resource.MustParse("0")),
					},
				},
			},
		),

		Entry("ewmaAlpha above maximum",
			migrationsv1.MigrationPolicySpec{
				ExperimentalMigrationOptions: &v1.ExperimentalMigrationOptions{
					StallDetector: &v1.StallDetectorOptions{
						EwmaAlpha: pointer.P(resource.MustParse("1.1")),
					},
				},
			},
		),

		Entry("ewmaAlpha with excessive decimal precision",
			migrationsv1.MigrationPolicySpec{
				ExperimentalMigrationOptions: &v1.ExperimentalMigrationOptions{
					StallDetector: &v1.StallDetectorOptions{
						EwmaAlpha: pointer.P(resource.MustParse("0.123456")),
					},
				},
			},
		),

		Entry("precopyPossibleFactor below minimum",
			migrationsv1.MigrationPolicySpec{
				ExperimentalMigrationOptions: &v1.ExperimentalMigrationOptions{
					StallDetector: &v1.StallDetectorOptions{
						PrecopyPossibleFactor: pointer.P(resource.MustParse("0.5")),
					},
				},
			},
		),

		Entry("patienceWindowDecayFactor above maximum",
			migrationsv1.MigrationPolicySpec{
				ExperimentalMigrationOptions: &v1.ExperimentalMigrationOptions{
					StallDetector: &v1.StallDetectorOptions{
						PatienceWindowDecayFactor: pointer.P(resource.MustParse("1.5")),
					},
				},
			},
		),
	)

	DescribeTable("should accept migration policy with", func(policySpec migrationsv1.MigrationPolicySpec) {
		By("Setting up a new policy")
		policy := kubecli.NewMinimalMigrationPolicy(policyName)
		policy.Spec = policySpec

		By("Expecting admitter would allow it")
		admitter.admitAndExpect(policy, true)
	},
		Entry("greater than zero BandwidthPerMigration",
			migrationsv1.MigrationPolicySpec{BandwidthPerMigration: resource.NewScaledQuantity(1, 1)},
		),

		Entry("greater than zero CompletionTimeoutPerGiB",
			migrationsv1.MigrationPolicySpec{CompletionTimeoutPerGiB: pointer.P(int64(1))},
		),

		Entry("zero CompletionTimeoutPerGiB",
			migrationsv1.MigrationPolicySpec{CompletionTimeoutPerGiB: pointer.P(int64(0))},
		),

		Entry("valid MaxDowntimeMs",
			migrationsv1.MigrationPolicySpec{MaxDowntimeMs: pointer.P(uint64(900))},
		),

		Entry("zero BandwidthPerMigration",
			migrationsv1.MigrationPolicySpec{BandwidthPerMigration: resource.NewScaledQuantity(0, 1)},
		),

		Entry("empty spec",
			migrationsv1.MigrationPolicySpec{},
		),

		Entry("completionTimeoutFactor without upper bound",
			migrationsv1.MigrationPolicySpec{
				ExperimentalMigrationOptions: &v1.ExperimentalMigrationOptions{
					StallDetector: &v1.StallDetectorOptions{
						CompletionTimeoutFactor: pointer.P(resource.MustParse("3")),
					},
				},
			},
		),

		Entry("precopyPossibleFactor without upper bound",
			migrationsv1.MigrationPolicySpec{
				ExperimentalMigrationOptions: &v1.ExperimentalMigrationOptions{
					StallDetector: &v1.StallDetectorOptions{
						PrecopyPossibleFactor: pointer.P(resource.MustParse("2")),
					},
				},
			},
		),

		Entry("ewmaAlpha with milli DecimalSI form",
			migrationsv1.MigrationPolicySpec{
				ExperimentalMigrationOptions: &v1.ExperimentalMigrationOptions{
					StallDetector: &v1.StallDetectorOptions{
						EwmaAlpha: pointer.P(resource.MustParse("500m")),
					},
				},
			},
		),
	)

	DescribeTable("maxDowntimeMs feature gate validation when feature gate is disabled",
		func(isUpdate bool, oldMs, newMs *uint64, expectAllowed bool) {
			disableFeatureGates()
			newPolicy := kubecli.NewMinimalMigrationPolicy(policyName)
			newPolicy.Spec.MaxDowntimeMs = newMs
			if !isUpdate {
				admitter.admitAndExpect(newPolicy, expectAllowed)
				return
			}
			oldPolicy := kubecli.NewMinimalMigrationPolicy(policyName)
			oldPolicy.Spec.MaxDowntimeMs = oldMs
			if expectAllowed {
				newPolicy.Spec.AllowAutoConverge = pointer.P(true)
			}
			admitter.admitUpdateAndExpect(oldPolicy, newPolicy, expectAllowed)
		},
		Entry("reject on create", false, nil, pointer.P(uint64(900)), false),
		Entry("allow unchanged update", true, pointer.P(uint64(900)), pointer.P(uint64(900)), true),
		Entry("reject changing value on update", true, pointer.P(uint64(500)), pointer.P(uint64(900)), false),
	)

	DescribeTable("experimental.stallDetector feature gate validation when feature gate is disabled",
		func(isUpdate bool, oldStallDetector, newStallDetector *v1.StallDetectorOptions, expectAllowed bool) {
			disableFeatureGates()
			newPolicy := kubecli.NewMinimalMigrationPolicy(policyName)
			if newStallDetector != nil {
				newPolicy.Spec.ExperimentalMigrationOptions = &v1.ExperimentalMigrationOptions{
					StallDetector: newStallDetector,
				}
			}
			if !isUpdate {
				admitter.admitAndExpect(newPolicy, expectAllowed)
				return
			}
			oldPolicy := kubecli.NewMinimalMigrationPolicy(policyName)
			if oldStallDetector != nil {
				oldPolicy.Spec.ExperimentalMigrationOptions = &v1.ExperimentalMigrationOptions{
					StallDetector: oldStallDetector,
				}
			}
			if expectAllowed {
				newPolicy.Spec.AllowAutoConverge = pointer.P(true)
			}
			admitter.admitUpdateAndExpect(oldPolicy, newPolicy, expectAllowed)
		},
		Entry("reject on create", false, nil, &v1.StallDetectorOptions{
			StallMargin: pointer.P(int64(4)),
		}, false),
		Entry("allow unchanged update", true,
			&v1.StallDetectorOptions{StallMargin: pointer.P(int64(4))},
			&v1.StallDetectorOptions{StallMargin: pointer.P(int64(4))},
			true,
		),
		Entry("reject changing value on update", true,
			&v1.StallDetectorOptions{StallMargin: pointer.P(int64(4))},
			&v1.StallDetectorOptions{StallMargin: pointer.P(int64(8))},
			false,
		),
	)

})

func createPolicyAdmissionReview(policy *migrationsv1.MigrationPolicy, namespace string) *admissionv1.AdmissionReview {
	policyBytes, _ := json.Marshal(policy)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Namespace: namespace,
			Resource: metav1.GroupVersionResource{
				Group:    migrationsv1.MigrationPolicyKind.Group,
				Resource: migrations.ResourceMigrationPolicies,
			},
			Object: runtime.RawExtension{
				Raw: policyBytes,
			},
		},
	}

	return ar
}

func (admitter *MigrationPolicyAdmitter) admitAndExpect(policy *migrationsv1.MigrationPolicy, expectAllowed bool) {
	ar := createPolicyAdmissionReview(policy, policy.Namespace)
	resp := admitter.Admit(context.Background(), ar)
	Expect(resp.Allowed).To(Equal(expectAllowed))
}

func (admitter *MigrationPolicyAdmitter) admitUpdateAndExpect(oldPolicy, newPolicy *migrationsv1.MigrationPolicy, expectAllowed bool) {
	ar := createPolicyAdmissionReview(newPolicy, newPolicy.Namespace)
	ar.Request.Operation = admissionv1.Update
	oldPolicyBytes, _ := json.Marshal(oldPolicy)
	ar.Request.OldObject = runtime.RawExtension{Raw: oldPolicyBytes}
	resp := admitter.Admit(context.Background(), ar)
	Expect(resp.Allowed).To(Equal(expectAllowed))
}

var _ = Describe("parseScalarFloatFromQuantity", func() {
	DescribeTable("parseScalarFloatFromQuantity", func(value string, precision int, expected float64, expectError bool) {
		q := resource.MustParse(value)
		result, err := parseScalarFloatFromQuantity(&q, precision)
		if expectError {
			Expect(err).To(HaveOccurred())
			return
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(expected))
	},
		Entry("accepts integer quantity", "2", 3, 2.0, false),
		Entry("accepts decimal within precision", "1.5", 1, 1.5, false),
		Entry("rejects negative quantity", "-1", 3, 0.0, true),
		Entry("rejects too many decimal places", "1.55", 1, 0.0, true),
		Entry("accepts exponential notation when value fits precision", "1.5e1", 0, 15.0, false),
	)
})
