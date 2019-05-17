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
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	//ControllerImageDefault - defualt value
	ControllerImageDefault = "cdi-controller"
	//ImporterImageDefault - defualt value
	ImporterImageDefault = "cdi-importer"
	//ClonerImageDefault - defualt value
	ClonerImageDefault = "cdi-cloner"
	//APIServerImageDefault - defualt value
	APIServerImageDefault = "cdi-apiserver"
	//UploadProxyImageDefault - defualt value
	UploadProxyImageDefault = "cdi-uploadproxy"
	//UploadServerImageDefault - defualt value
	UploadServerImageDefault = "cdi-uploadserver"
)

//CdiImages - images to be provied to cdi operator
type CdiImages struct {
	ControllerImage   string
	ImporterImage     string
	ClonerImage       string
	APIServerImage    string
	UplodaProxyImage  string
	UplodaServerImage string
}

//FillDefaults - fill image names with defaults
func (ci *CdiImages) FillDefaults() *CdiImages {
	if ci.ControllerImage == "" {
		ci.ControllerImage = ControllerImageDefault
	}
	if ci.ImporterImage == "" {
		ci.ImporterImage = ImporterImageDefault
	}
	if ci.ClonerImage == "" {
		ci.ClonerImage = ClonerImageDefault
	}
	if ci.APIServerImage == "" {
		ci.APIServerImage = APIServerImageDefault
	}
	if ci.UplodaProxyImage == "" {
		ci.UplodaProxyImage = UploadProxyImageDefault
	}
	if ci.UplodaServerImage == "" {
		ci.UplodaServerImage = UploadServerImageDefault
	}

	return ci
}

//NewCdiOperatorDeployment - provides operator deployment spec
func NewCdiOperatorDeployment(namespace string, repository string, tag string, imagePullPolicy string, verbosity string, cdiImages *CdiImages) (*appsv1.Deployment, error) {
	name := "cdi-operator"
	deployment := createOperatorDeployment(
		repository,
		namespace,
		"true",
		name,
		cdiImages.ControllerImage,
		cdiImages.ImporterImage,
		cdiImages.ClonerImage,
		cdiImages.APIServerImage,
		cdiImages.UplodaProxyImage,
		cdiImages.UplodaServerImage,
		tag,
		verbosity,
		imagePullPolicy)

	return deployment, nil
}

//NewCdiOperatorClusterRole - provides operator clusterRole
func NewCdiOperatorClusterRole() *rbacv1.ClusterRole {
	return createOperatorClusterRole(operatorClusterRoleName)
}

//NewCdiCrd - provides CDI CRD
func NewCdiCrd() *extv1beta1.CustomResourceDefinition {
	return createCDIListCRD()
}
