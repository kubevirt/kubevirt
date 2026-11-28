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

var _ = Describe("Deployment rollout reporting", func() {
	var kv *v1.KubeVirt
	var deployment *appsv1.Deployment
	var stores Stores
	var targets []*appsv1.Deployment

	condition := func() v1.KubeVirtCondition {
		for _, c := range kv.Status.Conditions {
			if c.Type == v1.KubeVirtConditionDeploymentsReady {
				return c
			}
		}
		Fail("DeploymentsReady condition missing")
		return v1.KubeVirtCondition{}
	}

	BeforeEach(func() {
		kv = &v1.KubeVirt{Status: v1.KubeVirtStatus{
			TargetKubeVirtVersion: "v1", TargetKubeVirtRegistry: "registry", TargetDeploymentID: "id",
		}}
		UpdateConditionsAvailable(kv)
		replicas := int32(2)
		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "virt-api", Namespace: "kubevirt", Generation: 2},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
			Status: appsv1.DeploymentStatus{
				ObservedGeneration: 2, Replicas: 2, UpdatedReplicas: 2, ReadyReplicas: 2, AvailableReplicas: 2,
			},
		}
		stores = Stores{
			DeploymentCache:        cache.NewStore(cache.MetaNamespaceKeyFunc),
			InfrastructurePodCache: cache.NewStore(cache.MetaNamespaceKeyFunc),
		}
		Expect(stores.DeploymentCache.Add(deployment)).To(Succeed())
		for _, name := range []string{"virt-api-a", "virt-api-b"} {
			Expect(stores.InfrastructurePodCache.Add(&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "kubevirt", Annotations: map[string]string{
					v1.InstallStrategyVersionAnnotation:    "v1",
					v1.InstallStrategyRegistryAnnotation:   "registry",
					v1.InstallStrategyIdentifierAnnotation: "id",
				}},
				Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Ready: true}}},
			})).To(Succeed())
		}
		targets = []*appsv1.Deployment{deployment.DeepCopy()}
	})

	DescribeTable("reports rollout readiness", func(change func(), expected corev1.ConditionStatus) {
		change()
		UpdateConditionsDeploymentsReady(kv, targets, stores)
		Expect(condition().Status).To(Equal(expected))
		// Full rollout reporting must never change service availability semantics.
		Expect(kv.Status.Conditions[0].Type).To(Equal(v1.KubeVirtConditionAvailable))
		Expect(kv.Status.Conditions[0].Status).To(Equal(corev1.ConditionTrue))
	},
		Entry("all desired replicas available", func() {}, corev1.ConditionTrue),
		Entry("waiting for strategy", func() { targets = nil }, corev1.ConditionFalse),
		Entry("missing deployment", func() { Expect(stores.DeploymentCache.Delete(deployment)).To(Succeed()) }, corev1.ConditionFalse),
		Entry("unobserved deployment generation", func() { deployment.Status.ObservedGeneration-- }, corev1.ConditionFalse),
		Entry("partially updated", func() { deployment.Status.UpdatedReplicas-- }, corev1.ConditionFalse),
		Entry("partially ready", func() { deployment.Status.ReadyReplicas-- }, corev1.ConditionFalse),
		Entry("partially available", func() { deployment.Status.AvailableReplicas-- }, corev1.ConditionFalse),
		Entry("old replicas remain", func() { deployment.Status.Replicas++ }, corev1.ConditionFalse),
		Entry("deleting deployment", func() { now := metav1.Now(); deployment.DeletionTimestamp = &now }, corev1.ConditionFalse),
		Entry("old-version pods", func() { kv.Status.TargetDeploymentID = "new-id" }, corev1.ConditionFalse),
		Entry("missing pods", func() { stores.InfrastructurePodCache = cache.NewStore(cache.MetaNamespaceKeyFunc) }, corev1.ConditionFalse),
		Entry("scaled to zero", func() {
			zero := int32(0)
			deployment.Spec.Replicas = &zero
			deployment.Status = appsv1.DeploymentStatus{ObservedGeneration: 2}
		}, corev1.ConditionTrue),
		Entry("default replica count", func() {
			deployment.Spec.Replicas = nil
			deployment.Status = appsv1.DeploymentStatus{
				ObservedGeneration: 2, Replicas: 1, UpdatedReplicas: 1, ReadyReplicas: 1, AvailableReplicas: 1,
			}
		}, corev1.ConditionTrue),
	)

	It("reports regressions and recovery without changing Available", func() {
		UpdateConditionsDeploymentsReady(kv, targets, stores)
		Expect(condition().Status).To(Equal(corev1.ConditionTrue))
		deployment.Status.AvailableReplicas = 1
		UpdateConditionsDeploymentsReady(kv, targets, stores)
		Expect(condition().Status).To(Equal(corev1.ConditionFalse))
		Expect(condition().Message).To(ContainSubstring("kubevirt/virt-api"))
		deployment.Status.AvailableReplicas = 2
		UpdateConditionsDeploymentsReady(kv, targets, stores)
		Expect(condition().Status).To(Equal(corev1.ConditionTrue))
	})

	It("only requires deployments in the enabled target list", func() {
		optional := deployment.DeepCopy()
		optional.Name = "disabled-component"
		optional.Status = appsv1.DeploymentStatus{}
		Expect(stores.DeploymentCache.Add(optional)).To(Succeed())
		UpdateConditionsDeploymentsReady(kv, targets, stores)
		Expect(condition().Status).To(Equal(corev1.ConditionTrue))
		targets = append(targets, optional)
		UpdateConditionsDeploymentsReady(kv, targets, stores)
		Expect(condition().Status).To(Equal(corev1.ConditionFalse))
	})
})
