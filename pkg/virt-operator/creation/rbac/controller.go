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

	pspcli := clientset.PolicyV1beta1()
	psp := newControllerPodSecurityPolicy()
	if _, exists, _ := stores.PodSecurityPolicyCache.Get(psp); !exists {
		expectations.PodSecurityPolicy.RaiseExpectations(kvkey, 1, 0)
		_, err := pspcli.PodSecurityPolicies().Create(psp)
		if err != nil {
			expectations.PodSecurityPolicy.LowerExpectations(kvkey, 1, 0)
			return objectsAdded, fmt.Errorf("unable to create PodSecurityPolicy %+v: %v", psp, err)
		} else if err == nil {
			objectsAdded++
		}
	} else {
		log.Log.V(4).Infof("PodSecurityPolicy %v already exists", psp.GetName())
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
		newControllerPodSecurityPolicy(),
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
				virtv1.AppLabel: "",
			},
		},
	}
}

func newControllerPodSecurityPolicy() *pspv1b1.PodSecurityPolicy {
	return &pspv1b1.PodSecurityPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1beta1",
			Kind:       "PodSecurityPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubevirt-controller-psp",
		},
		Spec: pspv1b1.PodSecurityPolicySpec{
			Privileged: true,
			AllowedCapabilities: []corev1.Capability{
				"NET_ADMIN",
				"SYS_NICE",
			},
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
			Volumes: []pspv1b1.FSType{
				"*",
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
				virtv1.AppLabel: "",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"policy",
				},
				Resources: []string{
					"poddisruptionbudgets",
				},
				Verbs: []string{
					"get", "list", "watch", "delete", "create",
				},
			},
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
			{
				APIGroups: []string{
					"policy",
				},
				Resources: []string{
					"podsecuritypolicies",
				},
				ResourceNames: []string{
					"kubevirt-controller-psp",
				},
				Verbs: []string{
					"use",
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
				virtv1.AppLabel: "",
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
