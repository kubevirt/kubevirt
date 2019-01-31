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
	"fmt"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

func CreateControllerRBAC(clientset kubecli.KubevirtClient, kv *virtv1.KubeVirt, stores util.Stores, expectations *util.Expectations) (int, error) {

	objectsAdded := 0
	core := clientset.CoreV1()
	kvkey, err := controller.KeyFunc(kv)
	if err != nil {
		return 0, err
	}

	sa := newControllerServiceAccount(kv.Namespace)
	if _, exists, _ := stores.ServiceAccountCache.Get(sa); !exists {
		expectations.ServiceAccount.RaiseExpectations(kvkey, 1, 0)
		_, err := core.ServiceAccounts(kv.Namespace).Create(sa)
		if err != nil {
			expectations.ServiceAccount.LowerExpectations(kvkey, 1, 0)
			return objectsAdded, fmt.Errorf("unable to create serviceaccount %+v: %v", sa, err)
		} else if err == nil {
			objectsAdded++
		}
	} else {
		log.Log.V(4).Infof("serviceaccount %v already exists", sa.GetName())
	}

	rbac := clientset.RbacV1()

	cr := newControllerClusterRole()
	if _, exists, _ := stores.ClusterRoleCache.Get(cr); !exists {
		expectations.ClusterRole.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.ClusterRoles().Create(cr)
		if err != nil {
			expectations.ClusterRole.LowerExpectations(kvkey, 1, 0)
			return objectsAdded, fmt.Errorf("unable to create clusterrole %+v: %v", cr, err)
		} else if err == nil {
			objectsAdded++
		}
	} else {
		log.Log.V(4).Infof("clusterrole %v already exists", cr.GetName())
	}

	crb := newControllerClusterRoleBinding(kv.Namespace)
	if _, exists, _ := stores.ClusterRoleBindingCache.Get(crb); !exists {
		expectations.ClusterRoleBinding.RaiseExpectations(kvkey, 1, 0)
		_, err := rbac.ClusterRoleBindings().Create(crb)
		if err != nil {
			expectations.ClusterRoleBinding.LowerExpectations(kvkey, 1, 0)
			return objectsAdded, fmt.Errorf("unable to create clusterrolebinding %+v: %v", crb, err)
		} else if err == nil {
			objectsAdded++
		}
	} else {
		log.Log.V(4).Infof("clusterrolebinding %v already exists", crb.GetName())
	}

	return objectsAdded, nil
}

func GetAllController(namespace string) []interface{} {
	return []interface{}{
		newControllerServiceAccount(namespace),
		newControllerClusterRole(),
		newControllerClusterRoleBinding(namespace),
	}
}

func newControllerServiceAccount(namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "kubevirt-controller",
			Labels: map[string]string{
				virtv1.AppLabel:       "",
				virtv1.ManagedByLabel: virtv1.ManagedByLabelOperatorValue,
			},
		},
	}
}

func newControllerClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-controller",
			Labels: map[string]string{
				virtv1.AppLabel:       "",
				virtv1.ManagedByLabel: virtv1.ManagedByLabelOperatorValue,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods", "configmaps", "endpoints",
				},
				Verbs: []string{
					"get", "list", "watch", "delete", "update", "create",
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
					"update", "create", "patch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods/finalizers",
				},
				Verbs: []string{
					"update",
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
					"get", "list", "watch", "update", "patch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"persistentvolumeclaims",
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
			{
				APIGroups: []string{
					"kubevirt.io",
				},
				Resources: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"cdi.kubevirt.io",
				},
				Resources: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"k8s.cni.cncf.io",
				},
				Resources: []string{
					"network-attachment-definitions",
				},
				Verbs: []string{
					"get", "list", "watch",
				},
			},
		},
	}
}

func newControllerClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-controller",
			Labels: map[string]string{
				virtv1.AppLabel:       "",
				virtv1.ManagedByLabel: virtv1.ManagedByLabelOperatorValue,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "kubevirt-controller",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      "kubevirt-controller",
			},
		},
	}
}
