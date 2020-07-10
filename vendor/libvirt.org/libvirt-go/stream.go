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
#include "stream_wrapper.h"
*/
import "C"
import (
	"io"
	"unsafe"
)

type StreamFlags int

const (
	STREAM_NONBLOCK = StreamFlags(C.VIR_STREAM_NONBLOCK)
)

type StreamEventType int

const (
	STREAM_EVENT_READABLE = StreamEventType(C.VIR_STREAM_EVENT_READABLE)
	STREAM_EVENT_WRITABLE = StreamEventType(C.VIR_STREAM_EVENT_WRITABLE)
	STREAM_EVENT_ERROR    = StreamEventType(C.VIR_STREAM_EVENT_ERROR)
	STREAM_EVENT_HANGUP   = StreamEventType(C.VIR_STREAM_EVENT_HANGUP)
)

type StreamRecvFlagsValues int

const (
	STREAM_RECV_STOP_AT_HOLE = StreamRecvFlagsValues(C.VIR_STREAM_RECV_STOP_AT_HOLE)
)

type Stream struct {
	ptr C.virStreamPtr
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamAbort
func (v *Stream) Abort() error {
	var err C.virError
	result := C.virStreamAbortWrapper(v.ptr, &err)
	if result == -1 {
		return makeError(&err)
	}

	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamFinish
func (v *Stream) Finish() error {
	var err C.virError
	result := C.virStreamFinishWrapper(v.ptr, &err)
	if result == -1 {
		return makeError(&err)
	}

	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamFree
func (v *Stream) Free() error {
	var err C.virError
	ret := C.virStreamFreeWrapper(v.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamRef
func (c *Stream) Ref() error {
	var err C.virError
	ret := C.virStreamRefWrapper(c.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamRecv
func (v *Stream) Recv(p []byte) (int, error) {
	var err C.virError
	n := C.virStreamRecvWrapper(v.ptr, (*C.char)(unsafe.Pointer(&p[0])), C.size_t(len(p)), &err)
	if n < 0 {
		return 0, makeError(&err)
	}
	if n == 0 {
		return 0, io.EOF
	}

	return int(n), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamRecvFlags
func (v *Stream) RecvFlags(p []byte, flags StreamRecvFlagsValues) (int, error) {
	if C.LIBVIR_VERSION_NUMBER < 3004000 {
		return 0, makeNotImplementedError("virStreamRecvFlags")
	}

	var err C.virError
	n := C.virStreamRecvFlagsWrapper(v.ptr, (*C.char)(unsafe.Pointer(&p[0])), C.size_t(len(p)), C.uint(flags), &err)
	if n < 0 {
		return 0, makeError(&err)
	}
	if n == 0 {
		return 0, io.EOF
	}

	return int(n), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamRecvHole
func (v *Stream) RecvHole(flags uint) (int64, error) {
	if C.LIBVIR_VERSION_NUMBER < 3004000 {
		return 0, makeNotImplementedError("virStreamSparseRecvHole")
	}

	var len C.longlong
	var err C.virError
	ret := C.virStreamRecvHoleWrapper(v.ptr, &len, C.uint(flags), &err)
	if ret < 0 {
		return 0, makeError(&err)
	}

	return int64(len), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamSend
func (v *Stream) Send(p []byte) (int, error) {
	var err C.virError
	n := C.virStreamSendWrapper(v.ptr, (*C.char)(unsafe.Pointer(&p[0])), C.size_t(len(p)), &err)
	if n < 0 {
		return 0, makeError(&err)
	}
	if n == 0 {
		return 0, io.EOF
	}

	return int(n), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamSendHole
func (v *Stream) SendHole(len int64, flags uint32) error {
	if C.LIBVIR_VERSION_NUMBER < 3004000 {
		return makeNotImplementedError("virStreamSendHole")
	}

	var err C.virError
	ret := C.virStreamSendHoleWrapper(v.ptr, C.longlong(len), C.uint(flags), &err)
	if ret < 0 {
		return makeError(&err)
	}

	return nil
}

type StreamSinkFunc func(*Stream, []byte) (int, error)
type StreamSinkHoleFunc func(*Stream, int64) error

//export streamSinkCallback
func streamSinkCallback(stream C.virStreamPtr, cdata *C.char, nbytes C.size_t, callbackID int) int {
	callbackFunc := getCallbackId(callbackID)

	callback, ok := callbackFunc.(StreamSinkFunc)
	if !ok {
		panic("Incorrect stream sink func callback")
	}

	data := make([]byte, int(nbytes))
	for i := 0; i < int(nbytes); i++ {
		cdatabyte := (*C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(cdata)) + (unsafe.Sizeof(*cdata) * uintptr(i))))
		data[i] = (byte)(*cdatabyte)
	}

	retnbytes, err := callback(&Stream{ptr: stream}, data)
	if err != nil {
		return -1
	}

	return retnbytes
}

//export streamSinkHoleCallback
func streamSinkHoleCallback(stream C.virStreamPtr, length C.longlong, callbackID int) int {
	callbackFunc := getCallbackId(callbackID)

	callback, ok := callbackFunc.(StreamSinkHoleFunc)
	if !ok {
		panic("Incorrect stream sink hole func callback")
	}

	err := callback(&Stream{ptr: stream}, int64(length))
	if err != nil {
		return -1
	}

	return 0
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamRecvAll
func (v *Stream) RecvAll(handler StreamSinkFunc) error {

	callbackID := registerCallbackId(handler)

	var err C.virError
	ret := C.virStreamRecvAllWrapper(v.ptr, (C.int)(callbackID), &err)
	freeCallbackId(callbackID)
	if ret == -1 {
		return makeError(&err)
	}

	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamSparseRecvAll
func (v *Stream) SparseRecvAll(handler StreamSinkFunc, holeHandler StreamSinkHoleFunc) error {
	if C.LIBVIR_VERSION_NUMBER < 3004000 {
		return makeNotImplementedError("virStreamSparseSendAll")
	}

	callbackID := registerCallbackId(handler)
	holeCallbackID := registerCallbackId(holeHandler)

	var err C.virError
	ret := C.virStreamSparseRecvAllWrapper(v.ptr, (C.int)(callbackID), (C.int)(holeCallbackID), &err)
	freeCallbackId(callbackID)
	freeCallbackId(holeCallbackID)
	if ret == -1 {
		return makeError(&err)
	}

	return nil
}

type StreamSourceFunc func(*Stream, int) ([]byte, error)
type StreamSourceHoleFunc func(*Stream) (bool, int64, error)
type StreamSourceSkipFunc func(*Stream, int64) error

//export streamSourceCallback
func streamSourceCallback(stream C.virStreamPtr, cdata *C.char, nbytes C.size_t, callbackID int) int {
	callbackFunc := getCallbackId(callbackID)

	callback, ok := callbackFunc.(StreamSourceFunc)
	if !ok {
		panic("Incorrect stream sink func callback")
	}

	data, err := callback(&Stream{ptr: stream}, (int)(nbytes))
	if err != nil {
		return -1
	}

	nretbytes := int(nbytes)
	if len(data) < nretbytes {
		nretbytes = len(data)
	}

	for i := 0; i < nretbytes; i++ {
		cdatabyte := (*C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(cdata)) + (unsafe.Sizeof(*cdata) * uintptr(i))))
		*cdatabyte = (C.char)(data[i])
	}

	return nretbytes
}

//export streamSourceHoleCallback
func streamSourceHoleCallback(stream C.virStreamPtr, cinData *C.int, clength *C.longlong, callbackID int) int {
	callbackFunc := getCallbackId(callbackID)

	callback, ok := callbackFunc.(StreamSourceHoleFunc)
	if !ok {
		panic("Incorrect stream sink hole func callback")
	}

	inData, length, err := callback(&Stream{ptr: stream})
	if err != nil {
		return -1
	}

	if inData {
		*cinData = 1
	} else {
		*cinData = 0
	}
	*clength = C.longlong(length)

	return 0
}

//export streamSourceSkipCallback
func streamSourceSkipCallback(stream C.virStreamPtr, length C.longlong, callbackID int) int {
	callbackFunc := getCallbackId(callbackID)

	callback, ok := callbackFunc.(StreamSourceSkipFunc)
	if !ok {
		panic("Incorrect stream sink skip func callback")
	}

	err := callback(&Stream{ptr: stream}, int64(length))
	if err != nil {
		return -1
	}

	return 0
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamSendAll
func (v *Stream) SendAll(handler StreamSourceFunc) error {

	callbackID := registerCallbackId(handler)

	var err C.virError
	ret := C.virStreamSendAllWrapper(v.ptr, (C.int)(callbackID), &err)
	freeCallbackId(callbackID)
	if ret == -1 {
		return makeError(&err)
	}

	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamSparseSendAll
func (v *Stream) SparseSendAll(handler StreamSourceFunc, holeHandler StreamSourceHoleFunc, skipHandler StreamSourceSkipFunc) error {
	if C.LIBVIR_VERSION_NUMBER < 3004000 {
		return makeNotImplementedError("virStreamSparseSendAll")
	}

	callbackID := registerCallbackId(handler)
	holeCallbackID := registerCallbackId(holeHandler)
	skipCallbackID := registerCallbackId(skipHandler)

	var err C.virError
	ret := C.virStreamSparseSendAllWrapper(v.ptr, (C.int)(callbackID), (C.int)(holeCallbackID), (C.int)(skipCallbackID), &err)
	freeCallbackId(callbackID)
	freeCallbackId(holeCallbackID)
	freeCallbackId(skipCallbackID)
	if ret == -1 {
		return makeError(&err)
	}

	return nil
}

type StreamEventCallback func(*Stream, StreamEventType)

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamEventAddCallback
func (v *Stream) EventAddCallback(events StreamEventType, callback StreamEventCallback) error {
	callbackID := registerCallbackId(callback)

	var err C.virError
	ret := C.virStreamEventAddCallbackWrapper(v.ptr, (C.int)(events), (C.int)(callbackID), &err)
	if ret == -1 {
		return makeError(&err)
	}

	return nil
}

//export streamEventCallback
func streamEventCallback(st C.virStreamPtr, events int, callbackID int) {
	callbackFunc := getCallbackId(callbackID)

	callback, ok := callbackFunc.(StreamEventCallback)
	if !ok {
		panic("Incorrect stream event func callback")
	}

	callback(&Stream{ptr: st}, StreamEventType(events))
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamEventUpdateCallback
func (v *Stream) EventUpdateCallback(events StreamEventType) error {
	var err C.virError
	ret := C.virStreamEventUpdateCallbackWrapper(v.ptr, (C.int)(events), &err)
	if ret == -1 {
		return makeError(&err)
	}

	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-stream.html#virStreamEventRemoveCallback
func (v *Stream) EventRemoveCallback() error {
	var err C.virError
	ret := C.virStreamEventRemoveCallbackWrapper(v.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}

	return nil
}
