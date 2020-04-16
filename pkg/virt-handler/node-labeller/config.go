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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//Config holds data about obsolete cpus and minimal baseline cpus
type Config struct {
	ObsoleteCPUs []string `yaml:"obsoleteCPUs"`
	MinCPU       string   `yaml:"minCPU"`
}

var configPath = "/var/lib/kubevirt-node-labeller/cpu-plugin-configmap.yaml"

//LoadConfig loads config yaml file with obsolete cpus and minimal baseline cpus
func (n *NodeLabeller) loadConfig() (Config, error) {
	config := Config{}
	cm, err := n.clientset.CoreV1().ConfigMaps(n.namespace).Get("kubevirt-cpu-plugin-configmap", metav1.GetOptions{})
	if err != nil {
		return config, err
	}

	if value, ok := cm.Data["cpu-plugin-configmap.yaml"]; ok {
		err := writeConfigFile(configPath, value)
		if err != nil {
			return config, err
		}
	}
	err = getStructureFromYamlFile(configPath, &config)
	if err != nil {
		return Config{}, err
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
