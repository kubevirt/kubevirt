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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package libnet

import (
	"fmt"
	"time"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/console"
)

func InterfaceExists(vmi *v1.VirtualMachineInstance, interfaceName string) error {
	const timeout = 15 * time.Second
	cmdCheck := fmt.Sprintf("ip link show %s\n", interfaceName)
	if err := console.RunCommand(vmi, cmdCheck, timeout); err != nil {
		return fmt.Errorf("could not check interface: interface %s was not found in the VMI %s: %w", interfaceName, vmi.Name, err)
	}
	return nil
}

func LookupNetworkByName(networks []v1.Network, name string) *v1.Network {
	for i, net := range networks {
		if net.Name == name {
			return &networks[i]
		}
	}

	return nil
}
