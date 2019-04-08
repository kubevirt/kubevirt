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
package helper

import (
	"fmt"
	"io/ioutil"

	yaml2 "github.com/ghodss/yaml"
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"gopkg.in/yaml.v2"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/json"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

type Channel struct {
	Name       string `yaml:"name"`
	CurrentCSV string `yaml:"currentCSV"`
}

type Pkg struct {
	PkgName  string    `yaml:"packageName"`
	Channels []Channel `yaml:"channels"`
}

type Data struct {
	CSVs     string `yaml:"clusterServiceVersions"`
	CRDs     string `yaml:"customResourceDefinitions"`
	Packages string `yaml:"packages"`
}

type Bundle struct {
	Data Data `yaml:"data"`
}

type BundleHelper struct {
	repo string
	Pkgs []Pkg
	CRDs []v1beta1.CustomResourceDefinition
	CSVs []yaml.MapSlice
}

func NewBundleHelper(repo string) (*BundleHelper, error) {
	bh := &BundleHelper{
		repo: repo,
	}
	if err := bh.downloadAndParseBundle(); err != nil {
		return nil, err
	}
	return bh, nil
}

func (bh *BundleHelper) downloadAndParseBundle() error {

	// prepare the app registry client
	options := appregistry.Options{
		Source:    "https://quay.io/cnr",
		AuthToken: "", // not needed for public applications
	}
	client, err := appregistry.NewClientFactory().New(options)
	if err != nil {
		return err
	}

	// get latest bundle info
	bundles, err := client.ListPackages(bh.repo)
	if err != nil {
		return err
	}

	if len(bundles) == 0 {
		fmt.Errorf("no old bundles found\n")
		return nil
	}
	bundleMetaData := bundles[0]

	// download bundle
	data, err := client.RetrieveOne(bundleMetaData.ID(), bundleMetaData.Release)
	if err != nil {
		return err
	}

	// parse bundle into package, CRDs and CSVs
	bundle := Bundle{}
	if err := yaml.Unmarshal(data.RawYAML, &bundle); err != nil {
		return err
	}

	if err := yaml.Unmarshal([]byte(bundle.Data.Packages), &bh.Pkgs); err != nil {
		return err
	}

	if err := yaml.Unmarshal([]byte(bundle.Data.CSVs), &bh.CSVs); err != nil {
		return err
	}

	// use k8s json unmarshaller for CRDs for filling metadata correctly
	crds, err := yaml2.YAMLToJSON([]byte(bundle.Data.CRDs))
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(crds), &bh.CRDs); err != nil {
		return err
	}

	return nil
}

func (bh *BundleHelper) AddOldManifests(outDir string, currentCSVVersion string) error {

	if err := bh.addOldCRDs(outDir); err != nil {
		return err
	}

	if err := bh.addOldCSVs(outDir, currentCSVVersion); err != nil {
		return err
	}

	return nil

}

func (bh *BundleHelper) addOldCRDs(outDir string) error {

	currentVersion := v1.KubeVirtGroupVersionKind.Version
	for _, crd := range bh.CRDs {
		if crd.Spec.Version == currentVersion {
			// the current version wil be generated
			continue
		}
		// write old CRD to the out dir
		bytes, err := json.Marshal(crd)
		if err != nil {
			return err
		}
		filename := fmt.Sprintf("%v/%v-%v.crd.yaml", outDir, crd.Name, crd.Spec.Version)
		err = ioutil.WriteFile(filename, bytes, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func (bh *BundleHelper) addOldCSVs(outDir string, currentCSVVersion string) error {

	for _, csv := range bh.CSVs {

		version := GetCSVVersion(csv)

		if version == currentCSVVersion {
			// the current version wil be generated
			continue
		}
		// write old CSV to the out dir
		bytes, err := yaml.Marshal(csv)
		if err != nil {
			return err
		}
		filename := fmt.Sprintf("%v/%v.csv.yaml", outDir, GetCSVName(csv))
		err = ioutil.WriteFile(filename, bytes, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetCSVVersion(csv yaml.MapSlice) string {
	for _, rootItem := range csv {
		if rootItem.Key == "spec" {
			if spec, ok := rootItem.Value.(yaml.MapSlice); ok {
				for _, specItem := range spec {
					if specItem.Key == "version" {
						return specItem.Value.(string)
					}
				}
			}
		}
	}
	return ""
}

func GetCSVName(csv yaml.MapSlice) string {
	for _, rootItem := range csv {
		if rootItem.Key == "metadata" {
			if metadata, ok := rootItem.Value.(yaml.MapSlice); ok {
				for _, metadataItem := range metadata {
					if metadataItem.Key == "name" {
						return metadataItem.Value.(string)
					}
				}
			}
		}
	}
	return ""
}

func (bh *BundleHelper) GetLatestPublishedCSVVersion() string {
	if len(bh.Pkgs) == 0 {
		return ""
	}
	// for the moment there is only 1 package with 1 channel
	return bh.Pkgs[0].Channels[0].CurrentCSV
}
