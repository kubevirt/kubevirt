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
#include <stdint.h>
#include "events_wrapper.h"
*/
import "C"

type EventHandleType int

const (
	EVENT_HANDLE_READABLE = EventHandleType(C.VIR_EVENT_HANDLE_READABLE)
	EVENT_HANDLE_WRITABLE = EventHandleType(C.VIR_EVENT_HANDLE_WRITABLE)
	EVENT_HANDLE_ERROR    = EventHandleType(C.VIR_EVENT_HANDLE_ERROR)
	EVENT_HANDLE_HANGUP   = EventHandleType(C.VIR_EVENT_HANDLE_HANGUP)
)

// See also https://libvirt.org/html/libvirt-libvirt-event.html#virEventRegisterDefaultImpl
func EventRegisterDefaultImpl() error {
	var err C.virError
	if i := int(C.virEventRegisterDefaultImplWrapper(&err)); i != 0 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-event.html#virEventRunDefaultImpl
func EventRunDefaultImpl() error {
	var err C.virError
	if i := int(C.virEventRunDefaultImplWrapper(&err)); i != 0 {
		return makeError(&err)
	}
	return nil
}

type EventHandleCallback func(watch int, file int, events EventHandleType)

//export eventHandleCallback
func eventHandleCallback(watch int, fd int, events int, callbackID int) {
	callbackFunc := getCallbackId(callbackID)

	callback, ok := callbackFunc.(EventHandleCallback)
	if !ok {
		panic("Incorrect event handle callback data")
	}

	callback(watch, fd, (EventHandleType)(events))
}

// See also https://libvirt.org/html/libvirt-libvirt-event.html#virEventAddHandle
func EventAddHandle(fd int, events EventHandleType, callback EventHandleCallback) (int, error) {
	callbackID := registerCallbackId(callback)

	var err C.virError
	ret := C.virEventAddHandleWrapper((C.int)(fd), (C.int)(events), (C.int)(callbackID), &err)
	if ret == -1 {
		return 0, makeError(&err)
	}

	return int(ret), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-event.html#virEventUpdateHandle
func EventUpdateHandle(watch int, events EventHandleType) {
	C.virEventUpdateHandle((C.int)(watch), (C.int)(events))
}

// See also https://libvirt.org/html/libvirt-libvirt-event.html#virEventRemoveHandle
func EventRemoveHandle(watch int) error {
	var err C.virError
	ret := C.virEventRemoveHandleWrapper((C.int)(watch), &err)
	if ret < 0 {
		return makeError(&err)
	}

	return nil
}

type EventTimeoutCallback func(timer int)

//export eventTimeoutCallback
func eventTimeoutCallback(timer int, callbackID int) {
	callbackFunc := getCallbackId(callbackID)

	callback, ok := callbackFunc.(EventTimeoutCallback)
	if !ok {
		panic("Incorrect event timeout callback data")
	}

	callback(timer)
}

// See also https://libvirt.org/html/libvirt-libvirt-event.html#virEventAddTimeout
func EventAddTimeout(freq int, callback EventTimeoutCallback) (int, error) {
	callbackID := registerCallbackId(callback)

	var err C.virError
	ret := C.virEventAddTimeoutWrapper((C.int)(freq), (C.int)(callbackID), &err)
	if ret == -1 {
		return 0, makeError(&err)
	}

	return int(ret), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-event.html#virEventUpdateTimeout
func EventUpdateTimeout(timer int, freq int) {
	C.virEventUpdateTimeout((C.int)(timer), (C.int)(freq))
}

// See also https://libvirt.org/html/libvirt-libvirt-event.html#virEventRemoveTimeout
func EventRemoveTimeout(timer int) error {
	var err C.virError
	ret := C.virEventRemoveTimeoutWrapper((C.int)(timer), &err)
	if ret < 0 {
		return makeError(&err)
	}

	return nil
}

type EventHandleCallbackInfo struct {
	callback uintptr
	opaque   uintptr
	free     uintptr
}

type EventTimeoutCallbackInfo struct {
	callback uintptr
	opaque   uintptr
	free     uintptr
}

func (i *EventHandleCallbackInfo) Invoke(watch int, fd int, event EventHandleType) {
	C.eventHandleCallbackInvoke(C.int(watch), C.int(fd), C.int(event), C.uintptr_t(i.callback), C.uintptr_t(i.opaque))
}

func (i *EventTimeoutCallbackInfo) Invoke(timer int) {
	C.eventTimeoutCallbackInvoke(C.int(timer), C.uintptr_t(i.callback), C.uintptr_t(i.opaque))
}

func (i *EventHandleCallbackInfo) Free() {
	C.eventHandleCallbackFree(C.uintptr_t(i.free), C.uintptr_t(i.opaque))
}

func (i *EventTimeoutCallbackInfo) Free() {
	C.eventTimeoutCallbackFree(C.uintptr_t(i.free), C.uintptr_t(i.opaque))
}

type EventLoop interface {
	AddHandleFunc(fd int, event EventHandleType, callback *EventHandleCallbackInfo) int
	UpdateHandleFunc(watch int, event EventHandleType)
	RemoveHandleFunc(watch int) int
	AddTimeoutFunc(freq int, callback *EventTimeoutCallbackInfo) int
	UpdateTimeoutFunc(timer int, freq int)
	RemoveTimeoutFunc(timer int) int
}

var eventLoopImpl EventLoop

// See also https://libvirt.org/html/libvirt-libvirt-event.html#virEventRegisterImpl
func EventRegisterImpl(impl EventLoop) {
	eventLoopImpl = impl
	C.virEventRegisterImplWrapper()
}

//export eventAddHandleFunc
func eventAddHandleFunc(fd C.int, event C.int, callback uintptr, opaque uintptr, free uintptr) C.int {
	if eventLoopImpl == nil {
		panic("Event loop impl is missing")
	}

	cbinfo := &EventHandleCallbackInfo{
		callback: callback,
		opaque:   opaque,
		free:     free,
	}

	return C.int(eventLoopImpl.AddHandleFunc(int(fd), EventHandleType(event), cbinfo))
}

//export eventUpdateHandleFunc
func eventUpdateHandleFunc(watch C.int, event C.int) {
	if eventLoopImpl == nil {
		panic("Event loop impl is missing")
	}

	eventLoopImpl.UpdateHandleFunc(int(watch), EventHandleType(event))
}

//export eventRemoveHandleFunc
func eventRemoveHandleFunc(watch C.int) {
	if eventLoopImpl == nil {
		panic("Event loop impl is missing")
	}

	eventLoopImpl.RemoveHandleFunc(int(watch))
}

//export eventAddTimeoutFunc
func eventAddTimeoutFunc(freq C.int, callback uintptr, opaque uintptr, free uintptr) C.int {
	if eventLoopImpl == nil {
		panic("Event loop impl is missing")
	}

	cbinfo := &EventTimeoutCallbackInfo{
		callback: callback,
		opaque:   opaque,
		free:     free,
	}

	return C.int(eventLoopImpl.AddTimeoutFunc(int(freq), cbinfo))
}

//export eventUpdateTimeoutFunc
func eventUpdateTimeoutFunc(timer C.int, freq C.int) {
	if eventLoopImpl == nil {
		panic("Event loop impl is missing")
	}

	eventLoopImpl.UpdateTimeoutFunc(int(timer), int(freq))
}

//export eventRemoveTimeoutFunc
func eventRemoveTimeoutFunc(timer C.int) {
	if eventLoopImpl == nil {
		panic("Event loop impl is missing")
	}

	eventLoopImpl.RemoveTimeoutFunc(int(timer))
}
