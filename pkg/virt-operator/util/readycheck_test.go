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

package util

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("DaemonsetIsReady", func() {
	const (
		targetVersion  = "1.0.0"
		targetRegistry = "registry.example.com"
		targetID       = "abc123"
	)

	var (
		kv        *v1.KubeVirt
		daemonset *appsv1.DaemonSet
	)

	BeforeEach(func() {
		kv = &v1.KubeVirt{
			Status: v1.KubeVirtStatus{
				TargetKubeVirtVersion:  targetVersion,
				TargetKubeVirtRegistry: targetRegistry,
				TargetDeploymentID:     targetID,
			},
		}
		daemonset = &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "virt-handler",
				Namespace: "kubevirt",
				Annotations: map[string]string{
					v1.InstallStrategyVersionAnnotation:    targetVersion,
					v1.InstallStrategyRegistryAnnotation:   targetRegistry,
					v1.InstallStrategyIdentifierAnnotation: targetID,
				},
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				UpdatedNumberScheduled: 3,
				NumberReady:            3,
				NumberAvailable:        3,
			},
		}
	})

	storeWith := func(obj interface{}) cache.Store {
		s := cache.NewStore(cache.MetaNamespaceKeyFunc)
		if obj != nil {
			Expect(s.Add(obj)).To(Succeed())
		}
		return s
	}

	stores := func(ds *appsv1.DaemonSet) Stores {
		return Stores{DaemonSetCache: storeWith(ds)}
	}

	DescribeTable("status and version checks",
		func(mutate func(*appsv1.DaemonSet), expectedReady bool) {
			if mutate != nil {
				mutate(daemonset)
			}
			Expect(DaemonsetIsReady(kv, daemonset, stores(daemonset))).To(Equal(expectedReady))
		},
		Entry("all conditions met",
			nil, true),
		Entry("annotations behind target version",
			func(ds *appsv1.DaemonSet) {
				ds.Annotations[v1.InstallStrategyVersionAnnotation] = "0.9.9"
			}, false),
		Entry("annotations behind target registry",
			func(ds *appsv1.DaemonSet) {
				ds.Annotations[v1.InstallStrategyRegistryAnnotation] = "wrong.registry.io"
			}, false),
		Entry("annotations behind target identifier",
			func(ds *appsv1.DaemonSet) {
				ds.Annotations[v1.InstallStrategyIdentifierAnnotation] = "wrong-id"
			}, false),
		Entry("ObservedGeneration behind Generation (controller not yet reconciled)",
			func(ds *appsv1.DaemonSet) {
				ds.Generation = 5
				ds.Status.ObservedGeneration = 4
			}, false),
		Entry("DesiredNumberScheduled is zero (no schedulable nodes)",
			func(ds *appsv1.DaemonSet) {
				ds.Status.DesiredNumberScheduled = 0
				ds.Status.UpdatedNumberScheduled = 0
				ds.Status.NumberReady = 0
				ds.Status.NumberAvailable = 0
			}, false),
		Entry("UpdatedNumberScheduled behind (rollout in progress)",
			func(ds *appsv1.DaemonSet) {
				ds.Status.UpdatedNumberScheduled = 2
			}, false),
		Entry("NumberAvailable behind (pods not yet available)",
			func(ds *appsv1.DaemonSet) {
				ds.Status.NumberAvailable = 2
			}, false),
	)

	It("returns false when DaemonSet is not in cache", func() {
		emptyStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
		Expect(DaemonsetIsReady(kv, daemonset, Stores{DaemonSetCache: emptyStore})).To(BeFalse())
	})
})

var _ = Describe("DeploymentIsReady", func() {
	const (
		targetVersion  = "1.0.0"
		targetRegistry = "registry.example.com"
		targetID       = "abc123"
	)

	var (
		kv         *v1.KubeVirt
		deployment *appsv1.Deployment
	)

	BeforeEach(func() {
		kv = &v1.KubeVirt{
			Status: v1.KubeVirtStatus{
				TargetKubeVirtVersion:  targetVersion,
				TargetKubeVirtRegistry: targetRegistry,
				TargetDeploymentID:     targetID,
			},
		}
		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "virt-api",
				Namespace: "kubevirt",
				Annotations: map[string]string{
					v1.InstallStrategyVersionAnnotation:    targetVersion,
					v1.InstallStrategyRegistryAnnotation:   targetRegistry,
					v1.InstallStrategyIdentifierAnnotation: targetID,
				},
			},
			Status: appsv1.DeploymentStatus{
				Conditions: []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable"},
					{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
				},
			},
		}
	})

	storeWith := func(obj interface{}) cache.Store {
		s := cache.NewStore(cache.MetaNamespaceKeyFunc)
		if obj != nil {
			Expect(s.Add(obj)).To(Succeed())
		}
		return s
	}

	stores := func(dep *appsv1.Deployment) Stores {
		return Stores{DeploymentCache: storeWith(dep)}
	}

	DescribeTable("condition and version checks",
		func(mutate func(*appsv1.Deployment), inCache bool, expectedReady bool) {
			if mutate != nil {
				mutate(deployment)
			}
			var s Stores
			if inCache {
				s = stores(deployment)
			} else {
				s = Stores{DeploymentCache: storeWith(nil)}
			}
			Expect(DeploymentIsReady(kv, deployment, s)).To(Equal(expectedReady))
		},
		Entry("not in cache",
			nil, false, false),
		Entry("all conditions met",
			nil, true, true),
		Entry("no conditions",
			func(d *appsv1.Deployment) {
				d.Status.Conditions = nil
			}, true, false),
		Entry("Progressing=False/ProgressDeadlineExceeded",
			func(d *appsv1.Deployment) {
				d.Status.Conditions = []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: "ProgressDeadlineExceeded"},
				}
			}, true, false),
		Entry("Progressing=True/ReplicaSetUpdated (rollout in progress)",
			func(d *appsv1.Deployment) {
				d.Status.Conditions = []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "ReplicaSetUpdated"},
				}
			}, true, false),
		Entry("rollout complete but pods crash-looping (Available=False)",
			func(d *appsv1.Deployment) {
				d.Status.Conditions = []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable"},
					{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionFalse},
				}
			}, true, false),
		Entry("rollout complete but Available condition absent",
			func(d *appsv1.Deployment) {
				d.Status.Conditions = []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable"},
				}
			}, true, false),
		Entry("Available=True but no Progressing condition",
			func(d *appsv1.Deployment) {
				d.Status.Conditions = []appsv1.DeploymentCondition{
					{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
				}
			}, true, false),
		Entry("rollout complete but deployment at wrong version",
			func(d *appsv1.Deployment) {
				d.Annotations[v1.InstallStrategyVersionAnnotation] = "0.9.9"
			}, true, false),
		Entry("rollout complete but deployment at wrong registry",
			func(d *appsv1.Deployment) {
				d.Annotations[v1.InstallStrategyRegistryAnnotation] = "wrong.registry.io"
			}, true, false),
		Entry("rollout complete but deployment at wrong identifier",
			func(d *appsv1.Deployment) {
				d.Annotations[v1.InstallStrategyIdentifierAnnotation] = "wrong-id"
			}, true, false),
		Entry("ObservedGeneration behind Generation (controller not yet reconciled)",
			func(d *appsv1.Deployment) {
				d.Generation = 2
				d.Status.ObservedGeneration = 1
			}, true, false),
	)
})
