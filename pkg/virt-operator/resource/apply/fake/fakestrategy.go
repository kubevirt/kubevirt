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

package fake

import (
	routev1 "github.com/openshift/api/route/v1"
	secv1 "github.com/openshift/api/security/v1"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

type FakeStrategy struct {
	FakeInstancetypes []*instancetypev1beta1.VirtualMachineClusterInstancetype
	FakePreferences   []*instancetypev1beta1.VirtualMachineClusterPreference
}

func (ins *FakeStrategy) ServiceAccounts() []*corev1.ServiceAccount {
	return nil
}

func (ins *FakeStrategy) ClusterRoles() []*rbacv1.ClusterRole {
	return nil
}

func (ins *FakeStrategy) ClusterRoleBindings() []*rbacv1.ClusterRoleBinding {
	return nil
}

func (ins *FakeStrategy) Roles() []*rbacv1.Role {
	return nil
}

func (ins *FakeStrategy) RoleBindings() []*rbacv1.RoleBinding {
	return nil
}

func (ins *FakeStrategy) Services() []*corev1.Service {
	return nil
}

func (ins *FakeStrategy) Deployments() []*appsv1.Deployment {
	return nil
}

func (ins *FakeStrategy) ApiDeployments() []*appsv1.Deployment {
	return nil
}

func (ins *FakeStrategy) ControllerDeployments() []*appsv1.Deployment {
	return nil
}

func (ins *FakeStrategy) ExportProxyDeployments() []*appsv1.Deployment {
	return nil
}

func (ins *FakeStrategy) DaemonSets() []*appsv1.DaemonSet {
	return nil
}

func (ins *FakeStrategy) ValidatingWebhookConfigurations() []*admissionregistrationv1.ValidatingWebhookConfiguration {
	return nil
}

func (ins *FakeStrategy) MutatingWebhookConfigurations() []*admissionregistrationv1.MutatingWebhookConfiguration {
	return nil
}

func (ins *FakeStrategy) APIServices() []*apiregv1.APIService {
	return nil
}

func (ins *FakeStrategy) CertificateSecrets() []*corev1.Secret {
	return nil
}

func (ins *FakeStrategy) SCCs() []*secv1.SecurityContextConstraints {
	return nil
}

func (ins *FakeStrategy) ServiceMonitors() []*promv1.ServiceMonitor {
	return nil
}

func (ins *FakeStrategy) PrometheusRules() []*promv1.PrometheusRule {
	return nil
}

func (ins *FakeStrategy) ConfigMaps() []*corev1.ConfigMap {
	return nil
}

func (ins *FakeStrategy) CRDs() []*extv1.CustomResourceDefinition {
	return nil
}

func (ins *FakeStrategy) Routes() []*routev1.Route {
	return nil
}

func (ins *FakeStrategy) Instancetypes() []*instancetypev1beta1.VirtualMachineClusterInstancetype {
	return ins.FakeInstancetypes
}

func (ins *FakeStrategy) Preferences() []*instancetypev1beta1.VirtualMachineClusterPreference {
	return ins.FakePreferences
}

func (ins *FakeStrategy) ValidatingAdmissionPolicyBindings() []*admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return nil
}

func (ins *FakeStrategy) ValidatingAdmissionPolicies() []*admissionregistrationv1.ValidatingAdmissionPolicy {
	return nil
}
