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
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/controller"
)

type Stores struct {
	ServiceAccountCache     cache.Store
	ClusterRoleCache        cache.Store
	ClusterRoleBindingCache cache.Store
	RoleCache               cache.Store
	RoleBindingCache        cache.Store
	CrdCache                cache.Store
	ServiceCache            cache.Store
	DeploymentCache         cache.Store
	DaemonSetCache          cache.Store
}

func (s *Stores) AllEmpty() bool {
	return IsStoreEmpty(s.ServiceAccountCache) &&
		IsStoreEmpty(s.ClusterRoleCache) &&
		IsStoreEmpty(s.ClusterRoleBindingCache) &&
		IsStoreEmpty(s.RoleCache) &&
		IsStoreEmpty(s.RoleBindingCache) &&
		IsStoreEmpty(s.CrdCache) &&
		IsStoreEmpty(s.ServiceCache) &&
		IsStoreEmpty(s.DeploymentCache) &&
		IsStoreEmpty(s.DaemonSetCache)
}

func IsStoreEmpty(store cache.Store) bool {
	return len(store.ListKeys()) == 0
}

type Expectations struct {
	ServiceAccount     *controller.UIDTrackingControllerExpectations
	ClusterRole        *controller.UIDTrackingControllerExpectations
	ClusterRoleBinding *controller.UIDTrackingControllerExpectations
	Role               *controller.UIDTrackingControllerExpectations
	RoleBinding        *controller.UIDTrackingControllerExpectations
	Crd                *controller.UIDTrackingControllerExpectations
	Service            *controller.UIDTrackingControllerExpectations
	Deployment         *controller.UIDTrackingControllerExpectations
	DaemonSet          *controller.UIDTrackingControllerExpectations
}

type Informers struct {
	ServiceAccount     cache.SharedIndexInformer
	ClusterRole        cache.SharedIndexInformer
	ClusterRoleBinding cache.SharedIndexInformer
	Role               cache.SharedIndexInformer
	RoleBinding        cache.SharedIndexInformer
	Crd                cache.SharedIndexInformer
	Service            cache.SharedIndexInformer
	Deployment         cache.SharedIndexInformer
	DaemonSet          cache.SharedIndexInformer
}

func (e *Expectations) DeleteExpectations(key string) {
	e.ServiceAccount.DeleteExpectations(key)
	e.ClusterRole.DeleteExpectations(key)
	e.ClusterRoleBinding.DeleteExpectations(key)
	e.Role.DeleteExpectations(key)
	e.RoleBinding.DeleteExpectations(key)
	e.Crd.DeleteExpectations(key)
	e.Service.DeleteExpectations(key)
	e.Deployment.DeleteExpectations(key)
	e.DaemonSet.DeleteExpectations(key)
}

func (e *Expectations) ResetExpectations(key string) {
	e.ServiceAccount.SetExpectations(key, 0, 0)
	e.ClusterRole.SetExpectations(key, 0, 0)
	e.ClusterRoleBinding.SetExpectations(key, 0, 0)
	e.Role.SetExpectations(key, 0, 0)
	e.RoleBinding.SetExpectations(key, 0, 0)
	e.Crd.SetExpectations(key, 0, 0)
	e.Service.SetExpectations(key, 0, 0)
	e.Deployment.SetExpectations(key, 0, 0)
	e.DaemonSet.SetExpectations(key, 0, 0)
}

func (e *Expectations) SatisfiedExpectations(key string) bool {
	return e.ServiceAccount.SatisfiedExpectations(key) &&
		e.ClusterRole.SatisfiedExpectations(key) &&
		e.ClusterRoleBinding.SatisfiedExpectations(key) &&
		e.Role.SatisfiedExpectations(key) &&
		e.RoleBinding.SatisfiedExpectations(key) &&
		e.Crd.SatisfiedExpectations(key) &&
		e.Service.SatisfiedExpectations(key) &&
		e.Deployment.SatisfiedExpectations(key) &&
		e.DaemonSet.SatisfiedExpectations(key)
}
