/*
Copyright 2019 The CDI Authors.

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

package cluster

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	uploadProxyResourceName = "cdi-uploadproxy"
)

func createUploadProxyResources(args *FactoryArgs) []runtime.Object {
	return []runtime.Object{
		createUploadProxyClusterRole(),
		createUploadProxyClusterRoleBinding(args.Namespace),
	}
}

//GetUploadProxyRolePermissions generates permissions for operator
func GetUploadProxyRolePermissions() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"",
			},
			Resources: []string{
				"pods",
			},
			Verbs: []string{
				"get",
			},
		},
	}
}

func createUploadProxyClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return CreateClusterRoleBinding(uploadProxyResourceName, uploadProxyResourceName, uploadProxyResourceName, namespace)
}

func createUploadProxyClusterRole() *rbacv1.ClusterRole {
	clusterRole := CreateClusterRole(uploadProxyResourceName)
	clusterRole.Rules = GetUploadProxyRolePermissions()
	return clusterRole
}
