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
	secv1 "github.com/openshift/api/security/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
)

type OperatorConfig struct {
	IsOnOpenshift                           bool
	ServiceMonitorEnabled                   bool
	PrometheusRulesEnabled                  bool
	ValidatingAdmissionPolicyBindingEnabled bool
	ValidatingAdmissionPolicyEnabled        bool
}

type Stores struct {
	KubeVirtCache                         cache.Store
	ServiceAccountCache                   cache.Store
	ClusterRoleCache                      cache.Store
	ClusterRoleBindingCache               cache.Store
	RoleCache                             cache.Store
	RoleBindingCache                      cache.Store
	OperatorCrdCache                      cache.Store
	ServiceCache                          cache.Store
	DeploymentCache                       cache.Store
	DaemonSetCache                        cache.Store
	ValidationWebhookCache                cache.Store
	MutatingWebhookCache                  cache.Store
	APIServiceCache                       cache.Store
	SCCCache                              cache.Store
	RouteCache                            cache.Store
	InstallStrategyConfigMapCache         cache.Store
	InstallStrategyJobCache               cache.Store
	InfrastructurePodCache                cache.Store
	PodDisruptionBudgetCache              cache.Store
	ServiceMonitorCache                   cache.Store
	NamespaceCache                        cache.Store
	PrometheusRuleCache                   cache.Store
	SecretCache                           cache.Store
	ConfigMapCache                        cache.Store
	ValidatingAdmissionPolicyBindingCache cache.Store
	ValidatingAdmissionPolicyCache        cache.Store
	ClusterInstancetype                   cache.Store
	ClusterPreference                     cache.Store
}

func (s *Stores) AllEmpty() bool {
	return IsStoreEmpty(s.ServiceAccountCache) &&
		IsStoreEmpty(s.ClusterRoleCache) &&
		IsStoreEmpty(s.ClusterRoleBindingCache) &&
		IsStoreEmpty(s.RoleCache) &&
		IsStoreEmpty(s.RoleBindingCache) &&
		IsStoreEmpty(s.OperatorCrdCache) &&
		IsStoreEmpty(s.ServiceCache) &&
		IsStoreEmpty(s.DeploymentCache) &&
		IsStoreEmpty(s.DaemonSetCache) &&
		IsStoreEmpty(s.ValidationWebhookCache) &&
		IsStoreEmpty(s.MutatingWebhookCache) &&
		IsStoreEmpty(s.APIServiceCache) &&
		IsStoreEmpty(s.PodDisruptionBudgetCache) &&
		IsSCCStoreEmpty(s.SCCCache) &&
		IsStoreEmpty(s.RouteCache) &&
		IsStoreEmpty(s.ServiceMonitorCache) &&
		IsStoreEmpty(s.PrometheusRuleCache) &&
		IsStoreEmpty(s.SecretCache) &&
		IsStoreEmpty(s.ConfigMapCache) &&
		IsStoreEmpty(s.ValidatingAdmissionPolicyBindingCache) &&
		IsStoreEmpty(s.ValidatingAdmissionPolicyCache)

	// Don't add InstallStrategyConfigMapCache to this list. The install
	// strategies persist even after deletion and updates.
}

func IsStoreEmpty(store cache.Store) bool {
	return len(store.ListKeys()) == 0
}

func IsManagedByOperator(labels map[string]string) bool {
	if v, ok := labels[v1.ManagedByLabel]; ok && (v == v1.ManagedByLabelOperatorValue || v == v1.ManagedByLabelOperatorOldValue) {
		return true
	}
	return false
}

func IsSCCStoreEmpty(store cache.Store) bool {
	cnt := 0
	for _, obj := range store.List() {
		if s, ok := obj.(*secv1.SecurityContextConstraints); ok && IsManagedByOperator(s.GetLabels()) {
			cnt++
		}
	}
	return cnt == 0
}

