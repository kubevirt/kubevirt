// TOOD: Remove this when Cluster Network Addons components are vendorable
package cnacomponents

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetDeployment(repository string, tag string, imagePullPolicy string) *appsv1.Deployment {
	name := "cluster-network-addons-operator"
	image := fmt.Sprintf("quay.io/%s/%s:%s", repository, name, tag)
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": name,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: name,
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           image,
							ImagePullPolicy: corev1.PullPolicy(imagePullPolicy),
							Env: []corev1.EnvVar{
								{
									Name:  "MULTUS_IMAGE",
									Value: "docker.io/nfvpe/multus:latest",
								},
								{
									Name:  "LINUX_BRIDGE_IMAGE",
									Value: "quay.io/kubevirt/cni-default-plugins",
								},
								{
									Name:  "SRIOV_DP_IMAGE",
									Value: "quay.io/booxter/sriov-device-plugin:latest",
								},
								{
									Name:  "SRIOV_CNI_IMAGE",
									Value: "docker.io/nfvpe/sriov-cni:latest",
								},
								{
									Name:  "SRIOV_ROOT_DEVICES",
									Value: "",
								},
								{
									Name:  "KUBEMACPOOL_IMAGE",
									Value: "quay.io/schseba/mac-controller:latest",
								},
								{
									Name:  "OPERATOR_IMAGE",
									Value: image,
								},
								{
									Name:  "OPERATOR_NAME",
									Value: name,
								},
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "WATCH_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return deployment
}

func GetClusterRole() *rbacv1.ClusterRole {
	name := "cluster-network-addons-operator"
	role := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"name": name,
			},
		},
		Rules: []rbacv1.PolicyRule{
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
					"pods",
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
					"get",
					"list",
					"watch",
					"create",
					"update",
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
					"route.openshift.io",
				},
				Resources: []string{
					"routes",
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
					"serviceaccounts",
					"services",
				},
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"update",
					"delete",
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
					"get",
					"list",
					"watch",
					"create",
					"delete",
				},
			},
			{
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"deployments",
					"daemonsets",
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
					"rbac.authorization.k8s.io",
				},
				Resources: []string{
					"clusterroles",
					"clusterrolebindings",
					"roles",
					"rolebindings",
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
					"apiextensions.k8s.io",
				},
				Resources: []string{
					"customresourcedefinitions",
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
		},
	}
	return role
}

func GetCrd() *extv1beta1.CustomResourceDefinition {
	crd := &extv1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1beta1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "networkaddonsconfigs.networkaddonsoperator.network.kubevirt.io",
		},
		Spec: extv1beta1.CustomResourceDefinitionSpec{
			Group:   "networkaddonsoperator.network.kubevirt.io",
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
				Plural:   "networkaddonsconfigs",
				Singular: "networkaddonsconfig",
				Kind:     "NetworkAddonsConfig",
				ListKind: "NetworkAddonsConfigList",
			},

			Subresources: &extv1beta1.CustomResourceSubresources{
				Status: &extv1beta1.CustomResourceSubresourceStatus{},
			},
		},
	}
	return crd
}

func int32Ptr(i int32) *int32 {
	return &i
}
