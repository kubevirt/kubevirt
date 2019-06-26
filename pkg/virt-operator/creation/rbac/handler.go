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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package rbac

import (
	corev1 "k8s.io/api/core/v1"
	pspv1b1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/client-go/api/v1"
)

func GetAllHandler(namespace string) []interface{} {
	return []interface{}{
		newHandlerServiceAccount(namespace),
		newHandlerClusterRole(),
		newHandlerClusterRoleBinding(namespace),
		newHandlerPsp(),
		newHandlerRole(namespace),
		newHandlerRoleBinding(namespace),
	}
}

func newHandlerServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-handler",
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
	}
}

func newHandlerClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-handler",
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"kubevirt.io",
				},
				Resources: []string{
					"virtualmachineinstances",
				},
				Verbs: []string{
					"update", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"secrets", "persistentvolumeclaims",
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"nodes",
				},
				Verbs: []string{
					"patch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"events",
				},
				Verbs: []string{
					"create", "patch",
				},
			},
		},
	}
}

func newHandlerClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-handler",
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "kubevirt-handler",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      "kubevirt-handler",
			},
		},
	}
}

func newHandlerPsp() *pspv1b1.PodSecurityPolicy {
	return &pspv1b1.PodSecurityPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1beta1",
			Kind:       "PodSecurityPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-privileged-psp",
		},
		Spec: pspv1b1.PodSecurityPolicySpec{
			Privileged: true,
			HostPID:    true,
			HostIPC:    true,
			SELinux: pspv1b1.SELinuxStrategyOptions{
				Rule: pspv1b1.SELinuxStrategyRunAsAny,
			},
			RunAsUser: pspv1b1.RunAsUserStrategyOptions{
				Rule: pspv1b1.RunAsUserStrategyRunAsAny,
			},
			SupplementalGroups: pspv1b1.SupplementalGroupsStrategyOptions{
				Rule: pspv1b1.SupplementalGroupsStrategyRunAsAny,
			},
			FSGroup: pspv1b1.FSGroupStrategyOptions{
				Rule: pspv1b1.FSGroupStrategyRunAsAny,
			},
			AllowedHostPaths: []pspv1b1.AllowedHostPath{
				{PathPrefix: "/var/run/kubevirt-libvirt-runtimes"},
				{PathPrefix: "/var/run/kubevirt"},
				{PathPrefix: "/var/run/kubevirt-private"},
				{PathPrefix: "/var/lib/kubelet/device-plugins"},
			},
			Volumes: []pspv1b1.FSType{
				"*",
			},
		},
	}
}

func newHandlerRole(namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-handler",
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"configmaps",
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"secrets",
				},
				Verbs: []string{
					"create",
				},
			},
			{
				APIGroups: []string{
					"policy",
				},
				Resources: []string{
					"podsecuritypolicies",
				},
				ResourceNames: []string{
					"kubevirt-privileged-psp",
				},
				Verbs: []string{
					"use",
				},
			},
		},
	}
}

func newHandlerRoleBinding(namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-handler",
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "kubevirt-handler",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      "kubevirt-handler",
			},
		},
	}
}
