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

package operator

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"kubevirt.io/containerized-data-importer/pkg/common"
	cluster "kubevirt.io/containerized-data-importer/pkg/operator/resources/cluster"
	utils "kubevirt.io/containerized-data-importer/pkg/operator/resources/utils"
)

const (
	operatorServiceAccountName = "cdi-operator"
	operatorClusterRoleName    = "cdi-operator-cluster"
	operatorNamespacedRoleName = "cdi-operator"
	privilegedAccountPrefix    = "system:serviceaccount"
	prometheusLabel            = common.PrometheusLabel
)

func createOperatorResources(args *FactoryArgs) []runtime.Object {
	return []runtime.Object{
		createCDIListCRD(),
		createOperatorServiceAccount(args.Namespace),
		createOperatorClusterRole(operatorClusterRoleName),
		createOperatorClusterRoleBinding(args.Namespace),
		createOperatorLeaderElectionConfigMap(args.Namespace),
		createOperatorDeployment(args.DockerRepo,
			args.Namespace,
			args.DeployClusterResources,
			args.OperatorImage,
			args.ControllerImage,
			args.ImporterImage,
			args.ClonerImage,
			args.APIServerImage,
			args.UploadProxyImage,
			args.UploadServerImage,
			args.DockerTag,
			args.Verbosity,
			args.PullPolicy),
	}
}

func getOperatorClusterRules() *[]rbacv1.PolicyRule {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"rbac.authorization.k8s.io",
			},
			Resources: []string{
				"roles",
				"rolebindings",
				"clusterrolebindings",
				"clusterroles",
			},
			Verbs: []string{
				"*",
			},
		},
		{
			APIGroups: []string{
				"security.openshift.io",
			},
			Resources: []string{
				"securitycontextconstraints",
			},
			Verbs: []string{
				"get",
				"list",
				"watch",
			},
		},
		{
			APIGroups: []string{
				"security.openshift.io",
			},
			Resources: []string{
				"securitycontextconstraints",
			},
			ResourceNames: []string{
				"privileged",
			},
			Verbs: []string{
				"get",
				"patch",
				"update",
			},
		},
		{
			APIGroups: []string{
				"",
			},
			Resources: []string{
				"serviceaccounts",
				"services",
			},
			Verbs: []string{
				"*",
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
				"get",
				"list",
				"watch",
				"update",
				"patch",
			},
		},
		{
			APIGroups: []string{
				"extensions",
			},
			Resources: []string{
				"deployments",
			},
			Verbs: []string{
				"*",
			},
		},
		{
			APIGroups: []string{
				"extensions",
			},
			Resources: []string{
				"ingresses",
			},
			Verbs: []string{
				"get",
				"list",
				"watch",
			},
		},
		{
			APIGroups: []string{
				"",
			},
			Resources: []string{
				"configmaps",
			},
			Verbs: []string{
				"watch",
				"create",
				"delete",
				"get",
				"update",
				"patch",
				"list",
			},
		},
		{
			APIGroups: []string{
				"batch",
			},
			Resources: []string{
				"jobs",
			},
			Verbs: []string{
				"create",
				"delete",
				"get",
				"update",
				"patch",
				"list",
			},
		},
		{
			APIGroups: []string{
				"apiextensions.k8s.io",
			},
			Resources: []string{
				"customresourcedefinitions",
			},
			Verbs: []string{
				"create",
				"delete",
				"get",
				"update",
				"patch",
				"list",
				"watch",
			},
		},
		{
			APIGroups: []string{
				"apps",
			},
			Resources: []string{
				"deployments",
				"daemonstes",
			},
			Verbs: []string{
				"create",
				"get",
				"list",
				"delete",
				"watch",
				"update",
			},
		},
		{
			APIGroups: []string{
				"admissionregistration.k8s.io",
			},
			Resources: []string{
				"validatingwebhookconfigurations",
			},
			Verbs: []string{
				"get",
				"create",
				"update",
			},
		},
		{
			APIGroups: []string{
				"apiregistration.k8s.io",
			},
			Resources: []string{
				"apiservices",
			},
			Verbs: []string{
				"get",
				"list",
				"watch",
				"create",
				"update",
				"patch",
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
				"storage.k8s.io",
			},
			Resources: []string{
				"storageclasses",
			},
			Verbs: []string{
				"get",
				"list",
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
				"create",
				"update",
				"patch",
			},
		},
		{
			APIGroups: []string{
				"",
			},
			Resources: []string{
				"pods",
				"persistentvolumeclaims",
			},
			Verbs: []string{
				"get",
				"list",
				"watch",
				"create",
				"update",
				"patch",
				"delete",
			},
		},
		{
			APIGroups: []string{
				"",
			},
			Resources: []string{
				"persistentvolumeclaims/finalizers",
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
				"services",
			},
			Verbs: []string{
				"get",
				"list",
				"watch",
				"create",
				"delete",
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
				"get",
				"list",
				"watch",
				"create",
			},
		},
		{
			APIGroups: []string{
				"",
			},
			Resources: []string{
				"namespaces",
			},
			Verbs: []string{
				"get",
				"list",
			},
		},
		{
			APIGroups: []string{
				"route.openshift.io",
			},
			Resources: []string{
				"routes",
			},
			Verbs: []string{
				"get",
				"list",
				"watch",
				"create",
				"update",
				"patch",
			},
		},
		{
			APIGroups: []string{
				"route.openshift.io",
			},
			Resources: []string{
				"routes/custom-host",
			},
			Verbs: []string{
				"create",
				"update",
			},
		},
	}

	return &rules
}

