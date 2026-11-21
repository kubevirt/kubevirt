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

package hyperv

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util"
)

// HypervModeHostModel is libvirt's hyperv host-model mode.
const HypervModeHostModel = "host-model"

// HypervModeAnnotation marks launcher pods of host-model candidates with
// HypervModeHostModel. Migration targets inherit it from their source pod, so
// domains that older launchers built in custom mode are never constrained.
const HypervModeAnnotation = "kubevirt.io/hyperv-mode"

// IsHostModelCandidate reports whether the VMI's Hyper-V features are rendered
// as libvirt host-model. A VMI that explicitly disables any enlightenment keeps
// its explicit feature list, since host-model could enable features that
// depend on the disabled one.
func IsHostModelCandidate(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Architecture != "amd64" {
		return false
	}
	features := vmi.Spec.Domain.Features
	if features == nil || features.Hyperv == nil {
		return false
	}
	return !util.UseLaunchSecurity(vmi) && !hasDisabledFeature(vmi)
}

// HostModelOverrides returns the Hyper-V features to set on top of host-model.
// Unset reenlightenment and vendor_id are turned off: host-model would enable
// reenlightenment, which needs an exposed TSC frequency to migrate, and expose
// the KVM vendor ID that an unset vendor_id hides today.
func HostModelOverrides(hyperv *v1.FeatureHyperv) *v1.FeatureHyperv {
	overrides := hyperv.DeepCopy()
	if overrides.Reenlightenment == nil {
		overrides.Reenlightenment = &v1.FeatureState{Enabled: pointer.P(false)}
	}
	if overrides.VendorID == nil {
		overrides.VendorID = &v1.FeatureVendorID{FeatureState: v1.FeatureState{Enabled: pointer.P(false)}}
	}
	return overrides
}

// hasDisabledFeature reports whether any Hyper-V feature, sub-feature or the
// hypervclock timer (hv-time) is explicitly disabled.
func hasDisabledFeature(vmi *v1.VirtualMachineInstance) bool {
	hyperv := vmi.Spec.Domain.Features.Hyperv
	states := []*v1.FeatureState{
		hyperv.Relaxed, hyperv.VAPIC, hyperv.VPIndex, hyperv.Runtime, hyperv.SyNIC,
		hyperv.Reset, hyperv.Frequencies, hyperv.Reenlightenment, hyperv.IPI, hyperv.EVMCS,
	}
	if hyperv.Spinlocks != nil {
		states = append(states, &hyperv.Spinlocks.FeatureState)
	}
	if hyperv.VendorID != nil {
		states = append(states, &hyperv.VendorID.FeatureState)
	}
	if hyperv.SyNICTimer != nil {
		states = append(states, &hyperv.SyNICTimer.FeatureState, hyperv.SyNICTimer.Direct)
	}
	if hyperv.TLBFlush != nil {
		states = append(states, &hyperv.TLBFlush.FeatureState, hyperv.TLBFlush.Direct, hyperv.TLBFlush.Extended)
	}
	// with host-model, libvirt would add a present hypervclock timer and turn hv-time back on
	if clock := vmi.Spec.Domain.Clock; clock != nil && clock.Timer != nil && clock.Timer.Hyperv != nil {
		states = append(states, &v1.FeatureState{Enabled: clock.Timer.Hyperv.Enabled})
	}

	for _, state := range states {
		if state != nil && state.Enabled != nil && !*state.Enabled {
			return true
		}
	}
	return false
}
