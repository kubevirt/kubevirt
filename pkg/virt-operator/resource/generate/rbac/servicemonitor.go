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

package rbac

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	virtv1 "kubevirt.io/api/core/v1"
)

const MONITOR_SERVICEACCOUNT_NAME = "kubevirt-monitoring"

func GetAllServiceMonitor(namespace string, monitorNamespace string, monitorServiceAccount string) []runtime.Object {
	return []runtime.Object{
		newServiceMonitorRole(namespace),
		newServiceMonitorRoleBinding(namespace, monitorNamespace, monitorServiceAccount),
	}
}

func newServiceMonitorRole(namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      MONITOR_SERVICEACCOUNT_NAME,
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
					"services",
					"endpoints",
					"pods",
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
		},
	}
}

func newServiceMonitorRoleBinding(namespace string, monitorNamespace string, monitorServiceAccount string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      MONITOR_SERVICEACCOUNT_NAME,
			Labels: map[string]string{
				virtv1.AppLabel: "",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     MONITOR_SERVICEACCOUNT_NAME,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: monitorNamespace,
				Name:      monitorServiceAccount,
			},
		},
	}
}