type Expectations struct {
	ServiceAccount                   *controller.UIDTrackingControllerExpectations
	ClusterRole                      *controller.UIDTrackingControllerExpectations
	ClusterRoleBinding               *controller.UIDTrackingControllerExpectations
	Role                             *controller.UIDTrackingControllerExpectations
	RoleBinding                      *controller.UIDTrackingControllerExpectations
	OperatorCrd                      *controller.UIDTrackingControllerExpectations
	Service                          *controller.UIDTrackingControllerExpectations
	Deployment                       *controller.UIDTrackingControllerExpectations
	DaemonSet                        *controller.UIDTrackingControllerExpectations
	ValidationWebhook                *controller.UIDTrackingControllerExpectations
	MutatingWebhook                  *controller.UIDTrackingControllerExpectations
	APIService                       *controller.UIDTrackingControllerExpectations
	SCC                              *controller.UIDTrackingControllerExpectations
	Route                            *controller.UIDTrackingControllerExpectations
	InstallStrategyConfigMap         *controller.UIDTrackingControllerExpectations
	InstallStrategyJob               *controller.UIDTrackingControllerExpectations
	PodDisruptionBudget              *controller.UIDTrackingControllerExpectations
	ServiceMonitor                   *controller.UIDTrackingControllerExpectations
	PrometheusRule                   *controller.UIDTrackingControllerExpectations
	Secrets                          *controller.UIDTrackingControllerExpectations
	ConfigMap                        *controller.UIDTrackingControllerExpectations
	ValidatingAdmissionPolicyBinding *controller.UIDTrackingControllerExpectations
	ValidatingAdmissionPolicy        *controller.UIDTrackingControllerExpectations
}

type Informers struct {
	KubeVirt                         cache.SharedIndexInformer
	CRD                              cache.SharedIndexInformer
	ServiceAccount                   cache.SharedIndexInformer
	ClusterRole                      cache.SharedIndexInformer
	ClusterRoleBinding               cache.SharedIndexInformer
	Role                             cache.SharedIndexInformer
	RoleBinding                      cache.SharedIndexInformer
	OperatorCrd                      cache.SharedIndexInformer
	Service                          cache.SharedIndexInformer
	Deployment                       cache.SharedIndexInformer
	DaemonSet                        cache.SharedIndexInformer
	ValidationWebhook                cache.SharedIndexInformer
	MutatingWebhook                  cache.SharedIndexInformer
	APIService                       cache.SharedIndexInformer
	SCC                              cache.SharedIndexInformer
	Route                            cache.SharedIndexInformer
	InstallStrategyConfigMap         cache.SharedIndexInformer
	InstallStrategyJob               cache.SharedIndexInformer
	InfrastructurePod                cache.SharedIndexInformer
	PodDisruptionBudget              cache.SharedIndexInformer
	ServiceMonitor                   cache.SharedIndexInformer
	Namespace                        cache.SharedIndexInformer
	PrometheusRule                   cache.SharedIndexInformer
	Secrets                          cache.SharedIndexInformer
	ConfigMap                        cache.SharedIndexInformer
	ValidatingAdmissionPolicyBinding cache.SharedIndexInformer
	ValidatingAdmissionPolicy        cache.SharedIndexInformer
	ClusterInstancetype              cache.SharedIndexInformer
	ClusterPreference                cache.SharedIndexInformer
	Leases                           cache.SharedIndexInformer
}

