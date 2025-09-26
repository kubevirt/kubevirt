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
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/hypervisor"
	virt_config "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libkubevirt"
)

// GetDevice returns the appropriate hypervisor device name
// based on the current KubeVirt configuration.
func GetDevice(virtClient kubecli.KubevirtClient) string {
	kv := libkubevirt.GetCurrentKv(virtClient)
	hypervisorName := virt_config.DefaultHypervisorName
	if checks.HasFeature(featuregate.ConfigurableHypervisor) && kv.Spec.Configuration.HypervisorConfiguration != nil {
		hypervisorName = kv.Spec.Configuration.HypervisorConfiguration.Name
	}
	return hypervisor.NewHypervisor(hypervisorName).GetDevice()

}
