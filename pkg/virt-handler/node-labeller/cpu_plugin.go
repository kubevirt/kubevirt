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
 * Copyright The KubeVirt Authors.
 *
 */

package nodelabeller

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

const (
	isSupported            string = "yes"
	isUnusable             string = "no"
	isRequired             string = "require"
	NodeLabellerVolumePath        = "/var/lib/kubevirt-node-labeller/"

	supportedFeaturesXml = "supported_features.xml"
)

func (n *NodeLabeller) getSupportedCpuModels(obsoleteCPUsx86 map[string]bool) []string {
	supportedCPUModels := make([]string, 0)

	if obsoleteCPUsx86 == nil {
		obsoleteCPUsx86 = DefaultObsoleteCPUModels
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

	for _, feature := range n.supportedFeatures {
		supportedCpuFeatures[feature] = true
	}

	return supportedCpuFeatures
}

func (n *NodeLabeller) GetHostCpuModel() hostCPUModel {
	return n.hostCPUModel
}

// loadDomCapabilities loads info about cpu models, which can host emulate
func (n *NodeLabeller) loadDomCapabilities() error {
	hostDomCapabilities, err := n.getDomCapabilities()
	if err != nil {
		return err
	}

	usableModels := make([]string, 0)
	for _, mode := range hostDomCapabilities.CPU.Mode {
		if mode.Name == v1.CPUModeHostModel {
			if !n.arch.supportsHostModel() {
				log.Log.Warningf("host-model cpu mode is not supported for %s architecture", n.arch.arch())
				continue
			}

			n.cpuModelVendor = mode.Vendor.Name
			if n.cpuModelVendor == "" {
				n.cpuModelVendor = n.arch.defaultVendor()
			}

			if len(mode.Model) < 1 {
				return fmt.Errorf("host model mode is expected to contain a model")
			}
			if len(mode.Model) > 1 {
				log.Log.Warning("host model mode is expected to contain only one model")
			}

			hostCpuModel := mode.Model[0]
			n.hostCPUModel.Name = hostCpuModel.Name
			n.hostCPUModel.fallback = hostCpuModel.Fallback

			for _, feature := range mode.Feature {
				if feature.Policy == isRequired {
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
	n.SEV = hostDomCapabilities.SEV

	return nil
}

// loadHostSupportedFeatures loads supported features
func (n *NodeLabeller) loadHostSupportedFeatures() error {
	featuresFile := filepath.Join(n.volumePath, supportedFeaturesXml)

	hostFeatures := SupportedHostFeature{}
	err := n.getStructureFromXMLFile(featuresFile, &hostFeatures)
	if err != nil {
		return err
	}

	usableFeatures := make([]string, 0)
	for _, f := range hostFeatures.Feature {
		if n.arch.requirePolicy(f.Policy) {
			usableFeatures = append(usableFeatures, f.Name)
		}
	}

	n.supportedFeatures = usableFeatures
	return nil
}

func (n *NodeLabeller) getDomCapabilities() (HostDomCapabilities, error) {
	domCapabilitiesFile := filepath.Join(n.volumePath, n.domCapabilitiesFileName)
	hostDomCapabilities := HostDomCapabilities{}
	err := n.getStructureFromXMLFile(domCapabilitiesFile, &hostDomCapabilities)
	if err != nil {
		return hostDomCapabilities, err
	}

	if hostDomCapabilities.SEV.Supported == "yes" && hostDomCapabilities.SEV.MaxESGuests > 0 {
		hostDomCapabilities.SEV.SupportedES = "yes"
	} else {
		hostDomCapabilities.SEV.SupportedES = "no"
	}

	return hostDomCapabilities, err
}

// GetStructureFromXMLFile load data from xml file and unmarshals them into given structure
// Given structure has to be pointer
func (n *NodeLabeller) getStructureFromXMLFile(path string, structure interface{}) error {
	rawFile, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	n.logger.V(4).Infof("node-labeller - loading data from xml file: %#v", string(rawFile))

	return xml.Unmarshal(rawFile, structure)
}
