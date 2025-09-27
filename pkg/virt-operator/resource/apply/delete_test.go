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

package apply

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	kubecli "kubevirt.io/client-go/kubecli"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/testing"
	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Deletion", func() {

	Context("CRD deletion", func() {
		It("Filter needs deletion", func() {
			crds := []*extv1.CustomResourceDefinition{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "does-not-deletion1",
						DeletionTimestamp: now(),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "needs-deletion",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "does-not-deletion2",
						DeletionTimestamp: now(),
					},
				},
			}

			needDeletion := crdFilterNeedDeletion(crds)

			Expect(needDeletion).To(HaveLen(1))
			Expect(needDeletion[0].Name).To(Equal("needs-deletion"))
		})

		It("Filter needs finalizer", func() {
			crds := []*extv1.CustomResourceDefinition{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "does-not-need-finalizer-1",
						DeletionTimestamp: now(),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "needs-finalizer",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "does-not-need-finalizer-2",
						Finalizers: []string{v1.VirtOperatorComponentFinalizer},
					},
				},
			}

			needAdded := crdFilterNeedFinalizerAdded(crds)

			Expect(needAdded).To(HaveLen(1))
			Expect(needAdded[0].Name).To(Equal("needs-finalizer"))
		})

		It("Filter needs finalizer removed", func() {
			crds := []*extv1.CustomResourceDefinition{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "1",
						Finalizers:        []string{v1.VirtOperatorComponentFinalizer},
						DeletionTimestamp: now(),
					},
					Status: extv1.CustomResourceDefinitionStatus{
						Conditions: []extv1.CustomResourceDefinitionCondition{
							instanceRemovedCondition(),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "2",
						Finalizers:        []string{v1.VirtOperatorComponentFinalizer},
						DeletionTimestamp: now(),
					},
					Status: extv1.CustomResourceDefinitionStatus{

						Conditions: []extv1.CustomResourceDefinitionCondition{
							instanceRemovedCondition(),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "3",
						Finalizers:        []string{v1.VirtOperatorComponentFinalizer},
						DeletionTimestamp: now(),
					},
					Status: extv1.CustomResourceDefinitionStatus{
						Conditions: []extv1.CustomResourceDefinitionCondition{
							instanceRemovedCondition(),
						},
					},
				},
			}

			needRemoved := crdFilterNeedFinalizerRemoved(crds)
			Expect(needRemoved).To(HaveLen(3))
		})

		It("Should block finalizer removal until all CRD CRs are removed", func() {
			crds := []*extv1.CustomResourceDefinition{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "1",
						Finalizers:        []string{v1.VirtOperatorComponentFinalizer},
						DeletionTimestamp: now(),
					},
					Status: extv1.CustomResourceDefinitionStatus{
						Conditions: []extv1.CustomResourceDefinitionCondition{
							instanceRemovedCondition(),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "2",
						Finalizers:        []string{v1.VirtOperatorComponentFinalizer},
						DeletionTimestamp: now(),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "3",
						Finalizers:        []string{v1.VirtOperatorComponentFinalizer},
						DeletionTimestamp: now(),
					},
					Status: extv1.CustomResourceDefinitionStatus{
						Conditions: []extv1.CustomResourceDefinitionCondition{
							instanceRemovedCondition(),
						},
					},
				},
			}

			needRemoved := crdFilterNeedFinalizerRemoved(crds)
			Expect(needRemoved).To(BeEmpty())
		})
	})

	Context("Node label deletion", func() {
		var clientset *fake.Clientset
		var kubevirtClient *kubecli.MockKubevirtClient
		var ctrl *gomock.Controller

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			clientset = fake.NewSimpleClientset()
			kubevirtClient = kubecli.NewMockKubevirtClient(ctrl)
			kubevirtClient.EXPECT().CoreV1().Return(clientset.CoreV1()).AnyTimes()
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should delete kubevirt labels from nodes", func() {
			nodes := &k8sv1.NodeList{
				Items: []k8sv1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "node1",
							Labels: map[string]string{
								"kubevirt.io/schedulable":          "true",
								"kubevirt.io/some-label":           "value",
								"prefix.kubevirt.io/another-label": "value",
								"other-label":                      "value",
							},
							Annotations: map[string]string{
								"kubevirt.io/some-annotation":           "value",
								"prefix.kubevirt.io/another-annotation": "value",
								"other-annotation":                      "value",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "node2",
							Labels: map[string]string{
								"kubevirt.io/schedulable":          "true",
								"kubevirt.io/some-label":           "value",
								"prefix.kubevirt.io/another-label": "value",
							},
							Annotations: map[string]string{
								"kubevirt.io/some-annotation":           "value",
								"prefix-kubevirt.io/another-annotation": "value",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:   "node3",
							Labels: map[string]string{},
						},
					},
				},
			}

			_, err := clientset.CoreV1().Nodes().Create(context.Background(), &nodes.Items[0], metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			_, err = clientset.CoreV1().Nodes().Create(context.Background(), &nodes.Items[1], metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			_, err = clientset.CoreV1().Nodes().Create(context.Background(), &nodes.Items[2], metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = deleteNodeLabelsAndAnnotations(kubevirtClient)
			Expect(err).ToNot(HaveOccurred())

			updatedNode1, err := clientset.CoreV1().Nodes().Get(context.Background(), "node1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedNode1.Labels).ToNot(HaveKey("kubevirt.io/some-label"))
			Expect(updatedNode1.Labels).ToNot(HaveKey("prefix.kubevirt.io/another-label"))
			Expect(updatedNode1.Labels).To(HaveKey("other-label"))
			Expect(updatedNode1.Annotations).ToNot(HaveKey("kubevirt.io/some-annotation"))
			Expect(updatedNode1.Annotations).ToNot(HaveKey("prefix.kubevirt.io/another-annotation"))
			Expect(updatedNode1.Annotations).To(HaveKey("other-annotation"))

			updatedNode2, err := clientset.CoreV1().Nodes().Get(context.Background(), "node2", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedNode2.Labels).To(BeEmpty())
			Expect(updatedNode2.Annotations).To(BeEmpty())

			Expect(clientset.Actions()).To(WithTransform(func(actions []testing.Action) []testing.Action {
				var node3patchActions []testing.Action
				for _, action := range actions {
					if action.GetVerb() == "patch" &&
						action.GetResource().Resource == "nodes" {
						patchAction := action.(testing.PatchAction)
						if patchAction.GetName() == "node3" {
							node3patchActions = append(node3patchActions, action)
						}
					}
				}
				return node3patchActions
			}, BeEmpty()))

		})
	})
})

func instanceRemovedCondition() extv1.CustomResourceDefinitionCondition {
	return extv1.CustomResourceDefinitionCondition{
		Type:   extv1.Terminating,
		Status: extv1.ConditionFalse,
		Reason: "InstanceDeletionCompleted",
	}
}

func now() *metav1.Time {
	now := metav1.Now()
	return &now
}
