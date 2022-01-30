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

	ext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	yaml2 "github.com/ghodss/yaml"
	"github.com/operator-framework/operator-marketplace/pkg/appregistry"
	"gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/util/json"

	v1 "kubevirt.io/api/core/v1"
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
	namespace   string
	packageName string
	Pkgs        []Pkg
	CRDs        []extv1.CustomResourceDefinition
	CSVs        []yaml.MapSlice
}

func NewBundleHelper(namespace string, packageName string) (*BundleHelper, error) {
	bh := &BundleHelper{
		namespace:   namespace,
		packageName: packageName,
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
	bundles, err := client.ListPackages(bh.namespace)
	if err != nil {
		return err
	}

	if len(bundles) == 0 {
		fmt.Println("no old bundles found")
		return nil
	}

	// since other projects are also pushing their bundle to the kubevirt namespace now, we need to find our own bundle
	for _, bundleMetaData := range bundles {

		// Quay repository name is always equal to package name
		if bundleMetaData.Repository != bh.packageName {
			fmt.Printf("skipping bundle: %v\n", bundleMetaData.Repository)
			continue
		}

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
		if err := json.Unmarshal(crds, &bh.CRDs); err != nil {
			// check if these are v1beta1 CRDs
			var v1beta1Crds []extv1beta1.CustomResourceDefinition
			if err = json.Unmarshal(crds, &v1beta1Crds); err != nil {
				return err
			}
			for _, v1beta1Crd := range v1beta1Crds {
				crd := &ext.CustomResourceDefinition{}
				if err = extv1beta1.Convert_v1beta1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(&v1beta1Crd, crd, nil); err != nil {
					return err
				}
				v1Crd := extv1.CustomResourceDefinition{}
				if err = extv1.Convert_apiextensions_CustomResourceDefinition_To_v1_CustomResourceDefinition(crd, &v1Crd, nil); err != nil {
					return err
				}
				bh.CRDs = append(bh.CRDs, v1Crd)
			}
		}

		// we found kubevirt, so no need to go on
		return nil
	}

	fmt.Println("no old kubevirt bundle found")
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
		for _, version := range crd.Spec.Versions {
			if version.Name == currentVersion {
				// the current version wil be generated
				continue
			}
			// write old CRD to the out dir
			bytes, err := json.Marshal(crd)
			if err != nil {
				return err
			}
			filename := fmt.Sprintf("%v/%v-%v.crd.yaml", outDir, crd.Name, version)
			err = ioutil.WriteFile(filename, bytes, 0644)
			if err != nil {
				return err
			}
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
