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
 */

package testsuite

import (
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/libkubevirt"
)

var (
	TestRunConfiguration RunConfiguration
)

type RunConfiguration struct {
	WarningToIgnoreList []string
}

func initRunConfiguration(virtClient kubecli.KubevirtClient) {
	kv := libkubevirt.GetCurrentKv(virtClient)
	runConfig := RunConfiguration{}
	if kv.Spec.Configuration.EvictionStrategy != nil &&
		*kv.Spec.Configuration.EvictionStrategy == v1.EvictionStrategyLiveMigrate {
		runConfig.WarningToIgnoreList = append(runConfig.WarningToIgnoreList, "EvictionStrategy is set but vmi is not migratable")
	}

	TestRunConfiguration = runConfig
}
