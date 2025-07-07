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
 * Copyright 2022 Red Hat, Inc.
 *
 */

// Originally inspired by: https://github.com/weaveworks/weave/blob/master/net/ethtool.go

package ethtool

import (
	"fmt"
	"syscall"
	"unsafe"

	"kubevirt.io/client-go/log"
)

type Ethtool struct{}

const (
	SIOCETHTOOL     = 0x8946     // linux/sockios.h
	ETHTOOL_GTXCSUM = 0x00000016 // linux/ethtool.h
	ETHTOOL_STXCSUM = 0x00000017 // linux/ethtool.h
	IFNAMSIZ        = 16         // linux/if.h
)

func (e Ethtool) TXChecksumOff(name string) error {
	return ethtoolTXOff(name)
}

func (e Ethtool) ReadTXChecksum(name string) (bool, error) {
	return readEthtoolTX(name)
}

// linux/if.h 'struct ifreq'
type IFReqData struct {
	Name [IFNAMSIZ]byte
	Data uintptr
}

// linux/ethtool.h 'struct ethtool_value'
type EthtoolValue struct {
	Cmd  uint32
	Data uint32
}

func ioctlEthtool(fd int, argp uintptr) error {
	_, _, errno := syscall.RawSyscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(SIOCETHTOOL), argp)
	if errno != 0 {
		return errno
	}
	return nil
}

// Disable TX checksum offload on specified interface
func ethtoolTXOff(name string) error {
	if len(name)+1 > IFNAMSIZ {
		return fmt.Errorf("name too long")
	}

	socket, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer closeSocketIgnoringError(socket)

	// Request current value
	value := EthtoolValue{Cmd: ETHTOOL_GTXCSUM}
	request := IFReqData{Data: uintptr(unsafe.Pointer(&value))} // #nosec Used for a RawSyscall
	copy(request.Name[:], name)

	if err := ioctlEthtool(socket, uintptr(unsafe.Pointer(&request))); err != nil { // #nosec Used for a RawSyscall
		return err
	}
	if value.Data == 0 { // if already off, don't try to change
		return nil
	}

	value = EthtoolValue{ETHTOOL_STXCSUM, 0}
	return ioctlEthtool(socket, uintptr(unsafe.Pointer(&request))) // #nosec Used for a RawSyscall
}

func readEthtoolTX(name string) (bool, error) {
	if len(name)+1 > IFNAMSIZ {
		return false, fmt.Errorf("name too long")
	}

	socket, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return false, err
	}
	defer closeSocketIgnoringError(socket)

	value := EthtoolValue{Cmd: ETHTOOL_GTXCSUM}
	request := IFReqData{Data: uintptr(unsafe.Pointer(&value))} // #nosec Used for a RawSyscall
	copy(request.Name[:], name)

	if err := ioctlEthtool(socket, uintptr(unsafe.Pointer(&request))); err != nil { // #nosec Used for a RawSyscall
		return false, err
	}
	return value.Data > 0, nil
}

func closeSocketIgnoringError(fd int) {
	if err := syscall.Close(fd); err != nil {
		log.Log.Warningf("failed to close socket file descriptor %d: %v", fd, err)
	}
}
