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
	"os"
	"path/filepath"
	"strings"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/api"

	util "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

const (
	isUnusable             string = "no"
	isRequired             string = "require"
	nodeLabellerVolumePath        = "/var/lib/kubevirt-node-labeller/"

	supportedFeaturesXml = "supported_features.xml"
)

func (n *NodeLabeller) getMinCpuFeature() cpuFeatures {
	minCPUModel := n.clusterConfig.GetMinCPUModel()
	if minCPUModel == "" {
		minCPUModel = util.DefaultMinCPUModel
	}
	return n.cpuInfo.models[minCPUModel]
}

func (n *NodeLabeller) getSupportedCpuModels() []string {
	supportedCPUModels := make([]string, 0)

	obsoleteCPUsx86 := n.clusterConfig.GetObsoleteCPUModels()
	if obsoleteCPUsx86 == nil {
		obsoleteCPUsx86 = util.DefaultObsoleteCPUModels
	}

	for _, model := range n.hostCapabilities.items {
		if _, ok := obsoleteCPUsx86[model]; ok {
			continue
		}
		supportedCPUModels = append(supportedCPUModels, model)
	}

	return supportedCPUModels
}

func (n *NodeLabeller) getSupportedCpuFeatures() cpuFeatures {
	supportedCpuFeatures := make(cpuFeatures)
	minCpuFeatures := n.getMinCpuFeature()

	for _, feature := range n.supportedFeatures {
		if _, exist := minCpuFeatures[feature]; !exist {
			supportedCpuFeatures[feature] = true
		}
	}

	return supportedCpuFeatures
}

func (n *NodeLabeller) getHostCpuModel() hostCPUModel {
	return n.hostCPUModel
}

//loadDomCapabilities loads info about cpu models, which can host emulate
func (n *NodeLabeller) loadDomCapabilities() error {
	hostDomCapabilities, err := n.getDomCapabilities()
	if err != nil {
		return err
	}

	usableModels := make([]string, 0)
	minCpuFeatures := n.getMinCpuFeature()
	log.Log.Infof("CPU features of a minimum baseline CPU model: %+v", minCpuFeatures)
	for _, mode := range hostDomCapabilities.CPU.Mode {
		if mode.Name == v1.CPUModeHostModel {
			n.cpuModelVendor = mode.Vendor.Name

			hostCpuModel := mode.Model[0]
			if len(mode.Model) > 0 {
				log.Log.Warning("host model mode is expected to contain only one model")
			}

			n.hostCPUModel.name = hostCpuModel.Name
			n.hostCPUModel.fallback = hostCpuModel.Fallback

			for _, feature := range mode.Feature {
				if _, isMinCpuFeature := minCpuFeatures[feature.Name]; !isMinCpuFeature && feature.Policy == isRequired {
					n.hostCPUModel.requiredFeatures[feature.Name] = true
				}
			}
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
	featuresFile := filepath.Join(n.volumePath, supportedFeaturesXml)

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

func (n *NodeLabeller) loadHostCapabilities() error {
	capsFile := filepath.Join(n.volumePath, "capabilities.xml")
	n.capabilities = &api.Capabilities{}
	err := n.getStructureFromXMLFile(capsFile, n.capabilities)
	if err != nil {
		return err
	}
	return nil
}

//loadCPUInfo load info about all cpu models
func (n *NodeLabeller) loadCPUInfo() error {
	files, err := os.ReadDir(filepath.Join(n.volumePath, "cpu_map"))
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
	domCapabilitiesFile := filepath.Join(n.volumePath, n.domCapabilitiesFileName)
	hostDomCapabilities := HostDomCapabilities{}
	err := n.getStructureFromXMLFile(domCapabilitiesFile, &hostDomCapabilities)

	return hostDomCapabilities, err
}

//LoadFeatures loads features for given cpu name
func (n *NodeLabeller) loadFeatures(fileName string) (cpuFeatures, error) {
	if fileName == "" {
		return nil, fmt.Errorf("file name can't be empty")
	}

	cpuFeaturepath := getPathCPUFeatures(n.volumePath, fileName)
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
func getPathCPUFeatures(volumePath string, name string) string {
	return filepath.Join(volumePath, "cpu_map", name)
}

//GetStructureFromXMLFile load data from xml file and unmarshals them into given structure
//Given structure has to be pointer
func (n *NodeLabeller) getStructureFromXMLFile(path string, structure interface{}) error {
	rawFile, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	n.logger.V(4).Infof("node-labeller - loading data from xml file: %#v", string(rawFile))

	return xml.Unmarshal(rawFile, structure)
}
