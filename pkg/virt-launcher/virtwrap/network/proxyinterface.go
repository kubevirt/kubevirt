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
	"os/exec"
	"strings"

	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	DefaultProtocol k8sv1.Protocol = "TCP"
)

var (
	DefaultVMCIDR = "10.0.2.0/24"
)

type ProxyBindMechanism interface {
	configPortForward() error
	configVMCIDR() error
	configDNSSearchName() error
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

	interfaces := domain.Spec.Devices.Interfaces

	// There should always be a pre-configured interface for the default pod interface.
	if len(interfaces) == 0 {
		return fmt.Errorf("failed to find a default interface configuration")
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

	return nil
}

func getProxyBinding(iface *v1.Interface, network *v1.Network, domain *api.Domain) (ProxyBindMechanism, error) {
	if iface.Slirp != nil {
		domain.Spec.QEMUCmd.QEMUEnv = append(domain.Spec.QEMUCmd.QEMUEnv, api.Env{Value: "-netdev"})
		slirpConfig := &api.Env{Value: fmt.Sprintf("user,id=%s", iface.Name)}
		domain.Spec.QEMUCmd.QEMUEnv = append(domain.Spec.QEMUCmd.QEMUEnv, *slirpConfig)
		return &SlirpProxyInterface{iface: iface, network: network, domain: domain, slirpConfig: slirpConfig}, nil
	}
	return nil, fmt.Errorf("Not implemented")
}

type SlirpProxyInterface struct {
	iface       *v1.Interface
	network     *v1.Network
	domain      *api.Domain
	slirpConfig *api.Env
}

func (s *SlirpProxyInterface) configPortForward() error {
	if s.iface.Slirp.Ports == nil {
		return nil
	}

	portForwardMap := make(map[int32]k8sv1.Protocol)

	for _, vmPort := range s.iface.Slirp.Ports {
		protocol := DefaultProtocol

		// Check protocol, its case sensitive like kubernetes
		if vmPort.Protocol != "" && vmPort.Protocol != "TCP" && vmPort.Protocol != "UDP" {
			return fmt.Errorf("Unknow protocol only TCP or UDP allowed")
		} else {
			protocol = vmPort.Protocol
		}

		//Check for duplicate pod port allocation
		if portProtocol, ok := portForwardMap[vmPort.PodPort]; ok && portProtocol == protocol {
			return fmt.Errorf("Duplicated pod port allocation")
		}

		portForwardMap[vmPort.PodPort] = protocol
		s.slirpConfig.Value += fmt.Sprintf(",hostfwd=%s::%d-:%d", strings.ToLower(string(protocol)), vmPort.PodPort, vmPort.VMPort)

	}

	return nil
}

func (s *SlirpProxyInterface) configVMCIDR() error {
	if s.network.Proxy == nil {
		return fmt.Errorf("Slirp works only with proxy network")
	}

	vmNetworkCIDR := ""
	if s.network.Proxy.VMNetworkCIDR != "" {
		_, _, err := net.ParseCIDR(s.network.Proxy.VMNetworkCIDR)
		if err != nil {
			return fmt.Errorf("Fail to parse CIDR")
		}
		vmNetworkCIDR = s.network.Proxy.VMNetworkCIDR
	} else {
		vmNetworkCIDR = DefaultVMCIDR
	}

	// Insert configuration to qemu commandline
	s.slirpConfig.Value += fmt.Sprintf(",net=%s", vmNetworkCIDR)

	return nil
}

func (s *SlirpProxyInterface) configDNSSearchName() error {
	// Get pod search names
	out, err := exec.Command("cat /etc/resolv.conf | grep \"^search\"").Output()
	if err != nil {
		return fmt.Errorf("Fail to get dns search name")
	}

	// remove the search string from the output and convert to string
	dnsSearchNames := strings.Split(string(out), " ")[1:]

	// Insert configuration to qemu commandline
	for _, dnsSearchName := range dnsSearchNames {
		s.slirpConfig.Value += fmt.Sprintf(",dnssearch=%s", dnsSearchName)
	}

	return nil
}
