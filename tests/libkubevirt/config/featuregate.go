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
	"slices"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
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
	kv := libkubevirt.GetCurrentKv(virtClient).DeepCopy()

	kv.Spec.Configuration.FeatureGates = slices.DeleteFunc(kv.Spec.Configuration.FeatureGates, func(v v1.FeatureGateConfiguration) bool {
		return v.Name == feature
	})
	kv.Spec.Configuration.FeatureGates = append(kv.Spec.Configuration.FeatureGates, v1.FeatureGateConfiguration{
		Name:    feature,
		Enabled: pointer.P(toEnable),
	})

	UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
}
