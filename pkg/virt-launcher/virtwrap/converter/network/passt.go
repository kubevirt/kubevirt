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

package network

import (
	"fmt"
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

const (
	//nolint:gosec //linter is confusing passt for password
	passtLogFilePath      = "/var/run/kubevirt/passt.log"
	istioInjectAnnotation = "sidecar.istio.io/inject"
	ifaceTypeVhostUser    = "vhostuser"
	passtBackendPasst     = "passt"
)

func createPasstInterface(vmi *v1.VirtualMachineInstance, iface *v1.Interface, d *DomainConfigurator) (api.Interface, error) {
	network := vmispec.LookupPodNetwork(vmi.Spec.Networks)
	if network == nil {
		return api.Interface{}, fmt.Errorf("pod network not found")
	}

	if network.Name != iface.Name {
		return api.Interface{}, fmt.Errorf("interface %s does not match pod network %s", iface.Name, network.Name)
	}

	ifaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, iface.Name)
	podIfaceName := namescheme.PrimaryPodInterfaceName
	if ifaceStatus != nil && ifaceStatus.PodInterfaceName != "" {
		podIfaceName = ifaceStatus.PodInterfaceName
	}

	ifaceModel := iface.Model
	if ifaceModel == "" {
		ifaceModel = v1.VirtIO
	}

	modelType := ifaceModel
	if ifaceModel == v1.VirtIO {
		modelType = d.virtioModel
	}

	istioProxyInjectionEnabled := false
	if val, ok := vmi.GetAnnotations()[istioInjectAnnotation]; ok {
		istioProxyInjectionEnabled = strings.EqualFold(val, "true")
	}

	domainIface := api.Interface{
		Alias:       api.NewUserDefinedAlias(iface.Name),
		Model:       &api.Model{Type: modelType},
		Type:        ifaceTypeVhostUser,
		Source:      api.InterfaceSource{Device: podIfaceName},
		Backend:     &api.InterfaceBackend{Type: passtBackendPasst, LogFile: passtLogFilePath},
		PortForward: GeneratePasstPortForward(iface, istioProxyInjectionEnabled),
	}

	if iface.PciAddress != "" {
		addr, err := device.NewPciAddressField(iface.PciAddress)
		if err != nil {
			return api.Interface{}, fmt.Errorf("failed to configure PCI address: %v", err)
		}
		domainIface.Address = addr
	}

	if iface.MacAddress != "" {
		domainIface.MAC = &api.MAC{MAC: iface.MacAddress}
	}

	if iface.ACPIIndex > 0 {
		domainIface.ACPI = &api.ACPI{Index: uint(iface.ACPIIndex)}
	}

	return domainIface, nil
}

// GeneratePasstPortForward generates port forwarding configuration for passt interfaces
func GeneratePasstPortForward(iface *v1.Interface, istioProxyInjectionEnabled bool) []api.InterfacePortForward {
	var tcpPortsRange, udpPortsRange []api.InterfacePortForwardRange

	if istioProxyInjectionEnabled {
		for _, port := range istio.ReservedPorts() {
			tcpPortsRange = append(tcpPortsRange, api.InterfacePortForwardRange{Start: port, Exclude: "yes"})
		}
	}

	const (
		protoTCP = "tcp"
		protoUDP = "udp"
	)

	for _, port := range iface.Ports {
		portNumber := port.Port
		if portNumber < 0 {
			log.Log.Errorf("port %d is illegal", portNumber)
			continue
		}
		if strings.EqualFold(port.Protocol, protoTCP) || port.Protocol == "" {
			tcpPortsRange = append(tcpPortsRange, api.InterfacePortForwardRange{Start: uint(portNumber)})
		} else if strings.EqualFold(port.Protocol, protoUDP) {
			udpPortsRange = append(udpPortsRange, api.InterfacePortForwardRange{Start: uint(portNumber)})
		} else {
			log.Log.Errorf("protocol %s is not supported by passt", port.Protocol)
		}
	}

	var portsFwd []api.InterfacePortForward
	if len(udpPortsRange) == 0 && len(tcpPortsRange) == 0 {
		portsFwd = append(
			portsFwd,
			api.InterfacePortForward{Proto: protoTCP},
			api.InterfacePortForward{Proto: protoUDP},
		)
	}
	if len(tcpPortsRange) > 0 {
		portsFwd = append(portsFwd, api.InterfacePortForward{Proto: protoTCP, Ranges: tcpPortsRange})
	}
	if len(udpPortsRange) > 0 {
		portsFwd = append(portsFwd, api.InterfacePortForward{Proto: protoUDP, Ranges: udpPortsRange})
	}

	return portsFwd
}
