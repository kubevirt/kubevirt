/*
 * This file is part of the libvirt-go project
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 * Copyright (c) 2013 Alex Zorin
 * Copyright (C) 2016 Red Hat, Inc.
 *
 */

package libvirt

/*
#cgo pkg-config: libvirt
#include <stdlib.h>
#include "network_wrapper.h"
*/
import "C"

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"
)

type IPAddrType int

const (
	IP_ADDR_TYPE_IPV4 = IPAddrType(C.VIR_IP_ADDR_TYPE_IPV4)
	IP_ADDR_TYPE_IPV6 = IPAddrType(C.VIR_IP_ADDR_TYPE_IPV6)
)

type NetworkXMLFlags int

const (
	NETWORK_XML_INACTIVE = NetworkXMLFlags(C.VIR_NETWORK_XML_INACTIVE)
)

type NetworkUpdateCommand int

const (
	NETWORK_UPDATE_COMMAND_NONE      = NetworkUpdateCommand(C.VIR_NETWORK_UPDATE_COMMAND_NONE)
	NETWORK_UPDATE_COMMAND_MODIFY    = NetworkUpdateCommand(C.VIR_NETWORK_UPDATE_COMMAND_MODIFY)
	NETWORK_UPDATE_COMMAND_DELETE    = NetworkUpdateCommand(C.VIR_NETWORK_UPDATE_COMMAND_DELETE)
	NETWORK_UPDATE_COMMAND_ADD_LAST  = NetworkUpdateCommand(C.VIR_NETWORK_UPDATE_COMMAND_ADD_LAST)
	NETWORK_UPDATE_COMMAND_ADD_FIRST = NetworkUpdateCommand(C.VIR_NETWORK_UPDATE_COMMAND_ADD_FIRST)
)

type NetworkUpdateSection int

const (
	NETWORK_SECTION_NONE              = NetworkUpdateSection(C.VIR_NETWORK_SECTION_NONE)
	NETWORK_SECTION_BRIDGE            = NetworkUpdateSection(C.VIR_NETWORK_SECTION_BRIDGE)
	NETWORK_SECTION_DOMAIN            = NetworkUpdateSection(C.VIR_NETWORK_SECTION_DOMAIN)
	NETWORK_SECTION_IP                = NetworkUpdateSection(C.VIR_NETWORK_SECTION_IP)
	NETWORK_SECTION_IP_DHCP_HOST      = NetworkUpdateSection(C.VIR_NETWORK_SECTION_IP_DHCP_HOST)
	NETWORK_SECTION_IP_DHCP_RANGE     = NetworkUpdateSection(C.VIR_NETWORK_SECTION_IP_DHCP_RANGE)
	NETWORK_SECTION_FORWARD           = NetworkUpdateSection(C.VIR_NETWORK_SECTION_FORWARD)
	NETWORK_SECTION_FORWARD_INTERFACE = NetworkUpdateSection(C.VIR_NETWORK_SECTION_FORWARD_INTERFACE)
	NETWORK_SECTION_FORWARD_PF        = NetworkUpdateSection(C.VIR_NETWORK_SECTION_FORWARD_PF)
	NETWORK_SECTION_PORTGROUP         = NetworkUpdateSection(C.VIR_NETWORK_SECTION_PORTGROUP)
	NETWORK_SECTION_DNS_HOST          = NetworkUpdateSection(C.VIR_NETWORK_SECTION_DNS_HOST)
	NETWORK_SECTION_DNS_TXT           = NetworkUpdateSection(C.VIR_NETWORK_SECTION_DNS_TXT)
	NETWORK_SECTION_DNS_SRV           = NetworkUpdateSection(C.VIR_NETWORK_SECTION_DNS_SRV)
)

type NetworkUpdateFlags int

const (
	NETWORK_UPDATE_AFFECT_CURRENT = NetworkUpdateFlags(C.VIR_NETWORK_UPDATE_AFFECT_CURRENT)
	NETWORK_UPDATE_AFFECT_LIVE    = NetworkUpdateFlags(C.VIR_NETWORK_UPDATE_AFFECT_LIVE)
	NETWORK_UPDATE_AFFECT_CONFIG  = NetworkUpdateFlags(C.VIR_NETWORK_UPDATE_AFFECT_CONFIG)
)

type NetworkEventLifecycleType int

const (
	NETWORK_EVENT_DEFINED   = NetworkEventLifecycleType(C.VIR_NETWORK_EVENT_DEFINED)
	NETWORK_EVENT_UNDEFINED = NetworkEventLifecycleType(C.VIR_NETWORK_EVENT_UNDEFINED)
	NETWORK_EVENT_STARTED   = NetworkEventLifecycleType(C.VIR_NETWORK_EVENT_STARTED)
	NETWORK_EVENT_STOPPED   = NetworkEventLifecycleType(C.VIR_NETWORK_EVENT_STOPPED)
)

