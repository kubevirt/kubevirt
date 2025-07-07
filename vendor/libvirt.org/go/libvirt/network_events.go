/*
 * This file is part of the libvirt-go-module project
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

import (
	"fmt"
	"unsafe"
)

/*
#cgo !libvirt_dlopen pkg-config: libvirt
#cgo libvirt_dlopen LDFLAGS: -ldl
#cgo libvirt_dlopen CFLAGS: -DLIBVIRT_DLOPEN
#include "network_events_helper.h"
*/
import "C"

type NetworkEventLifecycle struct {
	Event NetworkEventLifecycleType
	// TODO: we can make Detail typesafe somehow ?
	Detail int
}

type NetworkEventLifecycleCallback func(c *Connect, n *Network, event *NetworkEventLifecycle)

type NetworkEventMetadataChange struct {
	Type  NetworkMetadataType
	NSURI string
}

type NetworkEventMetadataChangeCallback func(c *Connect, n *Network, event *NetworkEventMetadataChange)

//export networkEventLifecycleCallback
func networkEventLifecycleCallback(c C.virConnectPtr, n C.virNetworkPtr,
	event int, detail int,
	goCallbackId int) {

	network := &Network{ptr: n}
	connection := &Connect{ptr: c}

	eventDetails := &NetworkEventLifecycle{
		Event:  NetworkEventLifecycleType(event),
		Detail: detail,
	}

	callbackFunc := getCallbackId(goCallbackId)
	callback, ok := callbackFunc.(NetworkEventLifecycleCallback)
	if !ok {
		panic("Inappropriate callback type called")
	}
	callback(connection, network, eventDetails)
}

//export networkEventMetadataChangeCallback
func networkEventMetadataChangeCallback(c C.virConnectPtr, d C.virNetworkPtr,
	mtype int, nsuri *C.char, goCallbackId int) {

	network := &Network{ptr: d}
	connection := &Connect{ptr: c}

	eventDetails := &NetworkEventMetadataChange{
		Type:  NetworkMetadataType(mtype),
		NSURI: C.GoString(nsuri),
	}
	callbackFunc := getCallbackId(goCallbackId)
	callback, ok := callbackFunc.(NetworkEventMetadataChangeCallback)
	if !ok {
		panic("Inappropriate callback type called")
	}
	callback(connection, network, eventDetails)
}

func (c *Connect) NetworkEventLifecycleRegister(net *Network, callback NetworkEventLifecycleCallback) (int, error) {
	goCallBackId := registerCallbackId(callback)
	callbackPtr := unsafe.Pointer(C.networkEventLifecycleCallbackHelper)
	var cnet C.virNetworkPtr
	if net != nil {
		cnet = net.ptr
	}
	var err C.virError
	ret := C.virConnectNetworkEventRegisterAnyHelper(c.ptr, cnet,
		C.VIR_NETWORK_EVENT_ID_LIFECYCLE,
		C.virConnectNetworkEventGenericCallback(callbackPtr),
		C.long(goCallBackId), &err)
	if ret == -1 {
		freeCallbackId(goCallBackId)
		return 0, makeError(&err)
	}
	return int(ret), nil
}

func (c *Connect) NetworkEventMetadataChangeRegister(net *Network, callback NetworkEventMetadataChangeCallback) (int, error) {
	goCallBackId := registerCallbackId(callback)
	callbackPtr := unsafe.Pointer(C.networkEventMetadataChangeCallbackHelper)
	var cnet C.virNetworkPtr
	if net != nil {
		cnet = net.ptr
	}
	var err C.virError
	ret := C.virConnectNetworkEventRegisterAnyHelper(c.ptr, cnet,
		C.VIR_NETWORK_EVENT_ID_METADATA_CHANGE,
		C.virConnectNetworkEventGenericCallback(callbackPtr),
		C.long(goCallBackId), &err)
	if ret == -1 {
		freeCallbackId(goCallBackId)
		return 0, makeError(&err)
	}
	return int(ret), nil
}

func (c *Connect) NetworkEventDeregister(callbackId int) error {
	// Deregister the callback
	var err C.virError
	ret := int(C.virConnectNetworkEventDeregisterAnyWrapper(c.ptr, C.int(callbackId), &err))
	if ret < 0 {
		return makeError(&err)
	}
	return nil
}

func (e NetworkEventLifecycle) String() string {
	var event string
	switch e.Event {
	case NETWORK_EVENT_DEFINED:
		event = "defined"

	case NETWORK_EVENT_UNDEFINED:
		event = "undefined"

	case NETWORK_EVENT_STARTED:
		event = "started"

	case NETWORK_EVENT_STOPPED:
		event = "stopped"

	default:
		event = "unknown"
	}

	return fmt.Sprintf("Network event=%q", event)
}
