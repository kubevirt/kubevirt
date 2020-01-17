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
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"
	"github.com/kubevirt/hyperconverged-cluster-operator/tools/util"

	csvv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const operatorName = "kubevirt-hyperconverged-operator"

// TODO: get rid of this once RelatedImages officially
// appears in github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators
type relatedImage struct {
	Name string `json:"name"`
	Ref  string `json:"image"`
}

type ClusterServiceVersionSpecExtended struct {
	csvv1alpha1.ClusterServiceVersionSpec
	RelatedImages []relatedImage `json:"relatedImages,omitempty"`
}

type ClusterServiceVersionExtended struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ClusterServiceVersionSpecExtended       `json:"spec"`
	Status csvv1alpha1.ClusterServiceVersionStatus `json:"status"`
}

var (
	cnaCsv              = flag.String("cna-csv", "", "Cluster Network Addons CSV string")
	virtCsv             = flag.String("virt-csv", "", "KubeVirt CSV string")
	sspCsv              = flag.String("ssp-csv", "", "Scheduling Scale Performance CSV string")
	cdiCsv              = flag.String("cdi-csv", "", "Containerized Data Importer CSV String")
	nmoCsv              = flag.String("nmo-csv", "", "Node Maintenance Operator CSV String")
	hppCsv              = flag.String("hpp-csv", "", "HostPath Provisioner Operator CSV String")
	operatorImage       = flag.String("operator-image-name", "", "HyperConverged Cluster Operator image")
	imsConversionImage  = flag.String("ims-conversion-image-name", "", "IMS conversion image")
	imsVMWareImage      = flag.String("ims-vmware-image-name", "", "IMS VMWare image")
	csvVersion          = flag.String("csv-version", "", "CSV version")
	replacesCsvVersion  = flag.String("replaces-csv-version", "", "CSV version to replace")
	metadataDescription = flag.String("metadata-description", "", "Metadata")
	specDescription     = flag.String("spec-description", "", "Description")
	specDisplayName     = flag.String("spec-displayname", "", "Display Name")
	relatedImagesList   = flag.String("related-images-list", "", "Comma separated list of all the images referred in the CSV")
	namespace           = flag.String("namespace", "kubevirt-hyperconverged", "Namespace")
	crdDisplay          = flag.String("crd-display", "KubeVirt HyperConverged Cluster", "Label show in OLM UI about the primary CRD")
	csvOverrides        = flag.String("csv-overrides", "", "CSV like string with punctual changes that will be recursively applied (if possible)")
)