func (e *Expectations) DeleteExpectations(key string) {
	e.ServiceAccount.DeleteExpectations(key)
	e.ClusterRole.DeleteExpectations(key)
	e.ClusterRoleBinding.DeleteExpectations(key)
	e.Role.DeleteExpectations(key)
	e.RoleBinding.DeleteExpectations(key)
	e.OperatorCrd.DeleteExpectations(key)
	e.Service.DeleteExpectations(key)
	e.Deployment.DeleteExpectations(key)
	e.DaemonSet.DeleteExpectations(key)
	e.ValidationWebhook.DeleteExpectations(key)
	e.MutatingWebhook.DeleteExpectations(key)
	e.APIService.DeleteExpectations(key)
	e.SCC.DeleteExpectations(key)
	e.Route.DeleteExpectations(key)
	e.InstallStrategyConfigMap.DeleteExpectations(key)
	e.InstallStrategyJob.DeleteExpectations(key)
	e.PodDisruptionBudget.DeleteExpectations(key)
	e.ServiceMonitor.DeleteExpectations(key)
	e.PrometheusRule.DeleteExpectations(key)
	e.Secrets.DeleteExpectations(key)
	e.ConfigMap.DeleteExpectations(key)
	e.ValidatingAdmissionPolicyBinding.DeleteExpectations(key)
	e.ValidatingAdmissionPolicy.DeleteExpectations(key)
}

func (e *Expectations) ResetExpectations(key string) {
	e.ServiceAccount.SetExpectations(key, 0, 0)
	e.ClusterRole.SetExpectations(key, 0, 0)
	e.ClusterRoleBinding.SetExpectations(key, 0, 0)
	e.Role.SetExpectations(key, 0, 0)
	e.RoleBinding.SetExpectations(key, 0, 0)
	e.OperatorCrd.SetExpectations(key, 0, 0)
	e.Service.SetExpectations(key, 0, 0)
	e.Deployment.SetExpectations(key, 0, 0)
	e.DaemonSet.SetExpectations(key, 0, 0)
	e.ValidationWebhook.SetExpectations(key, 0, 0)
	e.MutatingWebhook.SetExpectations(key, 0, 0)
	e.APIService.SetExpectations(key, 0, 0)
	e.SCC.SetExpectations(key, 0, 0)
	e.Route.SetExpectations(key, 0, 0)
	e.InstallStrategyConfigMap.SetExpectations(key, 0, 0)
	e.InstallStrategyJob.SetExpectations(key, 0, 0)
	e.PodDisruptionBudget.SetExpectations(key, 0, 0)
	e.ServiceMonitor.SetExpectations(key, 0, 0)
	e.PrometheusRule.SetExpectations(key, 0, 0)
	e.Secrets.SetExpectations(key, 0, 0)
	e.ConfigMap.SetExpectations(key, 0, 0)
	e.ValidatingAdmissionPolicyBinding.SetExpectations(key, 0, 0)
	e.ValidatingAdmissionPolicy.SetExpectations(key, 0, 0)
}

func (e *Expectations) SatisfiedExpectations(key string) bool {
	return e.ServiceAccount.SatisfiedExpectations(key) &&
		e.ClusterRole.SatisfiedExpectations(key) &&
		e.ClusterRoleBinding.SatisfiedExpectations(key) &&
		e.Role.SatisfiedExpectations(key) &&
		e.RoleBinding.SatisfiedExpectations(key) &&
		e.OperatorCrd.SatisfiedExpectations(key) &&
		e.Service.SatisfiedExpectations(key) &&
		e.Deployment.SatisfiedExpectations(key) &&
		e.DaemonSet.SatisfiedExpectations(key) &&
		e.ValidationWebhook.SatisfiedExpectations(key) &&
		e.MutatingWebhook.SatisfiedExpectations(key) &&
		e.APIService.SatisfiedExpectations(key) &&
		e.SCC.SatisfiedExpectations(key) &&
		e.Route.SatisfiedExpectations(key) &&
		e.InstallStrategyConfigMap.SatisfiedExpectations(key) &&
		e.InstallStrategyJob.SatisfiedExpectations(key) &&
		e.PodDisruptionBudget.SatisfiedExpectations(key) &&
		e.ServiceMonitor.SatisfiedExpectations(key) &&
		e.PrometheusRule.SatisfiedExpectations(key) &&
		e.Secrets.SatisfiedExpectations(key) &&
		e.ConfigMap.SatisfiedExpectations(key) &&
		e.ValidatingAdmissionPolicyBinding.SatisfiedExpectations(key) &&
		e.ValidatingAdmissionPolicy.SatisfiedExpectations(key)
}