func createOperatorClusterRole(roleName string) *rbacv1.ClusterRole {
	clusterRole := cluster.CreateClusterRole(roleName)
	clusterRole.Rules = *getOperatorClusterRules()

	return clusterRole
}

func createOperatorClusterRBAC(args *FactoryArgs) []runtime.Object {
	return []runtime.Object{
		createOperatorServiceAccount(args.Namespace),
		createOperatorClusterRole(operatorClusterRoleName),
		createOperatorClusterRoleBinding(args.Namespace),
	}
}

func createOperatorClusterDeployment(args *FactoryArgs) []runtime.Object {
	return []runtime.Object{
		createOperatorDeployment(args.DockerRepo,
			args.Namespace,
			args.DeployClusterResources,
			args.OperatorImage,
			args.ControllerImage,
			args.ImporterImage,
			args.ClonerImage,
			args.APIServerImage,
			args.UploadProxyImage,
			args.UploadServerImage,
			args.DockerTag,
			args.Verbosity,
			args.PullPolicy)}
}

func createOperatorClusterResources(args *FactoryArgs) []runtime.Object {
	return []runtime.Object{
		createCDIListCRD(),
		createOperatorLeaderElectionConfigMap(args.Namespace),
	}
}

func createOperatorCDIClusterResource(args *FactoryArgs) []runtime.Object {
	return []runtime.Object{
		createCDIListCRD(),
	}
}

func createOperatorConfigMapClusterResource(args *FactoryArgs) []runtime.Object {
	return []runtime.Object{
		createOperatorLeaderElectionConfigMap(args.Namespace),
	}
}

func createOperatorClusterRoleBinding(namespace string) *rbacv1.ClusterRoleBinding {
	return cluster.CreateClusterRoleBinding(operatorServiceAccountName, operatorClusterRoleName, operatorServiceAccountName, namespace)
}

func getOperatorPrivilegedAccounts(args *FactoryArgs) []string {
	return []string{
		fmt.Sprintf("%s:%s:%s", privilegedAccountPrefix, args.Namespace, operatorServiceAccountName),
	}
}

func createOperatorServiceAccount(namespace string) *corev1.ServiceAccount {
	return utils.CreateServiceNamespaceAccount(operatorServiceAccountName, namespace)
}

func createOperatorLeaderElectionConfigMap(namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cdi-operator-leader-election-helper",
			Namespace: namespace,
			Labels: map[string]string{
				"operator.cdi.kubevirt.io": "",
			},
		},
	}

}

