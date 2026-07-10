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
		}
	})

	storeWith := func(obj interface{}) cache.Store {
		s := cache.NewStore(cache.MetaNamespaceKeyFunc)
		if obj != nil {
			Expect(s.Add(obj)).To(Succeed())
		}
		return s
	}

	progressingCondition := func(status corev1.ConditionStatus, reason string) appsv1.DeploymentCondition {
		return appsv1.DeploymentCondition{
			Type:   appsv1.DeploymentProgressing,
			Status: status,
			Reason: reason,
		}
	}

	availableCondition := func(status corev1.ConditionStatus) appsv1.DeploymentCondition {
		return appsv1.DeploymentCondition{
			Type:   appsv1.DeploymentAvailable,
			Status: status,
		}
	}

	readyConditions := []appsv1.DeploymentCondition{
		progressingCondition(corev1.ConditionTrue, "NewReplicaSetAvailable"),
		availableCondition(corev1.ConditionTrue),
	}

	DescribeTable("condition and version checks",
		func(conditions []appsv1.DeploymentCondition, annotations map[string]string, inCache bool, expectedReady bool) {
			deployment.Status.Conditions = conditions
			if annotations != nil {
				deployment.Annotations = annotations
			}
			var store cache.Store
			if inCache {
				store = storeWith(deployment)
			} else {
				store = storeWith(nil)
			}
			stores := Stores{DeploymentCache: store}
			Expect(DeploymentIsReady(kv, deployment, stores)).To(Equal(expectedReady))
		},
		Entry("not in cache",
			nil, nil, false, false),
		Entry("no conditions",
			nil, nil, true, false),
		Entry("Progressing=False/ProgressDeadlineExceeded",
			[]appsv1.DeploymentCondition{progressingCondition(corev1.ConditionFalse, "ProgressDeadlineExceeded")}, nil, true, false),
		Entry("Progressing=True/ReplicaSetUpdated (rollout in progress)",
			[]appsv1.DeploymentCondition{progressingCondition(corev1.ConditionTrue, "ReplicaSetUpdated")}, nil, true, false),
		Entry("rollout complete and available",
			readyConditions, nil, true, true),
		Entry("rollout complete but pods crash-looping (Available=False)",
			[]appsv1.DeploymentCondition{
				progressingCondition(corev1.ConditionTrue, "NewReplicaSetAvailable"),
				availableCondition(corev1.ConditionFalse),
			}, nil, true, false),
		Entry("rollout complete but Available condition absent",
			[]appsv1.DeploymentCondition{progressingCondition(corev1.ConditionTrue, "NewReplicaSetAvailable")}, nil, true, false),
		Entry("Available=True but no Progressing condition",
			[]appsv1.DeploymentCondition{availableCondition(corev1.ConditionTrue)}, nil, true, false),
		Entry("rollout complete but deployment at wrong version",
			readyConditions,
			map[string]string{
				v1.InstallStrategyVersionAnnotation:    "0.9.9",
				v1.InstallStrategyRegistryAnnotation:   targetRegistry,
				v1.InstallStrategyIdentifierAnnotation: targetID,
			},
			true, false),
	)

	It("returns false when conditions are stale (ObservedGeneration < Generation)", func() {
		deployment.Generation = 2
		deployment.Status.ObservedGeneration = 1
		deployment.Status.Conditions = readyConditions
		store := storeWith(deployment)
		Expect(DeploymentIsReady(kv, deployment, Stores{DeploymentCache: store})).To(BeFalse())
	})
})
