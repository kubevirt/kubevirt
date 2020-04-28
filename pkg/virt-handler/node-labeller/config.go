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
 * Copyright 2020 Red Hat, Inc.
 */

package nodelabeller

import (
	"strings"

	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
)

//Config holds data about obsolete cpus and minimal baseline cpus
type Config struct {
	ObsoleteCPUs []string `yaml:"obsoleteCPUs"`
	MinCPU       string   `yaml:"minCPU"`
}

//LoadConfig loads config yaml file with obsolete cpus and minimal baseline cpus
func (n *NodeLabeller) loadConfig() (Config, error) {
	config := Config{}

	labellerCMKey := ""
	for _, key := range n.configMapInformer.GetStore().ListKeys() {
		if strings.Contains(key, "kubevirt-cpu-plugin-configmap") {
			labellerCMKey = key
		}
	}

	if labellerCMKey == "" {
		return config, nil
	}

	cmObj, exists, err := n.configMapInformer.GetStore().GetByKey(labellerCMKey)
	if !exists || err != nil {
		return config, err
	}
	var (
		cm *v1.ConfigMap
		ok bool
	)
	if cm, ok = cmObj.(*v1.ConfigMap); !ok {
		return config, nil
	}

	if value, ok := cm.Data["cpu-plugin-configmap.yaml"]; ok {
		err := yaml.Unmarshal([]byte(value), &config)
		if err != nil {
			return config, err
		}
	}

	return config, nil
}

//GetObsoleteCPUMap returns map of obsolete cpus
func (c *Config) getObsoleteCPUMap() map[string]bool {
	return convertStringSliceToMap(c.ObsoleteCPUs)
}

//GetMinCPU returns minimal baseline cpu. If minimal cpu is not defined,
//it returns for Intel vendor Penryn cpu model, for AMD it returns Opteron_G1.
func (c *Config) getMinCPU() string {
	return c.MinCPU
}
