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

package domain

import (
	"fmt"
	"slices"

	"libvirt.org/go/libvirtxml"

	vmschema "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const socketModeClient = "client"

type VhostUserNetworkConfigurator struct {
	vmiSpecIface *vmschema.Interface
	socketPath   string
}

func NewVhostUserNetworkConfigurator(
	ifaces []vmschema.Interface,
	networks []vmschema.Network,
	bindingPluginName string,
	socketPath string,
) (*VhostUserNetworkConfigurator, error) {
	idx := slices.IndexFunc(ifaces, func(iface vmschema.Interface) bool {
		return iface.Binding != nil && iface.Binding.Name == bindingPluginName
	})
	if idx == -1 {
		return nil, fmt.Errorf("no interface found with the %q network binding plugin", bindingPluginName)
	}
	iface := &ifaces[idx]

	network := vmispec.LookupNetworkByName(networks, iface.Name)
	if network == nil {
		return nil, fmt.Errorf("no network found for interface %q", iface.Name)
	}
	if !vmispec.IsDRANetwork(*network) {
		return nil, fmt.Errorf("network %q is not a DRA network", network.Name)
	}

	if socketPath == "" {
		return nil, fmt.Errorf("socket path is empty; the DRA network device was not provisioned")
	}

	return &VhostUserNetworkConfigurator{
		vmiSpecIface: iface,
		socketPath:   socketPath,
	}, nil
}

func (c VhostUserNetworkConfigurator) Mutate(domain *libvirtxml.Domain) (*libvirtxml.Domain, error) {
	const (
		sharedMemoryBackingAccessMode = "shared"
		memfdMemoryBackingSourceType  = "memfd"
	)

	if domain.MemoryBacking != nil &&
		domain.MemoryBacking.MemoryAccess != nil &&
		domain.MemoryBacking.MemoryAccess.Mode != sharedMemoryBackingAccessMode {
		return nil, fmt.Errorf("memory backing access mode must be 'shared'; cannot override existing mode: %q",
			domain.MemoryBacking.MemoryAccess.Mode)
	}

	generatedIface := c.generateInterface()

	if domain.Devices == nil {
		domain.Devices = &libvirtxml.DomainDeviceList{}
	}
	if iface := lookupIfaceByAliasName(domain.Devices.Interfaces, c.vmiSpecIface.Name); iface != nil {
		*iface = *generatedIface
	} else {
		domain.Devices.Interfaces = append(domain.Devices.Interfaces, *generatedIface)
	}

	// vhostuser interfaces require the guest memory to be shared with the
	// vhost-user backend process.
	if domain.MemoryBacking == nil {
		domain.MemoryBacking = &libvirtxml.DomainMemoryBacking{
			MemoryAccess: &libvirtxml.DomainMemoryAccess{
				Mode: sharedMemoryBackingAccessMode,
			},
			MemorySource: &libvirtxml.DomainMemorySource{
				Type: memfdMemoryBackingSourceType,
			},
		}
	}

	log.Log.Infof("vhostuser interface is added to domain spec successfully: %+v", generatedIface)

	return domain, nil
}

func lookupIfaceByAliasName(ifaces []libvirtxml.DomainInterface, name string) *libvirtxml.DomainInterface {
	for i := range ifaces {
		if ifaces[i].Alias != nil && ifaces[i].Alias.Name == api.UserAliasPrefix+name {
			return &ifaces[i]
		}
	}

	return nil
}

func (c VhostUserNetworkConfigurator) generateInterface() *libvirtxml.DomainInterface {
	return &libvirtxml.DomainInterface{
		Alias: &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + c.vmiSpecIface.Name},
		Model: &libvirtxml.DomainInterfaceModel{Type: "virtio-non-transitional"},
		Source: &libvirtxml.DomainInterfaceSource{
			VHostUser: &libvirtxml.DomainInterfaceSourceVHostUser{
				Chardev: &libvirtxml.DomainChardevSource{
					UNIX: &libvirtxml.DomainChardevSourceUNIX{
						Path: c.socketPath,
						Mode: socketModeClient,
					},
				},
			},
		},
	}
}
