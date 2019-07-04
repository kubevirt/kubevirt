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

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
)

type Versions struct {
	// the KubeVirt version
	// matches the image tag, if tags are used, either by the manifest, or by the KubeVirt CR
	// used on the KubeVirt CR status and on annotations, and for determing up-/downgrade path, even when using shasums for the images
	kubeVirtVersion string

	// the shasums of every image we use
	virtOperatorSha   string
	virtApiSha        string
	virtControllerSha string
	virtHandlerSha    string
	virtLauncherSha   string
}

func NewVersionsWithTag(tag string) *Versions {
	return &Versions{
		kubeVirtVersion: tag,
	}
}

func NewVersionsWithShasums(kubeVirtVersion string, operatorSha string, apiSha string, controllerSha string, handlerSha string, launcherSha string) *Versions {
	return &Versions{
		kubeVirtVersion:   kubeVirtVersion,
		virtOperatorSha:   operatorSha,
		virtApiSha:        apiSha,
		virtControllerSha: controllerSha,
		virtHandlerSha:    handlerSha,
		virtLauncherSha:   launcherSha,
	}
}

func (v *Versions) GetOperatorVersion() string {
	if v.UseShasums() {
		return v.virtOperatorSha
	}
	return v.kubeVirtVersion
}

func (v *Versions) GetApiVersion() string {
	if v.UseShasums() {
		return v.virtApiSha
	}
	return v.kubeVirtVersion
}

func (v *Versions) GetControllerVersion() string {
	if v.UseShasums() {
		return v.virtControllerSha
	}
	return v.kubeVirtVersion
}

func (v *Versions) GetHandlerVersion() string {
	if v.UseShasums() {
		return v.virtHandlerSha
	}
	return v.kubeVirtVersion
}

func (v *Versions) GetLauncherVersion() string {
	if v.UseShasums() {
		return v.virtLauncherSha
	}
	return v.kubeVirtVersion
}

func (v *Versions) GetKubeVirtVersion() string {
	return v.kubeVirtVersion
}

func (v *Versions) UseShasums() bool {
	return v.virtOperatorSha != "" && v.virtApiSha != "" && v.virtControllerSha != "" && v.virtHandlerSha != "" && v.virtLauncherSha != ""
}

func (v *Versions) SetTargetVersion(kv *v1.KubeVirt) {
	kv.Status.TargetKubeVirtVersion = v.GetKubeVirtVersion()
	if v.UseShasums() {
		kv.Status.TargetVirtApiSha = v.GetApiVersion()
		kv.Status.TargetVirtControllerSha = v.GetControllerVersion()
		kv.Status.TargetVirtHandlerSha = v.GetHandlerVersion()
		kv.Status.TargetVirtLauncherSha = v.GetLauncherVersion()
	} else {
		kv.Status.TargetVirtApiSha = ""
		kv.Status.TargetVirtControllerSha = ""
		kv.Status.TargetVirtHandlerSha = ""
		kv.Status.TargetVirtLauncherSha = ""
	}
}

func (v *Versions) SetObservedVersion(kv *v1.KubeVirt) {
	kv.Status.ObservedKubeVirtVersion = v.GetKubeVirtVersion()
	if v.UseShasums() {
		kv.Status.ObservedVirtApiSha = v.GetApiVersion()
		kv.Status.ObservedVirtControllerSha = v.GetControllerVersion()
		kv.Status.ObservedVirtHandlerSha = v.GetHandlerVersion()
	} else {
		kv.Status.ObservedVirtApiSha = ""
		kv.Status.ObservedVirtControllerSha = ""
		kv.Status.ObservedVirtHandlerSha = ""
	}
}

type Stores struct {
	ServiceAccountCache           cache.Store
	ClusterRoleCache              cache.Store
	ClusterRoleBindingCache       cache.Store
	RoleCache                     cache.Store
	RoleBindingCache              cache.Store
	CrdCache                      cache.Store
	ServiceCache                  cache.Store
	DeploymentCache               cache.Store
	DaemonSetCache                cache.Store
	ValidationWebhookCache        cache.Store
	SCCCache                      cache.Store
	InstallStrategyConfigMapCache cache.Store
	InstallStrategyJobCache       cache.Store
	InfrastructurePodCache        cache.Store
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
		IsStoreEmpty(s.DaemonSetCache) &&
		IsStoreEmpty(s.ValidationWebhookCache)
	// Don't add InstallStrategyConfigMapCache to this list. The install
	// strategies persist even after deletion and updates.
}

func IsStoreEmpty(store cache.Store) bool {
	return len(store.ListKeys()) == 0
}

type Expectations struct {
	ServiceAccount           *controller.UIDTrackingControllerExpectations
	ClusterRole              *controller.UIDTrackingControllerExpectations
	ClusterRoleBinding       *controller.UIDTrackingControllerExpectations
	Role                     *controller.UIDTrackingControllerExpectations
	RoleBinding              *controller.UIDTrackingControllerExpectations
	Crd                      *controller.UIDTrackingControllerExpectations
	Service                  *controller.UIDTrackingControllerExpectations
	Deployment               *controller.UIDTrackingControllerExpectations
	DaemonSet                *controller.UIDTrackingControllerExpectations
	ValidationWebhook        *controller.UIDTrackingControllerExpectations
	InstallStrategyConfigMap *controller.UIDTrackingControllerExpectations
	InstallStrategyJob       *controller.UIDTrackingControllerExpectations
}

type Informers struct {
	ServiceAccount           cache.SharedIndexInformer
	ClusterRole              cache.SharedIndexInformer
	ClusterRoleBinding       cache.SharedIndexInformer
	Role                     cache.SharedIndexInformer
	RoleBinding              cache.SharedIndexInformer
	Crd                      cache.SharedIndexInformer
	Service                  cache.SharedIndexInformer
	Deployment               cache.SharedIndexInformer
	DaemonSet                cache.SharedIndexInformer
	ValidationWebhook        cache.SharedIndexInformer
	SCC                      cache.SharedIndexInformer
	InstallStrategyConfigMap cache.SharedIndexInformer
	InstallStrategyJob       cache.SharedIndexInformer
	InfrastructurePod        cache.SharedIndexInformer
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
	e.ValidationWebhook.DeleteExpectations(key)
	e.InstallStrategyConfigMap.DeleteExpectations(key)
	e.InstallStrategyJob.DeleteExpectations(key)
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
	e.ValidationWebhook.SetExpectations(key, 0, 0)
	e.InstallStrategyConfigMap.SetExpectations(key, 0, 0)
	e.InstallStrategyJob.SetExpectations(key, 0, 0)
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
		e.DaemonSet.SatisfiedExpectations(key) &&
		e.ValidationWebhook.SatisfiedExpectations(key) &&
		e.InstallStrategyConfigMap.SatisfiedExpectations(key) &&
		e.InstallStrategyJob.SatisfiedExpectations(key)
}
