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

package apply

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	secv1 "github.com/openshift/api/security/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	secv1fake "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1/fake"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Apply Security Context Constraints", func() {

	Context("Manage users", func() {
		var stop chan struct{}
		var ctrl *gomock.Controller
		var stores util.Stores
		var informers util.Informers
		var virtClient *kubecli.MockKubevirtClient
		var secClient *secv1fake.FakeSecurityV1
		var err error

		namespace := "kubevirt-test"

		generateSCC := func(sccName string, usersList []string) *secv1.SecurityContextConstraints {
			return &secv1.SecurityContextConstraints{
				ObjectMeta: v12.ObjectMeta{
					Name: sccName,
				},
				Users: usersList,
			}
		}

		setupPrependReactor := func(sccName string, expectedPatch []byte) {
			secClient.Fake.PrependReactor("patch", "securitycontextconstraints",
				func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					patch, ok := action.(testing.PatchAction)
					Expect(ok).To(BeTrue())
					Expect(patch.GetName()).To(Equal(sccName), "Patch object name should match SCC name")
					Expect(patch.GetPatch()).To(Equal(expectedPatch))
					return true, nil, nil
				})
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			informers.SCC, _ = testutils.NewFakeInformerFor(&secv1.SecurityContextConstraints{})
			stores.SCCCache = informers.SCC.GetStore()
			secClient = &secv1fake.FakeSecurityV1{
				Fake: &fake.NewSimpleClientset().Fake,
			}
			virtClient.EXPECT().SecClient().Return(secClient).AnyTimes()
		})

		executeTest := func(scc *secv1.SecurityContextConstraints, expectedPatch string) {
			setupPrependReactor(scc.ObjectMeta.Name, []byte(expectedPatch))
			stores.SCCCache.Add(scc)

			r := &Reconciler{
				clientset: virtClient,
				stores:    stores,
			}

			err = r.removeKvServiceAccountsFromDefaultSCC(namespace)
			Expect(err).ToNot(HaveOccurred(), "Should successfully remove only the kubevirt service accounts")
		}

		AfterEach(func() {
			close(stop)
		})

		DescribeTable("Should remove Kubevirt service accounts from the default privileged SCC", func(additionalUserlist []string) {
			var serviceAccounts []string
			saMap := rbac.GetKubevirtComponentsServiceAccounts(namespace)
			for key := range saMap {
				serviceAccounts = append(serviceAccounts, key)
			}
			serviceAccounts = append(serviceAccounts, additionalUserlist...)
			scc := generateSCC("privileged", serviceAccounts)
			patchSet := patch.New()
			const usersPath = "/users"
			if len(additionalUserlist) != 0 {
				patchSet.AddOption(
					patch.WithTest(usersPath, serviceAccounts),
					patch.WithReplace(usersPath, additionalUserlist),
				)
			} else {
				patchSet.AddOption(
					patch.WithTest(usersPath, serviceAccounts),
					patch.WithReplace(usersPath, nil),
				)
			}
			patches, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred(), "Failed to generate patch payload")
			executeTest(scc, string(patches))
		},
			Entry("Without custom users", []string{}),
			Entry("With custom users", []string{"someuser"}),
		)
	})

})
