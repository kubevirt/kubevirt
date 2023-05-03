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
package csv

import (
	"encoding/json"
	"fmt"

	"k8s.io/utils/pointer"

	"github.com/coreos/go-semver/semver"
	csvv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"
)

const xDescriptorText = "urn:alm:descriptor:text"

type NewClusterServiceVersionData struct {
	Namespace             string
	KubeVirtVersion       string
	OperatorImageVersion  string
	DockerPrefix          string
	ImagePrefix           string
	ImagePullPolicy       string
	Verbosity             string
	CsvVersion            string
	VirtApiSha            string
	VirtControllerSha     string
	VirtHandlerSha        string
	VirtLauncherSha       string
	VirtExportProxySha    string
	VirtExportServerSha   string
	GsSha                 string
	PrHelperSha           string
	RunbookURLTemplate    string
	Replicas              int
	IconBase64            string
	ReplacesCsvVersion    string
	CreatedAtTimestamp    string
	VirtOperatorImage     string
	VirtApiImage          string
	VirtControllerImage   string
	VirtHandlerImage      string
	VirtLauncherImage     string
	VirtExportProxyImage  string
	VirtExportServerImage string
	GsImage               string
	PrHelperImage         string
}

type csvClusterPermissions struct {
	ServiceAccountName string              `json:"serviceAccountName"`
	Rules              []rbacv1.PolicyRule `json:"rules"`
}

type csvPermissions struct {
	ServiceAccountName string              `json:"serviceAccountName"`
	Rules              []rbacv1.PolicyRule `json:"rules"`
}

type csvDeployments struct {
	Name string                `json:"name"`
	Spec appsv1.DeploymentSpec `json:"spec,omitempty"`
}

type csvStrategySpec struct {
	ClusterPermissions []csvClusterPermissions `json:"clusterPermissions"`
	Permissions        []csvPermissions        `json:"permissions"`
	Deployments        []csvDeployments        `json:"deployments"`
}

var description = `
**KubeVirt** is a virtual machine management add-on for Kubernetes.
The aim is to provide a common ground for virtualization solutions on top of
Kubernetes.

# Virtualization extension for Kubernetes

At its core, KubeVirt extends [Kubernetes](https://kubernetes.io) by adding
additional virtualization resource types (especially the ` + "`VirtualMachine`" + ` type) through
[Kubernetes's Custom Resource Definitions API](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/).
By using this mechanism, the Kubernetes API can be used to manage these ` + "`VirtualMachine`" + `
resources alongside all other resources Kubernetes provides.

The resources themselves are not enough to launch virtual machines.
For this to happen the _functionality and business logic_ needs to be added to
the cluster. The functionality is not added to Kubernetes itself, but rather
added to a Kubernetes cluster by _running_ additional controllers and agents
on an existing cluster.

The necessary controllers and agents are provided by KubeVirt.

As of today KubeVirt can be used to declaratively

  * Create a predefined VM
  * Schedule a VM on a Kubernetes cluster
  * Launch a VM
  * Migrate a VM
  * Stop a VM
  * Delete a VM

# Start using KubeVirt

  * Try our quickstart at [kubevirt.io](https://kubevirt.io/get_kubevirt/).
  * See our user documentation at [kubevirt.io/docs](https://kubevirt.io/user-guide).

# Start developing KubeVirt

To set up a development environment please read our
[Getting Started Guide](https://github.com/kubevirt/kubevirt/blob/main/docs/getting-started.md).
To learn how to contribute, please read our [contribution guide](https://github.com/kubevirt/kubevirt/blob/main/CONTRIBUTING.md).

You can learn more about how KubeVirt is designed (and why it is that way),
and learn more about the major components by taking a look at our developer documentation:

  * [Architecture](https://github.com/kubevirt/kubevirt/blob/main/docs/architecture.md) - High-level view on the architecture
  * [Components](https://github.com/kubevirt/kubevirt/blob/main/docs/components.md) - Detailed look at all components
  * [API Reference](https://kubevirt.io/api-reference/)

# Community

If you got enough of code and want to speak to people, then you got a couple of options:

  * Follow us on [Twitter](https://twitter.com/kubevirt)
  * Chat with us in the #virtualization channel of the [Kubernetes Slack](https://slack.k8s.io/)
  * Discuss with us on the [kubevirt-dev Google Group](https://groups.google.com/forum/#!forum/kubevirt-dev)
  * Stay informed about designs and upcoming events by watching our [community content](https://github.com/kubevirt/community/)

# License

KubeVirt is distributed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.txt).
`

