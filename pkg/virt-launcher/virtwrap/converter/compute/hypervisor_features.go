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
)

type HypervisorFeaturesDomainConfigurator struct {
	hasVMPort            bool
	useLaunchSecurityTDX bool
}

func NewHypervisorFeaturesDomainConfigurator(hasVMPort, useLaunchSecurityTDX bool) HypervisorFeaturesDomainConfigurator {
	return HypervisorFeaturesDomainConfigurator{
		hasVMPort:            hasVMPort,
		useLaunchSecurityTDX: useLaunchSecurityTDX,
	}
}

func (h HypervisorFeaturesDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if vmi.Spec.Domain.Features != nil {
		domain.Spec.Features = &api.Features{}
		err := convert_v1_Features_To_api_Features(vmi.Spec.Domain.Features, domain.Spec.Features, h.useLaunchSecurityTDX)

		if h.hasVMPort {
			domain.Spec.Features.VMPort = &api.FeatureState{State: "off"}
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func convert_v1_Features_To_api_Features(source *v1.Features, features *api.Features, useLaunchSecurityTDX bool) error {
	if source.ACPI.Enabled == nil || *source.ACPI.Enabled {
		features.ACPI = &api.FeatureEnabled{}
	}
	if source.SMM != nil {
		if source.SMM.Enabled == nil || *source.SMM.Enabled {
			features.SMM = &api.FeatureEnabled{}
		}
	}
	if source.APIC != nil {
		if source.APIC.Enabled == nil || *source.APIC.Enabled {
			features.APIC = &api.FeatureEnabled{}
		}
	}
	if source.Hyperv != nil {
		features.Hyperv = &api.FeatureHyperv{}
		err := convert_v1_FeatureHyperv_To_api_FeatureHyperv(source.Hyperv, features.Hyperv)
		if err != nil {
			return nil
		}
	} else if source.HypervPassthrough != nil && *source.HypervPassthrough.Enabled {
		features.Hyperv = &api.FeatureHyperv{
			Mode: api.HypervModePassthrough,
		}
	}
	if source.KVM != nil {
		features.KVM = &api.FeatureKVM{
			Hidden: &api.FeatureState{
				State: boolToOnOff(&source.KVM.Hidden, false),
			},
		}
	}
	if source.Pvspinlock != nil {
		features.PVSpinlock = &api.FeaturePVSpinlock{
			State: boolToOnOff(source.Pvspinlock.Enabled, true),
		}
	}

	if useLaunchSecurityTDX {
		features.PMU = &api.FeatureState{
			State: "off",
		}
	}

	return nil
}

func convert_v1_FeatureHyperv_To_api_FeatureHyperv(source *v1.FeatureHyperv, hyperv *api.FeatureHyperv) error {
	if source.Spinlocks != nil {
		hyperv.Spinlocks = &api.FeatureSpinlocks{
			State:   boolToOnOff(source.Spinlocks.Enabled, true),
			Retries: source.Spinlocks.Retries,
		}
	}
	if source.VendorID != nil {
		hyperv.VendorID = &api.FeatureVendorID{
			State: boolToOnOff(source.VendorID.Enabled, true),
			Value: source.VendorID.VendorID,
		}
	}

	hyperv.Relaxed = convertFeatureState(source.Relaxed)
	hyperv.Reset = convertFeatureState(source.Reset)
	hyperv.Runtime = convertFeatureState(source.Runtime)
	hyperv.SyNIC = convertFeatureState(source.SyNIC)
	hyperv.SyNICTimer = convertV1ToAPISyNICTimer(source.SyNICTimer)
	hyperv.VAPIC = convertFeatureState(source.VAPIC)
	hyperv.VPIndex = convertFeatureState(source.VPIndex)
	hyperv.Frequencies = convertFeatureState(source.Frequencies)
	hyperv.Reenlightenment = convertFeatureState(source.Reenlightenment)
	hyperv.TLBFlush = convertFeatureState(source.TLBFlush)
	hyperv.IPI = convertFeatureState(source.IPI)
	hyperv.EVMCS = convertFeatureState(source.EVMCS)
	return nil
}

func convertFeatureState(source *v1.FeatureState) *api.FeatureState {
	if source != nil {
		return &api.FeatureState{
			State: boolToOnOff(source.Enabled, true),
		}
	}
	return nil
}

func convertV1ToAPISyNICTimer(syNICTimer *v1.SyNICTimer) *api.SyNICTimer {
	if syNICTimer == nil {
		return nil
	}

	result := &api.SyNICTimer{
		State: boolToOnOff(syNICTimer.Enabled, true),
	}

	if syNICTimer.Direct != nil {
		result.Direct = &api.FeatureState{
			State: boolToOnOff(syNICTimer.Direct.Enabled, true),
		}
	}
	return result
}