func main() {
	flag.Parse()

	if *specDisplayName == "" || *specDescription == "" {
		panic(errors.New("Must specify spec-displayname and spec-description"))
	}

	csvs := []string{
		*cnaCsv,
		*virtCsv,
		*sspCsv,
		*cdiCsv,
		*nmoCsv,
		*hppCsv,
	}

	version := semver.MustParse(*csvVersion)
	var replaces string
	if *replacesCsvVersion != "" {
		replaces = fmt.Sprintf("%v.v%v", operatorName, semver.MustParse(*replacesCsvVersion).String())
	}

	// This is the basic CSV without an InstallStrategy defined
	csvBase := components.GetCSVBase(
		operatorName,
		*namespace,
		*specDisplayName,
		*specDescription,
		*operatorImage,
		replaces,
		version,
		*crdDisplay,
	)
	csvExtended := ClusterServiceVersionExtended{
		TypeMeta:   csvBase.TypeMeta,
		ObjectMeta: csvBase.ObjectMeta,
		Spec:       ClusterServiceVersionSpecExtended{ClusterServiceVersionSpec: csvBase.Spec},
		Status:     csvBase.Status}

	// This is the base deployment + rbac for the HCO CSV
	installStrategyBase := components.GetInstallStrategyBase(
		*operatorImage,
		"IfNotPresent",
		*imsConversionImage,
		*imsVMWareImage,
	)

	for _, image := range strings.Split(*relatedImagesList, ",") {
		if image != "" {
			names := strings.Split(strings.Split(image, "@")[0], "/")
			name := names[len(names)-1]
			csvExtended.Spec.RelatedImages = append(
				csvExtended.Spec.RelatedImages,
				relatedImage{
					Name: name,
					Ref:  image,
				})
		}
	}

	for _, csvStr := range csvs {
		if csvStr != "" {
			csvBytes := []byte(csvStr)

			csvStruct := &csvv1alpha1.ClusterServiceVersion{}

			err := yaml.Unmarshal(csvBytes, csvStruct)
			if err != nil {
				panic(err)
			}

			strategySpec := &components.StrategyDetailsDeployment{}
			json.Unmarshal(csvStruct.Spec.InstallStrategy.StrategySpecRaw, strategySpec)

			installStrategyBase.DeploymentSpecs = append(installStrategyBase.DeploymentSpecs, strategySpec.DeploymentSpecs...)
			installStrategyBase.ClusterPermissions = append(installStrategyBase.ClusterPermissions, strategySpec.ClusterPermissions...)
			installStrategyBase.Permissions = append(installStrategyBase.Permissions, strategySpec.Permissions...)

			for _, owned := range csvStruct.Spec.CustomResourceDefinitions.Owned {
				csvExtended.Spec.CustomResourceDefinitions.Owned = append(
					csvExtended.Spec.CustomResourceDefinitions.Owned,
					csvv1alpha1.CRDDescription{
						Name:        owned.Name,
						Version:     owned.Version,
						Kind:        owned.Kind,
						Description: owned.Description,
						DisplayName: owned.DisplayName,
					},
				)
			}

			csv_base_alm_string := csvExtended.Annotations["alm-examples"]
			csv_struct_alm_string := csvStruct.Annotations["alm-examples"]
			var base_almcrs []interface{}
			var struct_almcrs []interface{}
			if err = json.Unmarshal([]byte(csv_base_alm_string), &base_almcrs); err != nil {
				panic(err)
			}
			if err = json.Unmarshal([]byte(csv_struct_alm_string), &struct_almcrs); err != nil {
				panic(err)
			}
			for _, cr := range struct_almcrs {
				base_almcrs = append(
					base_almcrs,
					cr,
				)
			}
			alm_b, err := json.Marshal(base_almcrs)
			if err != nil {
				panic(err)
			}
			csvExtended.Annotations["alm-examples"] = string(alm_b)

		}
	}

	dfound := false
	efound := false
	for _, deployment := range installStrategyBase.DeploymentSpecs {
		if deployment.Name == "hco-operator" {
			dfound = true
			deployment.Spec.Template.Spec.Containers[0].Image = *operatorImage
			for i, env := range deployment.Spec.Template.Spec.Containers[0].Env {
				if env.Name == "OPERATOR_IMAGE" {
					efound = true
					deployment.Spec.Template.Spec.Containers[0].Env[i].Value = *operatorImage
				}
				if env.Name == "CONVERSION_CONTAINER" {
					efound = true
					deployment.Spec.Template.Spec.Containers[0].Env[i].Value = *imsConversionImage
				}
				if env.Name == "VMWARE_CONTAINER" {
					efound = true
					deployment.Spec.Template.Spec.Containers[0].Env[i].Value = *imsVMWareImage
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
	updatedStrat, err := json.Marshal(installStrategyBase)
	if err != nil {
		panic(err)
	}
	csvExtended.Spec.InstallStrategy.StrategyName = "deployment"
	csvExtended.Spec.InstallStrategy.StrategySpecRaw = updatedStrat

	// These shouldn't be needed because it's in the csvExtended already
	// csvExtended.Annotations["createdAt"] = time.Now().Format("2006-01-02 15:04:05")
	// csvExtended.Annotations["containerImage"] = *operatorImage
	// csvExtended.Name = "kubevirt-hyperconverged-operator.v" + *csvVersion
	// csvExtended.Spec.Version = csvversion.OperatorVersion{semver.MustParse(*csvVersion)}

	if *metadataDescription != "" {
		csvExtended.Annotations["description"] = *metadataDescription
	}
	if *specDescription != "" {
		csvExtended.Spec.Description = *specDescription
	}
	if *specDisplayName != "" {
		csvExtended.Spec.DisplayName = *specDisplayName
	}

	if *csvOverrides != "" {
		csvOBytes := []byte(*csvOverrides)

		csvO := &ClusterServiceVersionExtended{}

		err := yaml.Unmarshal(csvOBytes, csvO)
		if err != nil {
			panic(err)
		}

		err = mergo.Merge(&csvExtended, csvO, mergo.WithOverride)
		if err != nil {
			panic(err)
		}

	}

	util.MarshallObject(csvExtended, os.Stdout)
}
