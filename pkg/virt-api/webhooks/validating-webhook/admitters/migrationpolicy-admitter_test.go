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
* Copyright 2021 Red Hat, Inc.
*
 */

package admitters

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/utils/pointer"

	"kubevirt.io/api/migrations"

	"kubevirt.io/kubevirt/pkg/psa"

	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("Validating MigrationPolicy Admitter", func() {
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var admitter *MigrationPolicyAdmitter
	var policyName string
	var kubeClient *fake.Clientset

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubeClient = fake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		admitter = &MigrationPolicyAdmitter{Client: virtClient}
		policyName = "test-policy"

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
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
			migrationsv1.MigrationPolicySpec{CompletionTimeoutPerGiB: pointer.Int64Ptr(-1)},
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
			migrationsv1.MigrationPolicySpec{CompletionTimeoutPerGiB: pointer.Int64Ptr(1)},
		),

		Entry("zero CompletionTimeoutPerGiB",
			migrationsv1.MigrationPolicySpec{CompletionTimeoutPerGiB: pointer.Int64Ptr(0)},
		),

		Entry("zero BandwidthPerMigration",
			migrationsv1.MigrationPolicySpec{BandwidthPerMigration: resource.NewScaledQuantity(0, 1)},
		),

		Entry("empty spec",
			migrationsv1.MigrationPolicySpec{},
		),
	)

	DescribeTable("migration policy with postcopy enabled should be", func(privilegedNamespace bool) {
		kubeClient.Fake.PrependReactor("get", "namespaces", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			_, ok := action.(testing.GetAction)
			Expect(ok).To(BeTrue())

			labels := map[string]string{}
			if privilegedNamespace {
				labels[psa.PSALabel] = "privileged"
			}

			return true, &k8sv1.Namespace{
				TypeMeta: metav1.TypeMeta{Kind: "Namespace"},
				ObjectMeta: metav1.ObjectMeta{
					Name:   "default",
					Labels: labels,
				},
			}, nil
		})

		By("Setting up a new policy")
		policy := kubecli.NewMinimalMigrationPolicy(policyName)
		policy.Spec = migrationsv1.MigrationPolicySpec{AllowPostCopy: pointer.BoolPtr(true)}

		By("Expecting admitter would allow it")
		admitter.admitAndExpect(policy, privilegedNamespace)
	},
		Entry("allowed in a privileged namespace", true),
		Entry("denied in a non-privileged namespace", false),
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
	resp := admitter.Admit(ar)
	Expect(resp.Allowed).To(Equal(expectAllowed))
}
