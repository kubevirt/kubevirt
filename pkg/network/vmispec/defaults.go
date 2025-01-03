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
 * Copyright 2024 Red Hat, Inc.
 *
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
