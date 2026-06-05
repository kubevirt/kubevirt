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
	"io"
	"os"
	"time"

	"kubevirt.io/kubevirt/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tools/perfscale-load-generator/flags"
)

const (
	DefaultMinSleepChurn = 20 * time.Second
	RequestTimeout       = 15 * time.Second

	// Set no limit for QPS and Burst
	Burst = 0
	QPS   = 0
)

// NewWorkload reads the test configuration file
func NewWorkload(testConfigPath string) *Workload {
	data, err := os.ReadFile(testConfigPath)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	w := &Workload{}
	if err := yaml.Unmarshal(data, w); err != nil {
		fmt.Println(err)
		panic(err)
	}
	w.LoadObjTemplate()
	return w
}

// LoadObjTemplate reads a YAML file with the object template for each object defined in the workload's jobs
func (w *Workload) LoadObjTemplate() {
	rootDir := flags.GetRootConfigDir()

	if w.Object != nil {
		templateFilePath := fmt.Sprintf("%s/%s", rootDir, w.Object.TemplateFile)
		f, err := os.Open(templateFilePath)
		if err != nil {
			panic(fmt.Errorf("unexpected error opening file %s: %v", templateFilePath, err))
		}
		objectTemplate, err := io.ReadAll(f)
		defer util.CloseIOAndCheckErr(f, nil)
		if err != nil {
			panic(fmt.Errorf("unexpected error reading file %s: %v", templateFilePath, err))
		}
		w.Object.ObjectTemplate = objectTemplate
	}
}

// NewKubeClient
func NewKubevirtClient() kubecli.KubevirtClient {
	config, err := clientcmd.BuildConfigFromFlags("", flags.Kubeconfig)
	if err != nil {
		panic(err)
	}
	config.QPS = QPS
	config.Burst = Burst
	config.Timeout = RequestTimeout
	clientSet, err := kubecli.GetKubevirtClientFromRESTConfig(config)
	if err != nil {
		panic(fmt.Errorf("unexpected error creating kubevirt client: %v", err))
	}
	return clientSet
}

func GetListOpts(label string, uuid string) *metav1.ListOptions {
	listOpts := &metav1.ListOptions{}
	listOpts.LabelSelector = fmt.Sprintf("%s=%s", label, uuid)
	return listOpts
}

func AddLabels(obj *unstructured.Unstructured, uuid string) {
	labels := map[string]string{
		WorkloadUUIDLabel: uuid,
	}
	obj.SetLabels(labels)
}
