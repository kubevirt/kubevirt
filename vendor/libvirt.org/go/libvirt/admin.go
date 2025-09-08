//go:build !libvirt_without_admin
// +build !libvirt_without_admin

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
 * Copyright (C) 2025 Red Hat, Inc.
 *
 */

package libvirt

/*
#cgo !libvirt_dlopen pkg-config: libvirt-admin
#cgo libvirt_dlopen LDFLAGS: -ldl
#cgo libvirt_dlopen CFLAGS: -DLIBVIRT_DLOPEN
#include <stdlib.h>
#include "admin_helper.h"
*/
import "C"

import (
	"reflect"
	"sync"
	"unsafe"
)

type AdmConnect struct {
	ptr C.virAdmConnectPtr
}

// Additional data associated to the connection.
type virAdmConnectionData struct {
	errCallbackId   *int
	closeCallbackId *int
}

var admConnections map[C.virAdmConnectPtr]*virAdmConnectionData
var admConnectionsLock sync.RWMutex

func init() {
	admConnections = make(map[C.virAdmConnectPtr]*virAdmConnectionData)
}

func admSaveConnectionData(c *AdmConnect, d *virAdmConnectionData) {
	if c.ptr == nil {
		return // Or panic?
	}
	admConnectionsLock.Lock()
	defer admConnectionsLock.Unlock()
	admConnections[c.ptr] = d
}

func admGetConnectionData(c *AdmConnect) *virAdmConnectionData {
	admConnectionsLock.RLock()
	d := admConnections[c.ptr]
	admConnectionsLock.RUnlock()
	if d != nil {
		return d
	}
	d = &virAdmConnectionData{}
	admSaveConnectionData(c, d)
	return d
}

