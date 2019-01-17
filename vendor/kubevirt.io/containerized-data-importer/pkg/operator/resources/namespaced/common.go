/*
Copyright 2018 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package namespaced

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var commonLabels = map[string]string{
	"cdi.kubevirt.io": "",
}

func withCommonLabels(labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}

	for k, v := range commonLabels {
		_, ok := labels[k]
		if !ok {
			labels[k] = v
		}
	}

	return labels
}

func createServiceAccount(name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: withCommonLabels(nil),
		},
	}
}

func createRoleBinding(name, roleRef, serviceAccount, serviceAccountNamespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: withCommonLabels(nil),
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     roleRef,
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: serviceAccountNamespace,
			},
		},
	}
}

func createRole(name string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: withCommonLabels(nil),
		},
	}
}

func createDeployment(name, matchKey, matchValue, serviceAccount string, numReplicas int32) *appsv1.Deployment {
	matchMap := map[string]string{matchKey: matchValue}
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: withCommonLabels(matchMap),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &numReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					matchKey: matchValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: withCommonLabels(matchMap),
				},
			},
		},
	}
	if serviceAccount != "" {
		deployment.Spec.Template.Spec.ServiceAccountName = serviceAccount
	}
	return deployment
}

func createContainer(name, repo, image, tag, verbosity string, pullPolicy corev1.PullPolicy) corev1.Container {
	return corev1.Container{
		Name:            name,
		Image:           fmt.Sprintf("%s/%s:%s", repo, image, tag),
		ImagePullPolicy: pullPolicy,
		Args:            []string{"-v=" + verbosity},
	}
}

func createService(name, matchKey, matchValue string) *corev1.Service {
	matchMap := map[string]string{matchKey: matchValue}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: withCommonLabels(matchMap),
		},
		Spec: corev1.ServiceSpec{
			Selector: matchMap,
		},
	}
}
