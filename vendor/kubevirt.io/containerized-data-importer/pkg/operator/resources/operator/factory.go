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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utils "kubevirt.io/containerized-data-importer/pkg/operator/resources/utils"
)

//FactoryArgs contains the required parameters to generate all cluster-scoped resources
type FactoryArgs struct {
	OperatorImage          string `required:"true" split_words:"true"`
	DockerRepo             string `required:"true" split_words:"true"`
	DockerTag              string `required:"true" split_words:"true"`
	DeployClusterResources string `required:"true" split_words:"true"`
	ControllerImage        string `required:"true" split_words:"true"`
	ImporterImage          string `required:"true" split_words:"true"`
	ClonerImage            string `required:"true" split_words:"true"`
	APIServerImage         string `required:"true" envconfig:"apiserver_image"`
	UploadProxyImage       string `required:"true" split_words:"true"`
	UploadServerImage      string `required:"true" split_words:"true"`
	Verbosity              string `required:"true"`
	PullPolicy             string `required:"true" split_words:"true"`
	Namespace              string
	CsvVersion             string `required:"true"`
	ReplacesCsvVersion     string
	CDILogo                string
}

type operatorFactoryFunc func(*FactoryArgs) []runtime.Object

const (
	//OperatorRBAC - operator rbac
	OperatorRBAC string = "operator-rbac"
	//OperatorDeployment - operator deployment
	OperatorDeployment string = "operator-deployment"
	//OperatorCdiCRD - operator CRDs
	OperatorCdiCRD string = "operator-cdi-crd"
	//OperatorConfigMapCR - operartor configmap
	OperatorConfigMapCR string = "operator-configmap-cr"
	//OperatorCSV - operator csv
	OperatorCSV string = "operator-csv"
)

var operatorFactoryFunctions = map[string]operatorFactoryFunc{
	OperatorRBAC:        createOperatorClusterRBAC,
	OperatorDeployment:  createOperatorClusterDeployment,
	OperatorCdiCRD:      createOperatorCDIClusterResource,
	OperatorConfigMapCR: createOperatorConfigMapClusterResource,
	OperatorCSV:         createOperatorClusterServiceVersion,
}

//IsFactoryResource returns true id codeGroupo belolngs to factory functions
func IsFactoryResource(codeGroup string) bool {
	for k := range operatorFactoryFunctions {
		if codeGroup == k {
			return true
		}
	}
	return false
}

//GetOperatorClusterRules returnes operator cluster rules
func GetOperatorClusterRules() *[]rbacv1.PolicyRule {
	return getOperatorClusterRules()
}

//GetOperatorDeploymentSpec returns operator deployment spce
func GetOperatorDeploymentSpec(args *FactoryArgs) *appsv1.DeploymentSpec {
	return createOperatorDeploymentSpec(args.DockerRepo,
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
		args.PullPolicy)
}

// CreateAllOperatorResources creates all cluster-wide resources
func CreateAllOperatorResources(args *FactoryArgs) ([]runtime.Object, error) {
	var resources []runtime.Object
	for group := range operatorFactoryFunctions {
		rs, err := CreateOperatorResourceGroup(group, args)
		if err != nil {
			return nil, err
		}
		resources = append(resources, rs...)
	}
	return resources, nil
}

// CreateOperatorResourceGroup creates all cluster resources fr a specific group/component
func CreateOperatorResourceGroup(group string, args *FactoryArgs) ([]runtime.Object, error) {
	f, ok := operatorFactoryFunctions[group]
	if !ok {
		return nil, fmt.Errorf("group %s does not exist", group)
	}
	resources := f(args)
	utils.ValidateGVKs(resources)
	return resources, nil
}
