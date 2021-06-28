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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package efi

import (
	"os"
	"path/filepath"
)

const (
	EFICode           = "OVMF_CODE.fd"
	EFIVars           = "OVMF_VARS.fd"
	EFICodeAARCH64    = "AAVMF_CODE.fd"
	EFIVarsAARCH64    = "AAVMF_VARS.fd"
	EFICodeSecureBoot = "OVMF_CODE.secboot.fd"
	EFIVarsSecureBoot = "OVMF_VARS.secboot.fd"
)

type EFIEnvironment struct {
	code           string
	vars           string
	codeSecureBoot string
	varsSecureBoot string
}

func (e *EFIEnvironment) Bootable(secureBoot bool) bool {
	if secureBoot {
		return e.varsSecureBoot != "" && e.codeSecureBoot != ""
	} else {
		return e.vars != "" && e.code != ""
	}
}

func (e *EFIEnvironment) EFICode(secureBoot bool) string {
	if secureBoot {
		return e.codeSecureBoot
	} else {
		return e.code
	}
}

func (e *EFIEnvironment) EFIVars(secureBoot bool) string {
	if secureBoot {
		return e.varsSecureBoot
	} else {
		return e.vars
	}
}

func DetectEFIEnvironment(arch, ovmfPath string) *EFIEnvironment {
	if arch == "arm64" {
		var codeArm64, varsArm64 string

		_, err := os.Stat(filepath.Join(ovmfPath, EFICodeAARCH64))
		if err == nil {
			codeArm64 = filepath.Join(ovmfPath, EFICodeAARCH64)
		}
		_, err = os.Stat(filepath.Join(ovmfPath, EFIVarsAARCH64))
		if err == nil {
			varsArm64 = filepath.Join(ovmfPath, EFIVarsAARCH64)
		}

		return &EFIEnvironment{
			code: codeArm64,
			vars: varsArm64,
		}
	}

	// detect EFI with SecureBoot
	var codeWithSB, varsWithSB string
	_, err := os.Stat(filepath.Join(ovmfPath, EFICodeSecureBoot))
	if err == nil {
		codeWithSB = filepath.Join(ovmfPath, EFICodeSecureBoot)
	}
	_, err = os.Stat(filepath.Join(ovmfPath, EFIVarsSecureBoot))
	if err == nil {
		varsWithSB = filepath.Join(ovmfPath, EFIVarsSecureBoot)
	}

	// detect EFI without SecureBoot
	var code, vars string
	_, err = os.Stat(filepath.Join(ovmfPath, EFICode))
	if err == nil {
		code = filepath.Join(ovmfPath, EFICode)
	} else {
		// The combination (EFICodeSecureBoot + EFIVars) is valid
		// for booting in EFI mode with SecureBoot disabled
		code = codeWithSB
	}
	_, err = os.Stat(filepath.Join(ovmfPath, EFIVars))
	if err == nil {
		vars = filepath.Join(ovmfPath, EFIVars)
	}

	return &EFIEnvironment{
		codeSecureBoot: codeWithSB,
		varsSecureBoot: varsWithSB,
		code:           code,
		vars:           vars,
	}
}
