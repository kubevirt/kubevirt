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
	"encoding/json"

	"github.com/blang/semver"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	csvv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
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
				"deployments/finalizers",
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
				"mutatingwebhookconfigurations",
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
				"volumesnapshots",
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
				"volumesnapshots/finalizers",
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
		{
			APIGroups: []string{
				"snapshot.storage.k8s.io",
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
				"apiextensions.k8s.io",
			},
			Resources: []string{
				"customresourcedefinitions",
			},
			Verbs: []string{
				"*",
			},
		},
	}

	return &rules
}

func createOperatorClusterRole(roleName string) *rbacv1.ClusterRole {
	clusterRole := cluster.CreateOperatorClusterRole(roleName)
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
	return cluster.CreateOperatorClusterRoleBinding(operatorServiceAccountName, operatorClusterRoleName, operatorServiceAccountName, namespace)
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
			Value: deployClusterResources,
		},
		{
			Name:  "DOCKER_REPO",
			Value: repo,
		},
		{
			Name:  "DOCKER_TAG",
			Value: tag,
		},
		{
			Name:  "CONTROLLER_IMAGE",
			Value: controllerImage,
		},
		{
			Name:  "IMPORTER_IMAGE",
			Value: importerImage,
		},
		{
			Name:  "CLONER_IMAGE",
			Value: clonerImage,
		},
		{
			Name:  "APISERVER_IMAGE",
			Value: apiServerImage,
		},
		{
			Name:  "UPLOAD_SERVER_IMAGE",
			Value: uploadServerImage,
		},
		{
			Name:  "UPLOAD_PROXY_IMAGE",
			Value: uploadProxyImage,
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

func createOperatorClusterServiceVersion(args *FactoryArgs) []runtime.Object {

	cdiImageNames := CdiImages{
		ControllerImage:   args.ControllerImage,
		ImporterImage:     args.ImporterImage,
		ClonerImage:       args.ClonerImage,
		APIServerImage:    args.APIServerImage,
		UplodaProxyImage:  args.UploadProxyImage,
		UplodaServerImage: args.UploadServerImage,
		OperatorImage:     args.OperatorImage,
	}

	data := NewClusterServiceVersionData{
		CsvVersion:         args.CsvVersion,
		ReplacesCsvVersion: args.ReplacesCsvVersion,
		Namespace:          args.Namespace,
		ImagePullPolicy:    args.PullPolicy,
		IconBase64:         args.CDILogo,
		Verbosity:          args.Verbosity,

		DockerPrefix:  args.DockerRepo,
		DockerTag:     args.DockerTag,
		CdiImageNames: cdiImageNames.FillDefaults(),
	}

	csv, err := createClusterServiceVersion(&data)
	if err != nil {
		panic(err)
	}
	return []runtime.Object{csv}

}

type csvClusterPermissions struct {
	ServiceAccountName string              `json:"serviceAccountName"`
	Rules              []rbacv1.PolicyRule `json:"rules"`
}
type csvDeployments struct {
	Name string                `json:"name"`
	Spec appsv1.DeploymentSpec `json:"spec,omitempty"`
}

type csvStrategySpec struct {
	ClusterPermissions []csvClusterPermissions `json:"clusterPermissions"`
	Deployments        []csvDeployments        `json:"deployments"`
}

func createClusterServiceVersion(data *NewClusterServiceVersionData) (*csvv1.ClusterServiceVersion, error) {

	description := `
CDI is a kubernetes extension that provides the ability to populate PVCs with VM images upon creation. Multiple image formats and sources are supported

_The CDI Operator does not support updates yet._
`

	deployment := createOperatorDeployment(
		data.DockerPrefix,
		data.Namespace,
		"true",
		data.CdiImageNames.OperatorImage,
		data.CdiImageNames.ControllerImage,
		data.CdiImageNames.ImporterImage,
		data.CdiImageNames.ClonerImage,
		data.CdiImageNames.APIServerImage,
		data.CdiImageNames.UplodaProxyImage,
		data.CdiImageNames.UplodaServerImage,
		data.DockerTag,
		data.Verbosity,
		data.ImagePullPolicy)

	rules := getOperatorClusterRules()

	strategySpec := csvStrategySpec{
		ClusterPermissions: []csvClusterPermissions{
			{
				ServiceAccountName: operatorServiceAccountName,
				Rules:              *rules,
			},
		},
		Deployments: []csvDeployments{
			{
				Name: "cdi-operator",
				Spec: deployment.Spec,
			},
		},
	}

	strategySpecJSONBytes, err := json.Marshal(strategySpec)
	if err != nil {
		return nil, err
	}

	csvVersion, err := semver.New(data.CsvVersion)
	if err != nil {
		return nil, err
	}

	return &csvv1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterServiceVersion",
			APIVersion: "operators.coreos.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cdioperator." + data.CsvVersion,
			Namespace: data.Namespace,
			Annotations: map[string]string{

				"capabilities": "Full Lifecycle",
				"categories":   "Storage,Virtualization",
				"alm-examples": `
      [
        {
          "apiVersion":"cdi.kubevirt.io/v1alpha1",
          "kind":"CDI",
          "metadata": {
            "name":"cdi",
            "namespace":"cdi"
          },
          "spec": {
            "imagePullPolicy":"IfNotPresent"
          }
        }
      ]`,
				"description": "Creates and maintains CDI deployments",
			},
		},

		Spec: csvv1.ClusterServiceVersionSpec{
			DisplayName: "CDI",
			Description: description,
			Keywords:    []string{"CDI", "Virtualization", "Storage"},
			Version:     version.OperatorVersion{Version: *csvVersion},
			Maturity:    "alpha",
			Replaces:    data.ReplacesCsvVersion,
			Maintainers: []csvv1.Maintainer{{
				Name:  "KubeVirt project",
				Email: "kubevirt-dev@googlegroups.com",
			}},
			Provider: csvv1.AppLink{
				Name: "KubeVirt/CDI project",
			},
			Links: []csvv1.AppLink{
				{
					Name: "CDI",
					URL:  "https://github.com/kubevirt/containerized-data-importer/blob/master/README.md",
				},
				{
					Name: "Source Code",
					URL:  "https://github.com/kubevirt/containerized-data-importer",
				},
			},
			Icon: []csvv1.Icon{{
				Data:      data.IconBase64,
				MediaType: "image/png",
			}},
			Labels: map[string]string{
				"alm-owner-cdi": "cdi-operator",
				"operated-by":   "cdi-operator",
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"alm-owner-cdi": "cdi-operator",
					"operated-by":   "cdi-operator",
				},
			},
			InstallModes: []csvv1.InstallMode{
				{
					Type:      csvv1.InstallModeTypeOwnNamespace,
					Supported: true,
				},
				{
					Type:      csvv1.InstallModeTypeSingleNamespace,
					Supported: true,
				},
				{
					Type:      csvv1.InstallModeTypeMultiNamespace,
					Supported: true,
				},
				{
					Type:      csvv1.InstallModeTypeAllNamespaces,
					Supported: true,
				},
			},
			InstallStrategy: csvv1.NamedInstallStrategy{
				StrategyName:    "deployment",
				StrategySpecRaw: json.RawMessage(strategySpecJSONBytes),
			},
			CustomResourceDefinitions: csvv1.CustomResourceDefinitions{

				Owned: []csvv1.CRDDescription{
					{
						Name:        "cdis.cdi.kubevirt.io",
						Version:     "v1alpha1",
						Kind:        "CDI",
						DisplayName: "CDI deployment",
						Description: "Represents a CDI deployment",
						Resources: []csvv1.APIResourceReference{
							{
								Kind:    "ConfigMap",
								Name:    "cdi-operator-leader-election-helper",
								Version: "v1",
							},
						},
						SpecDescriptors: []csvv1.SpecDescriptor{

							{
								Description:  "The ImageRegistry to use for the CDI components.",
								DisplayName:  "ImageRegistry",
								Path:         "imageRegistry",
								XDescriptors: []string{"urn:alm:descriptor:text"},
							},
							{
								Description:  "The ImageTag to use for the CDI components.",
								DisplayName:  "ImageTag",
								Path:         "imageTag",
								XDescriptors: []string{"urn:alm:descriptor:text"},
							},
							{
								Description:  "The ImagePullPolicy to use for the CDI components.",
								DisplayName:  "ImagePullPolicy",
								Path:         "imagePullPolicy",
								XDescriptors: []string{"urn:alm:descriptor:io.kubernetes:imagePullPolicy"},
							},
						},
						StatusDescriptors: []csvv1.StatusDescriptor{
							{
								Description:  "The deployment phase.",
								DisplayName:  "Phase",
								Path:         "phase",
								XDescriptors: []string{"urn:alm:descriptor:io.kubernetes.phase"},
							},
							{
								Description:  "Explanation for the current status of the CDI deployment.",
								DisplayName:  "Conditions",
								Path:         "conditions",
								XDescriptors: []string{"urn:alm:descriptor:io.kubernetes.conditions"},
							},
							{
								Description:  "The observed version of the CDI deployment.",
								DisplayName:  "Observed CDI Version",
								Path:         "observedVersion",
								XDescriptors: []string{"urn:alm:descriptor:text"},
							},
							{
								Description:  "The targeted version of the CDI deployment.",
								DisplayName:  "Target CDI Version",
								Path:         "targetVersion",
								XDescriptors: []string{"urn:alm:descriptor:text"},
							},
							{
								Description:  "The version of the CDI Operator",
								DisplayName:  "CDI Operator Version",
								Path:         "operatorVersion",
								XDescriptors: []string{"urn:alm:descriptor:text"},
							},
						},
					},
				},
			},
		},
	}, nil
}
