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

package domainspec

import (
	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func LookupIfaceByAliasName(ifaces []api.Interface, name string) *api.Interface {
	for i, iface := range ifaces {
		if iface.Alias != nil && iface.Alias.GetName() == name {
			return &ifaces[i]
		}
	}

	return nil
}

func DomainAttachmentByInterfaceName(vmiSpecIfaces []v1.Interface, networkBindings map[string]v1.InterfaceBindingPlugin) map[string]string {
	domainAttachmentByPluginName := map[string]string{}
	for name, binding := range networkBindings {
		if binding.DomainAttachmentType != "" {
			domainAttachmentByPluginName[name] = string(binding.DomainAttachmentType)
		}
	}

	domainAttachmentByInterfaceName := map[string]string{}
	for _, iface := range vmiSpecIfaces {
		if iface.Masquerade != nil || iface.Bridge != nil || iface.DeprecatedMacvtap != nil {
			// Macvtap is removed in v1.3.
			// The macvtap scenario is tracking old VMIs that are still processed in the reconcile loop.
			domainAttachmentByInterfaceName[iface.Name] = string(v1.Tap)
		} else if iface.Binding != nil {
			if domainAttachmentType, exist := domainAttachmentByPluginName[iface.Binding.Name]; exist {
				// For domain consumption, handle the `managedTap` type as `tap`.
				if domainAttachmentType == string(v1.ManagedTap) {
					domainAttachmentType = string(v1.Tap)
				}
				domainAttachmentByInterfaceName[iface.Name] = domainAttachmentType
			}
		}
	}
	return domainAttachmentByInterfaceName
}

func BindingMigrationByInterfaceName(vmiSpecIfaces []v1.Interface,
	networkBindings map[string]v1.InterfaceBindingPlugin,
) map[string]*cmdv1.InterfaceBindingMigration {
	bindingMigrationByPluginName := map[string]*cmdv1.InterfaceBindingMigration{}
	for name, binding := range networkBindings {
		if binding.Migration != nil {
			migration := &cmdv1.InterfaceBindingMigration{
				Method: string(binding.Migration.Method),
			}
			bindingMigrationByPluginName[name] = migration
		}
	}

	bindingMigrationByInterfaceName := map[string]*cmdv1.InterfaceBindingMigration{}
	for _, iface := range vmiSpecIfaces {
		if iface.Binding != nil {
			if bindingMigration, exist := bindingMigrationByPluginName[iface.Binding.Name]; exist {
				bindingMigrationByInterfaceName[iface.Name] = bindingMigration
			}
		}
	}
	return bindingMigrationByInterfaceName
}
