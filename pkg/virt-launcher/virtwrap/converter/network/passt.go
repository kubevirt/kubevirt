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
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	//nolint:gosec //linter is confusing passt for password
	passtLogFilePath      = "/var/run/kubevirt/passt.log"
	istioInjectAnnotation = "sidecar.istio.io/inject"
	ifaceTypeVhostUser    = "vhostuser"
	passtBackendPasst     = "passt"
)

func createPasstInterface(domainIface *api.Interface, vmi *v1.VirtualMachineInstance, iface *v1.Interface) (api.Interface, error) {
	ifaceStatus := vmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, iface.Name)
	podIfaceName := namescheme.PrimaryPodInterfaceName
	if ifaceStatus != nil && ifaceStatus.PodInterfaceName != "" {
		podIfaceName = ifaceStatus.PodInterfaceName
	}

	istioProxyInjectionEnabled := false
	if val, ok := vmi.GetAnnotations()[istioInjectAnnotation]; ok {
		istioProxyInjectionEnabled = strings.EqualFold(val, "true")
	}

	domainIface.Type = ifaceTypeVhostUser
	domainIface.Source = api.InterfaceSource{Device: podIfaceName}
	domainIface.Backend = &api.InterfaceBackend{Type: passtBackendPasst, LogFile: passtLogFilePath}
	domainIface.PortForward = generatePasstPortForward(iface, istioProxyInjectionEnabled)

	return *domainIface, nil
}

func generatePasstPortForward(iface *v1.Interface, istioProxyInjectionEnabled bool) []api.InterfacePortForward {
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
