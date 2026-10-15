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

package migration

import (
	"errors"
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

type clusterConfigurer interface {
	GetNetworkBindings() map[string]v1.InterfaceBindingPlugin
	NetworkDevicesWithDRAGateEnabled() bool
}

func VerifyVMIMigratable(vmi *v1.VirtualMachineInstance, config clusterConfigurer) error {
	ifaces := vmi.Spec.Domain.Devices.Interfaces
	if len(ifaces) == 0 {
		return nil
	}

	if config.NetworkDevicesWithDRAGateEnabled() && usesResourceClaim(vmi) {
		return errors.New("cannot migrate VMI whose DRA network is backed by a ResourceClaim; use a ResourceClaimTemplate instead")
	}

	podNetwork := vmispec.LookupPodNetwork(vmi.Spec.Networks)
	if podNetwork == nil {
		return nil
	}

	primaryIface := vmispec.LookupInterfaceByName(ifaces, podNetwork.Name)

	bindingPlugins := config.GetNetworkBindings()

	switch {
	case primaryIface == nil:
		return fmt.Errorf("no primary interface found for network %s", podNetwork.Name)
	case primaryIface.Masquerade != nil || primaryIface.PasstBinding != nil:
		return nil
	case primaryIface.Bridge != nil:
		if _, isLiveMigrationAllowed := vmi.Annotations[v1.AllowPodBridgeNetworkLiveMigrationAnnotation]; isLiveMigrationAllowed {
			return nil
		}
	case primaryIface.Binding != nil:
		if binding, exist := bindingPlugins[primaryIface.Binding.Name]; exist && binding.Migration != nil {
			return nil
		}
	default:
	}

	return errors.New("cannot migrate VMI which does not use masquerade or a migratable plugin to connect to the pod network")
}

// usesResourceClaim reports whether the VMI has a DRA network backed
// by a ResourceClaim rather than a ResourceClaimTemplate. A ResourceClaim is bound
// to a specific allocation and cannot be reserved by the migration target pod, so
// the VMI is not migratable. A ResourceClaimTemplate, in contrast, generates a
// fresh claim per pod and can be satisfied on the target.
func usesResourceClaim(vmi *v1.VirtualMachineInstance) bool {
	draNetworkClaimNames := map[string]struct{}{}
	for _, network := range vmi.Spec.Networks {
		if vmispec.IsDRANetwork(network) {
			draNetworkClaimNames[network.ResourceClaim.ClaimName] = struct{}{}
		}
	}

	for _, claim := range vmi.Spec.ResourceClaims {
		if _, isDRANetworkClaim := draNetworkClaimNames[claim.Name]; isDRANetworkClaim && claim.ResourceClaimName != nil {
			return true
		}
	}
	return false
}
