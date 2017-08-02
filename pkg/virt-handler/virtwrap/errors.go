/*
 * This file is part of the kubevirt project
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

package virtwrap

import (
	"reflect"

	"github.com/libvirt/libvirt-go"
)

func checkError(err interface{}, expectedError libvirt.ErrorNumber) bool {
	return err.(libvirt.Error).Code == expectedError
}

// IsNotFound detects libvirt's ERR_NO_DOMAIN. It accepts both error and libvirt.Error (as returned by GetLastError function).
func IsNotFound(err interface{}) bool {
	return checkError(err, libvirt.ERR_NO_DOMAIN)
}

// IsOk detects libvirt's ERR_OK. It accepts both error and libvirt.Error (as returned by GetLastError function).
func IsOk(err interface{}) bool {
	return checkError(err, libvirt.ERR_OK)
}
