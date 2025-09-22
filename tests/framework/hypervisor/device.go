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

package hypervisor

import (
	k8sv1 "k8s.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/tests/libkubevirt"
)

// GetDevice returns the appropriate hypervisor device resource name
// based on the current KubeVirt configuration. If HyperVLayered feature gate
// is enabled, it returns HyperVDevice, otherwise KvmDevice.
func GetDevice(virtClient kubecli.KubevirtClient) k8sv1.ResourceName {
	kv := libkubevirt.GetCurrentKv(virtClient)
	if kv.Spec.Configuration.DeveloperConfiguration != nil {
		featureGates := kv.Spec.Configuration.DeveloperConfiguration.FeatureGates
		for _, fg := range featureGates {
			if fg == featuregate.HyperVLayered {
				return services.HyperVDevice
			}
		}
	}
	return services.KvmDevice
}
