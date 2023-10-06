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

package domain

import (
	"fmt"
	"net"
	"strings"

	vmschema "kubevirt.io/api/core/v1"

	domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

// SlirpPluginName slirp binding plugin name should be registered to Kubevirt through Kubevirt CR
const SlirpPluginName = "slirp"

type SlirpNetworkConfigurator struct {
	vmiSpecIface   *vmschema.Interface
	vmiSpecNetwork *vmschema.Network
	searchDomains  []string
}

func NewSlirpNetworkConfigurator(ifaces []vmschema.Interface, networks []vmschema.Network, searchDomains []string) (*SlirpNetworkConfigurator, error) {
	network := vmispec.LookupPodNetwork(networks)
	if network == nil {
		return nil, fmt.Errorf("pod network not found")
	}

	iface := vmispec.LookupInterfaceByName(ifaces, network.Name)
	if iface == nil {
		return nil, fmt.Errorf("iface %q not found", network.Name)
	}
	if iface.Binding == nil && iface.Slirp == nil {
		return nil, fmt.Errorf("iface %q is not set with slirp network binding plugin or slirp binding method", network.Name)
	}
	if iface.Binding != nil && iface.Binding.Name != SlirpPluginName {
		return nil, fmt.Errorf("iface %q is not set with slirp network binding plugin", network.Name)
	}

	return &SlirpNetworkConfigurator{
		vmiSpecIface:   iface,
		vmiSpecNetwork: network,
		searchDomains:  searchDomains,
	}, nil
}

func (s SlirpNetworkConfigurator) Mutate(domainSpec *domainschema.DomainSpec) (*domainschema.DomainSpec, error) {
	slirpQemuCmdArgs, err := generateSlirpNetworkQemuCmdArgs(s.vmiSpecIface, s.vmiSpecNetwork, s.searchDomains)
	if err != nil {
		return nil, err
	}

	if len(slirpQemuCmdArgs) == 0 {
		log.Log.Warning("no qemu cmd args configured, domain spec did not change")
		return domainSpec, nil
	}

	domainSpecCopy := domainSpec.DeepCopy()
	if domainSpecCopy.QEMUCmd == nil {
		domainSpecCopy.QEMUCmd = &domainschema.Commandline{}
	}

	var currentCmdArgString string
	for _, arg := range domainSpecCopy.QEMUCmd.QEMUArg {
		currentCmdArgString += arg.Value
	}

	var generatedCmdArgString string
	for _, arg := range slirpQemuCmdArgs {
		generatedCmdArgString += arg.Value
	}

	if !strings.Contains(currentCmdArgString, generatedCmdArgString) {
		domainSpecCopy.QEMUCmd.QEMUArg = append(domainSpecCopy.QEMUCmd.QEMUArg, slirpQemuCmdArgs...)
		return domainSpecCopy, nil
	}

	return domainSpecCopy, nil
}

func generateSlirpNetworkQemuCmdArgs(iface *vmschema.Interface, network *vmschema.Network, searchDomains []string) ([]domainschema.Arg, error) {
	backendDeviceQemuArgs, err := generateBackendDeviceQemuArg(iface, network, searchDomains)
	if err != nil {
		return nil, err
	}

	var qemuArgs []domainschema.Arg
	qemuArgs = append(qemuArgs, backendDeviceQemuArgs...)
	qemuArgs = append(qemuArgs, generateNetDeviceQemuArg(iface)...)

	return qemuArgs, nil
}

func generateBackendDeviceQemuArg(iface *vmschema.Interface, network *vmschema.Network, dnsDomains []string) ([]domainschema.Arg, error) {
	backendDeviceConf := domainschema.Arg{Value: fmt.Sprintf("user,id=%s", iface.Name)}

	cidr, err := networkCIDR(network)
	if err != nil {
		return nil, err
	}
	backendDeviceConf.Value += fmt.Sprintf(",net=%s", cidr)

	for _, dnsDomain := range dnsDomains {
		backendDeviceConf.Value += fmt.Sprintf(",dnssearch=%s", dnsDomain)
	}

	ports, err := ifacePorts(iface)
	if err != nil {
		return nil, err
	}
	for _, port := range ports {
		backendDeviceConf.Value += fmt.Sprintf(",hostfwd=%[1]s::%[2]d-:%[2]d", strings.ToLower(port.Protocol), port.Port)
	}

	return []domainschema.Arg{{Value: "-netdev"}, backendDeviceConf}, nil
}

func networkCIDR(network *vmschema.Network) (string, error) {
	if network.Pod.VMNetworkCIDR != "" {
		if _, _, err := net.ParseCIDR(network.Pod.VMNetworkCIDR); err != nil {
			return "", fmt.Errorf("invalid network CIDR %q: %v", network.Pod.VMNetworkCIDR, err)
		}
		return network.Pod.VMNetworkCIDR, nil
	}

	return domainschema.DefaultVMCIDR, nil
}

func ifacePorts(iface *vmschema.Interface) ([]vmschema.Port, error) {
	if len(iface.Ports) == 0 {
		return nil, nil
	}

	var ports []vmschema.Port
	configuredPorts := make(map[string]struct{}, 0)
	for _, ifacePort := range iface.Ports {
		if ifacePort.Port <= 0 {
			return nil, fmt.Errorf("invalid port %q", ifacePort.Port)
		}

		if ifacePort.Protocol == "" {
			ifacePort.Protocol = domainschema.DefaultProtocol
		}

		// each port should be set once otherwise QEMU process crash
		portConfig := fmt.Sprintf("%s-%d", ifacePort.Protocol, ifacePort.Port)
		if _, ok := configuredPorts[portConfig]; !ok {
			ports = append(ports, ifacePort)
			configuredPorts[portConfig] = struct{}{}
		}
	}

	return ports, nil
}

func generateNetDeviceQemuArg(iface *vmschema.Interface) []domainschema.Arg {
	// slirp configuration works only with 'e1000' or 'rtl8139'
	ifaceModel := iface.Model
	if ifaceModel != "e1000" && ifaceModel != "rtl8139" {
		log.Log.Infof("interface (%q) model type (%q) is not supported by QEMU slirp Network, using 'e1000' model type instead",
			iface.Name, iface.Model)
		ifaceModel = "e1000"
	}
	netDeviceConf := fmt.Sprintf(`"driver":%[1]q,"netdev":%[2]q,"id":%[2]q`, ifaceModel, iface.Name)
	if iface.MacAddress != "" {
		// We assume address was already validated in API layer so just pass it to libvirt as-is.
		netDeviceConf += fmt.Sprintf(`,"mac":%q`, iface.MacAddress)
	}

	return []domainschema.Arg{{Value: "-device"}, {Value: fmt.Sprintf(`{%s}`, netDeviceConf)}}
}
