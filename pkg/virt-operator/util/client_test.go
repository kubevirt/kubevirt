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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package util

import (
	"strconv"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	secv1 "github.com/openshift/api/security/v1"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	discoveryFake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

var _ = Describe("Operator Client", func() {

	Describe("Conditions and Finalizers", func() {
		getKubeVirtWithCreatedConditionAndRandomFinalizer := func() *v1.KubeVirt {
			return &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "ns",
					Finalizers: []string{
						"oldFinalizer",
					},
				},
				Status: v1.KubeVirtStatus{
					Phase: v1.KubeVirtPhaseDeploying,
					Conditions: []v1.KubeVirtCondition{
						{
							Type:    v1.KubeVirtConditionCreated,
							Status:  k8sv1.ConditionFalse,
							Reason:  "OldReason",
							Message: "old message",
						},
					},
				},
			}
		}

		var kv *v1.KubeVirt

		BeforeEach(func() {
			kv = getKubeVirtWithCreatedConditionAndRandomFinalizer()
		})

		Describe("Updating a condition", func() {

			Context("When it doesn't exist yet", func() {
				It("Should add the condition", func() {
					UpdateCondition(kv, v1.KubeVirtConditionReady, k8sv1.ConditionTrue, "NewReason", "new message")
					Expect(len(kv.Status.Conditions)).To(Equal(2), "should have 2 conditions")
					condition1 := kv.Status.Conditions[0]
					Expect(condition1.Type).To(Equal(v1.KubeVirtConditionCreated), "should keep old condition type")
					condition2 := kv.Status.Conditions[1]
					Expect(condition2.Type).To(Equal(v1.KubeVirtConditionReady), "should set correct condition type")
					Expect(condition2.Status).To(Equal(k8sv1.ConditionTrue), "should set correct condition status")
					Expect(condition2.Reason).To(Equal("NewReason"), "should set correct condition reason")
					Expect(condition2.Message).To(Equal("new message"), "should set correct condition message")
				})
			})

			Context("When it exists", func() {
				It("Should update the condition", func() {
					UpdateCondition(kv, v1.KubeVirtConditionCreated, k8sv1.ConditionTrue, "NewReason", "new message")
					Expect(len(kv.Status.Conditions)).To(Equal(1), "should still have 1 condition")
					condition1 := kv.Status.Conditions[0]
					Expect(condition1.Type).To(Equal(v1.KubeVirtConditionCreated), "should keep old condition type")
					Expect(condition1.Status).To(Equal(k8sv1.ConditionTrue), "should update condition status")
					Expect(condition1.Reason).To(Equal("NewReason"), "should update condition reason")
					Expect(condition1.Message).To(Equal("new message"), "should update condition message")
				})
			})

		})

		Describe("Removing a condition", func() {

			Context("When it doesn't exist", func() {
				It("Should not change existing conditions", func() {
					RemoveCondition(kv, v1.KubeVirtConditionReady)
					Expect(len(kv.Status.Conditions)).To(Equal(1), "should still have 1 condition")
					condition1 := kv.Status.Conditions[0]
					Expect(condition1.Type).To(Equal(v1.KubeVirtConditionCreated))
					Expect(condition1.Status).To(Equal(k8sv1.ConditionFalse))
					Expect(condition1.Reason).To(Equal("OldReason"))
					Expect(condition1.Message).To(Equal("old message"))
				})
			})

			Context("When it exists", func() {
				It("Should remove the condition", func() {
					RemoveCondition(kv, v1.KubeVirtConditionCreated)
					Expect(kv.Status.Conditions).To(BeEmpty(), "should have no condition")
				})
			})

		})

		Describe("Adding a finalizer", func() {
			Context("When another one already exists", func() {
				It("Should add it", func() {
					AddFinalizer(kv)
					Expect(len(kv.Finalizers)).To(Equal(2), "should have 2 finalizers")
					Expect(kv.Finalizers[0]).To(Equal("oldFinalizer"), "should keep first old finalizer")
					Expect(kv.Finalizers[1]).To(Equal(KubeVirtFinalizer), "should add new finalizer")
				})
				It("Should not add it again", func() {
					AddFinalizer(kv)
					Expect(len(kv.Finalizers)).To(Equal(2), "should still have 2 finalizers")
					Expect(kv.Finalizers[0]).To(Equal("oldFinalizer"), "should keep first old finalizer")
					Expect(kv.Finalizers[1]).To(Equal(KubeVirtFinalizer), "should keep second old finalizer")
				})
			})
		})

	})

	Describe("OpenShift Test", func() {

		var ctrl *gomock.Controller
		var virtClient *kubecli.MockKubevirtClient
		var discoveryClient *discoveryFake.FakeDiscovery

		BeforeEach(func() {

			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			discoveryClient = &discoveryFake.FakeDiscovery{
				Fake: &fake.NewSimpleClientset().Fake,
			}
			virtClient.EXPECT().DiscoveryClient().Return(discoveryClient).AnyTimes()

		})

		getServerResources := func(onOpenShift bool) []*metav1.APIResourceList {
			list := []*metav1.APIResourceList{
				{
					GroupVersion: v1.GroupVersion.String(),
					APIResources: []metav1.APIResource{
						{
							Name: "kubevirts",
						},
					},
				},
			}
			if onOpenShift {
				list = append(list, &metav1.APIResourceList{
					GroupVersion: secv1.GroupVersion.String(),
					APIResources: []metav1.APIResource{
						{
							Name: "securitycontextconstraints",
						},
					},
				})
			}
			return list
		}

		table.DescribeTable("Testing for OpenShift", func(onOpenShift bool) {

			discoveryClient.Fake.Resources = getServerResources(onOpenShift)
			isOnOpenShift, err := IsOnOpenshift(virtClient)
			Expect(err).ToNot(HaveOccurred(), "should not return an error")
			Expect(isOnOpenShift).To(Equal(onOpenShift), "should return "+strconv.FormatBool(onOpenShift))

		},
			table.Entry("on Kubernetes", false),
			table.Entry("on OpenShift", true),
		)

	})

})
