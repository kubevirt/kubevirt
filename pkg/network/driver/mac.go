/*
 * This file is part of the KubeVirt project
 * Most of it originates from https://github.com/subgraph/libmacouflage
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
 * Copyright (c) 2020 Subgraph
 * Copyright 2022 Red Hat, Inc.
 *
 */

package driver

import "C"
import (
	"crypto/rand"
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

type NetInfo struct {
	name   [16]byte
	family uint16
	data   [6]byte
}
type EthtoolPermAddr struct {
	cmd  uint32
	size uint32
	data [6]byte
}

type ifreq struct {
	name [16]byte
	epa  *EthtoolPermAddr
}

const (
	SIOCSIFHWADDR    = 0x8924
	SIOCETHTOOL      = 0x8946
	ETHTOOLGPERMADDR = 0x00000020
	IFHWADDRLEN      = 6
)

func getMacDetails(name string) (net.HardwareAddr, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	return iface.HardwareAddr, nil
}

func setMac(name string, mac string) (err error) {
	sockfd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	defer syscall.Close(sockfd)
	iface, err := net.ParseMAC(mac)
	if err != nil {
		return
	}
	var netinfo NetInfo
	copy(netinfo.name[:], name)
	netinfo.family = syscall.AF_UNIX
	copy(netinfo.data[:], iface)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sockfd), SIOCSIFHWADDR, uintptr(unsafe.Pointer(&netinfo)))
	if errno != 0 {
		err = errno
	}
	return
}

func getPermanentMac(name string) (mac net.HardwareAddr, err error) {
	sockfd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	defer syscall.Close(sockfd)
	var ifr ifreq
	copy(ifr.name[:], name)
	var epa EthtoolPermAddr
	epa.cmd = ETHTOOLGPERMADDR
	epa.size = IFHWADDRLEN
	ifr.epa = &epa
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sockfd), SIOCETHTOOL, uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		err = errno
		return
	}
	mac = net.HardwareAddr(C.GoBytes(unsafe.Pointer(&ifr.epa.data), 6))
	return
}

func macChanged(iface string) (changed bool, err error) {
	current, err := getMacDetails(iface)
	if err != nil {
		return false, err
	}
	permanent, err := getPermanentMac(iface)
	if err != nil {
		return false, err
	}
	if current.String() == permanent.String() {
		return false, err
	}
	return true, nil
}

func spoofMacSameVendor(name string, mac net.HardwareAddr) (changed bool, err error) {
	if len(mac) != 6 {
		err = fmt.Errorf("invalid size for macbytes byte array: %d",
			len(mac))
		return
	}
	for i := 3; i < 6; i++ {
		buf := make([]byte, 1)
		_, err = rand.Read(buf)
		if err != nil {
			return false, err
		}
		mac[i] = buf[0]
	}
	mac[0] |= 2

	err = setMac(name, mac.String())
	if err != nil {
		return false, err
	}
	changed, err = macChanged(name)
	if err != nil {
		return false, err
	}
	return changed, nil
}
