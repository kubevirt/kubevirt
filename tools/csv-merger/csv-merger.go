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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"

	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"
	"github.com/kubevirt/hyperconverged-cluster-operator/tools/util"

	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const operatorName = "kubevirt-hyperconverged-operator"

const CSVMode = "CSV"
const CRDMode = "CRDs"

var validOutputModes = []string{CSVMode, CRDMode}

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

type EnvVarFlags []corev1.EnvVar

func (i *EnvVarFlags) String() string {
	es := []string{}
	for _, ev := range *i {
		es = append(es, fmt.Sprintf("%s=%s", ev.Name, ev.Value))
	}
	return strings.Join(es, ",")
}

func (i *EnvVarFlags) Set(value string) error {
	kv := strings.Split(value, "=")
	*i = append(*i, corev1.EnvVar{
		Name:  kv[0],
		Value: kv[1],
	})
	return nil
}

var (
	cwd, _              = os.Getwd()
	outputMode          = flag.String("output-mode", CSVMode, "Working mode: "+strings.Join(validOutputModes, "|"))
	cnaCsv              = flag.String("cna-csv", "", "Cluster Network Addons CSV string")
	virtCsv             = flag.String("virt-csv", "", "KubeVirt CSV string")
	sspCsv              = flag.String("ssp-csv", "", "Scheduling Scale Performance CSV string")
	cdiCsv              = flag.String("cdi-csv", "", "Containerized Data Importer CSV String")
	hppCsv              = flag.String("hpp-csv", "", "HostPath Provisioner Operator CSV String")
	vmImportCsv         = flag.String("vmimport-csv", "", "Virtual Machine Import Operator CSV String")
	operatorImage       = flag.String("operator-image-name", "", "HyperConverged Cluster Operator image")
	imsConversionImage  = flag.String("ims-conversion-image-name", "", "IMS conversion image")
	imsVMWareImage      = flag.String("ims-vmware-image-name", "", "IMS VMWare image")
	smbios              = flag.String("smbios", "", "Custom SMBIOS string for KubeVirt ConfigMap")
	machinetype         = flag.String("machinetype", "", "Custom MACHINETYPE string for KubeVirt ConfigMap")
	csvVersion          = flag.String("csv-version", "", "CSV version")
	replacesCsvVersion  = flag.String("replaces-csv-version", "", "CSV version to replace")
	metadataDescription = flag.String("metadata-description", "", "Metadata")
	specDescription     = flag.String("spec-description", "", "Description")
	specDisplayName     = flag.String("spec-displayname", "", "Display Name")
	namespace           = flag.String("namespace", "kubevirt-hyperconverged", "Namespace")
	crdDisplay          = flag.String("crd-display", "KubeVirt HyperConverged Cluster", "Label show in OLM UI about the primary CRD")
	csvOverrides        = flag.String("csv-overrides", "", "CSV like string with punctual changes that will be recursively applied (if possible)")
	visibleCRDList      = flag.String("visible-crds-list", "hyperconvergeds.hco.kubevirt.io,hostpathprovisioners.hostpathprovisioner.kubevirt.io",
		"Comma separated list of all the CRDs that should be visible in OLM console")
	relatedImagesList = flag.String("related-images-list", "",
		"Comma separated list of all the images referred in the CSV (just the image pull URLs or eventually a set of 'image|name' collations)")
	crdDir          = flag.String("crds-dir", "", "the directory containing the CRDs for apigroup validation. The validation will be performed if and only if the value is non-empty.")
	hcoKvIoVersion  = flag.String("hco-kv-io-version", "", "KubeVirt version")
	kubevirtVersion = flag.String("kubevirt-version", "", "Kubevirt operator version")
	cdiVersion      = flag.String("cdi-version", "", "CDI operator version")
	cnaoVersion     = flag.String("cnao-version", "", "CNA operator version")
	sspVersion      = flag.String("ssp-version", "", "SSP operator version")
	hppoVersion     = flag.String("hppo-version", "", "HPP operator version")
	vmImportVersion = flag.String("vm-import-version", "", "VM-Import operator version")
	apiSources      = flag.String("api-sources", cwd+"/...", "Project sources")
	envVars         EnvVarFlags
)

func gen_hco_crds() {
	// Write out CRDs and CR
	util.MarshallObject(components.GetOperatorCRD(*apiSources), os.Stdout)
	util.MarshallObject(components.GetV2VCRD(), os.Stdout)
	util.MarshallObject(components.GetV2VOvirtProviderCRD(), os.Stdout)
}

