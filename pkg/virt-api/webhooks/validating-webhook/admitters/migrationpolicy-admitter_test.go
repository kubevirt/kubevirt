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

	v12 "kubevirt.io/client-go/apis/core/v1"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("Validating MigrationPolicy Admitter", func() {
	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var admitter *MigrationPolicyAdmitter
	var _true *bool
	var mockMigrationPolies map[string]*kubecli.MockMigrationPolicyInterface
	var addedPolicies map[string][]v12.MigrationPolicy

	addMigrationPolicy := func(policy *v12.MigrationPolicy) {
		By("Setting expectation for migration policy")

		// Adding policy to added policies list
		alreadyExists := false
		for idx, existingPolicy := range addedPolicies[policy.Namespace] {
			if existingPolicy.Name == policy.Name {
				alreadyExists = true
				addedPolicies[policy.Namespace][idx] = *policy
			}
		}

		if !alreadyExists {
			addedPolicies[policy.Namespace] = append(addedPolicies[policy.Namespace], *policy)
		}

		mockMigrationPolicy, exists := mockMigrationPolies[policy.Namespace]
		if !exists {
			mockMigrationPolicy = kubecli.NewMockMigrationPolicyInterface(ctrl)
			mockMigrationPolies[policy.Namespace] = mockMigrationPolicy
		}

		mockMigrationPolicy.EXPECT().Get(policy.Name, gomock.Any()).Return(policy, nil)
		policyList := kubecli.NewMinimalMigrationPolicyList(addedPolicies[policy.Namespace]...)
		mockMigrationPolicy.EXPECT().List(gomock.Any()).Return(policyList, nil)
		virtClient.EXPECT().MigrationPolicy(policy.Namespace).Times(1).Return(mockMigrationPolicy)
	}
	setupMockPolicies := func(namespace string) {
		mockPolicy := kubecli.NewMockMigrationPolicyInterface(ctrl)

		minimalList := kubecli.NewMinimalMigrationPolicyList()
		mockPolicy.EXPECT().List(gomock.Any()).Return(minimalList, nil)
		mockPolicy.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, nil)

		mockMigrationPolies[namespace] = mockPolicy
		virtClient.EXPECT().MigrationPolicy(namespace).Times(1).Return(mockPolicy)
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		mockMigrationPolies = make(map[string]*kubecli.MockMigrationPolicyInterface)

		setupMockPolicies(metav1.NamespaceDefault)

		admitter = &MigrationPolicyAdmitter{Client: virtClient}

		t := true
		_true = &t
		addedPolicies = make(map[string][]v12.MigrationPolicy)
	})

	It("Should reject admitting two migration policies to the same namespace", func() {
		const policyName = "testpolicy"
		const namespace = metav1.NamespaceDefault

		By("Introducing a first policy to current namespace")
		policy := kubecli.NewMinimalMigrationPolicy(policyName, namespace)
		policy.Spec.AllowPostCopy = _true

		By("Admitting migration policy and expecting it to be allowed")
		admitter.admitAndExpect(policy, true)

		By("Introducing a second policy to current namespace")
		anotherPolicy := kubecli.NewMinimalMigrationPolicy(policyName, namespace)
		addMigrationPolicy(anotherPolicy)

		By("Admitting migration policy and expecting it to be denied")
		admitter.admitAndExpect(anotherPolicy, false)
	})

})

func createPolicyAdmissionReview(policy *v12.MigrationPolicy, namespace string) *admissionv1.AdmissionReview {
	policyBytes, _ := json.Marshal(policy)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Namespace: namespace,
			Resource: metav1.GroupVersionResource{
				Group:    v1.MigrationPolicyKind.Group,
				Resource: "migrationpolicies",
			},
			Object: runtime.RawExtension{
				Raw: policyBytes,
			},
		},
	}

	return ar
}

func createMigrationPolicyUpdateAdmissionReview(old, current *snapshotv1.VirtualMachineSnapshot, namespace string) *admissionv1.AdmissionReview {
	oldBytes, _ := json.Marshal(old)
	currentBytes, _ := json.Marshal(current)

	ar := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Operation: admissionv1.Update,
			Namespace: namespace,
			Resource: metav1.GroupVersionResource{
				Group:    v1.MigrationPolicyKind.Group,
				Resource: "migrationpolicies",
			},
			Object: runtime.RawExtension{
				Raw: currentBytes,
			},
			OldObject: runtime.RawExtension{
				Raw: oldBytes,
			},
		},
	}

	return ar
}

func (admitter *MigrationPolicyAdmitter) admitAndExpect(policy *v12.MigrationPolicy, expectAllowed bool) {
	ar := createPolicyAdmissionReview(policy, policy.Namespace)
	resp := admitter.Admit(ar)
	Expect(resp.Allowed).To(Equal(expectAllowed))
}
