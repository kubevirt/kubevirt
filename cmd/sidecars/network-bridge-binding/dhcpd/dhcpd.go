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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package dhcpd

import (
	"crypto/sha1"
	"fmt"
	vmschema "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	dhcpserver "kubevirt.io/kubevirt/pkg/network/dhcp/server"
	"kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"net"
	"os"

	"kubevirt.io/client-go/log"
)

const (
	lockFile       = "/tmp/dhcpd.pid"
	advInterface   = "k6t-eth0"
	dummyInterface = "eth0"
)

type Server struct {
	MacChan chan string
	Driver  driver.NetworkUtilsHandler
	HwAddr  net.HardwareAddr
	//
	nic           cache.DHCPConfig
	dhcpOptions   *vmschema.DHCPOptions
	nameservers   [][]byte
	searchDomains []string
}

func NewDhcpServer(macChan chan string) *Server {
	return &Server{
		Driver:  driver.NetworkUtilsHandler{},
		MacChan: macChan,
	}
}

func (m *Server) Start() error {
	var err error
	mac := <-m.MacChan

	if m.HwAddr, err = net.ParseMAC(mac); err != nil {
		return fmt.Errorf("cannot parse MAC address: %v", err)
	}

	log.Log.Infof("Got new mac request: %v", m.HwAddr)

	if err = m.PrepareConfig(); err != nil {
		return fmt.Errorf("cannot prepare DHCP config: %s", err)
	}

	if err = m.RunDhcpServer(); err != nil {
		return fmt.Errorf("cannot run DHCP server: %s", err)
	}

	return nil
}

// GenerateMac generates a MAC address based on the vmi.UID
func (m *Server) GenerateMac(vmi *vmschema.VirtualMachineInstance) net.HardwareAddr {
	hash := sha1.New()
	hash.Write([]byte(vmi.UID))
	hashed := hash.Sum(nil)

	mac := make([]byte, 6)

	// Set the first 2 bytes to the QEMU prefix (52:54)
	mac[0] = 0x52
	mac[1] = 0x54

	copy(mac[2:], hashed[:4])

	return net.HardwareAddr(mac)
}

func (m *Server) PrepareConfig() error {
	ipv4, _, err := m.Driver.ReadIPAddressesFromLink(dummyInterface)
	if err != nil {
		return fmt.Errorf("cannot parse IP address from %s: %s", dummyInterface, err)
	}

	log.Log.Infof("Prepare DhcpdConfig: Interface %s , IP: %+v, MAC: %s", advInterface, ipv4, m.HwAddr)

	addr, err := m.Driver.ParseAddr(ipv4 + "/32")
	if err != nil {
		return fmt.Errorf("cannot parse %s to link.Addr %s", dummyInterface, err)
	}

	m.nic = cache.DHCPConfig{
		Name:              dummyInterface,
		MAC:               m.HwAddr,
		IP:                *addr,
		AdvertisingIPAddr: net.ParseIP("169.254.75.10"),
		Gateway:           net.ParseIP("169.254.1.1"),
		Mtu:               uint16(1480),
	}

	m.nameservers, m.searchDomains, err = converter.GetResolvConfDetailsFromPod()
	if err != nil {
		return fmt.Errorf("cannot get resolv.conf details from pod: %s", err)
	}
	return nil
}

func (m *Server) RunDhcpServer() error {
	// Check if the lock file exists
	if _, err := os.Stat(lockFile); err == nil {
		return fmt.Errorf("dhcpd lock file already exists")
	}

	log.Log.Infof("RunDhcpServer for VM MAC: %s", m.HwAddr.String())

	go func() {
		if err := dhcpserver.SingleClientDHCPServer(
			m.nic.MAC,
			m.nic.IP.IP,
			m.nic.IP.Mask,
			advInterface,
			m.nic.AdvertisingIPAddr,
			m.nic.Gateway,
			m.nameservers,
			m.nic.Routes,
			m.searchDomains,
			m.nic.Mtu,
			m.dhcpOptions,
		); err != nil {
			log.Log.Errorf("failed to run DHCP Server: %v", err)
			panic(err)
		}
	}()

	// Create the lock file
	file, err := os.Create(lockFile)
	if err != nil {
		return fmt.Errorf("error creating lock file: %v", err)
	}
	defer file.Close()

	return nil
}
