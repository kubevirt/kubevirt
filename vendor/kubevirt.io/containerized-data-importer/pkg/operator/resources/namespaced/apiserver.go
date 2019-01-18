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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	"kubevirt.io/containerized-data-importer/pkg/common"
)

const (
	cdiLabel                  = common.CDIComponentLabel
	apiServerRessouceName     = "cdi-apiserver"
	extensionAPIResourceName  = "cdi-extension-apiserver-authentication"
	extensionAPIConfigMapName = "extension-apiserver-authentication"
)

func createAPIServerResources(args *FactoryArgs) []runtime.Object {
	return []runtime.Object{
		createAPIServerServiceAccount(),
		createAPIServerRoleBinding(args.Namespace),
		createAPIServerRole(),
		createExtensionAPIServerRoleBinding(args.Namespace),
		createExtensionAPIServerRole(),
		createAPIServerService(),
		createAPIServerDeployment(args.DockerRepo, args.APIServerImage, args.DockerTag, args.Verbosity, args.PullPolicy),
	}
}

func createAPIServerServiceAccount() *corev1.ServiceAccount {
	return createServiceAccount(apiServerRessouceName)
}

func createAPIServerRoleBinding(serviceAccountNamespace string) *rbacv1.RoleBinding {
	return createRoleBinding(apiServerRessouceName, apiServerRessouceName, apiServerRessouceName, serviceAccountNamespace)
}

func createAPIServerRole() *rbacv1.Role {
	role := createRole(apiServerRessouceName)
	role.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"",
			},
			Resources: []string{
				"secrets",
				"configmaps",
			},
			Verbs: []string{
				"get",
				"create",
			},
		},
	}
	return role
}

func createExtensionAPIServerRoleBinding(serviceAccountNamespace string) *rbacv1.RoleBinding {
	roleBinding := createRoleBinding(
		extensionAPIResourceName,
		extensionAPIResourceName,
		apiServerRessouceName,
		serviceAccountNamespace,
	)
	roleBinding.Namespace = "kube-system"
	return roleBinding
}

func createExtensionAPIServerRole() *rbacv1.Role {
	role := createRole(extensionAPIResourceName)
	role.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"",
			},
			Resources: []string{
				"configmaps",
			},
			Verbs: []string{
				"get",
			},
			ResourceNames: []string{
				extensionAPIConfigMapName,
			},
		},
	}
	role.Namespace = "kube-system"
	return role
}

func createAPIServerService() *corev1.Service {
	service := createService("cdi-api", cdiLabel, apiServerRessouceName)
	service.Spec.Ports = []corev1.ServicePort{
		{
			Port: 443,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 8443,
			},
			Protocol: corev1.ProtocolTCP,
		},
	}
	return service
}

func createAPIServerDeployment(repo, image, tag, verbosity, pullPolicy string) *appsv1.Deployment {
	deployment := createDeployment(apiServerRessouceName, cdiLabel, apiServerRessouceName, apiServerRessouceName, 1)
	container := createContainer(apiServerRessouceName, repo, image, tag, verbosity, corev1.PullPolicy(pullPolicy))
	deployment.Spec.Template.Spec.Containers = []corev1.Container{container}
	return deployment
}
