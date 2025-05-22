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

package config

import (
	"fmt"
	"slices"
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
)

func DisableFeatureGate(feature string) {
	setFeatureGateState(feature, false)
}

func EnableFeatureGate(feature string) {
	setFeatureGateState(feature, true)
}

func setFeatureGateState(feature string, toEnable bool) {
	if toEnable == checks.HasFeature(feature) {
		return
	}
	virtClient := kubevirt.Client()

	kv := libkubevirt.GetCurrentKv(virtClient)
	if kv.Spec.Configuration.DeveloperConfiguration == nil {
		kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{
			FeatureGates: []string{},
		}
	}

	featureGates := slices.DeleteFunc(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates, func(definedFeatureGate string) bool {
		return definedFeatureGate == feature || strings.HasPrefix(definedFeatureGate, feature+"=")
	})

	featureGates = append(featureGates, fmt.Sprintf("%s=%t", feature, toEnable))
	kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = featureGates

	UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
}