func admReleaseConnectionData(c *AdmConnect) {
	if c.ptr == nil {
		return
	}
	admConnectionsLock.Lock()
	defer admConnectionsLock.Unlock()
	delete(admConnections, c.ptr)
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectOpen
func NewAdmConnect(uri string, flags ConnectFlags) (*AdmConnect, error) {
	var cUri *C.char
	if uri != "" {
		cUri = C.CString(uri)
		defer C.free(unsafe.Pointer(cUri))
	}
	var err C.virError
	ptr := C.virAdmConnectOpenWrapper(cUri, C.uint(flags), &err)
	if ptr == nil {
		return nil, makeError(&err)
	}
	return &AdmConnect{ptr: ptr}, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectClose
func (c *AdmConnect) Close() (int, error) {
	var err C.virError
	result := int(C.virAdmConnectCloseWrapper(c.ptr, &err))
	if result == -1 {
		return result, makeError(&err)
	}
	if result == 0 {
		// No more reference to this connection, release data.
		admReleaseConnectionData(c)
		c.ptr = nil
	}
	return result, nil
}

type AdmCloseCallback func(conn *AdmConnect, reason ConnectCloseReason)

// Register a close callback for the given destination. Only one
// callback per connection is allowed. Setting a callback will remove
// the previous one.
// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectRegisterCloseCallback
func (c *AdmConnect) ConnectRegisterCloseCallback(callback AdmCloseCallback) error {
	c.UnregisterCloseCallback()
	goCallbackId := registerCallbackId(callback)
	var err C.virError
	res := C.virAdmConnectRegisterCloseCallbackHelper(c.ptr, C.long(goCallbackId), &err)
	if res != 0 {
		freeCallbackId(goCallbackId)
		return makeError(&err)
	}
	connData := admGetConnectionData(c)
	connData.closeCallbackId = &goCallbackId
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectUnregisterCloseCallback
func (c *AdmConnect) UnregisterCloseCallback() error {
	connData := admGetConnectionData(c)
	if connData.closeCallbackId == nil {
		return nil
	}
	var err C.virError
	res := C.virAdmConnectUnregisterCloseCallbackHelper(c.ptr, &err)
	if res != 0 {
		return makeError(&err)
	}
	connData.closeCallbackId = nil
	return nil
}

//export admCloseCallback
func admCloseCallback(conn C.virAdmConnectPtr, reason ConnectCloseReason, goCallbackId int) {
	callbackFunc := getCallbackId(goCallbackId)
	callback, ok := callbackFunc.(AdmCloseCallback)
	if !ok {
		panic("Inappropriate callback type called")
	}
	callback(&AdmConnect{ptr: conn}, reason)
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectRef
func (c *AdmConnect) Ref() error {
	var err C.virError
	ret := C.virAdmConnectRefWrapper(c.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectGetURI
func (c *AdmConnect) GetURI() (string, error) {
	var err C.virError
	cStr := C.virAdmConnectGetURIWrapper(c.ptr, &err)
	if cStr == nil {
		return "", makeError(&err)
	}
	uri := C.GoString(cStr)
	C.free(unsafe.Pointer(cStr))
	return uri, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectIsAlive
func (c *AdmConnect) IsAlive() (bool, error) {
	var err C.virError
	result := C.virAdmConnectIsAliveWrapper(c.ptr, &err)
	if result == -1 {
		return false, makeError(&err)
	}
	if result == 1 {
		return true, nil
	}
	return false, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmGetVersion
func AdmGetVersion() (uint64, error) {
	var version C.ulonglong
	var err C.virError
	ret := C.virAdmGetVersionWrapper(&version, &err)
	if ret < 0 {
		return 0, makeError(&err)
	}
	return uint64(version), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectGetLibVersion
func (c *AdmConnect) GetLibVersion() (uint64, error) {
	var version C.ulonglong
	var err C.virError
	ret := C.virAdmConnectGetLibVersionWrapper(c.ptr, &version, &err)
	if ret < 0 {
		return 0, makeError(&err)
	}
	return uint64(version), nil
}

type DaemonShutdownFlags int

const (
	DAEMON_SHUTDOWN_PRESERVE = C.VIR_DAEMON_SHUTDOWN_PRESERVE
)

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectDaemonShutdown
func (c *AdmConnect) DaemonShutdown(flags DaemonShutdownFlags) error {
	var err C.virError
	ret := C.virAdmConnectDaemonShutdownWrapper(c.ptr, C.uint(flags), &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectGetLoggingFilters
func (c *AdmConnect) GetLoggingFilters(flags uint32) (string, error) {
	var cFilters *C.char
	var err C.virError
	ret := C.virAdmConnectGetLoggingFiltersWrapper(c.ptr, &cFilters, C.uint(flags), &err)
	if ret == -1 {
		return "", makeError(&err)
	}

	filters := C.GoString(cFilters)
	defer C.free(unsafe.Pointer(cFilters))

	return filters, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectSetLoggingFilters
func (c *AdmConnect) SetLoggingFilters(filters string, flags uint32) error {
	var cFilters *C.char
	if filters != "" {
		cFilters = C.CString(filters)
		defer C.free(unsafe.Pointer(cFilters))
	}
	var err C.virError
	ret := C.virAdmConnectSetLoggingFiltersWrapper(c.ptr, cFilters, C.uint(flags), &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectGetLoggingOutputs
func (c *AdmConnect) GetLoggingOutputs(flags uint32) (string, error) {
	var cOutputs *C.char
	var err C.virError
	ret := C.virAdmConnectGetLoggingOutputsWrapper(c.ptr, &cOutputs, C.uint(flags), &err)
	if ret == -1 {
		return "", makeError(&err)
	}

	outputs := C.GoString(cOutputs)
	defer C.free(unsafe.Pointer(cOutputs))

	return outputs, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectSetLoggingOutputs
func (c *AdmConnect) SetLoggingOutputs(outputs string, flags uint32) error {
	var cOutputs *C.char
	if outputs != "" {
		cOutputs = C.CString(outputs)
		defer C.free(unsafe.Pointer(cOutputs))
	}
	var err C.virError
	ret := C.virAdmConnectSetLoggingOutputsWrapper(c.ptr, cOutputs, C.uint(flags), &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectSetDaemonTimeout
func (c *AdmConnect) SetDaemonTimeout(timeout uint, flags uint32) error {
	var err C.virError
	ret := C.virAdmConnectSetDaemonTimeoutWrapper(c.ptr, C.uint(timeout), C.uint(flags), &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

type AdmServer struct {
	ptr C.virAdmServerPtr
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectListServers
func (c *AdmConnect) ListServers(flags uint32) ([]AdmServer, error) {
	var cServers *C.virAdmServerPtr
	var err C.virError
	numServers := C.virAdmConnectListServersWrapper(c.ptr, &cServers, C.uint(flags), &err)
	if numServers == -1 {
		return nil, makeError(&err)
	}
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(cServers)),
		Len:  int(numServers),
		Cap:  int(numServers),
	}
	var servers []AdmServer
	slice := *(*[]C.virAdmServerPtr)(unsafe.Pointer(&hdr))
	for _, ptr := range slice {
		servers = append(servers, AdmServer{ptr})
	}
	C.free(unsafe.Pointer(cServers))
	return servers, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmConnectLookupServer
func (c *AdmConnect) LookupServer(name string, flags uint32) (*AdmServer, error) {
	var cName *C.char
	if name != "" {
		cName = C.CString(name)
		defer C.free(unsafe.Pointer(cName))
	}
	var err C.virError
	ptr := C.virAdmConnectLookupServerWrapper(c.ptr, cName, C.uint(flags), &err)
	if ptr == nil {
		return nil, makeError(&err)
	}
	return &AdmServer{ptr: ptr}, nil
}

// See https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmServerFree
func (s *AdmServer) Free() error {
	var err C.virError
	ret := C.virAdmServerFreeWrapper(s.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmServerGetName
func (s *AdmServer) GetName() (string, error) {
	var err C.virError
	cName := C.virAdmServerGetNameWrapper(s.ptr, &err)
	if cName == nil {
		return "", makeError(&err)
	}
	name := C.GoString(cName)
	return name, nil
}

type ThreadPoolParameters struct {
	MinWorkersSet     bool
	MinWorkers        uint
	MaxWorkersSet     bool
	MaxWorkers        uint
	PrioWorkersSet    bool
	PrioWorkers       uint
	FreeWorkersSet    bool
	FreeWorkers       uint
	CurrentWorkersSet bool
	CurrentWorkers    uint
	JobQueueDepthSet  bool
	JobQueueDepth     uint
}

func getThreadPoolParametersFieldInfo(params *ThreadPoolParameters) map[string]typedParamsFieldInfo {
	return map[string]typedParamsFieldInfo{
		C.VIR_THREADPOOL_WORKERS_MIN: typedParamsFieldInfo{
			set: &params.MinWorkersSet,
			ui:  &params.MinWorkers,
		},
		C.VIR_THREADPOOL_WORKERS_MAX: typedParamsFieldInfo{
			set: &params.MaxWorkersSet,
			ui:  &params.MaxWorkers,
		},
		C.VIR_THREADPOOL_WORKERS_PRIORITY: typedParamsFieldInfo{
			set: &params.PrioWorkersSet,
			ui:  &params.PrioWorkers,
		},
		C.VIR_THREADPOOL_WORKERS_FREE: typedParamsFieldInfo{
			set: &params.FreeWorkersSet,
			ui:  &params.FreeWorkers,
		},
		C.VIR_THREADPOOL_WORKERS_CURRENT: typedParamsFieldInfo{
			set: &params.CurrentWorkersSet,
			ui:  &params.CurrentWorkers,
		},
		C.VIR_THREADPOOL_JOB_QUEUE_DEPTH: typedParamsFieldInfo{
			set: &params.JobQueueDepthSet,
			ui:  &params.JobQueueDepth,
		},
	}
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmServerGetThreadPoolParameters
func (s *AdmServer) GetThreadPoolParameters(flags uint32) (*ThreadPoolParameters, error) {
	params := &ThreadPoolParameters{}
	info := getThreadPoolParametersFieldInfo(params)

	var cparams *C.virTypedParameter
	var cnparams C.int

	var err C.virError
	ret := C.virAdmServerGetThreadPoolParametersWrapper(s.ptr, (*C.virTypedParameterPtr)(unsafe.Pointer(&cparams)), &cnparams, C.uint(flags), &err)
	if ret == -1 {
		return nil, makeError(&err)
	}

	defer C.virTypedParamsFreeWrapper(cparams, cnparams)

	_, gerr := typedParamsUnpack(cparams, cnparams, info)
	if gerr != nil {
		return nil, gerr
	}

	return params, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmServerSetThreadPoolParameters
func (s *AdmServer) SetThreadPoolParameters(params *ThreadPoolParameters, flags uint32) error {
	info := getThreadPoolParametersFieldInfo(params)

	cparams, cnparams, gerr := typedParamsPackNew(info)
	if gerr != nil {
		return gerr
	}

	defer C.virTypedParamsFreeWrapper(cparams, cnparams)

	var err C.virError
	ret := C.virAdmServerSetThreadPoolParametersWrapper(s.ptr, cparams, cnparams, C.uint(flags), &err)
	if ret == -1 {
		return makeError(&err)
	}

	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmServerUpdateTlsFiles
func (s *AdmServer) UpdateTlsFiles(flags uint32) error {
	var err C.virError
	ret := C.virAdmServerUpdateTlsFilesWrapper(s.ptr, C.uint(flags), &err)
	if ret == -1 {
		return makeError(&err)
	}

	return nil
}

type ClientLimitsParameters struct {
	MaxClientsSet           bool
	MaxClients              uint
	CurrentClientsSet       bool
	CurrentClients          uint
	MaxUnauthClientsSet     bool
	MaxUnauthClients        uint
	CurrentUnauthClientsSet bool
	CurrentUnauthClients    uint
}

func getClientLimitsParametersFieldInfo(params *ClientLimitsParameters) map[string]typedParamsFieldInfo {
	return map[string]typedParamsFieldInfo{
		C.VIR_SERVER_CLIENTS_MAX: typedParamsFieldInfo{
			set: &params.MaxClientsSet,
			ui:  &params.MaxClients,
		},
		C.VIR_SERVER_CLIENTS_CURRENT: typedParamsFieldInfo{
			set: &params.CurrentClientsSet,
			ui:  &params.CurrentClients,
		},
		C.VIR_SERVER_CLIENTS_UNAUTH_MAX: typedParamsFieldInfo{
			set: &params.MaxUnauthClientsSet,
			ui:  &params.MaxUnauthClients,
		},
		C.VIR_SERVER_CLIENTS_UNAUTH_CURRENT: typedParamsFieldInfo{
			set: &params.CurrentUnauthClientsSet,
			ui:  &params.CurrentUnauthClients,
		},
	}
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmServerGetClientLimits
func (s *AdmServer) GetClientLimits(flags uint32) (*ClientLimitsParameters, error) {
	params := &ClientLimitsParameters{}
	info := getClientLimitsParametersFieldInfo(params)

	var cparams *C.virTypedParameter
	var cnparams C.int

	var err C.virError
	ret := C.virAdmServerGetClientLimitsWrapper(s.ptr, (*C.virTypedParameterPtr)(unsafe.Pointer(&cparams)), &cnparams, C.uint(flags), &err)
	if ret == -1 {
		return nil, makeError(&err)
	}

	defer C.virTypedParamsFreeWrapper(cparams, cnparams)

	_, gerr := typedParamsUnpack(cparams, cnparams, info)
	if gerr != nil {
		return nil, gerr
	}

	return params, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmServerSetClientLimits
func (s *AdmServer) SetClientLimits(params *ClientLimitsParameters, flags uint32) error {
	info := getClientLimitsParametersFieldInfo(params)

	cparams, cnparams, gerr := typedParamsPackNew(info)
	if gerr != nil {
		return gerr
	}

	defer C.virTypedParamsFreeWrapper(cparams, cnparams)

	var err C.virError
	ret := C.virAdmServerSetClientLimitsWrapper(s.ptr, cparams, cnparams, C.uint(flags), &err)
	if ret == -1 {
		return makeError(&err)
	}

	return nil
}

type AdmClient struct {
	ptr C.virAdmClientPtr
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmServerListClients
func (s *AdmServer) ListClients(flags uint32) ([]AdmClient, error) {
	var cClients *C.virAdmClientPtr
	var err C.virError
	numClients := C.virAdmServerListClientsWrapper(s.ptr, &cClients, C.uint(flags), &err)
	if numClients == -1 {
		return nil, makeError(&err)
	}
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(cClients)),
		Len:  int(numClients),
		Cap:  int(numClients),
	}
	var clients []AdmClient
	slice := *(*[]C.virAdmClientPtr)(unsafe.Pointer(&hdr))
	for _, ptr := range slice {
		clients = append(clients, AdmClient{ptr})
	}
	C.free(unsafe.Pointer(cClients))
	return clients, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmServerLookupClient
func (s *AdmServer) LookupClient(id uint64, flags uint32) (*AdmClient, error) {
	var err C.virError
	ptr := C.virAdmServerLookupClientWrapper(s.ptr, C.ulonglong(id), C.uint(flags), &err)
	if ptr == nil {
		return nil, makeError(&err)
	}
	return &AdmClient{ptr: ptr}, nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmClientFree
func (c *AdmClient) Free() error {
	var err C.virError
	ret := C.virAdmClientFreeWrapper(c.ptr, &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmClientClose
func (c *AdmClient) Close(flags int32) error {
	var err C.virError
	ret := C.virAdmClientCloseWrapper(c.ptr, C.uint(flags), &err)
	if ret == -1 {
		return makeError(&err)
	}
	return nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmClientGetID
func (c *AdmClient) GetID() (uint64, error) {
	var err C.virError
	ret := C.virAdmClientGetIDWrapper(c.ptr, &err)
	if ret == ^C.ulonglong(0) {
		return uint64(ret), makeError(&err)
	}
	return uint64(ret), nil
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmClientGetTimestamp
func (c *AdmClient) GetTimestamp() (int64, error) {
	var err C.virError
	ret := C.virAdmClientGetTimestampWrapper(c.ptr, &err)
	if ret == -1 {
		return int64(ret), makeError(&err)
	}
	return int64(ret), nil
}

type ClientTransport int

const (
	CLIENT_TRANS_TCP  = ClientTransport(C.VIR_CLIENT_TRANS_TCP)
	CLIENT_TRANS_TLS  = ClientTransport(C.VIR_CLIENT_TRANS_TLS)
	CLIENT_TRANS_UNIX = ClientTransport(C.VIR_CLIENT_TRANS_UNIX)
)

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmClientGetTransport
func (c *AdmClient) GetTransport() (ClientTransport, error) {
	var err C.virError
	ret := C.virAdmClientGetTransportWrapper(c.ptr, &err)
	if ret == -1 {
		return -1, makeError(&err)
	}
	return ClientTransport(ret), nil
}

type ClientInfo struct {
	ReadonlySet                 bool
	Readonly                    bool
	SocketAddressSet            bool
	SocketAddress               string
	SaslUsernameSet             bool
	SaslUsername                string
	TlsX509DistinguishedNameSet bool
	TlsX509DistinguishedName    string
	UnixUserIdSet               bool
	UnixUserId                  int
	UnixUsernameSet             bool
	UnixUsername                string
	UnixGroupIdSet              bool
	UnixGroupId                 int
	UnixGroupnameSet            bool
	UnixGroupname               string
	UnixProcessIdSet            bool
	UnixProcessId               int
	SelinuxContextSet           bool
	SelinuxContext              string
}

func getClientInfoFieldInfo(params *ClientInfo) map[string]typedParamsFieldInfo {
	return map[string]typedParamsFieldInfo{
		C.VIR_CLIENT_INFO_READONLY: typedParamsFieldInfo{
			set: &params.ReadonlySet,
			b:   &params.Readonly,
		},
		C.VIR_CLIENT_INFO_SOCKET_ADDR: typedParamsFieldInfo{
			set: &params.SocketAddressSet,
			s:   &params.SocketAddress,
		},
		C.VIR_CLIENT_INFO_SASL_USER_NAME: typedParamsFieldInfo{
			set: &params.SaslUsernameSet,
			s:   &params.SaslUsername,
		},
		C.VIR_CLIENT_INFO_X509_DISTINGUISHED_NAME: typedParamsFieldInfo{
			set: &params.TlsX509DistinguishedNameSet,
			s:   &params.TlsX509DistinguishedName,
		},
		C.VIR_CLIENT_INFO_UNIX_USER_ID: typedParamsFieldInfo{
			set: &params.UnixUserIdSet,
			i:   &params.UnixUserId,
		},
		C.VIR_CLIENT_INFO_UNIX_USER_NAME: typedParamsFieldInfo{
			set: &params.UnixUsernameSet,
			s:   &params.UnixUsername,
		},
		C.VIR_CLIENT_INFO_UNIX_GROUP_ID: typedParamsFieldInfo{
			set: &params.UnixGroupIdSet,
			i:   &params.UnixGroupId,
		},
		C.VIR_CLIENT_INFO_UNIX_GROUP_NAME: typedParamsFieldInfo{
			set: &params.UnixGroupnameSet,
			s:   &params.UnixGroupname,
		},
		C.VIR_CLIENT_INFO_UNIX_PROCESS_ID: typedParamsFieldInfo{
			set: &params.UnixProcessIdSet,
			i:   &params.UnixProcessId,
		},
		C.VIR_CLIENT_INFO_SELINUX_CONTEXT: typedParamsFieldInfo{
			set: &params.SelinuxContextSet,
			s:   &params.SelinuxContext,
		},
	}
}

// See also https://libvirt.org/html/libvirt-libvirt-admin.html#virAdmClientGetInfo
func (c *AdmClient) GetInfo(flags uint32) (*ClientInfo, error) {
	params := &ClientInfo{}
	info := getClientInfoFieldInfo(params)

	var cparams *C.virTypedParameter
	var cnparams C.int

	var err C.virError
	ret := C.virAdmClientGetInfoWrapper(c.ptr, (*C.virTypedParameterPtr)(unsafe.Pointer(&cparams)), &cnparams, C.uint(flags), &err)
	if ret == -1 {
		return nil, makeError(&err)
	}

	defer C.virTypedParamsFreeWrapper(cparams, cnparams)

	_, gerr := typedParamsUnpack(cparams, cnparams, info)
	if gerr != nil {
		return nil, gerr
	}

	return params, nil
}

func adminInitialize() error {
	var err C.virError
	if C.virAdmInitializeWrapper(&err) < 0 {
		return makeError(&err)
	}
	return nil
}
