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

package compute

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/iothreads"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
)

type IOThreadsDomainConfigurator struct {
	scsiMultiIOEnabled bool
	multiIOAutoEnabled bool
}

type ioThreadsOption func(*IOThreadsDomainConfigurator)

func NewIOThreadsDomainConfigurator(options ...ioThreadsOption) IOThreadsDomainConfigurator {
	var configurator IOThreadsDomainConfigurator
	for _, f := range options {
		f(&configurator)
	}
	return configurator
}

func IOThreadsWithSCSIMultiIOThreadEnabled(enabled bool) ioThreadsOption {
	return func(i *IOThreadsDomainConfigurator) {
		i.scsiMultiIOEnabled = enabled
	}
}

func IOThreadsWithMultiIOThreadAutoPolicyEnabled(enabled bool) ioThreadsOption {
	return func(i *IOThreadsDomainConfigurator) {
		i.multiIOAutoEnabled = enabled
	}
}

func (i IOThreadsDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if !iothreads.HasIOThreads(vmi) {
		return nil
	}

	totalThreads, autoThreads := iothreads.GetIOThreadsCountType(vmi)
	if i.scsiMultiIOEnabled && i.multiIOAutoEnabled &&
		vmi.Spec.Domain.IOThreadsPolicy != nil &&
		*vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicyAuto {
		vcpus := uint(vcpu.CalculateRequestedVCPUs(vcpu.GetCPUTopology(vmi)))
		dedicatedThreads := totalThreads - autoThreads
		totalThreads = iothreads.AutoPoolSize(autoThreads, vcpus) + dedicatedThreads
	}

	if totalThreads == 0 {
		return nil
	}

	if domain.Spec.IOThreads == nil {
		domain.Spec.IOThreads = &api.IOThreads{}
	}
	domain.Spec.IOThreads.IOThreads = uint(totalThreads)

	return nil
}
