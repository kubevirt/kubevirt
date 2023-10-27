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
 * Copyright the KubeVirt Authors.
 *
 */

package config

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

// Object template global input variable name
const (
	Replica   = "replica"
	Namespace = "namespace"
)

// Default config values
const (
	// WorkloadUUIDLabel identifies all namespaces and objects created within the workload
	WorkloadUUIDLabel = "kubevirt-load-generator-workload"
	Type              = "burst"

	Timeout = time.Duration(5 * time.Minute)
)

var (
	ContainerPrefix = "registry:5000/kubevirt"
	ContainerTag    = "devel"
)

func init() {
	if dockerPrefixEnv := os.Getenv("DOCKER_PREFIX"); dockerPrefixEnv != "" {
		ContainerPrefix = dockerPrefixEnv
	}
	if dockerTagEnv := os.Getenv("DOCKER_TAG"); dockerTagEnv != "" {
		ContainerTag = dockerTagEnv
	}
}

type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}

// ObjectSpec defines an object spec that the load generator will create (e.g. VMs, VMIs, etc)
type ObjectSpec struct {
	// TemplateFile relative path to a valid YAML definition of a kubevirt resource
	TemplateFile string `yaml:"templateFile" json:"templateFile,omitempty"`
	// ObjectTemplate contains the object template that must be fill with global and arbitrary user-defined input variables
	ObjectTemplate []byte `yaml:"objectTemplate" json:"objectTemplate,omitempty"`
	// InputVars contains a map of arbitrary user-define input variables
	// that can be introduced in the template by users
	InputVars map[string]interface{} `yaml:"inputVars" json:"inputVars,omitempty"`
}

type TestType string

// Workload defines a load generator workload
type Workload struct {
	Name          string      `yaml:"name" json:"name"`
	Object        *ObjectSpec `yaml:"object" json:"object"`
	Type          TestType    `yaml:"type" json:"type"`
	Timeout       Duration    `yaml:"timeout" json:"timeout,omitempty"`
	Count         int         `yaml:"count" json:"count"`
	Churn         int         `yaml:"churn" json:"churn,omitempty"`
	MinChurnSleep *Duration   `yaml:"minChurnSleep" json:"minChurnSleep,omitempty"`
}
