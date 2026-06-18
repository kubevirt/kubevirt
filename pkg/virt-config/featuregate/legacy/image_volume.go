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

package legacy

import (
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// Owner: @Barakmor1
// Alpha: v1.6.0
// Beta: v1.7.0
//
// ImageVolume uses Kubernetes ImageVolume FG to eliminate
// the need for an extra container for containerDisk, improving security by avoiding
// bind mounts in virt-handler.
const ImageVolume = "ImageVolume"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: ImageVolume, State: featuregate.Beta})
}

// ImageVolumeEnabled returns true when the ImageVolume feature gate is enabled.
func (g LegacyFeatureGates) ImageVolumeEnabled() bool {
	return featuregate.GateEnabled(ImageVolume, g.ConfigReader)
}