func NewClusterServiceVersion(data *NewClusterServiceVersionData) (*csvv1.ClusterServiceVersion, error) {

	deployment, err := components.NewOperatorDeployment(
		data.Namespace,
		data.DockerPrefix,
		data.ImagePrefix,
		data.OperatorImageVersion,
		data.Verbosity,
		data.KubeVirtVersion,
		data.VirtApiSha,
		data.VirtControllerSha,
		data.VirtHandlerSha,
		data.VirtLauncherSha,
		data.VirtExportProxySha,
		data.VirtExportServerSha,
		data.GsSha,
		data.PrHelperSha,
		data.RunbookURLTemplate,
		data.VirtApiImage,
		data.VirtControllerImage,
		data.VirtHandlerImage,
		data.VirtLauncherImage,
		data.VirtExportProxyImage,
		data.VirtExportServerImage,
		data.GsImage,
		data.PrHelperImage,
		data.VirtOperatorImage,
		v1.PullPolicy(data.ImagePullPolicy))
	if err != nil {
		return nil, err
	}

	imageVersion := components.AddVersionSeparatorPrefix(data.OperatorImageVersion)

	if data.Replicas > 0 && *deployment.Spec.Replicas != int32(data.Replicas) {
		deployment.Spec.Replicas = pointer.Int32(int32(data.Replicas))
	}

	clusterRules := rbac.NewOperatorClusterRole().Rules
	rules := rbac.NewOperatorRole(data.Namespace).Rules

	strategySpec := csvStrategySpec{
		ClusterPermissions: []csvClusterPermissions{
			{
				ServiceAccountName: "kubevirt-operator",
				Rules:              clusterRules,
			},
		},
		Permissions: []csvPermissions{
			{
				ServiceAccountName: "kubevirt-operator",
				Rules:              rules,
			},
		},
		Deployments: []csvDeployments{
			{
				Name: "virt-operator",
				Spec: deployment.Spec,
			},
		},
	}

	strategySpecJsonBytes, err := json.Marshal(strategySpec)
	if err != nil {
		return nil, err
	}

	almExampleFmt := `
      [
        {
          "apiVersion":"kubevirt.io/%s",
          "kind":"KubeVirt",
          "metadata": {
            "name":"kubevirt",
            "namespace":"kubevirt"
          },
          "spec": {
            "imagePullPolicy":"Always"
          }
        }
      ]`

	almExample := fmt.Sprintf(almExampleFmt, virtv1.ApiLatestVersion)

	return &csvv1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterServiceVersion",
			APIVersion: "operators.coreos.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirtoperator." + data.CsvVersion,
			Namespace: data.Namespace,
			Annotations: map[string]string{

				"capabilities":   "Seamless Upgrades",
				"categories":     "OpenShift Optional",
				"containerImage": data.DockerPrefix + "/virt-operator" + imageVersion,
				"createdAt":      data.CreatedAtTimestamp,
				"repository":     "https://github.com/kubevirt/kubevirt",
				"certified":      "false",
				"support":        "KubeVirt",
				"alm-examples":   almExample,
				"description":    "Creates and maintains KubeVirt deployments",
			},
		},

		Spec: csvv1.ClusterServiceVersionSpec{
			DisplayName: "KubeVirt",
			Description: description,
			Keywords:    []string{"KubeVirt", "Virtualization"},
			Version:     *semver.New(data.CsvVersion),
			Maturity:    "alpha",
			Replaces:    data.ReplacesCsvVersion,
			Maintainers: []csvv1.Maintainer{{
				Name:  "KubeVirt project",
				Email: "kubevirt-dev@googlegroups.com",
			}},
			Provider: csvv1.AppLink{
				Name: "KubeVirt project",
			},
			Links: []csvv1.AppLink{
				{
					Name: "KubeVirt",
					URL:  "https://kubevirt.io",
				},
				{
					Name: "Source Code",
					URL:  "https://github.com/kubevirt/kubevirt",
				},
			},
			Icon: []csvv1.Icon{{
				Data:      data.IconBase64,
				MediaType: "image/png",
			}},
			Labels: map[string]string{
				"alm-owner-kubevirt": "kubevirtoperator",
				"operated-by":        "kubevirtoperator",
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"alm-owner-kubevirt": "kubevirtoperator",
					"operated-by":        "kubevirtoperator",
				},
			},
			InstallModes: []csvv1.InstallMode{
				{
					Type:      csvv1.InstallModeTypeOwnNamespace,
					Supported: true,
				},
				{
					Type:      csvv1.InstallModeTypeSingleNamespace,
					Supported: false,
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
				StrategySpecRaw: json.RawMessage(strategySpecJsonBytes),
			},
			CustomResourceDefinitions: csvv1.CustomResourceDefinitions{

				Owned: []csvv1.CRDDescription{
					{
						Name:        "kubevirts.kubevirt.io",
						Version:     virtv1.ApiLatestVersion,
						Kind:        "KubeVirt",
						DisplayName: "KubeVirt deployment",
						Description: "Represents a KubeVirt deployment",
						SpecDescriptors: []csvv1.SpecDescriptor{

							{
								Description:  "The ImagePullPolicy to use for the KubeVirt components.",
								DisplayName:  "ImagePullPolicy",
								Path:         "imagePullPolicy",
								XDescriptors: []string{"urn:alm:descriptor:io.kubernetes:imagePullPolicy"},
							},
							{
								Description:  "The ImageRegistry to use for the KubeVirt components.",
								DisplayName:  "ImageRegistry",
								Path:         "imageRegistry",
								XDescriptors: []string{xDescriptorText},
							},
							{
								Description:  "The ImageTag to use for the KubeVirt components.",
								DisplayName:  "ImageTag",
								Path:         "imageTag",
								XDescriptors: []string{xDescriptorText},
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
								Description:  "Explanation for the current status of the cluster.",
								DisplayName:  "Conditions",
								Path:         "conditions",
								XDescriptors: []string{"urn:alm:descriptor:io.kubernetes.conditions"},
							},
							{
								Description:  "The observed version of the KubeVirt deployment.",
								DisplayName:  "Observed KubeVirt Version",
								Path:         "observedKubeVirtVersion",
								XDescriptors: []string{xDescriptorText},
							},
							{
								Description:  "The targeted version of the KubeVirt deployment.",
								DisplayName:  "Target KubeVirt Version",
								Path:         "targetKubeVirtVersion",
								XDescriptors: []string{xDescriptorText},
							},
							{
								Description:  "The observed registry of the KubeVirt deployment.",
								DisplayName:  "Observed KubeVirt registry",
								Path:         "ObservedKubeVirtRegistry",
								XDescriptors: []string{xDescriptorText},
							},
							{
								Description:  "The targeted registry of the KubeVirt deployment.",
								DisplayName:  "Target KubeVirt registry",
								Path:         "TargetKubeVirtRegistry",
								XDescriptors: []string{xDescriptorText},
							},
							{
								Description:  "The version of the KubeVirt Operator.",
								DisplayName:  "KubeVirt Operator Version",
								Path:         "operatorVersion",
								XDescriptors: []string{xDescriptorText},
							},
						},
					},
				},
			},
		},
	}, nil
}
