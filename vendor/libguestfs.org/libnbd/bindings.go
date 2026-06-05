/* NBD client library in userspace
 * WARNING: THIS FILE IS GENERATED FROM
 * generator/generator
 * ANY CHANGES YOU MAKE TO THIS FILE WILL BE LOST.
 *
 * Copyright Red Hat
 *
 * This library is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Lesser General Public
 * License as published by the Free Software Foundation; either
 * version 2 of the License, or (at your option) any later version.
 *
 * This library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public
 * License along with this library; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA
 */

package libnbd

/*
#cgo pkg-config: libnbd
#cgo CFLAGS: -D_GNU_SOURCE=1

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "libnbd.h"
#include "wrappers.h"

// There must be no blank line between end comment and import!
// https://github.com/golang/go/issues/9733
*/
import "C"

import (
	"runtime"
	"unsafe"
)

/* Enums. */
type Tls int

const (
	TLS_DISABLE = Tls(0)
	TLS_ALLOW   = Tls(1)
	TLS_REQUIRE = Tls(2)
)

type Size int

const (
	SIZE_MINIMUM   = Size(0)
	SIZE_PREFERRED = Size(1)
	SIZE_MAXIMUM   = Size(2)
	SIZE_PAYLOAD   = Size(3)
)

/* Flags. */
type CmdFlag uint32

const (
	CMD_FLAG_FUA         = CmdFlag(0x01)
	CMD_FLAG_NO_HOLE     = CmdFlag(0x02)
	CMD_FLAG_DF          = CmdFlag(0x04)
	CMD_FLAG_REQ_ONE     = CmdFlag(0x08)
	CMD_FLAG_FAST_ZERO   = CmdFlag(0x10)
	CMD_FLAG_PAYLOAD_LEN = CmdFlag(0x20)
	CMD_FLAG_MASK        = CmdFlag(0x3f)
)

type HandshakeFlag uint32

const (
	HANDSHAKE_FLAG_FIXED_NEWSTYLE = HandshakeFlag(0x01)
	HANDSHAKE_FLAG_NO_ZEROES      = HandshakeFlag(0x02)
	HANDSHAKE_FLAG_MASK           = HandshakeFlag(0x03)
)

type Strict uint32

const (
	STRICT_COMMANDS  = Strict(0x01)
	STRICT_FLAGS     = Strict(0x02)
	STRICT_BOUNDS    = Strict(0x04)
	STRICT_ZERO_SIZE = Strict(0x08)
	STRICT_ALIGN     = Strict(0x10)
	STRICT_PAYLOAD   = Strict(0x20)
	STRICT_AUTO_FLAG = Strict(0x40)
	STRICT_MASK      = Strict(0x7f)
)

type AllowTransport uint32

const (
	ALLOW_TRANSPORT_TCP   = AllowTransport(0x01)
	ALLOW_TRANSPORT_UNIX  = AllowTransport(0x02)
	ALLOW_TRANSPORT_VSOCK = AllowTransport(0x04)
	ALLOW_TRANSPORT_SSH   = AllowTransport(0x08)
	ALLOW_TRANSPORT_MASK  = AllowTransport(0x0f)
)

type Shutdown uint32

const (
	SHUTDOWN_ABANDON_PENDING = Shutdown(0x10000)
	SHUTDOWN_MASK            = Shutdown(0x10000)
)

/* Constants. */
const (
	AIO_DIRECTION_READ  uint32 = 1
	AIO_DIRECTION_WRITE uint32 = 2
	AIO_DIRECTION_BOTH  uint32 = 3
	READ_DATA           uint32 = 1
	READ_HOLE           uint32 = 2
	READ_ERROR          uint32 = 3
	/* Meta-context namespace "base" */
	NAMESPACE_BASE          = "base:"
	CONTEXT_BASE_ALLOCATION = "base:allocation"
	/* Defined bits in "base:allocation" */
	STATE_HOLE uint32 = 1
	STATE_ZERO uint32 = 2
	/* Meta-context namespace "qemu" */
	NAMESPACE_QEMU            = "qemu:"
	CONTEXT_QEMU_DIRTY_BITMAP = "qemu:dirty-bitmap:"
	/* Defined bits in "qemu:dirty-bitmap:" */
	STATE_DIRTY                   uint32 = 1
	CONTEXT_QEMU_ALLOCATION_DEPTH        = "qemu:allocation-depth"
)

/* SetDebug: set or clear the debug flag */
func (h *Libnbd) SetDebug(debug bool) error {
	if h.h == nil {
		return closed_handle_error("set_debug")
	}

	var c_err C.struct_error
	c_debug := C.bool(debug)

	ret := C._nbd_set_debug_wrapper(&c_err, h.h, c_debug)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_debug", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetDebug: return the state of the debug flag */
func (h *Libnbd) GetDebug() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("get_debug")
	}

	var c_err C.struct_error

	ret := C._nbd_get_debug_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_debug", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* SetDebugCallback: set the debug callback */
func (h *Libnbd) SetDebugCallback(debug DebugCallback) error {
	if h.h == nil {
		return closed_handle_error("set_debug_callback")
	}

	var c_err C.struct_error
	var c_debug C.nbd_debug_callback
	c_debug.callback = (*[0]byte)(C._nbd_debug_callback_wrapper)
	c_debug.free = (*[0]byte)(C._nbd_debug_callback_free)
	debug_cbid := registerCallbackId(debug)
	c_debug.user_data = C.alloc_cbid(C.long(debug_cbid))

	ret := C._nbd_set_debug_callback_wrapper(&c_err, h.h, c_debug)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_debug_callback", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* ClearDebugCallback: clear the debug callback */
func (h *Libnbd) ClearDebugCallback() error {
	if h.h == nil {
		return closed_handle_error("clear_debug_callback")
	}

	var c_err C.struct_error

	ret := C._nbd_clear_debug_callback_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("clear_debug_callback", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* StatsBytesSent: statistics of bytes sent over connection so far */
func (h *Libnbd) StatsBytesSent() (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("stats_bytes_sent")
	}

	var c_err C.struct_error

	ret := C._nbd_stats_bytes_sent_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	return uint64(ret), nil
}

/* StatsChunksSent: statistics of chunks sent over connection so far */
func (h *Libnbd) StatsChunksSent() (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("stats_chunks_sent")
	}

	var c_err C.struct_error

	ret := C._nbd_stats_chunks_sent_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	return uint64(ret), nil
}

/* StatsBytesReceived: statistics of bytes received over connection so far */
func (h *Libnbd) StatsBytesReceived() (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("stats_bytes_received")
	}

	var c_err C.struct_error

	ret := C._nbd_stats_bytes_received_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	return uint64(ret), nil
}

/* StatsChunksReceived: statistics of chunks received over connection so far */
func (h *Libnbd) StatsChunksReceived() (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("stats_chunks_received")
	}

	var c_err C.struct_error

	ret := C._nbd_stats_chunks_received_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	return uint64(ret), nil
}

/* SetHandleName: set the handle name */
func (h *Libnbd) SetHandleName(handle_name string) error {
	if h.h == nil {
		return closed_handle_error("set_handle_name")
	}

	var c_err C.struct_error
	c_handle_name := C.CString(handle_name)
	defer C.free(unsafe.Pointer(c_handle_name))

	ret := C._nbd_set_handle_name_wrapper(&c_err, h.h, c_handle_name)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_handle_name", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetHandleName: get the handle name */
func (h *Libnbd) GetHandleName() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_handle_name")
	}

	var c_err C.struct_error

	ret := C._nbd_get_handle_name_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_handle_name", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	r := C.GoString(ret)
	C.free(unsafe.Pointer(ret))
	return &r, nil
}

/* SetPrivateData: set the per-handle private data */
func (h *Libnbd) SetPrivateData(private_data uint) (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("set_private_data")
	}

	var c_err C.struct_error
	c_private_data := C.uintptr_t(private_data)

	ret := C._nbd_set_private_data_wrapper(&c_err, h.h, c_private_data)
	runtime.KeepAlive(h.h)
	return uint(ret), nil
}

/* GetPrivateData: get the per-handle private data */
func (h *Libnbd) GetPrivateData() (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("get_private_data")
	}

	var c_err C.struct_error

	ret := C._nbd_get_private_data_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	return uint(ret), nil
}

/* GetHandleSize: return an estimate of the memory allocated for the handle */
func (h *Libnbd) GetHandleSize() (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("get_handle_size")
	}

	var c_err C.struct_error

	ret := C._nbd_get_handle_size_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_handle_size", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* SetExportName: set the export name */
func (h *Libnbd) SetExportName(export_name string) error {
	if h.h == nil {
		return closed_handle_error("set_export_name")
	}

	var c_err C.struct_error
	c_export_name := C.CString(export_name)
	defer C.free(unsafe.Pointer(c_export_name))

	ret := C._nbd_set_export_name_wrapper(&c_err, h.h, c_export_name)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_export_name", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetExportName: get the export name */
func (h *Libnbd) GetExportName() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_export_name")
	}

	var c_err C.struct_error

	ret := C._nbd_get_export_name_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_export_name", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	r := C.GoString(ret)
	C.free(unsafe.Pointer(ret))
	return &r, nil
}

