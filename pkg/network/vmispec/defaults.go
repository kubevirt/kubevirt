/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package vmispec

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
)

type netClusterConfigurer interface {
	GetDefaultNetworkInterface() string
	IsBridgeInterfaceOnPodNetworkEnabled() bool
}

func SetDefaultNetworkInterface(config netClusterConfigurer, spec *v1.VirtualMachineInstanceSpec) error {
	if autoAttach := spec.Domain.Devices.AutoattachPodInterface; autoAttach != nil && !*autoAttach {
		return nil
	}

	// Override only when nothing is specified
	if len(spec.Networks) != 0 || len(spec.Domain.Devices.Interfaces) != 0 {
		return nil
	}

	switch v1.NetworkInterfaceType(config.GetDefaultNetworkInterface()) {
	case v1.BridgeInterface:
		if !config.IsBridgeInterfaceOnPodNetworkEnabled() {
			return fmt.Errorf("bridge interface is not enabled in kubevirt-config")
		}
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
	case v1.MasqueradeInterface:
		spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
	case v1.DeprecatedSlirpInterface:
		return fmt.Errorf("slirp interface is deprecated as of v1.3")
	}

	spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

	return nil
}