func IOReadDir(root string) ([]string, error) {
	var files []string
	fileInfo, err := ioutil.ReadDir(root)
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		files = append(files, filepath.Join(root, file.Name()))
	}
	return files, nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func validateNoApiOverlap(crdDir string) bool {
	var (
		crdFiles []string
		err      error
	)
	crdFiles, err = IOReadDir(crdDir)
	if err != nil {
		panic(err)
	}

	// crdMap is populated with operator names as keys and a slice of associated api groups as values.
	crdMap := make(map[string][]string)

	for _, crdFilePath := range crdFiles {
		file, err := os.Open(crdFilePath)
		if err != nil {
			panic(err)
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}
		err = file.Close()
		if err != nil {
			panic(err)
		}

		crdFileName := filepath.Base(crdFilePath)
		reg := regexp.MustCompile(`([^\d]+)`)
		operator := reg.FindString(crdFileName)

		var crd apiextensions.CustomResourceDefinition
		err = yaml.Unmarshal(content, &crd)
		if err != nil {
			panic(err)
		}
		if !stringInSlice(crd.Spec.Group, crdMap[operator]) {
			crdMap[operator] = append(crdMap[operator], crd.Spec.Group)
		}
	}

	// overlapsMap is populated with collisions found - API Groups as keys,
	// and slice containing operators using them, as values.
	overlapsMap := make(map[string][]string)
	for operator := range crdMap {
		for _, apigroup := range crdMap[operator] {
			for comparedOperator := range crdMap {
				if operator == comparedOperator {
					continue
				}
				if stringInSlice(apigroup, crdMap[comparedOperator]) {
					overlappingOperators := []string{operator, comparedOperator}
					for _, o := range overlappingOperators {
						// We work on replacement for current v2v. Remove this check when vmware import is removed
						if !stringInSlice(o, overlapsMap[apigroup]) && apigroup != "v2v.kubevirt.io" {
							overlapsMap[apigroup] = append(overlapsMap[apigroup], o)
						}
					}
				}
			}
		}
	}

	// if at least one overlap found - emit an error.
	if len(overlapsMap) != 0 {
		log.Print("ERROR: Overlapping API Groups were found between different operators.")
		for apigroup := range overlapsMap {
			fmt.Print("The API Group " + apigroup + " is being used by these operators: " + strings.Join(overlapsMap[apigroup], ", ") + "\n")
			return false
		}
	}
	return true
}

func main() {
	flag.Var(&envVars, "env-var", "HCO environment variable (key=value), may be used multiple times")

	flag.Parse()

	if *crdDir != "" {
		result := validateNoApiOverlap(*crdDir)
		if result {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	switch *outputMode {
	case CRDMode:
		gen_hco_crds()
	case CSVMode:
		if *specDisplayName == "" || *specDescription == "" {
			panic(errors.New("Must specify spec-displayname and spec-description"))
		}

		csvs := []string{
			*cnaCsv,
			*virtCsv,
			*sspCsv,
			*cdiCsv,
			*hppCsv,
			*vmImportCsv,
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
			*namespace,
			*operatorImage,
			"IfNotPresent",
			*imsConversionImage,
			*imsVMWareImage,
			*smbios,
			*machinetype,
			*hcoKvIoVersion,
			*kubevirtVersion,
			*cdiVersion,
			*cnaoVersion,
			*sspVersion,
			*hppoVersion,
			*vmImportVersion,
			envVars,
		)

		for _, image := range strings.Split(*relatedImagesList, ",") {
			if image != "" {
				name := ""
				if strings.Contains(image, "|") {
					image_s := strings.Split(image, "|")
					image = image_s[0]
					name = image_s[1]
				} else {
					names := strings.Split(strings.Split(image, "@")[0], "/")
					name = names[len(names)-1]
				}
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

				strategySpec := csvStruct.Spec.InstallStrategy.StrategySpec

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

		hidden_crds := []string{}
		visible_crds := strings.Split(*visibleCRDList, ",")
		for _, owned := range csvExtended.Spec.CustomResourceDefinitions.Owned {
			found := false
			for _, name := range visible_crds {
				if owned.Name == name {
					found = true
				}
			}
			if !found {
				hidden_crds = append(
					hidden_crds,
					owned.Name,
				)
			}
		}

		hidden_crds_j, err := json.Marshal(hidden_crds)
		if err != nil {
			panic(err)
		}
		csvExtended.Annotations["operators.operatorframework.io/internal-objects"] = string(hidden_crds_j)

		// Update csv strategy.
		csvExtended.Spec.InstallStrategy.StrategyName = "deployment"
		csvExtended.Spec.InstallStrategy.StrategySpec = *installStrategyBase

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

	default:
		panic("Unsupported output mode: " + *outputMode)
	}

}