/* SetRequestBlockSize: control whether NBD_OPT_GO requests block size */
func (h *Libnbd) SetRequestBlockSize(request bool) error {
	if h.h == nil {
		return closed_handle_error("set_request_block_size")
	}

	var c_err C.struct_error
	c_request := C.bool(request)

	ret := C._nbd_set_request_block_size_wrapper(&c_err, h.h, c_request)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_request_block_size", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetRequestBlockSize: see if NBD_OPT_GO requests block size */
func (h *Libnbd) GetRequestBlockSize() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("get_request_block_size")
	}

	var c_err C.struct_error

	ret := C._nbd_get_request_block_size_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_request_block_size", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* SetFullInfo: control whether NBD_OPT_GO requests extra details */
func (h *Libnbd) SetFullInfo(request bool) error {
	if h.h == nil {
		return closed_handle_error("set_full_info")
	}

	var c_err C.struct_error
	c_request := C.bool(request)

	ret := C._nbd_set_full_info_wrapper(&c_err, h.h, c_request)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_full_info", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetFullInfo: see if NBD_OPT_GO requests extra details */
func (h *Libnbd) GetFullInfo() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("get_full_info")
	}

	var c_err C.struct_error

	ret := C._nbd_get_full_info_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_full_info", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* GetCanonicalExportName: return the canonical export name, if the server has one */
func (h *Libnbd) GetCanonicalExportName() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_canonical_export_name")
	}

	var c_err C.struct_error

	ret := C._nbd_get_canonical_export_name_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_canonical_export_name", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	r := C.GoString(ret)
	C.free(unsafe.Pointer(ret))
	return &r, nil
}

/* GetExportDescription: return the export description, if the server has one */
func (h *Libnbd) GetExportDescription() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_export_description")
	}

	var c_err C.struct_error

	ret := C._nbd_get_export_description_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_export_description", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	r := C.GoString(ret)
	C.free(unsafe.Pointer(ret))
	return &r, nil
}

/* SetTls: enable or require TLS (authentication and encryption) */
func (h *Libnbd) SetTls(tls Tls) error {
	if h.h == nil {
		return closed_handle_error("set_tls")
	}

	var c_err C.struct_error
	c_tls := C.int(tls)

	ret := C._nbd_set_tls_wrapper(&c_err, h.h, c_tls)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_tls", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetTls: get the TLS request setting */
func (h *Libnbd) GetTls() (Tls, error) {
	if h.h == nil {
		return 0, closed_handle_error("get_tls")
	}

	var c_err C.struct_error

	ret := C._nbd_get_tls_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	return Tls(ret), nil
}

/* GetTlsNegotiated: find out if TLS was negotiated on a connection */
func (h *Libnbd) GetTlsNegotiated() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("get_tls_negotiated")
	}

	var c_err C.struct_error

	ret := C._nbd_get_tls_negotiated_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_tls_negotiated", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* SetTlsCertificates: set the path to the TLS certificates directory */
func (h *Libnbd) SetTlsCertificates(dir string) error {
	if h.h == nil {
		return closed_handle_error("set_tls_certificates")
	}

	var c_err C.struct_error
	c_dir := C.CString(dir)
	defer C.free(unsafe.Pointer(c_dir))

	ret := C._nbd_set_tls_certificates_wrapper(&c_err, h.h, c_dir)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_tls_certificates", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* SetTlsVerifyPeer: set whether we verify the identity of the server */
func (h *Libnbd) SetTlsVerifyPeer(verify bool) error {
	if h.h == nil {
		return closed_handle_error("set_tls_verify_peer")
	}

	var c_err C.struct_error
	c_verify := C.bool(verify)

	ret := C._nbd_set_tls_verify_peer_wrapper(&c_err, h.h, c_verify)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_tls_verify_peer", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetTlsVerifyPeer: get whether we verify the identity of the server */
func (h *Libnbd) GetTlsVerifyPeer() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("get_tls_verify_peer")
	}

	var c_err C.struct_error

	ret := C._nbd_get_tls_verify_peer_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_tls_verify_peer", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* SetTlsUsername: set the TLS username */
func (h *Libnbd) SetTlsUsername(username string) error {
	if h.h == nil {
		return closed_handle_error("set_tls_username")
	}

	var c_err C.struct_error
	c_username := C.CString(username)
	defer C.free(unsafe.Pointer(c_username))

	ret := C._nbd_set_tls_username_wrapper(&c_err, h.h, c_username)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_tls_username", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetTlsUsername: get the current TLS username */
func (h *Libnbd) GetTlsUsername() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_tls_username")
	}

	var c_err C.struct_error

	ret := C._nbd_get_tls_username_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_tls_username", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	r := C.GoString(ret)
	C.free(unsafe.Pointer(ret))
	return &r, nil
}

/* SetTlsHostname: set the TLS hostname */
func (h *Libnbd) SetTlsHostname(hostname string) error {
	if h.h == nil {
		return closed_handle_error("set_tls_hostname")
	}

	var c_err C.struct_error
	c_hostname := C.CString(hostname)
	defer C.free(unsafe.Pointer(c_hostname))

	ret := C._nbd_set_tls_hostname_wrapper(&c_err, h.h, c_hostname)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_tls_hostname", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetTlsHostname: get the effective TLS hostname */
func (h *Libnbd) GetTlsHostname() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_tls_hostname")
	}

	var c_err C.struct_error

	ret := C._nbd_get_tls_hostname_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_tls_hostname", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	r := C.GoString(ret)
	C.free(unsafe.Pointer(ret))
	return &r, nil
}

/* SetTlsPskFile: set the TLS Pre-Shared Keys (PSK) filename */
func (h *Libnbd) SetTlsPskFile(filename string) error {
	if h.h == nil {
		return closed_handle_error("set_tls_psk_file")
	}

	var c_err C.struct_error
	c_filename := C.CString(filename)
	defer C.free(unsafe.Pointer(c_filename))

	ret := C._nbd_set_tls_psk_file_wrapper(&c_err, h.h, c_filename)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_tls_psk_file", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* SetTlsPriority: set the TLS priority string */
func (h *Libnbd) SetTlsPriority(priority string) error {
	if h.h == nil {
		return closed_handle_error("set_tls_priority")
	}

	var c_err C.struct_error
	c_priority := C.CString(priority)
	defer C.free(unsafe.Pointer(c_priority))

	ret := C._nbd_set_tls_priority_wrapper(&c_err, h.h, c_priority)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_tls_priority", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetTlsPriority: get the current TLS priority */
func (h *Libnbd) GetTlsPriority() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_tls_priority")
	}

	var c_err C.struct_error

	ret := C._nbd_get_tls_priority_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_tls_priority", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	r := C.GoString(ret)
	C.free(unsafe.Pointer(ret))
	return &r, nil
}

/* SetRequestExtendedHeaders: control use of extended headers */
func (h *Libnbd) SetRequestExtendedHeaders(request bool) error {
	if h.h == nil {
		return closed_handle_error("set_request_extended_headers")
	}

	var c_err C.struct_error
	c_request := C.bool(request)

	ret := C._nbd_set_request_extended_headers_wrapper(&c_err, h.h, c_request)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_request_extended_headers", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetRequestExtendedHeaders: see if extended headers are attempted */
func (h *Libnbd) GetRequestExtendedHeaders() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("get_request_extended_headers")
	}

	var c_err C.struct_error

	ret := C._nbd_get_request_extended_headers_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_request_extended_headers", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* GetExtendedHeadersNegotiated: see if extended headers are in use */
func (h *Libnbd) GetExtendedHeadersNegotiated() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("get_extended_headers_negotiated")
	}

	var c_err C.struct_error

	ret := C._nbd_get_extended_headers_negotiated_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_extended_headers_negotiated", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* SetRequestStructuredReplies: control use of structured replies */
func (h *Libnbd) SetRequestStructuredReplies(request bool) error {
	if h.h == nil {
		return closed_handle_error("set_request_structured_replies")
	}

	var c_err C.struct_error
	c_request := C.bool(request)

	ret := C._nbd_set_request_structured_replies_wrapper(&c_err, h.h, c_request)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_request_structured_replies", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetRequestStructuredReplies: see if structured replies are attempted */
func (h *Libnbd) GetRequestStructuredReplies() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("get_request_structured_replies")
	}

	var c_err C.struct_error

	ret := C._nbd_get_request_structured_replies_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_request_structured_replies", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* GetStructuredRepliesNegotiated: see if structured replies are in use */
func (h *Libnbd) GetStructuredRepliesNegotiated() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("get_structured_replies_negotiated")
	}

	var c_err C.struct_error

	ret := C._nbd_get_structured_replies_negotiated_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_structured_replies_negotiated", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* SetRequestMetaContext: control whether connect automatically requests meta contexts */
