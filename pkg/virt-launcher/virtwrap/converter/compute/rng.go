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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
)

type RNGDomainConfigurator struct {
	architecture          string
	useVirtioTransitional bool
	useLaunchSecuritySEV  bool
	useLaunchSecurityPV   bool
}

type rngOption func(*RNGDomainConfigurator)

func NewRNGDomainConfigurator(options ...rngOption) RNGDomainConfigurator {
	var configurator RNGDomainConfigurator

	for _, f := range options {
		f(&configurator)
	}

	return configurator
}

func (r RNGDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if vmi.Spec.Domain.Devices.Rng != nil {
		newRng := &api.Rng{}
		err := convert_v1_Rng_To_api_Rng(
			vmi.Spec.Domain.Devices.Rng,
			newRng,
			r.architecture,
			r.useVirtioTransitional,
			r.useLaunchSecuritySEV,
			r.useLaunchSecurityPV,
		)
		if err != nil {
			return err
		}
		domain.Spec.Devices.Rng = newRng
	}

	return nil
}

func convert_v1_Rng_To_api_Rng(_ *v1.Rng, rng *api.Rng, architecture string, useVirtioTransitional, useLaunchSecuritySEV, useLaunchSecurityPV bool) error {

	// default rng model for KVM/QEMU virtualization
	rng.Model = virtio.InterpretTransitionalModelType(&useVirtioTransitional, architecture)

	// default backend model, random
	rng.Backend = &api.RngBackend{
		Model: "random",
	}

	// the default source for rng is dev urandom
	rng.Backend.Source = "/dev/urandom"

	if useLaunchSecuritySEV || useLaunchSecurityPV {
		rng.Driver = &api.RngDriver{
			IOMMU: "on",
		}
	}

	return nil
}

func RNGWithArchitecture(architecture string) rngOption {
	return func(r *RNGDomainConfigurator) {
		r.architecture = architecture
	}
}

func RNGWithUseVirtioTransitional(useVirtioTransitional bool) rngOption {
	return func(r *RNGDomainConfigurator) {
		r.useVirtioTransitional = useVirtioTransitional
	}
}

func RNGWithUseLaunchSecuritySEV(useLaunchSecuritySEV bool) rngOption {
	return func(r *RNGDomainConfigurator) {
		r.useLaunchSecuritySEV = useLaunchSecuritySEV
	}
}

func RNGWithUseLaunchSecurityPV(useLaunchSecurityPV bool) rngOption {
	return func(r *RNGDomainConfigurator) {
		r.useLaunchSecurityPV = useLaunchSecurityPV
	}
}