func createCDIListCRD() *extv1beta1.CustomResourceDefinition {
	return &extv1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1beta1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cdis.cdi.kubevirt.io",
			Labels: map[string]string{
				"operator.cdi.kubevirt.io": "",
			},
		},
		Spec: extv1beta1.CustomResourceDefinitionSpec{
			Group:   "cdi.kubevirt.io",
			Version: "v1alpha1",
			Scope:   "Cluster",

			Versions: []extv1beta1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
				},
			},
			Names: extv1beta1.CustomResourceDefinitionNames{
				Kind:     "CDI",
				ListKind: "CDIList",
				Plural:   "cdis",
				Singular: "cdi",
				Categories: []string{
					"all",
				},
				ShortNames: []string{"cdi", "cdis"},
			},

			AdditionalPrinterColumns: []extv1beta1.CustomResourceColumnDefinition{
				{Name: "Age", Type: "date", JSONPath: ".metadata.creationTimestamp"},
				{Name: "Phase", Type: "string", JSONPath: ".status.phase"},
			},
		},
	}
}

const (
	uploadProxyResourceName = "cdi-uploadproxy"
)

func createOperatorDeploymentSpec(repo, namespace, deployClusterResources, operatorImage, controllerImage, importerImage, clonerImage, apiServerImage, uploadProxyImage, uploadServerImage, tag, verbosity, pullPolicy string) *appsv1.DeploymentSpec {
	deployment := createOperatorDeployment(repo,
		namespace,
		deployClusterResources,
		operatorImage,
		controllerImage,
		importerImage,
		clonerImage,
		apiServerImage,
		uploadProxyImage,
		uploadServerImage,
		tag,
		verbosity,
		pullPolicy)
	return &deployment.Spec
}

func createOperatorEnvVar(repo, deployClusterResources, operatorImage, controllerImage, importerImage, clonerImage, apiServerImage, uploadProxyImage, uploadServerImage, tag, verbosity, pullPolicy string) *[]corev1.EnvVar {
	return &[]corev1.EnvVar{
		{
			Name:  "DEPLOY_CLUSTER_RESOURCES",
			Value: fmt.Sprintf("%s", deployClusterResources),
		},
		{
			Name:  "DOCKER_REPO",
			Value: fmt.Sprintf("%s", repo),
		},
		{
			Name:  "DOCKER_TAG",
			Value: fmt.Sprintf("%s", tag),
		},
		{
			Name:  "CONTROLLER_IMAGE",
			Value: fmt.Sprintf("%s", controllerImage),
		},
		{
			Name:  "IMPORTER_IMAGE",
			Value: fmt.Sprintf("%s", importerImage),
		},
		{
			Name:  "CLONER_IMAGE",
			Value: fmt.Sprintf("%s", clonerImage),
		},
		{
			Name:  "APISERVER_IMAGE",
			Value: fmt.Sprintf("%s", apiServerImage),
		},
		{
			Name:  "UPLOAD_SERVER_IMAGE",
			Value: fmt.Sprintf("%s", uploadServerImage),
		},
		{
			Name:  "UPLOAD_PROXY_IMAGE",
			Value: fmt.Sprintf("%s", uploadProxyImage),
		},
		{
			Name:  "VERBOSITY",
			Value: verbosity,
		},
		{
			Name:  "PULL_POLICY",
			Value: pullPolicy,
		},
	}
}

func createOperatorDeployment(repo, namespace, deployClusterResources, operatorImage, controllerImage, importerImage, clonerImage, apiServerImage, uploadProxyImage, uploadServerImage, tag, verbosity, pullPolicy string) *appsv1.Deployment {
	deployment := utils.CreateOperatorDeployment("cdi-operator", namespace, "name", "cdi-operator", operatorServiceAccountName, int32(1))
	container := utils.CreatePortsContainer("cdi-operator", repo, operatorImage, tag, verbosity, corev1.PullPolicy(pullPolicy), createPrometheusPorts())
	container.Env = *createOperatorEnvVar(repo, deployClusterResources, operatorImage, controllerImage, importerImage, clonerImage, apiServerImage, uploadProxyImage, uploadServerImage, tag, verbosity, pullPolicy)
	deployment.Spec.Template.Spec.Containers = []corev1.Container{container}
	return deployment
}

func createPrometheusPorts() *[]corev1.ContainerPort {
	return &[]corev1.ContainerPort{
		{
			Name:          "metrics",
			ContainerPort: 60000,
			Protocol:      "TCP",
		},
	}
}
