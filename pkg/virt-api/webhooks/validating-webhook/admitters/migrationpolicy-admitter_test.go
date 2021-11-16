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

//
//import (
//	"encoding/json"
//
//	"kubevirt.io/api/migrations"
//
//	"k8s.io/client-go/testing"
//
//	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
//
//	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"
//
//	"github.com/golang/mock/gomock"
//	. "github.com/onsi/ginkgo"
//	. "github.com/onsi/gomega"
//	admissionv1 "k8s.io/api/admission/v1"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/runtime"
//
//	"kubevirt.io/client-go/kubecli"
//)
//
//var _ = Describe("Validating MigrationPolicy Admitter", func() {
//	var ctrl *gomock.Controller
//	var virtClient *kubecli.MockKubevirtClient
//	var admitter *MigrationPolicyAdmitter
//	var _true *bool
//	var mockMigrationPolies map[string]*kubevirtfake.Clientset
//	var addedPolicies map[string][]migrationsv1.MigrationPolicy
//
//	addMigrationPolicy := func(policy *migrationsv1.MigrationPolicy) {
//		By("Setting expectation for migration policy")
//
//		// Adding policy to added policies list
//		alreadyExists := false
//		for idx, existingPolicy := range addedPolicies[policy.Namespace] {
//			if existingPolicy.Name == policy.Name {
//				alreadyExists = true
//				addedPolicies[policy.Namespace][idx] = *policy
//			}
//		}
//
//		if !alreadyExists {
//			addedPolicies[policy.Namespace] = append(addedPolicies[policy.Namespace], *policy)
//		}
//	}
//
//	setupMockPolicies := func(namespace string) {
//		migrationsClient := kubevirtfake.NewSimpleClientset()
//		virtClient.EXPECT().MigrationPolicy().Return(
//			migrationsClient.MigrationsV1alpha1().MigrationPolicies()).AnyTimes()
//
//		migrationsClient.Fake.PrependReactor("get", migrations.ResourceMigrationPolicies, func(action testing.Action) (handled bool, obj runtime.Object, err error) {
//			a, ok := action.(testing.GetAction)
//			Expect(ok).To(BeTrue())
//			obj = nil
//
//			for _, policy := range addedPolicies[a.GetNamespace()] {
//				if policy.Name == a.GetName() {
//					obj = policy.DeepCopyObject()
//					break
//				}
//			}
//
//			return true, obj, nil
//		})
//
//		migrationsClient.Fake.PrependReactor("list", migrations.ResourceMigrationPolicies, func(action testing.Action) (handled bool, obj runtime.Object, err error) {
//			a, ok := action.(testing.ListAction)
//			Expect(ok).To(BeTrue())
//
//			policyList := kubecli.NewMinimalMigrationPolicyList(addedPolicies[a.GetNamespace()]...)
//
//			return true, policyList, nil
//		})
//
//		mockMigrationPolies[namespace] = migrationsClient
//	}
//
//	BeforeEach(func() {
//		ctrl = gomock.NewController(GinkgoT())
//		virtClient = kubecli.NewMockKubevirtClient(ctrl)
//		mockMigrationPolies = make(map[string]*kubevirtfake.Clientset)
//
//		setupMockPolicies(metav1.NamespaceDefault)
//
//		admitter = &MigrationPolicyAdmitter{Client: virtClient}
//
//		t := true
//		_true = &t
//		addedPolicies = make(map[string][]migrationsv1.MigrationPolicy)
//	})
//
//	It("Should reject admitting two migration policies to the same namespace", func() {
//		const policyName = "testpolicy"
//		const namespace = metav1.NamespaceDefault
//
//		By("Introducing a first policy to current namespace")
//		policy := kubecli.NewMinimalMigrationPolicy(policyName)
//		policy.Spec.AllowPostCopy = _true
//
//		By("Admitting migration policy and expecting it to be allowed")
//		admitter.admitAndExpect(policy, true)
//
//		By("Introducing a second policy to current namespace")
//		anotherPolicy := kubecli.NewMinimalMigrationPolicy(policyName)
//		addMigrationPolicy(anotherPolicy)
//
//		By("Admitting migration policy and expecting it to be denied")
//		admitter.admitAndExpect(anotherPolicy, false)
//	})
//
//})
//
//func createPolicyAdmissionReview(policy *migrationsv1.MigrationPolicy, namespace string) *admissionv1.AdmissionReview {
//	policyBytes, _ := json.Marshal(policy)
//
//	ar := &admissionv1.AdmissionReview{
//		Request: &admissionv1.AdmissionRequest{
//			Operation: admissionv1.Create,
//			Namespace: namespace,
//			Resource: metav1.GroupVersionResource{
//				Group:    migrationsv1.MigrationPolicyKind.Group,
//				Resource: migrations.ResourceMigrationPolicies,
//			},
//			Object: runtime.RawExtension{
//				Raw: policyBytes,
//			},
//		},
//	}
//
//	return ar
//}
//
//func (admitter *MigrationPolicyAdmitter) admitAndExpect(policy *migrationsv1.MigrationPolicy, expectAllowed bool) {
//	ar := createPolicyAdmissionReview(policy, policy.Namespace)
//	resp := admitter.Admit(ar)
//	Expect(resp.Allowed).To(Equal(expectAllowed))
//}