func (h *Libnbd) SetRequestMetaContext(request bool) error {
	if h.h == nil {
		return closed_handle_error("set_request_meta_context")
	}

	var c_err C.struct_error
	c_request := C.bool(request)

	ret := C._nbd_set_request_meta_context_wrapper(&c_err, h.h, c_request)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_request_meta_context", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetRequestMetaContext: see if connect automatically requests meta contexts */
func (h *Libnbd) GetRequestMetaContext() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("get_request_meta_context")
	}

	var c_err C.struct_error

	ret := C._nbd_get_request_meta_context_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_request_meta_context", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* SetHandshakeFlags: control use of handshake flags */
func (h *Libnbd) SetHandshakeFlags(flags HandshakeFlag) error {
	if h.h == nil {
		return closed_handle_error("set_handshake_flags")
	}

	var c_err C.struct_error
	c_flags := C.uint32_t(flags)

	ret := C._nbd_set_handshake_flags_wrapper(&c_err, h.h, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_handshake_flags", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetHandshakeFlags: see which handshake flags are supported */
func (h *Libnbd) GetHandshakeFlags() (HandshakeFlag, error) {
	if h.h == nil {
		return 0, closed_handle_error("get_handshake_flags")
	}

	var c_err C.struct_error

	ret := C._nbd_get_handshake_flags_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	return HandshakeFlag(ret), nil
}

/* SetPreadInitialize: control whether libnbd pre-initializes read buffers */
func (h *Libnbd) SetPreadInitialize(request bool) error {
	if h.h == nil {
		return closed_handle_error("set_pread_initialize")
	}

	var c_err C.struct_error
	c_request := C.bool(request)

	ret := C._nbd_set_pread_initialize_wrapper(&c_err, h.h, c_request)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_pread_initialize", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetPreadInitialize: see whether libnbd pre-initializes read buffers */
func (h *Libnbd) GetPreadInitialize() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("get_pread_initialize")
	}

	var c_err C.struct_error

	ret := C._nbd_get_pread_initialize_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_pread_initialize", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* SetStrictMode: control how strictly to follow NBD protocol */
func (h *Libnbd) SetStrictMode(flags Strict) error {
	if h.h == nil {
		return closed_handle_error("set_strict_mode")
	}

	var c_err C.struct_error
	c_flags := C.uint32_t(flags)

	ret := C._nbd_set_strict_mode_wrapper(&c_err, h.h, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_strict_mode", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetStrictMode: see which strictness flags are in effect */
func (h *Libnbd) GetStrictMode() (Strict, error) {
	if h.h == nil {
		return 0, closed_handle_error("get_strict_mode")
	}

	var c_err C.struct_error

	ret := C._nbd_get_strict_mode_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	return Strict(ret), nil
}

/* SetOptMode: control option mode, for pausing during option negotiation */
func (h *Libnbd) SetOptMode(enable bool) error {
	if h.h == nil {
		return closed_handle_error("set_opt_mode")
	}

	var c_err C.struct_error
	c_enable := C.bool(enable)

	ret := C._nbd_set_opt_mode_wrapper(&c_err, h.h, c_enable)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_opt_mode", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetOptMode: return whether option mode was enabled */
func (h *Libnbd) GetOptMode() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("get_opt_mode")
	}

	var c_err C.struct_error

	ret := C._nbd_get_opt_mode_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_opt_mode", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* OptGo: end negotiation and move on to using an export */
func (h *Libnbd) OptGo() error {
	if h.h == nil {
		return closed_handle_error("opt_go")
	}

	var c_err C.struct_error

	ret := C._nbd_opt_go_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("opt_go", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* OptAbort: end negotiation and close the connection */
func (h *Libnbd) OptAbort() error {
	if h.h == nil {
		return closed_handle_error("opt_abort")
	}

	var c_err C.struct_error

	ret := C._nbd_opt_abort_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("opt_abort", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* OptStarttls: request the server to initiate TLS */
func (h *Libnbd) OptStarttls() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("opt_starttls")
	}

	var c_err C.struct_error

	ret := C._nbd_opt_starttls_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("opt_starttls", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* OptExtendedHeaders: request the server to enable extended headers */
func (h *Libnbd) OptExtendedHeaders() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("opt_extended_headers")
	}

	var c_err C.struct_error

	ret := C._nbd_opt_extended_headers_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("opt_extended_headers", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* OptStructuredReply: request the server to enable structured replies */
func (h *Libnbd) OptStructuredReply() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("opt_structured_reply")
	}

	var c_err C.struct_error

	ret := C._nbd_opt_structured_reply_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("opt_structured_reply", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* OptList: request the server to list all exports during negotiation */
func (h *Libnbd) OptList(list ListCallback) (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("opt_list")
	}

	var c_err C.struct_error
	var c_list C.nbd_list_callback
	c_list.callback = (*[0]byte)(C._nbd_list_callback_wrapper)
	c_list.free = (*[0]byte)(C._nbd_list_callback_free)
	list_cbid := registerCallbackId(list)
	c_list.user_data = C.alloc_cbid(C.long(list_cbid))

	ret := C._nbd_opt_list_wrapper(&c_err, h.h, c_list)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("opt_list", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* OptInfo: request the server for information about an export */
func (h *Libnbd) OptInfo() error {
	if h.h == nil {
		return closed_handle_error("opt_info")
	}

	var c_err C.struct_error

	ret := C._nbd_opt_info_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("opt_info", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* OptListMetaContext: list available meta contexts, using implicit query list */
func (h *Libnbd) OptListMetaContext(context ContextCallback) (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("opt_list_meta_context")
	}

	var c_err C.struct_error
	var c_context C.nbd_context_callback
	c_context.callback = (*[0]byte)(C._nbd_context_callback_wrapper)
	c_context.free = (*[0]byte)(C._nbd_context_callback_free)
	context_cbid := registerCallbackId(context)
	c_context.user_data = C.alloc_cbid(C.long(context_cbid))

	ret := C._nbd_opt_list_meta_context_wrapper(&c_err, h.h, c_context)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("opt_list_meta_context", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* OptListMetaContextQueries: list available meta contexts, using explicit query list */
func (h *Libnbd) OptListMetaContextQueries(queries []string, context ContextCallback) (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("opt_list_meta_context_queries")
	}

	var c_err C.struct_error
	c_queries := arg_string_list(queries)
	defer free_string_list(c_queries)
	var c_context C.nbd_context_callback
	c_context.callback = (*[0]byte)(C._nbd_context_callback_wrapper)
	c_context.free = (*[0]byte)(C._nbd_context_callback_free)
	context_cbid := registerCallbackId(context)
	c_context.user_data = C.alloc_cbid(C.long(context_cbid))

	ret := C._nbd_opt_list_meta_context_queries_wrapper(&c_err, h.h, &c_queries[0], c_context)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("opt_list_meta_context_queries", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* OptSetMetaContext: select specific meta contexts, using implicit query list */
func (h *Libnbd) OptSetMetaContext(context ContextCallback) (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("opt_set_meta_context")
	}

	var c_err C.struct_error
	var c_context C.nbd_context_callback
	c_context.callback = (*[0]byte)(C._nbd_context_callback_wrapper)
	c_context.free = (*[0]byte)(C._nbd_context_callback_free)
	context_cbid := registerCallbackId(context)
	c_context.user_data = C.alloc_cbid(C.long(context_cbid))

	ret := C._nbd_opt_set_meta_context_wrapper(&c_err, h.h, c_context)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("opt_set_meta_context", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* OptSetMetaContextQueries: select specific meta contexts, using explicit query list */
func (h *Libnbd) OptSetMetaContextQueries(queries []string, context ContextCallback) (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("opt_set_meta_context_queries")
	}

	var c_err C.struct_error
	c_queries := arg_string_list(queries)
	defer free_string_list(c_queries)
	var c_context C.nbd_context_callback
	c_context.callback = (*[0]byte)(C._nbd_context_callback_wrapper)
	c_context.free = (*[0]byte)(C._nbd_context_callback_free)
	context_cbid := registerCallbackId(context)
	c_context.user_data = C.alloc_cbid(C.long(context_cbid))

	ret := C._nbd_opt_set_meta_context_queries_wrapper(&c_err, h.h, &c_queries[0], c_context)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("opt_set_meta_context_queries", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* AddMetaContext: ask server to negotiate metadata context */
func (h *Libnbd) AddMetaContext(name string) error {
	if h.h == nil {
		return closed_handle_error("add_meta_context")
	}

	var c_err C.struct_error
	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_name))

	ret := C._nbd_add_meta_context_wrapper(&c_err, h.h, c_name)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("add_meta_context", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetNrMetaContexts: return the current number of requested meta contexts */
func (h *Libnbd) GetNrMetaContexts() (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("get_nr_meta_contexts")
	}

	var c_err C.struct_error

	ret := C._nbd_get_nr_meta_contexts_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_nr_meta_contexts", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* GetMetaContext: return the i'th meta context request */
func (h *Libnbd) GetMetaContext(i int) (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_meta_context")
	}

	var c_err C.struct_error
	c_i := C.size_t(i)

	ret := C._nbd_get_meta_context_wrapper(&c_err, h.h, c_i)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_meta_context", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	r := C.GoString(ret)
	C.free(unsafe.Pointer(ret))
	return &r, nil
}

/* ClearMetaContexts: reset the list of requested meta contexts */
func (h *Libnbd) ClearMetaContexts() error {
	if h.h == nil {
		return closed_handle_error("clear_meta_contexts")
	}

	var c_err C.struct_error

	ret := C._nbd_clear_meta_contexts_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("clear_meta_contexts", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* SetUriAllowTransports: set the allowed transports in NBD URIs */
func (h *Libnbd) SetUriAllowTransports(mask AllowTransport) error {
	if h.h == nil {
		return closed_handle_error("set_uri_allow_transports")
	}

	var c_err C.struct_error
	c_mask := C.uint32_t(mask)

	ret := C._nbd_set_uri_allow_transports_wrapper(&c_err, h.h, c_mask)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_uri_allow_transports", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* SetUriAllowTls: set the allowed TLS settings in NBD URIs */
func (h *Libnbd) SetUriAllowTls(tls Tls) error {
	if h.h == nil {
		return closed_handle_error("set_uri_allow_tls")
	}

	var c_err C.struct_error
	c_tls := C.int(tls)

	ret := C._nbd_set_uri_allow_tls_wrapper(&c_err, h.h, c_tls)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_uri_allow_tls", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* SetUriAllowTlsPriority: set if 'tls-priority' is allowed in NBD URIs */
func (h *Libnbd) SetUriAllowTlsPriority(allow bool) error {
	if h.h == nil {
		return closed_handle_error("set_uri_allow_tls_priority")
	}

	var c_err C.struct_error
	c_allow := C.bool(allow)

	ret := C._nbd_set_uri_allow_tls_priority_wrapper(&c_err, h.h, c_allow)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_uri_allow_tls_priority", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* SetUriAllowLocalFile: set the allowed transports in NBD URIs */
func (h *Libnbd) SetUriAllowLocalFile(allow bool) error {
	if h.h == nil {
		return closed_handle_error("set_uri_allow_local_file")
	}

	var c_err C.struct_error
	c_allow := C.bool(allow)

	ret := C._nbd_set_uri_allow_local_file_wrapper(&c_err, h.h, c_allow)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_uri_allow_local_file", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* ConnectUri: connect to NBD URI */
func (h *Libnbd) ConnectUri(uri string) error {
	if h.h == nil {
		return closed_handle_error("connect_uri")
	}

	var c_err C.struct_error
	c_uri := C.CString(uri)
	defer C.free(unsafe.Pointer(c_uri))

	ret := C._nbd_connect_uri_wrapper(&c_err, h.h, c_uri)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("connect_uri", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* ConnectUnix: connect to NBD server over a Unix domain socket */
func (h *Libnbd) ConnectUnix(unixsocket string) error {
	if h.h == nil {
		return closed_handle_error("connect_unix")
	}

	var c_err C.struct_error
	c_unixsocket := C.CString(unixsocket)
	defer C.free(unsafe.Pointer(c_unixsocket))

	ret := C._nbd_connect_unix_wrapper(&c_err, h.h, c_unixsocket)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("connect_unix", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* ConnectVsock: connect to NBD server over AF_VSOCK protocol */
func (h *Libnbd) ConnectVsock(cid uint32, port uint32) error {
	if h.h == nil {
		return closed_handle_error("connect_vsock")
	}

	var c_err C.struct_error
	c_cid := C.uint32_t(cid)
	c_port := C.uint32_t(port)

	ret := C._nbd_connect_vsock_wrapper(&c_err, h.h, c_cid, c_port)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("connect_vsock", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* ConnectTcp: connect to NBD server over a TCP port */
func (h *Libnbd) ConnectTcp(hostname string, port string) error {
	if h.h == nil {
		return closed_handle_error("connect_tcp")
	}

	var c_err C.struct_error
	c_hostname := C.CString(hostname)
	defer C.free(unsafe.Pointer(c_hostname))
	c_port := C.CString(port)
	defer C.free(unsafe.Pointer(c_port))

	ret := C._nbd_connect_tcp_wrapper(&c_err, h.h, c_hostname, c_port)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("connect_tcp", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* ConnectSocket: connect directly to a connected socket */
func (h *Libnbd) ConnectSocket(sock int) error {
	if h.h == nil {
		return closed_handle_error("connect_socket")
	}

	var c_err C.struct_error
	c_sock := C.int(sock)

	ret := C._nbd_connect_socket_wrapper(&c_err, h.h, c_sock)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("connect_socket", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* ConnectCommand: connect to NBD server command */
func (h *Libnbd) ConnectCommand(argv []string) error {
	if h.h == nil {
		return closed_handle_error("connect_command")
	}

	var c_err C.struct_error
	c_argv := arg_string_list(argv)
	defer free_string_list(c_argv)

	ret := C._nbd_connect_command_wrapper(&c_err, h.h, &c_argv[0])
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("connect_command", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* ConnectSystemdSocketActivation: connect using systemd socket activation */
func (h *Libnbd) ConnectSystemdSocketActivation(argv []string) error {
	if h.h == nil {
		return closed_handle_error("connect_systemd_socket_activation")
	}

	var c_err C.struct_error
	c_argv := arg_string_list(argv)
	defer free_string_list(c_argv)

	ret := C._nbd_connect_systemd_socket_activation_wrapper(&c_err, h.h, &c_argv[0])
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("connect_systemd_socket_activation", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* SetSocketActivationName: set the socket activation name */
func (h *Libnbd) SetSocketActivationName(socket_name string) error {
	if h.h == nil {
		return closed_handle_error("set_socket_activation_name")
	}

	var c_err C.struct_error
	c_socket_name := C.CString(socket_name)
	defer C.free(unsafe.Pointer(c_socket_name))

	ret := C._nbd_set_socket_activation_name_wrapper(&c_err, h.h, c_socket_name)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("set_socket_activation_name", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetSocketActivationName: get the socket activation name */
func (h *Libnbd) GetSocketActivationName() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_socket_activation_name")
	}

	var c_err C.struct_error

	ret := C._nbd_get_socket_activation_name_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_socket_activation_name", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	r := C.GoString(ret)
	C.free(unsafe.Pointer(ret))
	return &r, nil
}

/* IsReadOnly: is the NBD export read-only? */
func (h *Libnbd) IsReadOnly() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("is_read_only")
	}

	var c_err C.struct_error

	ret := C._nbd_is_read_only_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("is_read_only", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* CanFlush: does the server support the flush command? */
func (h *Libnbd) CanFlush() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("can_flush")
	}

	var c_err C.struct_error

	ret := C._nbd_can_flush_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("can_flush", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* CanFua: does the server support the FUA flag? */
func (h *Libnbd) CanFua() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("can_fua")
	}

	var c_err C.struct_error

	ret := C._nbd_can_fua_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("can_fua", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* IsRotational: is the NBD disk rotational (like a disk)? */
func (h *Libnbd) IsRotational() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("is_rotational")
	}

	var c_err C.struct_error

	ret := C._nbd_is_rotational_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("is_rotational", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* CanTrim: does the server support the trim command? */
func (h *Libnbd) CanTrim() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("can_trim")
	}

	var c_err C.struct_error

	ret := C._nbd_can_trim_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("can_trim", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* CanZero: does the server support the zero command? */
func (h *Libnbd) CanZero() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("can_zero")
	}

	var c_err C.struct_error

	ret := C._nbd_can_zero_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("can_zero", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* CanFastZero: does the server support the fast zero flag? */
func (h *Libnbd) CanFastZero() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("can_fast_zero")
	}

	var c_err C.struct_error

	ret := C._nbd_can_fast_zero_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("can_fast_zero", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* CanBlockStatusPayload: does the server support the block status payload flag? */
func (h *Libnbd) CanBlockStatusPayload() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("can_block_status_payload")
	}

	var c_err C.struct_error

	ret := C._nbd_can_block_status_payload_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("can_block_status_payload", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* CanDf: does the server support the don't fragment flag to pread? */
func (h *Libnbd) CanDf() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("can_df")
	}

	var c_err C.struct_error

	ret := C._nbd_can_df_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("can_df", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* CanMultiConn: does the server support multi-conn? */
func (h *Libnbd) CanMultiConn() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("can_multi_conn")
	}

	var c_err C.struct_error

	ret := C._nbd_can_multi_conn_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("can_multi_conn", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* CanCache: does the server support the cache command? */
func (h *Libnbd) CanCache() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("can_cache")
	}

	var c_err C.struct_error

	ret := C._nbd_can_cache_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("can_cache", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* CanMetaContext: does the server support a specific meta context? */
func (h *Libnbd) CanMetaContext(metacontext string) (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("can_meta_context")
	}

	var c_err C.struct_error
	c_metacontext := C.CString(metacontext)
	defer C.free(unsafe.Pointer(c_metacontext))

	ret := C._nbd_can_meta_context_wrapper(&c_err, h.h, c_metacontext)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("can_meta_context", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* GetProtocol: return the NBD protocol variant */
func (h *Libnbd) GetProtocol() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_protocol")
	}

	var c_err C.struct_error

	ret := C._nbd_get_protocol_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_protocol", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	/* ret is statically allocated, do not free it. */
	r := C.GoString(ret)
	return &r, nil
}

/* GetSize: return the export size */
func (h *Libnbd) GetSize() (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("get_size")
	}

	var c_err C.struct_error

	ret := C._nbd_get_size_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_size", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* GetBlockSize: return a specific server block size constraint */
func (h *Libnbd) GetBlockSize(size_type Size) (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("get_block_size")
	}

	var c_err C.struct_error
	c_size_type := C.int(size_type)

	ret := C._nbd_get_block_size_wrapper(&c_err, h.h, c_size_type)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_block_size", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* Struct carrying optional arguments for Pread. */
type PreadOptargs struct {
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* Pread: read from the NBD server */
func (h *Libnbd) Pread(buf []byte, offset uint64, optargs *PreadOptargs) error {
	if h.h == nil {
		return closed_handle_error("pread")
	}

	var c_err C.struct_error
	c_buf := unsafe.Pointer(&buf[0])
	c_count := C.size_t(len(buf))
	c_offset := C.uint64_t(offset)
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_pread_wrapper(&c_err, h.h, c_buf, c_count, c_offset, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("pread", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for PreadStructured. */
type PreadStructuredOptargs struct {
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* PreadStructured: read from the NBD server */
func (h *Libnbd) PreadStructured(buf []byte, offset uint64, chunk ChunkCallback, optargs *PreadStructuredOptargs) error {
	if h.h == nil {
		return closed_handle_error("pread_structured")
	}

	var c_err C.struct_error
	c_buf := unsafe.Pointer(&buf[0])
	c_count := C.size_t(len(buf))
	c_offset := C.uint64_t(offset)
	var c_chunk C.nbd_chunk_callback
	c_chunk.callback = (*[0]byte)(C._nbd_chunk_callback_wrapper)
	c_chunk.free = (*[0]byte)(C._nbd_chunk_callback_free)
	chunk_cbid := registerCallbackId(chunk)
	c_chunk.user_data = C.alloc_cbid(C.long(chunk_cbid))
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_pread_structured_wrapper(&c_err, h.h, c_buf, c_count, c_offset, c_chunk, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("pread_structured", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for Pwrite. */
type PwriteOptargs struct {
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* Pwrite: write to the NBD server */
func (h *Libnbd) Pwrite(buf []byte, offset uint64, optargs *PwriteOptargs) error {
	if h.h == nil {
		return closed_handle_error("pwrite")
	}

	var c_err C.struct_error
	c_buf := unsafe.Pointer(&buf[0])
	c_count := C.size_t(len(buf))
	c_offset := C.uint64_t(offset)
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_pwrite_wrapper(&c_err, h.h, c_buf, c_count, c_offset, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("pwrite", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for Shutdown. */
type ShutdownOptargs struct {
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    Shutdown
}

/* Shutdown: disconnect from the NBD server */
func (h *Libnbd) Shutdown(optargs *ShutdownOptargs) error {
	if h.h == nil {
		return closed_handle_error("shutdown")
	}

	var c_err C.struct_error
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_shutdown_wrapper(&c_err, h.h, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("shutdown", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for Flush. */
type FlushOptargs struct {
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* Flush: send flush command to the NBD server */
func (h *Libnbd) Flush(optargs *FlushOptargs) error {
	if h.h == nil {
		return closed_handle_error("flush")
	}

	var c_err C.struct_error
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_flush_wrapper(&c_err, h.h, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("flush", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for Trim. */
type TrimOptargs struct {
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* Trim: send trim command to the NBD server */
func (h *Libnbd) Trim(count uint64, offset uint64, optargs *TrimOptargs) error {
	if h.h == nil {
		return closed_handle_error("trim")
	}

	var c_err C.struct_error
	c_count := C.uint64_t(count)
	c_offset := C.uint64_t(offset)
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_trim_wrapper(&c_err, h.h, c_count, c_offset, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("trim", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for Cache. */
type CacheOptargs struct {
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* Cache: send cache (prefetch) command to the NBD server */
func (h *Libnbd) Cache(count uint64, offset uint64, optargs *CacheOptargs) error {
	if h.h == nil {
		return closed_handle_error("cache")
	}

	var c_err C.struct_error
	c_count := C.uint64_t(count)
	c_offset := C.uint64_t(offset)
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_cache_wrapper(&c_err, h.h, c_count, c_offset, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("cache", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for Zero. */
type ZeroOptargs struct {
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* Zero: send write zeroes command to the NBD server */
func (h *Libnbd) Zero(count uint64, offset uint64, optargs *ZeroOptargs) error {
	if h.h == nil {
		return closed_handle_error("zero")
	}

	var c_err C.struct_error
	c_count := C.uint64_t(count)
	c_offset := C.uint64_t(offset)
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_zero_wrapper(&c_err, h.h, c_count, c_offset, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("zero", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for BlockStatus. */
type BlockStatusOptargs struct {
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* BlockStatus: send block status command, with 32-bit callback */
func (h *Libnbd) BlockStatus(count uint64, offset uint64, extent ExtentCallback, optargs *BlockStatusOptargs) error {
	if h.h == nil {
		return closed_handle_error("block_status")
	}

	var c_err C.struct_error
	c_count := C.uint64_t(count)
	c_offset := C.uint64_t(offset)
	var c_extent C.nbd_extent_callback
	c_extent.callback = (*[0]byte)(C._nbd_extent_callback_wrapper)
	c_extent.free = (*[0]byte)(C._nbd_extent_callback_free)
	extent_cbid := registerCallbackId(extent)
	c_extent.user_data = C.alloc_cbid(C.long(extent_cbid))
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_block_status_wrapper(&c_err, h.h, c_count, c_offset, c_extent, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("block_status", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for BlockStatus64. */
type BlockStatus64Optargs struct {
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* BlockStatus64: send block status command, with 64-bit callback */
func (h *Libnbd) BlockStatus64(count uint64, offset uint64, extent64 Extent64Callback, optargs *BlockStatus64Optargs) error {
	if h.h == nil {
		return closed_handle_error("block_status_64")
	}

	var c_err C.struct_error
	c_count := C.uint64_t(count)
	c_offset := C.uint64_t(offset)
	var c_extent64 C.nbd_extent64_callback
	c_extent64.callback = (*[0]byte)(C._nbd_extent64_callback_wrapper)
	c_extent64.free = (*[0]byte)(C._nbd_extent64_callback_free)
	extent64_cbid := registerCallbackId(extent64)
	c_extent64.user_data = C.alloc_cbid(C.long(extent64_cbid))
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_block_status_64_wrapper(&c_err, h.h, c_count, c_offset, c_extent64, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("block_status_64", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for BlockStatusFilter. */
type BlockStatusFilterOptargs struct {
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* BlockStatusFilter: send filtered block status command, with 64-bit callback */
func (h *Libnbd) BlockStatusFilter(count uint64, offset uint64, contexts []string, extent64 Extent64Callback, optargs *BlockStatusFilterOptargs) error {
	if h.h == nil {
		return closed_handle_error("block_status_filter")
	}

	var c_err C.struct_error
	c_count := C.uint64_t(count)
	c_offset := C.uint64_t(offset)
	c_contexts := arg_string_list(contexts)
	defer free_string_list(c_contexts)
	var c_extent64 C.nbd_extent64_callback
	c_extent64.callback = (*[0]byte)(C._nbd_extent64_callback_wrapper)
	c_extent64.free = (*[0]byte)(C._nbd_extent64_callback_free)
	extent64_cbid := registerCallbackId(extent64)
	c_extent64.user_data = C.alloc_cbid(C.long(extent64_cbid))
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_block_status_filter_wrapper(&c_err, h.h, c_count, c_offset, &c_contexts[0], c_extent64, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("block_status_filter", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Poll: poll the handle once */
func (h *Libnbd) Poll(timeout int) (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("poll")
	}

	var c_err C.struct_error
	c_timeout := C.int(timeout)

	ret := C._nbd_poll_wrapper(&c_err, h.h, c_timeout)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("poll", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* Poll2: poll the handle once, with fd */
func (h *Libnbd) Poll2(fd int, timeout int) (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("poll2")
	}

	var c_err C.struct_error
	c_fd := C.int(fd)
	c_timeout := C.int(timeout)

	ret := C._nbd_poll2_wrapper(&c_err, h.h, c_fd, c_timeout)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("poll2", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* AioConnect: connect to the NBD server */
func (h *Libnbd) AioConnect(addr string) error {
	if h.h == nil {
		return closed_handle_error("aio_connect")
	}

	var c_err C.struct_error
	panic("SockAddrAndLen not supported")
	var c_addr *C.struct_sockaddr
	var c_addrlen C.uint

	ret := C._nbd_aio_connect_wrapper(&c_err, h.h, c_addr, c_addrlen)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_connect", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* AioConnectUri: connect to an NBD URI */
func (h *Libnbd) AioConnectUri(uri string) error {
	if h.h == nil {
		return closed_handle_error("aio_connect_uri")
	}

	var c_err C.struct_error
	c_uri := C.CString(uri)
	defer C.free(unsafe.Pointer(c_uri))

	ret := C._nbd_aio_connect_uri_wrapper(&c_err, h.h, c_uri)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_connect_uri", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* AioConnectUnix: connect to the NBD server over a Unix domain socket */
func (h *Libnbd) AioConnectUnix(unixsocket string) error {
	if h.h == nil {
		return closed_handle_error("aio_connect_unix")
	}

	var c_err C.struct_error
	c_unixsocket := C.CString(unixsocket)
	defer C.free(unsafe.Pointer(c_unixsocket))

	ret := C._nbd_aio_connect_unix_wrapper(&c_err, h.h, c_unixsocket)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_connect_unix", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* AioConnectVsock: connect to the NBD server over AF_VSOCK socket */
func (h *Libnbd) AioConnectVsock(cid uint32, port uint32) error {
	if h.h == nil {
		return closed_handle_error("aio_connect_vsock")
	}

	var c_err C.struct_error
	c_cid := C.uint32_t(cid)
	c_port := C.uint32_t(port)

	ret := C._nbd_aio_connect_vsock_wrapper(&c_err, h.h, c_cid, c_port)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_connect_vsock", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* AioConnectTcp: connect to the NBD server over a TCP port */
func (h *Libnbd) AioConnectTcp(hostname string, port string) error {
	if h.h == nil {
		return closed_handle_error("aio_connect_tcp")
	}

	var c_err C.struct_error
	c_hostname := C.CString(hostname)
	defer C.free(unsafe.Pointer(c_hostname))
	c_port := C.CString(port)
	defer C.free(unsafe.Pointer(c_port))

	ret := C._nbd_aio_connect_tcp_wrapper(&c_err, h.h, c_hostname, c_port)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_connect_tcp", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* AioConnectSocket: connect directly to a connected socket */
func (h *Libnbd) AioConnectSocket(sock int) error {
	if h.h == nil {
		return closed_handle_error("aio_connect_socket")
	}

	var c_err C.struct_error
	c_sock := C.int(sock)

	ret := C._nbd_aio_connect_socket_wrapper(&c_err, h.h, c_sock)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_connect_socket", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* AioConnectCommand: connect to the NBD server */
func (h *Libnbd) AioConnectCommand(argv []string) error {
	if h.h == nil {
		return closed_handle_error("aio_connect_command")
	}

	var c_err C.struct_error
	c_argv := arg_string_list(argv)
	defer free_string_list(c_argv)

	ret := C._nbd_aio_connect_command_wrapper(&c_err, h.h, &c_argv[0])
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_connect_command", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* AioConnectSystemdSocketActivation: connect using systemd socket activation */
func (h *Libnbd) AioConnectSystemdSocketActivation(argv []string) error {
	if h.h == nil {
		return closed_handle_error("aio_connect_systemd_socket_activation")
	}

	var c_err C.struct_error
	c_argv := arg_string_list(argv)
	defer free_string_list(c_argv)

	ret := C._nbd_aio_connect_systemd_socket_activation_wrapper(&c_err, h.h, &c_argv[0])
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_connect_systemd_socket_activation", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for AioOptGo. */
type AioOptGoOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
}

/* AioOptGo: end negotiation and move on to using an export */
func (h *Libnbd) AioOptGo(optargs *AioOptGoOptargs) error {
	if h.h == nil {
		return closed_handle_error("aio_opt_go")
	}

	var c_err C.struct_error
	var c_completion C.nbd_completion_callback
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
	}

	ret := C._nbd_aio_opt_go_wrapper(&c_err, h.h, c_completion)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_opt_go", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* AioOptAbort: end negotiation and close the connection */
func (h *Libnbd) AioOptAbort() error {
	if h.h == nil {
		return closed_handle_error("aio_opt_abort")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_opt_abort_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_opt_abort", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for AioOptStarttls. */
type AioOptStarttlsOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
}

/* AioOptStarttls: request the server to initiate TLS */
func (h *Libnbd) AioOptStarttls(optargs *AioOptStarttlsOptargs) error {
	if h.h == nil {
		return closed_handle_error("aio_opt_starttls")
	}

	var c_err C.struct_error
	var c_completion C.nbd_completion_callback
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
	}

	ret := C._nbd_aio_opt_starttls_wrapper(&c_err, h.h, c_completion)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_opt_starttls", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for AioOptExtendedHeaders. */
type AioOptExtendedHeadersOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
}

/* AioOptExtendedHeaders: request the server to enable extended headers */
func (h *Libnbd) AioOptExtendedHeaders(optargs *AioOptExtendedHeadersOptargs) error {
	if h.h == nil {
		return closed_handle_error("aio_opt_extended_headers")
	}

	var c_err C.struct_error
	var c_completion C.nbd_completion_callback
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
	}

	ret := C._nbd_aio_opt_extended_headers_wrapper(&c_err, h.h, c_completion)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_opt_extended_headers", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for AioOptStructuredReply. */
type AioOptStructuredReplyOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
}

/* AioOptStructuredReply: request the server to enable structured replies */
func (h *Libnbd) AioOptStructuredReply(optargs *AioOptStructuredReplyOptargs) error {
	if h.h == nil {
		return closed_handle_error("aio_opt_structured_reply")
	}

	var c_err C.struct_error
	var c_completion C.nbd_completion_callback
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
	}

	ret := C._nbd_aio_opt_structured_reply_wrapper(&c_err, h.h, c_completion)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_opt_structured_reply", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for AioOptList. */
type AioOptListOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
}

/* AioOptList: request the server to list all exports during negotiation */
func (h *Libnbd) AioOptList(list ListCallback, optargs *AioOptListOptargs) error {
	if h.h == nil {
		return closed_handle_error("aio_opt_list")
	}

	var c_err C.struct_error
	var c_list C.nbd_list_callback
	c_list.callback = (*[0]byte)(C._nbd_list_callback_wrapper)
	c_list.free = (*[0]byte)(C._nbd_list_callback_free)
	list_cbid := registerCallbackId(list)
	c_list.user_data = C.alloc_cbid(C.long(list_cbid))
	var c_completion C.nbd_completion_callback
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
	}

	ret := C._nbd_aio_opt_list_wrapper(&c_err, h.h, c_list, c_completion)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_opt_list", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for AioOptInfo. */
type AioOptInfoOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
}

/* AioOptInfo: request the server for information about an export */
func (h *Libnbd) AioOptInfo(optargs *AioOptInfoOptargs) error {
	if h.h == nil {
		return closed_handle_error("aio_opt_info")
	}

	var c_err C.struct_error
	var c_completion C.nbd_completion_callback
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
	}

	ret := C._nbd_aio_opt_info_wrapper(&c_err, h.h, c_completion)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_opt_info", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for AioOptListMetaContext. */
type AioOptListMetaContextOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
}

/* AioOptListMetaContext: request list of available meta contexts, using implicit query */
func (h *Libnbd) AioOptListMetaContext(context ContextCallback, optargs *AioOptListMetaContextOptargs) (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_opt_list_meta_context")
	}

	var c_err C.struct_error
	var c_context C.nbd_context_callback
	c_context.callback = (*[0]byte)(C._nbd_context_callback_wrapper)
	c_context.free = (*[0]byte)(C._nbd_context_callback_free)
	context_cbid := registerCallbackId(context)
	c_context.user_data = C.alloc_cbid(C.long(context_cbid))
	var c_completion C.nbd_completion_callback
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
	}

	ret := C._nbd_aio_opt_list_meta_context_wrapper(&c_err, h.h, c_context, c_completion)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_opt_list_meta_context", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* Struct carrying optional arguments for AioOptListMetaContextQueries. */
type AioOptListMetaContextQueriesOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
}

/* AioOptListMetaContextQueries: request list of available meta contexts, using explicit query */
func (h *Libnbd) AioOptListMetaContextQueries(queries []string, context ContextCallback, optargs *AioOptListMetaContextQueriesOptargs) (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_opt_list_meta_context_queries")
	}

	var c_err C.struct_error
	c_queries := arg_string_list(queries)
	defer free_string_list(c_queries)
	var c_context C.nbd_context_callback
	c_context.callback = (*[0]byte)(C._nbd_context_callback_wrapper)
	c_context.free = (*[0]byte)(C._nbd_context_callback_free)
	context_cbid := registerCallbackId(context)
	c_context.user_data = C.alloc_cbid(C.long(context_cbid))
	var c_completion C.nbd_completion_callback
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
	}

	ret := C._nbd_aio_opt_list_meta_context_queries_wrapper(&c_err, h.h, &c_queries[0], c_context, c_completion)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_opt_list_meta_context_queries", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* Struct carrying optional arguments for AioOptSetMetaContext. */
type AioOptSetMetaContextOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
}

/* AioOptSetMetaContext: select specific meta contexts, with implicit query list */
func (h *Libnbd) AioOptSetMetaContext(context ContextCallback, optargs *AioOptSetMetaContextOptargs) (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_opt_set_meta_context")
	}

	var c_err C.struct_error
	var c_context C.nbd_context_callback
	c_context.callback = (*[0]byte)(C._nbd_context_callback_wrapper)
	c_context.free = (*[0]byte)(C._nbd_context_callback_free)
	context_cbid := registerCallbackId(context)
	c_context.user_data = C.alloc_cbid(C.long(context_cbid))
	var c_completion C.nbd_completion_callback
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
	}

	ret := C._nbd_aio_opt_set_meta_context_wrapper(&c_err, h.h, c_context, c_completion)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_opt_set_meta_context", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* Struct carrying optional arguments for AioOptSetMetaContextQueries. */
type AioOptSetMetaContextQueriesOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
}

/* AioOptSetMetaContextQueries: select specific meta contexts, with explicit query list */
func (h *Libnbd) AioOptSetMetaContextQueries(queries []string, context ContextCallback, optargs *AioOptSetMetaContextQueriesOptargs) (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_opt_set_meta_context_queries")
	}

	var c_err C.struct_error
	c_queries := arg_string_list(queries)
	defer free_string_list(c_queries)
	var c_context C.nbd_context_callback
	c_context.callback = (*[0]byte)(C._nbd_context_callback_wrapper)
	c_context.free = (*[0]byte)(C._nbd_context_callback_free)
	context_cbid := registerCallbackId(context)
	c_context.user_data = C.alloc_cbid(C.long(context_cbid))
	var c_completion C.nbd_completion_callback
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
	}

	ret := C._nbd_aio_opt_set_meta_context_queries_wrapper(&c_err, h.h, &c_queries[0], c_context, c_completion)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_opt_set_meta_context_queries", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* Struct carrying optional arguments for AioPread. */
type AioPreadOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* AioPread: read from the NBD server */
func (h *Libnbd) AioPread(buf AioBuffer, offset uint64, optargs *AioPreadOptargs) (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_pread")
	}

	var c_err C.struct_error
	c_buf := buf.P
	c_count := C.size_t(buf.Size)
	c_offset := C.uint64_t(offset)
	var c_completion C.nbd_completion_callback
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_aio_pread_wrapper(&c_err, h.h, c_buf, c_count, c_offset, c_completion, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_pread", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* Struct carrying optional arguments for AioPreadStructured. */
type AioPreadStructuredOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* AioPreadStructured: read from the NBD server */
func (h *Libnbd) AioPreadStructured(buf AioBuffer, offset uint64, chunk ChunkCallback, optargs *AioPreadStructuredOptargs) (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_pread_structured")
	}

	var c_err C.struct_error
	c_buf := buf.P
	c_count := C.size_t(buf.Size)
	c_offset := C.uint64_t(offset)
	var c_chunk C.nbd_chunk_callback
	c_chunk.callback = (*[0]byte)(C._nbd_chunk_callback_wrapper)
	c_chunk.free = (*[0]byte)(C._nbd_chunk_callback_free)
	chunk_cbid := registerCallbackId(chunk)
	c_chunk.user_data = C.alloc_cbid(C.long(chunk_cbid))
	var c_completion C.nbd_completion_callback
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_aio_pread_structured_wrapper(&c_err, h.h, c_buf, c_count, c_offset, c_chunk, c_completion, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_pread_structured", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* Struct carrying optional arguments for AioPwrite. */
type AioPwriteOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* AioPwrite: write to the NBD server */
func (h *Libnbd) AioPwrite(buf AioBuffer, offset uint64, optargs *AioPwriteOptargs) (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_pwrite")
	}

	var c_err C.struct_error
	c_buf := buf.P
	c_count := C.size_t(buf.Size)
	c_offset := C.uint64_t(offset)
	var c_completion C.nbd_completion_callback
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_aio_pwrite_wrapper(&c_err, h.h, c_buf, c_count, c_offset, c_completion, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_pwrite", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* Struct carrying optional arguments for AioDisconnect. */
type AioDisconnectOptargs struct {
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* AioDisconnect: disconnect from the NBD server */
func (h *Libnbd) AioDisconnect(optargs *AioDisconnectOptargs) error {
	if h.h == nil {
		return closed_handle_error("aio_disconnect")
	}

	var c_err C.struct_error
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_aio_disconnect_wrapper(&c_err, h.h, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_disconnect", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* Struct carrying optional arguments for AioFlush. */
type AioFlushOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* AioFlush: send flush command to the NBD server */
func (h *Libnbd) AioFlush(optargs *AioFlushOptargs) (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_flush")
	}

	var c_err C.struct_error
	var c_completion C.nbd_completion_callback
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_aio_flush_wrapper(&c_err, h.h, c_completion, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_flush", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* Struct carrying optional arguments for AioTrim. */
type AioTrimOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* AioTrim: send trim command to the NBD server */
func (h *Libnbd) AioTrim(count uint64, offset uint64, optargs *AioTrimOptargs) (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_trim")
	}

	var c_err C.struct_error
	c_count := C.uint64_t(count)
	c_offset := C.uint64_t(offset)
	var c_completion C.nbd_completion_callback
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_aio_trim_wrapper(&c_err, h.h, c_count, c_offset, c_completion, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_trim", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* Struct carrying optional arguments for AioCache. */
type AioCacheOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* AioCache: send cache (prefetch) command to the NBD server */
func (h *Libnbd) AioCache(count uint64, offset uint64, optargs *AioCacheOptargs) (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_cache")
	}

	var c_err C.struct_error
	c_count := C.uint64_t(count)
	c_offset := C.uint64_t(offset)
	var c_completion C.nbd_completion_callback
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_aio_cache_wrapper(&c_err, h.h, c_count, c_offset, c_completion, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_cache", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* Struct carrying optional arguments for AioZero. */
type AioZeroOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* AioZero: send write zeroes command to the NBD server */
func (h *Libnbd) AioZero(count uint64, offset uint64, optargs *AioZeroOptargs) (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_zero")
	}

	var c_err C.struct_error
	c_count := C.uint64_t(count)
	c_offset := C.uint64_t(offset)
	var c_completion C.nbd_completion_callback
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_aio_zero_wrapper(&c_err, h.h, c_count, c_offset, c_completion, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_zero", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* Struct carrying optional arguments for AioBlockStatus. */
type AioBlockStatusOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* AioBlockStatus: send block status command, with 32-bit callback */
func (h *Libnbd) AioBlockStatus(count uint64, offset uint64, extent ExtentCallback, optargs *AioBlockStatusOptargs) (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_block_status")
	}

	var c_err C.struct_error
	c_count := C.uint64_t(count)
	c_offset := C.uint64_t(offset)
	var c_extent C.nbd_extent_callback
	c_extent.callback = (*[0]byte)(C._nbd_extent_callback_wrapper)
	c_extent.free = (*[0]byte)(C._nbd_extent_callback_free)
	extent_cbid := registerCallbackId(extent)
	c_extent.user_data = C.alloc_cbid(C.long(extent_cbid))
	var c_completion C.nbd_completion_callback
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_aio_block_status_wrapper(&c_err, h.h, c_count, c_offset, c_extent, c_completion, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_block_status", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* Struct carrying optional arguments for AioBlockStatus64. */
type AioBlockStatus64Optargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* AioBlockStatus64: send block status command, with 64-bit callback */
func (h *Libnbd) AioBlockStatus64(count uint64, offset uint64, extent64 Extent64Callback, optargs *AioBlockStatus64Optargs) (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_block_status_64")
	}

	var c_err C.struct_error
	c_count := C.uint64_t(count)
	c_offset := C.uint64_t(offset)
	var c_extent64 C.nbd_extent64_callback
	c_extent64.callback = (*[0]byte)(C._nbd_extent64_callback_wrapper)
	c_extent64.free = (*[0]byte)(C._nbd_extent64_callback_free)
	extent64_cbid := registerCallbackId(extent64)
	c_extent64.user_data = C.alloc_cbid(C.long(extent64_cbid))
	var c_completion C.nbd_completion_callback
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_aio_block_status_64_wrapper(&c_err, h.h, c_count, c_offset, c_extent64, c_completion, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_block_status_64", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* Struct carrying optional arguments for AioBlockStatusFilter. */
type AioBlockStatusFilterOptargs struct {
	/* CompletionCallback field is ignored unless CompletionCallbackSet == true. */
	CompletionCallbackSet bool
	CompletionCallback    CompletionCallback
	/* Flags field is ignored unless FlagsSet == true. */
	FlagsSet bool
	Flags    CmdFlag
}

/* AioBlockStatusFilter: send filtered block status command to the NBD server */
func (h *Libnbd) AioBlockStatusFilter(count uint64, offset uint64, contexts []string, extent64 Extent64Callback, optargs *AioBlockStatusFilterOptargs) (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_block_status_filter")
	}

	var c_err C.struct_error
	c_count := C.uint64_t(count)
	c_offset := C.uint64_t(offset)
	c_contexts := arg_string_list(contexts)
	defer free_string_list(c_contexts)
	var c_extent64 C.nbd_extent64_callback
	c_extent64.callback = (*[0]byte)(C._nbd_extent64_callback_wrapper)
	c_extent64.free = (*[0]byte)(C._nbd_extent64_callback_free)
	extent64_cbid := registerCallbackId(extent64)
	c_extent64.user_data = C.alloc_cbid(C.long(extent64_cbid))
	var c_completion C.nbd_completion_callback
	var c_flags C.uint32_t
	if optargs != nil {
		if optargs.CompletionCallbackSet {
			c_completion.callback = (*[0]byte)(C._nbd_completion_callback_wrapper)
			c_completion.free = (*[0]byte)(C._nbd_completion_callback_free)
			completion_cbid := registerCallbackId(optargs.CompletionCallback)
			c_completion.user_data = C.alloc_cbid(C.long(completion_cbid))
		}
		if optargs.FlagsSet {
			c_flags = C.uint32_t(optargs.Flags)
		}
	}

	ret := C._nbd_aio_block_status_filter_wrapper(&c_err, h.h, c_count, c_offset, &c_contexts[0], c_extent64, c_completion, c_flags)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_block_status_filter", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* AioGetFd: return file descriptor associated with this connection */
func (h *Libnbd) AioGetFd() (int, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_get_fd")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_get_fd_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_get_fd", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return int(ret), nil
}

/* AioGetDirection: return the read or write direction */
func (h *Libnbd) AioGetDirection() (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_get_direction")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_get_direction_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	return uint(ret), nil
}

/* AioNotifyRead: notify that the connection is readable */
func (h *Libnbd) AioNotifyRead() error {
	if h.h == nil {
		return closed_handle_error("aio_notify_read")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_notify_read_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_notify_read", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* AioNotifyWrite: notify that the connection is writable */
func (h *Libnbd) AioNotifyWrite() error {
	if h.h == nil {
		return closed_handle_error("aio_notify_write")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_notify_write_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_notify_write", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* AioIsCreated: check if the connection has just been created */
func (h *Libnbd) AioIsCreated() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("aio_is_created")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_is_created_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_is_created", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* AioIsConnecting: check if the connection is connecting or handshaking */
func (h *Libnbd) AioIsConnecting() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("aio_is_connecting")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_is_connecting_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_is_connecting", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* AioIsNegotiating: check if connection is ready to send handshake option */
func (h *Libnbd) AioIsNegotiating() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("aio_is_negotiating")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_is_negotiating_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_is_negotiating", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* AioIsReady: check if the connection is in the ready state */
func (h *Libnbd) AioIsReady() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("aio_is_ready")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_is_ready_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_is_ready", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* AioIsProcessing: check if the connection is processing a command */
func (h *Libnbd) AioIsProcessing() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("aio_is_processing")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_is_processing_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_is_processing", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* AioIsDead: check if the connection is dead */
func (h *Libnbd) AioIsDead() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("aio_is_dead")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_is_dead_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_is_dead", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* AioIsClosed: check if the connection is closed */
func (h *Libnbd) AioIsClosed() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("aio_is_closed")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_is_closed_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_is_closed", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* AioCommandCompleted: check if the command completed */
func (h *Libnbd) AioCommandCompleted(cookie uint64) (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("aio_command_completed")
	}

	var c_err C.struct_error
	c_cookie := C.uint64_t(cookie)

	ret := C._nbd_aio_command_completed_wrapper(&c_err, h.h, c_cookie)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_command_completed", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* AioPeekCommandCompleted: check if any command has completed */
func (h *Libnbd) AioPeekCommandCompleted() (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_peek_command_completed")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_peek_command_completed_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_peek_command_completed", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* AioInFlight: check how many aio commands are still in flight */
func (h *Libnbd) AioInFlight() (uint, error) {
	if h.h == nil {
		return 0, closed_handle_error("aio_in_flight")
	}

	var c_err C.struct_error

	ret := C._nbd_aio_in_flight_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("aio_in_flight", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint(ret), nil
}

/* ConnectionState: return string describing the state of the connection */
func (h *Libnbd) ConnectionState() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("connection_state")
	}

	var c_err C.struct_error

	ret := C._nbd_connection_state_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("connection_state", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	/* ret is statically allocated, do not free it. */
	r := C.GoString(ret)
	return &r, nil
}

/* GetPackageName: return the name of the library */
func (h *Libnbd) GetPackageName() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_package_name")
	}

	var c_err C.struct_error

	ret := C._nbd_get_package_name_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_package_name", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	/* ret is statically allocated, do not free it. */
	r := C.GoString(ret)
	return &r, nil
}

/* GetVersion: return the version of the library */
func (h *Libnbd) GetVersion() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_version")
	}

	var c_err C.struct_error

	ret := C._nbd_get_version_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_version", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	/* ret is statically allocated, do not free it. */
	r := C.GoString(ret)
	return &r, nil
}

/* GetVersionExtra: return the extra version of the library */
func (h *Libnbd) GetVersionExtra() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_version_extra")
	}

	var c_err C.struct_error

	ret := C._nbd_get_version_extra_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_version_extra", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	/* ret is statically allocated, do not free it. */
	r := C.GoString(ret)
	return &r, nil
}

/* KillSubprocess: kill server running as a subprocess */
func (h *Libnbd) KillSubprocess(signum int) error {
	if h.h == nil {
		return closed_handle_error("kill_subprocess")
	}

	var c_err C.struct_error
	c_signum := C.int(signum)

	ret := C._nbd_kill_subprocess_wrapper(&c_err, h.h, c_signum)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("kill_subprocess", c_err)
		C.free_error(&c_err)
		return err
	}
	return nil
}

/* GetSubprocessPid: get the process ID of the subprocess */
func (h *Libnbd) GetSubprocessPid() (uint64, error) {
	if h.h == nil {
		return 0, closed_handle_error("get_subprocess_pid")
	}

	var c_err C.struct_error

	ret := C._nbd_get_subprocess_pid_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("get_subprocess_pid", c_err)
		C.free_error(&c_err)
		return 0, err
	}
	return uint64(ret), nil
}

/* SupportsTls: true if libnbd was compiled with support for TLS */
func (h *Libnbd) SupportsTls() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("supports_tls")
	}

	var c_err C.struct_error

	ret := C._nbd_supports_tls_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("supports_tls", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* SupportsVsock: true if libnbd was compiled with support for AF_VSOCK */
func (h *Libnbd) SupportsVsock() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("supports_vsock")
	}

	var c_err C.struct_error

	ret := C._nbd_supports_vsock_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("supports_vsock", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* SupportsUri: true if libnbd was compiled with support for NBD URIs */
func (h *Libnbd) SupportsUri() (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("supports_uri")
	}

	var c_err C.struct_error

	ret := C._nbd_supports_uri_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("supports_uri", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}

/* GetUri: construct an NBD URI for a connection */
func (h *Libnbd) GetUri() (*string, error) {
	if h.h == nil {
		return nil, closed_handle_error("get_uri")
	}

	var c_err C.struct_error

	ret := C._nbd_get_uri_wrapper(&c_err, h.h)
	runtime.KeepAlive(h.h)
	if ret == nil {
		err := get_error("get_uri", c_err)
		C.free_error(&c_err)
		return nil, err
	}
	r := C.GoString(ret)
	C.free(unsafe.Pointer(ret))
	return &r, nil
}

/* IsUri: detect if a string could be an NBD URI */
func (h *Libnbd) IsUri(uri string) (bool, error) {
	if h.h == nil {
		return false, closed_handle_error("is_uri")
	}

	var c_err C.struct_error
	c_uri := C.CString(uri)
	defer C.free(unsafe.Pointer(c_uri))

	ret := C._nbd_is_uri_wrapper(&c_err, h.h, c_uri)
	runtime.KeepAlive(h.h)
	if ret == -1 {
		err := get_error("is_uri", c_err)
		C.free_error(&c_err)
		return false, err
	}
	return int(ret) != 0, nil
}
