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
 * Copyright 2021 IBM, Inc.
 *
 */

package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ghodss/yaml"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tools/perfscale-load-generator/flags"
)

const RequestTimeout = 15 * time.Second

// NewConfig returns a new config object
func NewConfig() *TestConfig {
	t := &TestConfig{}
	t.LoadWorkloadConfig(flags.WorkloadConfigFile)
	t.LoadObjTemplate()
	t.LoadKubeClient()
	return t
}

//LoadWorkloadConfig reads the test configuration file
func (t *TestConfig) LoadWorkloadConfig(testConfigPath string) {
	data, err := ioutil.ReadFile(testConfigPath)
	if err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal(data, t); err != nil {
		panic(err)
	}
}

// LoadObjTemplate reads an YAML file with the object template for each object defined in the workload's jobs
func (t *TestConfig) LoadObjTemplate() {
	rootDir := flags.GetRootConfigDir()
	for _, workload := range t.Workloads {
		for _, obj := range workload.Objects {
			templateFilePath := fmt.Sprintf("%s/%s", rootDir, obj.TemplateFile)
			f, err := os.Open(templateFilePath)
			if err != nil {
				panic(fmt.Errorf("unexpected error opening file %s: %v", templateFilePath, err))
			}
			objectTemplate, err := ioutil.ReadAll(f)
			if err != nil {
				panic(fmt.Errorf("unexpected error reading file %s: %v", templateFilePath, err))
			}
			obj.ObjectTemplate = objectTemplate
		}
	}
}

// LoadKubeClient configured the kubernetes client-go
func (t *TestConfig) LoadKubeClient() {
	config, err := clientcmd.BuildConfigFromFlags("", flags.Kubeconfig)
	if err != nil {
		panic(err)
	}
	config.QPS = float32(t.Global.QPS)
	config.Burst = t.Global.Burst
	config.Timeout = RequestTimeout
	clientSet, err := kubecli.GetKubevirtClientFromRESTConfig(config)
	if err != nil {
		panic(fmt.Errorf("unexpected error creating kubevirt client: %v", err))
	}
	t.Global.Client = clientSet
}
