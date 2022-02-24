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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
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
					updateCondition(kv, v1.KubeVirtConditionAvailable, k8sv1.ConditionTrue, "NewReason", "new message")
					Expect(len(kv.Status.Conditions)).To(Equal(2), "should have 2 conditions")
					condition1 := kv.Status.Conditions[0]
					Expect(condition1.Type).To(Equal(v1.KubeVirtConditionCreated), "should keep old condition type")
					condition2 := kv.Status.Conditions[1]
					Expect(condition2.Type).To(Equal(v1.KubeVirtConditionAvailable), "should set correct condition type")
					Expect(condition2.Status).To(Equal(k8sv1.ConditionTrue), "should set correct condition status")
					Expect(condition2.Reason).To(Equal("NewReason"), "should set correct condition reason")
					Expect(condition2.Message).To(Equal("new message"), "should set correct condition message")
				})
			})

			Context("When it exists", func() {
				It("Should update the condition", func() {
					updateCondition(kv, v1.KubeVirtConditionCreated, k8sv1.ConditionTrue, "NewReason", "new message")
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
					removeCondition(kv, v1.KubeVirtConditionAvailable)
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
					removeCondition(kv, v1.KubeVirtConditionCreated)
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

		Describe("Setting condition timestamps", func() {
			var kv1 *v1.KubeVirt
			var kv2 *v1.KubeVirt
			empty := metav1.Time{}
			now := metav1.Time{
				Time: time.Now(),
			}

			BeforeEach(func() {
				kv1 = &v1.KubeVirt{
					Status: v1.KubeVirtStatus{
						Conditions: []v1.KubeVirtCondition{},
					},
				}
				kv2 = &v1.KubeVirt{
					Status: v1.KubeVirtStatus{
						Conditions: []v1.KubeVirtCondition{},
					},
				}
			})
			Context("For new condition", func() {
				It("should set lastProbeTime only", func() {
					kv2.Status.Conditions = append(kv1.Status.Conditions, v1.KubeVirtCondition{
						Type: v1.KubeVirtConditionAvailable,
					})
					SetConditionTimestamps(kv1, kv2)
					Expect(kv2.Status.Conditions[0].LastProbeTime).ToNot(Equal(empty))
					Expect(kv2.Status.Conditions[0].LastProbeTime.Time).To(BeTemporally(">", now.Time))
					Expect(kv2.Status.Conditions[0].LastTransitionTime).To(Equal(empty))
				})
			})
			Context("For unmodified condition", func() {
				It("should keep timestamps", func() {
					kv1.Status.Conditions = append(kv1.Status.Conditions, v1.KubeVirtCondition{
						Type:   v1.KubeVirtConditionAvailable,
						Status: k8sv1.ConditionTrue,
					})
					kv2.Status.Conditions = append(kv2.Status.Conditions, v1.KubeVirtCondition{
						Type:          v1.KubeVirtConditionAvailable,
						Status:        k8sv1.ConditionTrue,
						LastProbeTime: now,
					})
					SetConditionTimestamps(kv1, kv2)
					Expect(kv2.Status.Conditions[0].LastProbeTime).To(Equal(now))
					Expect(kv2.Status.Conditions[0].LastTransitionTime).To(Equal(empty))
				})
			})
			Context("For modified condition", func() {
				It("should update both timestamps", func() {
					now := metav1.Time{
						Time: time.Now(),
					}
					kv1.Status.Conditions = append(kv1.Status.Conditions, v1.KubeVirtCondition{
						Type:   v1.KubeVirtConditionAvailable,
						Status: k8sv1.ConditionTrue,
					})
					kv2.Status.Conditions = append(kv2.Status.Conditions, v1.KubeVirtCondition{
						Type:          v1.KubeVirtConditionAvailable,
						Status:        k8sv1.ConditionFalse,
						LastProbeTime: now,
					})
					SetConditionTimestamps(kv1, kv2)
					Expect(kv2.Status.Conditions[0].LastProbeTime.Time).To(BeTemporally(">", now.Time))
					Expect(kv2.Status.Conditions[0].LastTransitionTime).ToNot(Equal(empty))
					Expect(kv2.Status.Conditions[0].LastTransitionTime).To(Equal(kv2.Status.Conditions[0].LastProbeTime))
				})
			})
		})

	})
})
