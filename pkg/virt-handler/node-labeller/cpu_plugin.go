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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package nodelabeller

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	util "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

const (
	isUnusable             string = "no"
	nodeLabellerVolumePath        = "/var/lib/kubevirt-node-labeller/"
)

// getCPUInfo processes all cpu info data and returns
// slice of usable cpu models and features.
func (n *NodeLabeller) getCPUInfo() ([]string, cpuFeatures) {
	minCPUModel := n.clusterConfig.GetMinCPUModel()
	if minCPUModel == "" {
		minCPUModel = util.DefaultMinCPUModel
	}

	obsoleteCPUsx86 := n.clusterConfig.GetObsoleteCPUModels()
	if obsoleteCPUsx86 == nil {
		obsoleteCPUsx86 = util.DefaultObsoleteCPUModels
	}

	basicFeaturesMap := n.cpuInfo.models[minCPUModel]

	cpus := make([]string, 0)
	features := make(cpuFeatures)

	for _, model := range n.hostCapabilities.items {
		if _, ok := obsoleteCPUsx86[model]; ok {
			continue
		}
		cpus = append(cpus, model)
	}

	for _, feature := range n.supportedFeatures {
		if _, exist := basicFeaturesMap[feature]; !exist {
			features[feature] = true
		}
	}

	return cpus, features
}

//loadHostCapabilities loads info about cpu models, which can host emulate
func (n *NodeLabeller) loadHostCapabilities() error {
	hostDomCapabilities, err := n.getDomCapabilities()
	if err != nil {
		return err
	}

	usableModels := make([]string, 0)
	for _, mode := range hostDomCapabilities.CPU.Mode {
		if mode.Vendor.Name != "" {
			n.cpuModelVendor = mode.Vendor.Name
		}

		for _, model := range mode.Model {
			if model.Usable == isUnusable || model.Usable == "" {
				continue
			}
			usableModels = append(usableModels, model.Name)
		}
	}

	n.hostCapabilities.items = usableModels

	return nil
}

//loadHostSupportedFeatures loads supported features
func (n *NodeLabeller) loadHostSupportedFeatures() error {
	featuresFile := path.Join(nodeLabellerVolumePath + "supported_features.xml")

	hostFeatures := SupportedHostFeature{}
	err := n.getStructureFromXMLFile(featuresFile, &hostFeatures)
	if err != nil {
		return err
	}

	usableFeatures := make([]string, 0)
	for _, f := range hostFeatures.Feature {
		if f.Policy != util.RequirePolicy {
			continue
		}

		usableFeatures = append(usableFeatures, f.Name)
	}

	n.supportedFeatures = usableFeatures
	return nil
}

//loadCPUInfo load info about all cpu models
func (n *NodeLabeller) loadCPUInfo() error {
	files, err := ioutil.ReadDir(path.Join(nodeLabellerVolumePath, "cpu_map"))
	if err != nil {
		return err
	}

	models := make(map[string]cpuFeatures)
	for _, f := range files {
		fileName := f.Name()
		if strings.HasPrefix(fileName, "x86_") {
			features, err := n.loadFeatures(fileName)
			if err != nil {
				return err
			}
			cpuName := strings.TrimSuffix(strings.TrimPrefix(fileName, "x86_"), ".xml")

			models[cpuName] = features
		}
	}

	n.cpuInfo.models = models
	return nil
}

func (n *NodeLabeller) getDomCapabilities() (HostDomCapabilities, error) {
	domCapabilitiesFile := path.Join(nodeLabellerVolumePath + "virsh_domcapabilities.xml")
	hostDomCapabilities := HostDomCapabilities{}
	err := n.getStructureFromXMLFile(domCapabilitiesFile, &hostDomCapabilities)

	return hostDomCapabilities, err
}

//LoadFeatures loads features for given cpu name
func (n *NodeLabeller) loadFeatures(fileName string) (cpuFeatures, error) {
	if fileName == "" {
		return nil, fmt.Errorf("file name can't be empty")
	}

	cpuFeaturepath := getPathCPUFeatures(fileName)
	features := FeatureModel{}
	err := n.getStructureFromXMLFile(cpuFeaturepath, &features)
	if err != nil {
		return nil, err
	}

	modelFeatures := cpuFeatures{}
	for _, f := range features.Model.Features {
		modelFeatures[f.Name] = true
	}
	return modelFeatures, nil
}

//getPathCPUFeatures creates path where folder with cpu models is
func getPathCPUFeatures(name string) string {
	return path.Join(nodeLabellerVolumePath, "cpu_map", name)
}

//GetStructureFromXMLFile load data from xml file and unmarshals them into given structure
//Given structure has to be pointer
func (n *NodeLabeller) getStructureFromXMLFile(path string, structure interface{}) error {
	rawFile, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	n.logger.V(4).Infof("node-labeller - loading data from xml file: %#v", string(rawFile))

	return xml.Unmarshal(rawFile, structure)
}
