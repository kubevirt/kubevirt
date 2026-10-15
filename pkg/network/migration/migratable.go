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

func VerifyVMIMigratable(vmi *v1.VirtualMachineInstance, bindingPlugins map[string]v1.InterfaceBindingPlugin) error {
	ifaces := vmi.Spec.Domain.Devices.Interfaces
	if len(ifaces) == 0 {
		return nil
	}

	podNetwork := vmispec.LookupPodNetwork(vmi.Spec.Networks)
	if podNetwork == nil {
		return nil
	}

	primaryIface := vmispec.LookupInterfaceByName(ifaces, podNetwork.Name)

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
