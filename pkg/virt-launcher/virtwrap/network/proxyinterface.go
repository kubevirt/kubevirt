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
 * Copyright 2018 Red Hat, Inc.
 *
 */

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

package network

import (
	"fmt"
	"net"
	"strings"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// DefaultProtocol is the default port protocol
const DefaultProtocol string = "TCP"

// DefaultVMCIDR is the default CIRD for vm network
const DefaultVMCIDR = "10.0.2.0/24"

type ProxyBindMechanism interface {
	configPortForward() error
	configVMCIDR() error
	configDNSSearchName() error
	CommitConfiguration() error
}

type ProxyInterface struct{}

func (l *ProxyInterface) Unplug() {}

// Plug connect a Pod network device to the virtual machine
func (l *ProxyInterface) Plug(iface *v1.Interface, network *v1.Network, domain *api.Domain) error {
	precond.MustNotBeNil(domain)
	initHandler()

	driver, err := getProxyBinding(iface, network, domain)
	if err != nil {
		return err
	}

	err = driver.configVMCIDR()
	if err != nil {
		return err
	}

	err = driver.configDNSSearchName()
	if err != nil {
		return err
	}

	err = driver.configPortForward()
	if err != nil {
		return err
	}

	err = driver.CommitConfiguration()
	if err != nil {
		return err
	}

	return nil
}

func getProxyBinding(iface *v1.Interface, network *v1.Network, domain *api.Domain) (ProxyBindMechanism, error) {
	if iface.Proxy != nil {
		proxyConfig := api.Arg{Value: fmt.Sprintf("user,id=%s", iface.Name)}
		return &ProxyPodInterface{iface: iface, network: network, domain: domain, proxyConfig: proxyConfig}, nil
	}
	return nil, fmt.Errorf("Interface Type not implemented for proxy network")
}

type ProxyPodInterface struct {
	iface       *v1.Interface
	network     *v1.Network
	domain      *api.Domain
	proxyConfig api.Arg
}

func (p *ProxyPodInterface) configPortForward() error {
	if p.iface.Proxy.Ports == nil {
		return nil
	}

	portForwardMap := make(map[int32]string)

	for _, forwardPort := range p.iface.Proxy.Ports {
		protocol := DefaultProtocol

		// Check protocol, its case sensitive like kubernetes
		if forwardPort.Protocol != "" {
			if forwardPort.Protocol != "TCP" && forwardPort.Protocol != "UDP" {
				return fmt.Errorf("Unknow protocol only TCP or UDP allowed")
			} else {
				protocol = forwardPort.Protocol
			}
		}
		//Check for duplicate pod port allocation
		if portProtocol, ok := portForwardMap[forwardPort.PodPort]; ok && portProtocol == protocol {
			return fmt.Errorf("Duplicated pod port allocation")
		}

		// Check if PodPort is configure If not Get the same Port as the vm port
		if forwardPort.VMPort == 0 {
			forwardPort.VMPort = forwardPort.PodPort
		}

		portForwardMap[forwardPort.PodPort] = protocol
		p.proxyConfig.Value += fmt.Sprintf(",hostfwd=%s::%d-:%d", strings.ToLower(string(protocol)), forwardPort.PodPort, forwardPort.VMPort)

	}

	return nil
}

func (p *ProxyPodInterface) configVMCIDR() error {
	if p.network.Pod == nil {
		return fmt.Errorf("Proxy works only with proxy network")
	}

	vmNetworkCIDR := ""
	if p.network.Pod.VMNetworkCIDR != "" {
		_, _, err := net.ParseCIDR(p.network.Pod.VMNetworkCIDR)
		if err != nil {
			return fmt.Errorf("Failed parsing CIDR")
		}
		vmNetworkCIDR = p.network.Pod.VMNetworkCIDR
	} else {
		vmNetworkCIDR = DefaultVMCIDR
	}

	// Insert configuration to qemu commandline
	p.proxyConfig.Value += fmt.Sprintf(",net=%s", vmNetworkCIDR)

	return nil
}

func (p *ProxyPodInterface) configDNSSearchName() error {
	// remove the search string from the output and convert to string
	_, dnsSearchNames, err := getResolvConfDetailsFromPod()
	if err != nil {
		return err
	}

	// Insert configuration to qemu commandline
	for _, dnsSearchName := range dnsSearchNames {
		p.proxyConfig.Value += fmt.Sprintf(",dnssearch=%s", dnsSearchName)
	}

	return nil
}

func (p *ProxyPodInterface) CommitConfiguration() error {
	p.domain.Spec.QEMUCmd.QEMUArg = append(p.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-netdev"})
	p.domain.Spec.QEMUCmd.QEMUArg = append(p.domain.Spec.QEMUCmd.QEMUArg, p.proxyConfig)

	return nil
}
