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

	vmschema "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
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

func (c VhostUserNetworkConfigurator) Mutate(domainSpec *domainschema.DomainSpec) (*domainschema.DomainSpec, error) {
	const (
		sharedMemoryBackingAccessMode = "shared"
		memfdMemoryBackingSourceType  = "memfd"
	)

	if domainSpec.MemoryBacking != nil &&
		domainSpec.MemoryBacking.Access != nil &&
		domainSpec.MemoryBacking.Access.Mode != sharedMemoryBackingAccessMode {
		return nil, fmt.Errorf("memory backing access mode must be 'shared'; cannot override existing mode: %q",
			domainSpec.MemoryBacking.Access.Mode)
	}

	generatedIface := c.generateInterface()

	domainSpecCopy := domainSpec.DeepCopy()
	if iface := lookupIfaceByAliasName(domainSpecCopy.Devices.Interfaces, c.vmiSpecIface.Name); iface != nil {
		*iface = *generatedIface
	} else {
		domainSpecCopy.Devices.Interfaces = append(domainSpecCopy.Devices.Interfaces, *generatedIface)
	}

	// vhostuser interfaces require the guest memory to be shared with the
	// vhost-user backend process.
	if domainSpecCopy.MemoryBacking == nil {
		domainSpecCopy.MemoryBacking = &domainschema.MemoryBacking{
			Access: &domainschema.MemoryBackingAccess{
				Mode: sharedMemoryBackingAccessMode,
			},
			Source: &domainschema.MemoryBackingSource{
				Type: memfdMemoryBackingSourceType,
			},
		}
	}

	log.Log.Infof("vhostuser interface is added to domain spec successfully: %+v", generatedIface)

	return domainSpecCopy, nil
}

func lookupIfaceByAliasName(ifaces []domainschema.Interface, name string) *domainschema.Interface {
	for i := range ifaces {
		if ifaces[i].Alias != nil && ifaces[i].Alias.GetName() == name {
			return &ifaces[i]
		}
	}

	return nil
}

func (c VhostUserNetworkConfigurator) generateInterface() *domainschema.Interface {
	var ifaceModelType string
	switch {
	case c.vmiSpecIface.Model != "" && c.vmiSpecIface.Model != vmschema.VirtIO:
		ifaceModelType = c.vmiSpecIface.Model
	case c.useVirtioTransitional:
		ifaceModelType = "virtio-transitional"
	default:
		ifaceModelType = "virtio-non-transitional"
	}

	var mac *domainschema.MAC
	if c.vmiSpecIface.MacAddress != "" {
		mac = &domainschema.MAC{MAC: c.vmiSpecIface.MacAddress}
	}

	var pciAddress *domainschema.Address
	if c.vmiSpecIface.PciAddress != "" {
		if addr, err := device.NewPciAddressField(c.vmiSpecIface.PciAddress); err == nil {
			pciAddress = addr
		} else {
			log.Log.Reason(err).Warningf("failed to parse PCI address %q", c.vmiSpecIface.PciAddress)
		}
	}

	var acpi *domainschema.ACPI
	if acpiIndex := c.vmiSpecIface.ACPIIndex; acpiIndex > 0 {
		acpi = &domainschema.ACPI{Index: uint(acpiIndex)}
	}

	return &domainschema.Interface{
		Alias:   domainschema.NewUserDefinedAlias(c.vmiSpecIface.Name),
		Model:   &domainschema.Model{Type: ifaceModelType},
		Address: pciAddress,
		MAC:     mac,
		ACPI:    acpi,
		Type:    "vhostuser",
		Source: domainschema.InterfaceSource{
			Type: "unix",
			Path: c.socketPath,
			Mode: c.socketMode,
		},
	}
}
