package components

import (
	"encoding/json"
	"fmt"

	"github.com/blang/semver"
	csvv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/version"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const description = `The **machine remediation operator** deploys components to monitor and remediate unhealthy machines for different platforms, it works on top of cluster-api controllers.

It should deploy three controllers:

* [machine-health-check](https://github.com/kubevirt/machine-remediation-operator/blob/master/docs/machine-health-check.md) controller
* [machine-disruption-budget](https://github.com/kubevirt/machine-remediation-operator/blob/master/docs/machine-disruption-budget.md) controller
* [machine-remediation](https://github.com/kubevirt/machine-remediation-operator/blob/master/docs/machine-remediation.md) controller`

const almExamples = `[
  {
    "apiVersion":"machineremediation.kubevirt.io/v1alpha1",
    "kind":"MachineRemediationOperator",
    "metadata": {
      "name":"mro",
      "namespace":"openshift-machine-api"
    },
    "spec": {
      "imagePullPolicy":"IfNotPresent",
    }
  }
]`

// ClusterServiceVersionData contains all data needed for CSV generation
type ClusterServiceVersionData struct {
	Namespace          string
	ContainerPrefix    string
	ContainerTag       string
	ImagePullPolicy    corev1.PullPolicy
	Verbosity          string
	CSVVersion         string
	ReplacesCSVVersion string
	CreatedAtTimestamp string
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

// NewClusterServiceVersion returns new ClusterServiceVersion object
func NewClusterServiceVersion(data *ClusterServiceVersionData) (*csvv1.ClusterServiceVersion, error) {
	operatorData := &DeploymentData{
		Name:            ComponentMachineRemediationOperator,
		Namespace:       data.Namespace,
		ImageRepository: data.ContainerPrefix,
		PullPolicy:      corev1.PullPolicy(data.ImagePullPolicy),
		Verbosity:       data.Verbosity,
		OperatorVersion: data.ContainerTag,
	}
	operator := NewDeployment(operatorData)

	strategySpec := csvStrategySpec{
		ClusterPermissions: []csvClusterPermissions{
			{
				ServiceAccountName: ComponentMachineRemediationOperator,
				Rules:              Rules[ComponentMachineRemediationOperator],
			},
		},
		Deployments: []csvDeployments{
			{
				Name: ComponentMachineRemediationOperator,
				Spec: operator.Spec,
			},
		},
	}

	strategySpecJSONBytes, err := json.Marshal(strategySpec)
	if err != nil {
		return nil, err
	}

	csvVersion, err := semver.Make(data.CSVVersion)
	if err != nil {
		return nil, err
	}

	csv := &csvv1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       csvv1.ClusterServiceVersionKind,
			APIVersion: csvv1.ClusterServiceVersionAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s.%s", ComponentMachineRemediationOperator, data.CSVVersion),
			Namespace: "placeholder",
			Annotations: map[string]string{
				"capabilities":   "Full Lifecycle",
				"categories":     "OpenShift Optional",
				"containerImage": getImage(ComponentMachineRemediationOperator, data.ContainerPrefix, data.ContainerTag),
				"createdAt":      data.CreatedAtTimestamp,
				"repository":     "https://github.com/kubevirt/machine-remediation-operator",
				"certified":      "false",
				"support":        "KubeVirt",
				"alm-examples":   almExamples,
				"description":    "Deploys components to monitor and remediate unhealthy machines",
			},
		},
		Spec: csvv1.ClusterServiceVersionSpec{
			DisplayName: "Machine Remediation Operator",
			Description: description,
			Keywords:    []string{"remediation", "fencing", "HA", "health", "cluster-api"},
			Version:     version.OperatorVersion{Version: csvVersion},
			Maturity:    "alpha",
			Maintainers: []csvv1.Maintainer{{
				Name:  "KubeVirt project",
				Email: "kubevirt-dev@googlegroups.com",
			}},
			Provider: csvv1.AppLink{
				Name: "Machine Remediation Operator project",
			},
			Links: []csvv1.AppLink{
				{
					Name: "KubeVirt",
					URL:  "https://kubevirt.io",
				},
				{
					Name: "Source Code",
					URL:  "https://github.com/kubevirt/machine-remediation-operator",
				},
			},
			Labels: map[string]string{
				"alm-owner-kubevirt": "machineremediationoperator",
				"operated-by":        "machineremediationoperator",
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"alm-owner-kubevirt": "machineremediationoperator",
					"operated-by":        "machineremediationoperator",
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
						Name:        "machineremediationoperators.machineremediation.kubevirt.io",
						Version:     "v1alpha1",
						Kind:        "MachineRemediationOperator",
						DisplayName: "Machine Remediation Operator deployment",
						Description: "Represents a Machine Remediation Operator deployment",
						SpecDescriptors: []csvv1.SpecDescriptor{

							{
								Description:  "The ImagePullPolicy to use for the Machine Remediation components.",
								DisplayName:  "ImagePullPolicy",
								Path:         "imagePullPolicy",
								XDescriptors: []string{"urn:alm:descriptor:io.kubernetes:imagePullPolicy"},
							},
							{
								Description:  "The ImageRegistry to use for the Machine Remediation components.",
								DisplayName:  "ImageRegistry",
								Path:         "imageRegistry",
								XDescriptors: []string{"urn:alm:descriptor:text"},
							},
						},
						StatusDescriptors: []csvv1.StatusDescriptor{
							{
								Description:  "Explanation for the current status of the cluster.",
								DisplayName:  "Conditions",
								Path:         "conditions",
								XDescriptors: []string{"urn:alm:descriptor:io.kubernetes.conditions"},
							},
						},
					},
				},
			},
		},
	}

	if data.ReplacesCSVVersion != "" {
		csv.Spec.Replaces = fmt.Sprintf("%s.%s", ComponentMachineRemediationOperator, data.ReplacesCSVVersion)
	}

	return csv, nil
}
