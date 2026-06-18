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

package storage

import "kubevirt.io/kubevirt/pkg/virt-config/featuregate"

// Owner: sig-storage
// Alpha: v0.36.0
// Deprecated: v1.9.0
const HotplugVolumesGate = "HotplugVolumes"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{
		Name:    HotplugVolumesGate,
		State:   featuregate.Deprecated,
		Message: "HotplugVolumes has been deprecated since v1.9.0 and has been replaced by DeclarativeHotplugVolumes",
	})
}

// HotplugVolumesEnabled returns true when the HotplugVolumes feature gate is enabled.
func (g StorageFeatureGates) HotplugVolumesEnabled() bool {
	return featuregate.GateEnabled(HotplugVolumesGate, g.ConfigReader)
}