type NetworkEventID int

const (
	NETWORK_EVENT_ID_LIFECYCLE = NetworkEventID(C.VIR_NETWORK_EVENT_ID_LIFECYCLE)
)

type Network struct {
	ptr C.virNetworkPtr
}

type NetworkDHCPLease struct {
	Iface      string
	ExpiryTime time.Time
	Type       IPAddrType
	Mac        string
	Iaid       string
	IPaddr     string
	Prefix     uint
	Hostname   string
	Clientid   string
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkFree
func (n *Network) Free() error {
	var err C.virError
	ret := C.virNetworkFreeWrapper(n.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkRef
func (c *Network) Ref() error {
	var err C.virError
	ret := C.virNetworkRefWrapper(c.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkCreate
func (n *Network) Create() error {
	var err C.virError
	result := C.virNetworkCreateWrapper(n.ptr, &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkDestroy
func (n *Network) Destroy() error {
	var err C.virError
	result := C.virNetworkDestroyWrapper(n.ptr, &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkIsActive
func (n *Network) IsActive() (bool, error) {
	var err C.virError
	result := C.virNetworkIsActiveWrapper(n.ptr, &err)
	if result == -1 {
		return false, makeError(&err)
	}
	if result == 1 {
		return true, nil
	}
	return false, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkIsPersistent
func (n *Network) IsPersistent() (bool, error) {
	var err C.virError
	result := C.virNetworkIsPersistentWrapper(n.ptr, &err)
	if result == -1 {
		return false, makeError(&err)
	}
	if result == 1 {
		return true, nil
	}
	return false, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkGetAutostart
func (n *Network) GetAutostart() (bool, error) {
	var out C.int
	var err C.virError
	result := C.virNetworkGetAutostartWrapper(n.ptr, (*C.int)(unsafe.Pointer(&out)), &err)
	if result == -1 {
		return false, makeError(&err)
	}
	switch out {
	case 1:
		return true, nil
	default:
		return false, nil
	}
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkSetAutostart
func (n *Network) SetAutostart(autostart bool) error {
	var cAutostart C.int
	switch autostart {
	case true:
		cAutostart = 1
	default:
		cAutostart = 0
	}
	var err C.virError
	result := C.virNetworkSetAutostartWrapper(n.ptr, cAutostart, &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkGetName
func (n *Network) GetName() (string, error) {
	var err C.virError
	name := C.virNetworkGetNameWrapper(n.ptr, &err)
	if name == nil {
		return "", makeError(&err)
	}
	return C.GoString(name), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkGetUUID
func (n *Network) GetUUID() ([]byte, error) {
	var cUuid [C.VIR_UUID_BUFLEN](byte)
	cuidPtr := unsafe.Pointer(&cUuid)
	var err C.virError
	result := C.virNetworkGetUUIDWrapper(n.ptr, (*C.uchar)(cuidPtr), &err)
	if result != 0 {
		return []byte{}, makeError(&err)
	}
	return C.GoBytes(cuidPtr, C.VIR_UUID_BUFLEN), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkGetUUIDString
func (n *Network) GetUUIDString() (string, error) {
	var cUuid [C.VIR_UUID_STRING_BUFLEN](C.char)
	cuidPtr := unsafe.Pointer(&cUuid)
	var err C.virError
	result := C.virNetworkGetUUIDStringWrapper(n.ptr, (*C.char)(cuidPtr), &err)
	if result != 0 {
		return "", makeError(&err)
	}
	return C.GoString((*C.char)(cuidPtr)), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkGetBridgeName
func (n *Network) GetBridgeName() (string, error) {
	var err C.virError
	result := C.virNetworkGetBridgeNameWrapper(n.ptr, &err)
	if result == nil {
		return "", makeError(&err)
	}
	bridge := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return bridge, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkGetXMLDesc
func (n *Network) GetXMLDesc(flags NetworkXMLFlags) (string, error) {
	var err C.virError
	result := C.virNetworkGetXMLDescWrapper(n.ptr, C.uint(flags), &err)
	if result == nil {
		return "", makeError(&err)
	}
	xml := C.GoString(result)
	C.free(unsafe.Pointer(result))
	return xml, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkUndefine
func (n *Network) Undefine() error {
	var err C.virError
	result := C.virNetworkUndefineWrapper(n.ptr, &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkUpdate
func (n *Network) Update(cmd NetworkUpdateCommand, section NetworkUpdateSection, parentIndex int, xml string, flags NetworkUpdateFlags) error {
	cxml := C.CString(xml)
	defer C.free(unsafe.Pointer(cxml))
	var err C.virError
	result := C.virNetworkUpdateWrapper(n.ptr, C.uint(cmd), C.uint(section), C.int(parentIndex), cxml, C.uint(flags), &err)
	if result == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkGetDHCPLeases
func (n *Network) GetDHCPLeases() ([]NetworkDHCPLease, error) {
	if C.LIBVIR_VERSION_NUMBER < 1002006 {
		return []NetworkDHCPLease{}, makeNotImplementedError("virNetworkGetDHCPLeases")
	}
	var cLeases *C.virNetworkDHCPLeasePtr
	var err C.virError
	numLeases := C.virNetworkGetDHCPLeasesWrapper(n.ptr, nil, (**C.virNetworkDHCPLeasePtr)(&cLeases), C.uint(0), &err)
	if numLeases == -1 {
		return nil, makeError(&err)
	}
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(cLeases)),
		Len:  int(numLeases),
		Cap:  int(numLeases),
	}
	var leases []NetworkDHCPLease
	slice := *(*[]C.virNetworkDHCPLeasePtr)(unsafe.Pointer(&hdr))
	for _, clease := range slice {
		leases = append(leases, NetworkDHCPLease{
			Iface:      C.GoString(clease.iface),
			ExpiryTime: time.Unix(int64(clease.expirytime), 0),
			Type:       IPAddrType(clease._type),
			Mac:        C.GoString(clease.mac),
			Iaid:       C.GoString(clease.iaid),
			IPaddr:     C.GoString(clease.ipaddr),
			Prefix:     uint(clease.prefix),
			Hostname:   C.GoString(clease.hostname),
			Clientid:   C.GoString(clease.clientid),
		})
		C.virNetworkDHCPLeaseFreeWrapper(clease)
	}
	C.free(unsafe.Pointer(cLeases))
	return leases, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkPortLookupByUUIDString
func (n *Network) LookupNetworkPortByUUIDString(uuid string) (*NetworkPort, error) {
	if C.LIBVIR_VERSION_NUMBER < 5005000 {
		return nil, makeNotImplementedError("virNetworkPortLookupByUUIDString")
	}

	cUuid := C.CString(uuid)
	defer C.free(unsafe.Pointer(cUuid))
	var err C.virError
	ptr := C.virNetworkPortLookupByUUIDStringWrapper(n.ptr, cUuid, &err)
	if ptr == nil {
		return nil, makeError(&err)
	}
	return &NetworkPort{ptr: ptr}, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkPortLookupByUUID
func (n *Network) LookupNetworkPortByUUID(uuid []byte) (*NetworkPort, error) {
	if C.LIBVIR_VERSION_NUMBER < 5005000 {
		return nil, makeNotImplementedError("virNetworkPortLookupByUUID")
	}

	if len(uuid) != C.VIR_UUID_BUFLEN {
		return nil, fmt.Errorf("UUID must be exactly %d bytes in size",
			int(C.VIR_UUID_BUFLEN))
	}
	cUuid := make([]C.uchar, C.VIR_UUID_BUFLEN)
	for i := 0; i < C.VIR_UUID_BUFLEN; i++ {
		cUuid[i] = C.uchar(uuid[i])
	}
	var err C.virError
	ptr := C.virNetworkPortLookupByUUIDWrapper(n.ptr, &cUuid[0], &err)
	if ptr == nil {
		return nil, makeError(&err)
	}
	return &NetworkPort{ptr: ptr}, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkPortCreateXML
func (n *Network) PortCreateXML(xmlConfig string, flags uint) (*NetworkPort, error) {
	if C.LIBVIR_VERSION_NUMBER < 5005000 {
		return nil, makeNotImplementedError("virNetworkPortCreateXML")
	}
	cXml := C.CString(string(xmlConfig))
	defer C.free(unsafe.Pointer(cXml))
	var err C.virError
	ptr := C.virNetworkPortCreateXMLWrapper(n.ptr, cXml, C.uint(flags), &err)
	if ptr == nil {
		return nil, makeError(&err)
	}
	return &NetworkPort{ptr: ptr}, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkListAllPorts
func (n *Network) ListAllPorts(flags uint) ([]NetworkPort, error) {
	if C.LIBVIR_VERSION_NUMBER < 5005000 {
		return []NetworkPort{}, makeNotImplementedError("virNetworkListAllPorts")
	}

	var cList *C.virNetworkPortPtr
	var err C.virError
	numPorts := C.virNetworkListAllPortsWrapper(n.ptr, (**C.virNetworkPortPtr)(&cList), C.uint(flags), &err)
	if numPorts == -1 {
		return []NetworkPort{}, makeError(&err)
	}
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(cList)),
		Len:  int(numPorts),
		Cap:  int(numPorts),
	}
	var ports []NetworkPort
	slice := *(*[]C.virNetworkPortPtr)(unsafe.Pointer(&hdr))
	for _, ptr := range slice {
		ports = append(ports, NetworkPort{ptr})
	}
	C.free(unsafe.Pointer(cList))
	return ports, nil
}
