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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path"
	"time"

	yaml "github.com/ghodss/yaml"
	csvv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/kubevirt/hyperconverged-cluster-operator/tools/util"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/version"
	"github.com/blang/semver"
)

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
	ClusterPermissions []csvClusterPermissions  `json:"clusterPermissions"`
	Permissions        []csvPermissions         `json:"permissions"`
	Deployments        []csvDeployments         `json:"deployments"`
}

var (
	cnaCsv              = flag.String("cna-csv", "", "")
	virtCsv             = flag.String("virt-csv", "", "")
	sspCsv              = flag.String("ssp-csv", "", "")
	mroCsv              = flag.String("mro-csv", "", "")
	cdiCsv              = flag.String("cdi-csv", "", "")
	nmoCsv              = flag.String("nmo-csv", "", "")
	operatorImage       = flag.String("operator-image-name", "", "")
	csvVersion          = flag.String("csv-version", "", "")
	replacesCsvVersion  = flag.String("replaces-csv-version", "", "")
)

func main() {
	flag.Parse()

	csvs := []string{
		*cnaCsv,
		*virtCsv,
		*sspCsv,
		*mroCsv,
		*cdiCsv,
		*nmoCsv,
	}

	// The template name and dir are configured in build/Dockerfile
	templateDir := "/var"
	templateName := "hco-csv-template.yaml.in"

	templateCSVBytes, err := ioutil.ReadFile(path.Join(templateDir, templateName))
	if err != nil {
		panic(err)
	}

	templateStruct := &csvv1.ClusterServiceVersion{}
	err = yaml.Unmarshal(templateCSVBytes, templateStruct)
	if err != nil {
		panic(err)
	}

	templateStrategySpec := &csvStrategySpec{}
	json.Unmarshal(templateStruct.Spec.InstallStrategy.StrategySpecRaw, templateStrategySpec)

	for _, csvStr := range csvs {
		if csvStr != "" {
			csvBytes := []byte(csvStr)

			csvStruct := &csvv1.ClusterServiceVersion{}

			err = yaml.Unmarshal(csvBytes, csvStruct)
			if err != nil {
				panic(err)
			}

			strategySpec := &csvStrategySpec{}
			json.Unmarshal(csvStruct.Spec.InstallStrategy.StrategySpecRaw, strategySpec)

			deployments := strategySpec.Deployments
			clusterPermissions := strategySpec.ClusterPermissions
			permissions := strategySpec.Permissions

			templateStrategySpec.Deployments = append(templateStrategySpec.Deployments, deployments...)
			templateStrategySpec.ClusterPermissions = append(templateStrategySpec.ClusterPermissions, clusterPermissions...)
			templateStrategySpec.Permissions = append(templateStrategySpec.Permissions, permissions...)

			for _, owned := range csvStruct.Spec.CustomResourceDefinitions.Owned {
				templateStruct.Spec.CustomResourceDefinitions.Owned = append(
					templateStruct.Spec.CustomResourceDefinitions.Owned,
					csvv1.CRDDescription{
						Name:    owned.Name,
						Version: owned.Version,
						Kind:    owned.Kind,
					})
			}
		}
	}

	dfound := false
	efound := false
	for _, deployment := range templateStrategySpec.Deployments {
		if deployment.Name == "hco-operator" {
			dfound = true
			deployment.Spec.Template.Spec.Containers[0].Image = *operatorImage
			for i, env := range deployment.Spec.Template.Spec.Containers[0].Env {
				if env.Name == "OPERATOR_IMAGE" {
					efound = true
					deployment.Spec.Template.Spec.Containers[0].Env[i].Value = *operatorImage
				}
			}
		}
	}

	if !dfound {
		panic("Failed identifying hco-operator deployment")
	}
	if !efound {
		panic("Failed identifying OPERATOR_IMAGE env value for hco-operator")
	}

	// Re-serialize deployments and permissions into csv strategy.
	updatedStrat, err := json.Marshal(templateStrategySpec)
	if err != nil {
		panic(err)
	}
	templateStruct.Spec.InstallStrategy.StrategySpecRaw = updatedStrat

	templateStruct.Annotations["createdAt"] = time.Now().Format("2006-01-02 15:04:05")
	templateStruct.Annotations["containerImage"] = *operatorImage
	templateStruct.Name = "kubevirt-hyperconverged-operator.v" + *csvVersion
	templateStruct.Spec.Version = version.OperatorVersion{semver.MustParse(*csvVersion)}
	templateStruct.Spec.Replaces = "kubevirt-hyperconverged-operator.v" + *replacesCsvVersion

	util.MarshallObject(templateStruct, os.Stdout)

}
