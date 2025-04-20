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

	"kubevirt.io/api/migrations"

	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Validating MigrationPolicy Admitter", func() {
	var admitter *MigrationPolicyAdmitter
	var policyName string

	BeforeEach(func() {
		admitter = &MigrationPolicyAdmitter{}
		policyName = "test-policy"
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

		Entry("zero BandwidthPerMigration",
			migrationsv1.MigrationPolicySpec{BandwidthPerMigration: resource.NewScaledQuantity(0, 1)},
		),

		Entry("empty spec",
			migrationsv1.MigrationPolicySpec{},
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
