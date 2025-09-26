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
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type Hypervisor interface {
	AdjustDomain(vmi *v1.VirtualMachineInstance, domain *api.Domain)
}

func NewHypervisor(hypervisor string) Hypervisor {
	switch hypervisor {
	case v1.HyperVLayeredHypervisorName:
		log.Log.Info("Creating Hypervisor instance for hyperv-layered implementation")
		return &HyperVLayeredHypervisor{}
	default:
		log.Log.Infof("Creating Hypervisor instance for default KVM implementation. Provided hypervisor: %s", hypervisor)
		return &KVMHypervisor{}
	}
}
