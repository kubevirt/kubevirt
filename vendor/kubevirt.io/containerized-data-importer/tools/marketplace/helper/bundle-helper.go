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

	"github.com/operator-framework/operator-marketplace/pkg/datastore"
	v1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"

	"k8s.io/klog"
)

//Channel from which OLM bundle is consumed
type Channel struct {
	Name       string `yaml:"name"`
	CurrentCSV string `yaml:"currentCSV"`
}

//Pkg - Package to be consumed
type Pkg struct {
	PkgName  string    `yaml:"packageName"`
	Channels []Channel `yaml:"channels"`
}

//Data - OLM bundle wrapper
type Data struct {
	CSVs     string `yaml:"clusterServiceVersions"`
	CRDs     string `yaml:"customResourceDefinitions"`
	Packages string `yaml:"packages"`
}

//Bundle - OLM bundle
type Bundle struct {
	Data Data `yaml:"data"`
}

//BundleHelper - object that provides logic of fetching OLM bundle from marketplace quay repo
type BundleHelper struct {
	repo      string
	namespace string
	Pkgs      []Pkg
	CRDs      []v1beta1.CustomResourceDefinition
	CSVs      []yaml.MapSlice
}

//NewBundleHelper - Ctor
func NewBundleHelper(repo, namespace string) (*BundleHelper, error) {
	bh := &BundleHelper{
		repo:      repo,
		namespace: namespace,
	}
	if err := bh.downloadAndParseBundle(); err != nil {
		klog.Errorf("Failed to  download bundle %v", err)
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
		klog.Errorf("Failed to  create new appregistry client %v", err)
		return err
	}

	// get latest bundle info
	bundles, err := client.ListPackages(bh.namespace)
	if err != nil {
		klog.Errorf("Failed to  list packages %v", err)
		return err
	}

	if len(bundles) == 0 {
		klog.Infoln("no new bundles")
		return nil
	}
	cdiBundles, err := bh.getRepoBundles(bundles)
	if err != nil {
		klog.Errorf("Failed to  get repo bundles %v", err)
		return err
	}

	if len(cdiBundles) == 0 || cdiBundles[0] == nil {
		klog.Infoln("no new bundles")
		return nil
	}

	bundleMetaData := cdiBundles[0]

	// download bundle
	data, err := client.RetrieveOne(bundleMetaData.ID(), bundleMetaData.Release)
	if err != nil {
		klog.Errorf("Failed to retriev bundle data %v", err)
		return err
	}

	// parse bundle into package, CRDs and CSVs
	bundle := Bundle{}
	if err := yaml.Unmarshal(data.RawYAML, &bundle); err != nil {
		klog.Errorf("Failed to unmarshal bundle data %v", err)
		return err
	}

	if err := yaml.Unmarshal([]byte(bundle.Data.Packages), &bh.Pkgs); err != nil {
		klog.Errorf("Failed to unmarshal package data %v", err)
		return err
	}

	if err := yaml.Unmarshal([]byte(bundle.Data.CSVs), &bh.CSVs); err != nil {
		klog.Errorf("Failed to unmarshal CSV data %v", err)
		return err
	}

	// use k8s json unmarshaller for CRDs for filling metadata correctly
	crds, err := yaml2.YAMLToJSON([]byte(bundle.Data.CRDs))
	if err != nil {
		klog.Errorf("Failed to convert CRD data to json format %v", err)
		return err
	}
	if err := json.Unmarshal([]byte(crds), &bh.CRDs); err != nil {
		klog.Errorf("Failed to unmarshal CRD data %v", err)
		return err
	}

	return nil
}

func (bh *BundleHelper) getRepoBundles(bundles []*datastore.RegistryMetadata) ([]*datastore.RegistryMetadata, error) {

	list := make([]*datastore.RegistryMetadata, 0)
	for _, bundle := range bundles {
		if bh.repo == bundle.Repository {
			metadata := &datastore.RegistryMetadata{
				Namespace:  bundle.Namespace,
				Repository: bundle.Repository,

				// 'Default' points to the latest release pushed.
				Release: bundle.Release,

				// Getting 'Digest' would require an additional call to the app
				// registry, so it is being defaulted.
			}
			list = append(list, metadata)
		} //copy bundle info only if repo matches
	}

	return list, nil
}

//AddOldManifests - downloads old CRDs and CSVs to outDir with respect to currentCSVVersion
func (bh *BundleHelper) AddOldManifests(outDir string, currentCSVVersion string) error {

	if err := bh.addOldCRDs(outDir); err != nil {
		klog.Errorf("Failed to add old CRDs to %s, %v", outDir, err)
		return err
	}

	if err := bh.addOldCSVs(outDir, currentCSVVersion); err != nil {
		klog.Errorf("Failed to add old CSVs to %s, %v", outDir, err)
		return err
	}

	return nil

}

func (bh *BundleHelper) addOldCRDs(outDir string) error {

	currentVersion := v1.CDIGroupVersionKind.Version
	for _, crd := range bh.CRDs {
		if crd.Spec.Version == currentVersion {
			// the current version wil be generated
			continue
		}
		// write old CRD to the out dir
		bytes, err := json.Marshal(crd)
		if err != nil {
			klog.Errorf("Failed to marshal old CRDs %v", err)
			return err
		}
		filename := fmt.Sprintf("%v/%v-%v.crd.yaml", outDir, crd.Name, crd.Spec.Version)
		err = ioutil.WriteFile(filename, bytes, 0644)
		if err != nil {
			klog.Errorf("Failed to write old CRDs in %s %v", filename, err)
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
			klog.Errorf("Failed to marshal old CSVs %v", err)
			return err
		}
		filename := fmt.Sprintf("%v/%v.csv.yaml", outDir, GetCSVName(csv))
		err = ioutil.WriteFile(filename, bytes, 0644)
		if err != nil {
			klog.Errorf("Failed to write old CSVs in %s, %v", filename, err)
			return err
		}
	}

	return nil
}

//GetCSVVersion - retrievs CSV Version from provided CSV manifest
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

//GetCSVName -retrieves CSV name from provided CSV manifest
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

//GetLatestPublishedCSVVersion - return latest CSV version that exists in quay
func (bh *BundleHelper) GetLatestPublishedCSVVersion() string {
	if len(bh.Pkgs) == 0 {
		return ""
	}
	// for the moment there is only 1 package with 1 channel
	return bh.Pkgs[0].Channels[0].CurrentCSV
}

//VerifyNotPublishedCSVVersion - returns latest CSV version that exists
func (bh *BundleHelper) VerifyNotPublishedCSVVersion(currentCSVVersion string) bool {

	for _, csv := range bh.CSVs {

		version := GetCSVVersion(csv)

		if version == currentCSVVersion {
			// the current version already exist
			klog.Infof("CSV version %s already published in marketplace", currentCSVVersion)
			return false
		}
	}

	return true

}
