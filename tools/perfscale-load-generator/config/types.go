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
	"time"

	"kubevirt.io/client-go/kubecli"
)

// VMISpec describe the VMI configuration
type VMISpec struct {
	// VMImage is the image type will reflect in the storage size
	VMImage string `yaml:"vm_image_name"`
	// CPULimit is the number of CPUs to allocate (100m = .1 cores)
	CPULimit string `yaml:"cpu_limit"`
	// MEMLimit is the amount of Memory to allocate
	MEMLimit string `yaml:"mem_limit"`
}

// Scenario defines a load generator scenario
type Scenario struct {
	// UUID scenario id
	UUID string `yaml:"uuid" json:"uuid"`
	// Name scenario name
	Name string `yaml:"name" json:"name"`
	// Namespace namespace base name to use
	Namespace string `yaml:"namespace" json:"namespace"`
	// NamespacedIterations create a namespace per scenario iteration
	NamespacedIterations bool `yaml:"namespacedIterations" json:"namespacedIterations"`
	// VMISpec total number of VMs to be created
	VMISpec VMISpec `yaml:"vmiSpec" json:"vmiSpec"`
	// VMICount total number of VMs to be created
	VMICount int `yaml:"count" json:"count"`
	// IterationCount how many times to execute the scenario
	IterationCount int `yaml:"iterationCount" json:"iterationCount"`
	// IterationInterval how much time to wait between each scenario iteration
	IterationInterval time.Duration `yaml:"iterationInterval" json:"iterationInterval"`
	// IterationWaitForDeletion wait for VMIs to be deleted and all objects disapear in each iteration
	IterationWaitForDeletion bool `yaml:"iterationwaitForDeletion" json:"iterationWaitForDeletion"`
	// IterationVMIWait wait for all vmis to be running before moving forward to the next iteration
	IterationVMIWait bool `yaml:"iterationvmiWait" json:"iterationVmiWait"`
	// IterationCleanup clean up old tests, e.g., namespaces, nodes, configurations, before moving forward to the next iteration
	IterationCleanup bool `yaml:"iterationcleanup" json:"iterationcleanup"`
	// MaxWaitTimeout maximum wait period for all iterations
	MaxWaitTimeout time.Duration `yaml:"maxWaitTimeout" json:"maxWaitTimeout"`
	// QPS is the max number of queries per second
	QPS float64 `yaml:"qps" json:"qps"`
	// Burst is the maximum burst for throttle
	Burst int `yaml:"burst" json:"burst"`
	// WaitWhenFinished delays the termination of the scenario
	WaitWhenFinished time.Duration `yaml:"waitWhenFinished" json:"waitWhenFinished"`
}

// ConfigSpec is the test configuration specification
type ConfigSpec struct {
	// ClientSet defines global configuration parameters
	ClientSet kubecli.KubevirtClient `yaml:"kubevirtClient,omitempty" json:"kubevirtClient,omitempty"`
	// Scenario defines how the load generator will create VMs
	Scenario Scenario `yaml:"scenarios" json:"scenarios"`
}
