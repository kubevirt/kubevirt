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
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"libvirt.org/go/libvirtxml"

	vmschema "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	hwutil "kubevirt.io/kubevirt/pkg/util/hardware"
	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	SocketDirEnvVar      = "KUBEVIRT_HOSTPATH_MOUNTPOINT"
	SocketFileNameEnvVar = "KUBEVIRT_HOSTPATH_SOCKET"
	SocketModeEnvVar     = "KUBEVIRT_VHOSTUSER_MODE"

	DefaultSocketFileName = "vhost.sock"

	socketModeClient = "client"
	socketModeServer = "server"
)

type VhostUserNetworkConfigurator struct {
	vmiSpecIface          *vmschema.Interface
	socketPath            string
	socketMode            string
	useVirtioTransitional bool
}

func NewVhostUserNetworkConfigurator(
	ifaces []vmschema.Interface,
	networks []vmschema.Network,
	bindingPluginName string,
	useVirtioTransitional bool,
) (*VhostUserNetworkConfigurator, error) {
	idx := slices.IndexFunc(networks, vmispec.IsDRANetwork)
	if idx == -1 {
		return nil, fmt.Errorf("DRA network not found")
	}
	network := networks[idx]

	iface := vmispec.LookupInterfaceByName(ifaces, network.Name)
	if iface == nil {
		return nil, fmt.Errorf("no interface found for network %q", network.Name)
	}
	if iface.Binding == nil || iface.Binding.Name != bindingPluginName {
		return nil, fmt.Errorf("interface %q is not set with the %q network binding plugin",
			network.Name, bindingPluginName)
	}

	socketDir := os.Getenv(SocketDirEnvVar)
	if socketDir == "" {
		return nil, fmt.Errorf("env var %q is not set; the DRA network device was not provisioned", SocketDirEnvVar)
	}

	socketFileName := os.Getenv(SocketFileNameEnvVar)
	if socketFileName == "" {
		socketFileName = DefaultSocketFileName
	}

	socketMode := os.Getenv(SocketModeEnvVar)
	if socketMode == "" {
		socketMode = socketModeClient
	}
	if socketMode != socketModeClient && socketMode != socketModeServer {
		return nil, fmt.Errorf("invalid socket mode %q; must be %q or %q",
			socketMode, socketModeClient, socketModeServer)
	}

	return &VhostUserNetworkConfigurator{
		vmiSpecIface:          iface,
		socketPath:            filepath.Join(socketDir, socketFileName),
		socketMode:            socketMode,
		useVirtioTransitional: useVirtioTransitional,
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
	var ifaceModelType string
	switch {
	case c.vmiSpecIface.Model != "" && c.vmiSpecIface.Model != vmschema.VirtIO:
		ifaceModelType = c.vmiSpecIface.Model
	case c.useVirtioTransitional:
		ifaceModelType = "virtio-transitional"
	default:
		ifaceModelType = "virtio-non-transitional"
	}

	domainIface := &libvirtxml.DomainInterface{
		Alias: &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + c.vmiSpecIface.Name},
		Model: &libvirtxml.DomainInterfaceModel{Type: ifaceModelType},
		Source: &libvirtxml.DomainInterfaceSource{
			VHostUser: &libvirtxml.DomainInterfaceSourceVHostUser{
				Chardev: &libvirtxml.DomainChardevSource{
					UNIX: &libvirtxml.DomainChardevSourceUNIX{
						Path: c.socketPath,
						Mode: c.socketMode,
					},
				},
			},
		},
	}

	if c.vmiSpecIface.MacAddress != "" {
		domainIface.MAC = &libvirtxml.DomainInterfaceMAC{Address: c.vmiSpecIface.MacAddress}
	}

	if c.vmiSpecIface.PciAddress != "" {
		if addr, err := newPCIAddress(c.vmiSpecIface.PciAddress); err == nil {
			domainIface.Address = &libvirtxml.DomainAddress{PCI: addr}
		} else {
			log.Log.Reason(err).Warningf("failed to parse PCI address %q", c.vmiSpecIface.PciAddress)
		}
	}

	if acpiIndex := c.vmiSpecIface.ACPIIndex; acpiIndex > 0 {
		domainIface.ACPI = &libvirtxml.DomainDeviceACPI{Index: uint(acpiIndex)}
	}

	return domainIface
}

func newPCIAddress(address string) (*libvirtxml.DomainAddressPCI, error) {
	fields, err := hwutil.ParsePciAddress(address)
	if err != nil {
		return nil, err
	}

	values := make([]uint, len(fields))
	for i, field := range fields {
		v, err := strconv.ParseUint(field, 16, 32)
		if err != nil {
			return nil, err
		}
		values[i] = uint(v)
	}

	return &libvirtxml.DomainAddressPCI{
		Domain:   &values[0],
		Bus:      &values[1],
		Slot:     &values[2],
		Function: &values[3],
	}, nil
}
