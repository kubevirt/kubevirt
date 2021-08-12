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
	"encoding/json"
	"errors"
	"time"

	"kubevirt.io/client-go/kubecli"
)

// Object template global input variable name
const (
	// Replicas number of replicas to create of the given object
	Replica = "replica"
	// Iteration how many times to execute the workload
	Iteration = "iteration"
	// Namespace prefix to be create to create objects
	Namespace = "namespace"
	// WorkloadLabel identifies all namespaces and objects created within the workload
	// It is mostly used to cleanup
	WorkloadLabel = "kubevirt-load-generator-workload"
)

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
	// Replicas number of replicas to create of the given object
	Replicas int `yaml:"replicas" json:"replicas,omitempty"`
	// InputVars contains a map of arbitrary user-define input variables
	// that can be introduced in the template by users
	InputVars map[string]interface{} `yaml:"inputVars" json:"inputVars,omitempty"`
}

// Workload defines a load generator workload to create a list of objects
type Workload struct {
	// UUID workload id
	UUID string `yaml:"uuid" json:"uuid,omitempty"`
	// Name workload name
	Name string `yaml:"name" json:"name"`
	// Objects list of object spec
	Objects []*ObjectSpec `yaml:"objects" json:"objects"`
	// NamespacedIterations create a namespace per workload iteration
	NamespacedIterations bool `yaml:"namespacedIterations" json:"namespacedIterations,omitempty"`
	// IterationCount how many times to execute the workload
	IterationCount int `yaml:"iterationCount" json:"iterationCount"`
	// IterationInterval how much time to wait between each workload iteration
	IterationInterval Duration `yaml:"iterationInterval" json:"iterationInterval,omitempty"`
	// IterationCleanup clean up old tests, e.g., namespaces, nodes, configurations, before moving forward to the next iteration
	IterationCleanup bool `yaml:"iterationCleanup" json:"iterationCleanup,omitempty"`
	// IterationCreationWait wait for all objects to be running before moving forward to the next iteration
	IterationCreationWait bool `yaml:"iterationCreationWait" json:"iterationCreationWait,omitempty"`
	// IterationDeletionWait wait for objects to be deleted in each iteration
	IterationDeletionWait bool `yaml:"iterationDeletionWait" json:"iterationDeletionWait,omitempty"`
	// MaxWaitTimeout maximum wait period for all iterations
	MaxWaitTimeout Duration `yaml:"maxWaitTimeout" json:"maxWaitTimeout,omitempty"`
	// QPS is the max number of queries per second to control the job creation rate
	QPS float64 `yaml:"qps" json:"qps"`
	// Burst is the maximum burst for throttle to control the job creation rate
	Burst int `yaml:"burst" json:"burst"`
	// WaitWhenFinished delays the termination of the workload
	WaitWhenFinished Duration `yaml:"waitWhenFinished" json:"waitWhenFinished,omitempty"`
}

// GlobalConfig defined the kubernetes client-go configuration
type GlobalConfig struct {
	// Client defines global configuration parameters
	Client kubecli.KubevirtClient `yaml:"kubevirtClient,omitempty" json:"kubevirtClient,omitempty"`
	// QPS is the max number of queries per second to configure the kubernetes client-go
	QPS float64 `yaml:"qps" json:"qps"`
	// Burst is the maximum burst for throttle to configure the kubernetes client-go
	Burst int `yaml:"burst" json:"burst"`
}

// TestConfig is the test configuration specification
type TestConfig struct {
	// Global defines global configuration parameters
	Global GlobalConfig `yaml:"globalConfig" json:"globalConfig"`
	// Workloads contains a set of jobs that define how the load generator will create objects (e.g. VMs, VMIs, etc)
	Workloads []Workload `yaml:"workloads" json:"workloads"`
}
