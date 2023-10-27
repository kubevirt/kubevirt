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
 * Copyright the KubeVirt Authors.
 *
 */

package apply

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
