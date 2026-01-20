/* SPDX-License-Identifier: LGPL-3.0-or-later */

package VirtLint

import (
	"fmt"
	"unsafe"

	libvirt "libvirt.org/go/libvirt"
)

/*
#cgo pkg-config: virt_lint
#include "virt_lint.h"
*/
import "C"

type Error struct {
	msg string
}

func (err Error) Error() string {
	return err.msg
}

func makeError(err **C.VirtLintError) Error {
	cmsg := C.virt_lint_error_get_message(*err)
	defer C.virt_lint_string_free(cmsg)
	defer C.virt_lint_error_free(err)

	return Error{C.GoString(cmsg)}
}

func Virt_lint_version() uint {
	result := C.virt_lint_version()
	return uint(result)
}

func List_validator_tags() ([]string, error) {
	var vlErr *C.VirtLintError = nil
	var ctags **C.char

	ret := C.virt_lint_list_tags(&ctags, &vlErr)
	if ret < 0 {
		return []string{}, makeError(&vlErr)
	}

	defer C.virt_lint_string_free((*C.char)(unsafe.Pointer(ctags)))

	ctags_slice := unsafe.Slice(ctags, ret)

	tags := make([]string, int(ret))

	for i := 0; i < len(ctags_slice); i++ {
		ctag := ctags_slice[i]

		defer C.virt_lint_string_free(ctag)
		tags[i] = C.GoString(ctag)
	}

	return tags, nil
}

type VirtLint struct {
	ptr *C.VirtLint
}

func New(conn *libvirt.Connect) (*VirtLint, error) {
	var raw_ptr C.virConnectPtr = nil

	if conn != nil {
		conn_raw_ptr, err := conn.RawPtr()
		if err != nil {
			return nil, err
		}
		defer libvirt.CloseRawPtr(conn_raw_ptr)

		raw_ptr = (C.virConnectPtr)(unsafe.Pointer(conn_raw_ptr))
	}

	vl := C.virt_lint_new(raw_ptr)
	return &VirtLint{ptr: vl}, nil
}

func (vl *VirtLint) Close() {
	C.virt_lint_free(vl.ptr)
	vl.ptr = nil
}

func (vl *VirtLint) CapabilitiesSet(capsxml string) error {
	var vlErr *C.VirtLintError = nil

	caps := C.CString(capsxml)

	if C.virt_lint_capabilities_set(vl.ptr, caps, &vlErr) < 0 {
		return makeError(&vlErr)
	}

	return nil
}

func (vl *VirtLint) OsShortidSet(shortid string) error {
	var vlErr *C.VirtLintError = nil

	caps := C.CString(shortid)

	if C.virt_lint_os_shortid_set(vl.ptr, caps, &vlErr) < 0 {
		return makeError(&vlErr)
	}

	return nil
}

func (vl *VirtLint) DomainCapabilitiesClear() error {
	var vlErr *C.VirtLintError = nil

	if C.virt_lint_domain_capabilities_clear(vl.ptr, &vlErr) < 0 {
		return makeError(&vlErr)
	}

	return nil
}

func (vl *VirtLint) DomainCapabilitiesAdd(domcapsxml string) error {
	var vlErr *C.VirtLintError = nil

	domcaps := C.CString(domcapsxml)

	if C.virt_lint_domain_capabilities_add(vl.ptr, domcaps, &vlErr) < 0 {
		return makeError(&vlErr)
	}

	return nil
}

func (vl *VirtLint) Validate(xml string, tags []string, error_on_no_connect bool) error {
	var vlErr *C.VirtLintError = nil
	var rc C.int

	domxml := C.CString(xml)

	if len(tags) == 0 {
		rc = C.virt_lint_validate(vl.ptr, domxml, nil, 0,
			C.bool(error_on_no_connect), &vlErr)
	} else {
		ctags := make([](*C.char), len(tags))

		for i := 0; i < len(tags); i++ {
			ctags[i] = C.CString(tags[i])
			defer C.free(unsafe.Pointer(ctags[i]))
		}

		nctags := C.size_t(len(ctags))

		rc = C.virt_lint_validate(vl.ptr, domxml,
			(**C.char)(unsafe.Pointer(&ctags[0])), nctags,
			C.bool(error_on_no_connect), &vlErr)
	}

	if rc < 0 {
		return makeError(&vlErr)
	}

	return nil
}

type WarningDomain int

const (
	DOMAIN WarningDomain = WarningDomain(C.Domain)
	NODE   WarningDomain = WarningDomain(C.Node)
)

func (e WarningDomain) String() string {
	switch e {
	case DOMAIN:
		return "DOMAIN"
	case NODE:
		return "NODE"
	default:
		return fmt.Sprintf("%d", int(e))
	}
}

type WarningLevel int

const (
	ERROR   WarningLevel = WarningLevel(C.Error)
	WARNING WarningLevel = WarningLevel(C.Warning)
	NOTICE  WarningLevel = WarningLevel(C.Notice)
)

func (e WarningLevel) String() string {
	switch e {
	case ERROR:
		return "ERROR"
	case WARNING:
		return "WARNING"
	case NOTICE:
		return "NOTICE"
	default:
		return fmt.Sprintf("%d", int(e))
	}
}

type VirtLintWarning struct {
	Tags   []string
	Domain WarningDomain
	Level  WarningLevel
	Msg    string
}

func (vl *VirtLint) GetWarnings() ([]VirtLintWarning, error) {
	var vlErr *C.VirtLintError = nil
	var cwarnings *C.CVirtLintWarning = nil

	ncwarnings := C.virt_lint_get_warnings(vl.ptr, &cwarnings, &vlErr)
	defer C.virt_lint_warnings_free(&cwarnings, &ncwarnings)

	if ncwarnings < 0 {
		return []VirtLintWarning{}, makeError(&vlErr)
	}

	cwarninges_slice := unsafe.Slice(cwarnings, ncwarnings)

	warnings := make([]VirtLintWarning, ncwarnings)

	for i := 0; i < len(cwarninges_slice); i++ {
		cwarn := cwarninges_slice[i]
		cwarn_tags := unsafe.Slice(cwarn.tags, cwarn.ntags)

		tags := make([]string, cwarn.ntags)
		for j := 0; j < len(cwarn_tags); j++ {
			tags[j] = C.GoString(cwarn_tags[j])
		}

		msg := C.GoString(cwarn.msg)

		warnings[i] = VirtLintWarning{
			Tags:   tags,
			Domain: WarningDomain(cwarn.domain),
			Level:  WarningLevel(cwarn.level),
			Msg:    msg,
		}
	}

	return warnings, nil
}
