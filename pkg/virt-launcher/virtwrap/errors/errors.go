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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package errors

import (
	"fmt"

	libvirt "github.com/libvirt/libvirt-go"
)

func checkError(err error, expectedError libvirt.ErrorNumber) bool {
	libvirtError, ok := err.(libvirt.Error)
	if ok {
		return libvirtError.Code == expectedError
	}

	return false
}

// IsNotFound detects libvirt's ERR_NO_DOMAIN. It accepts both error and libvirt.Error (as returned by GetLastError function).
func IsNotFound(err error) bool {
	return checkError(err, libvirt.ERR_NO_DOMAIN)
}

// IsInvalidOperation detects libvirt's VIR_ERR_OPERATION_INVALID. It accepts both error and libvirt.Error (as returned by GetLastError function).
func IsInvalidOperation(err error) bool {
	return checkError(err, libvirt.ERR_OPERATION_INVALID)
}

// IsOk detects libvirt's ERR_OK. It accepts both error and libvirt.Error (as returned by GetLastError function).
func IsOk(err error) bool {
	return checkError(err, libvirt.ERR_OK)
}

func FormatLibvirtError(err error) string {
	var libvirtError string
	lerr, ok := err.(libvirt.Error)
	if ok {
		libvirtError = fmt.Sprintf("LibvirtError(Code=%d, Domain=%d, Message='%s')",
			lerr.Code, lerr.Domain, lerr.Message)
	}

	return libvirtError
}
