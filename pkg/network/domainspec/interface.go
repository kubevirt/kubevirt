/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
		if iface.Masquerade != nil || iface.Bridge != nil {
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
