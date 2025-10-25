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
 * Copyright The KubeVirt Authors.
 *
 */

package efi

import (
	"os"
	"path/filepath"
)

const (
	EFICode              = "OVMF_CODE.fd"
	EFIVars              = "OVMF_VARS.fd"
	EFICodeAARCH64       = "AAVMF_CODE.fd"
	EFIVarsAARCH64       = "AAVMF_VARS.fd"
	EFICodeSecureBoot    = "OVMF_CODE.secboot.fd"
	EFIVarsSecureBoot    = "OVMF_VARS.secboot.fd"
	EFICodeSEV           = "OVMF_CODE.cc.fd"
	EFIVarsSEV           = EFIVars
	EFICodeSNP           = "OVMF.amdsev.fd"
	EFICodeTDX           = "OVMF.inteltdx.fd"
	EFICodeTDXSecureBoot = "OVMF.inteltdx.secboot.fd"
)

type EFIEnvironment struct {
	code              string
	vars              string
	codeSecureBoot    string
	varsSecureBoot    string
	codeSEV           string
	varsSEV           string
	codeSNP           string
	codeTDX           string
	codeTDXSecureBoot string
}

type SecureVMType int

const (
	None SecureVMType = iota // Regular VM without confidential computing
	SEV                      // AMD SEV/SEV-ES VM
	SNP                      // AMD SNP VM
	TDX                      // Intel TDX VM
)

func (e *EFIEnvironment) Bootable(secureBoot bool, vmType SecureVMType) bool {
	switch vmType {
	case SEV:
		if secureBoot {
			// secure boot cannot work with SEV
			return false
		} else {
			return e.varsSEV != "" && e.codeSEV != ""
		}
	case SNP:
		if secureBoot {
			// secure boot not compatible with SNP
			return false
		} else {
			return e.codeSNP != ""
		}
	case TDX:
		if secureBoot {
			return e.codeTDXSecureBoot != ""
		} else {
			return e.codeTDX != ""
		}
	default:
		if secureBoot {
			return e.varsSecureBoot != "" && e.codeSecureBoot != ""
		} else {
			return e.vars != "" && e.code != ""
		}
	}
}

func (e *EFIEnvironment) EFICode(secureBoot bool, vmType SecureVMType) string {
	switch vmType {
	case SEV:
		if secureBoot {
			// secure boot cannot work with SEV
			return ""
		} else {
			return e.codeSEV
		}
	case SNP:
		if secureBoot {
			// secure boot cannot work with SNP
			return ""
		} else {
			return e.codeSNP
		}
	case TDX:
		if secureBoot {
			return e.codeTDXSecureBoot
		} else {
			return e.codeTDX
		}
	default:
		if secureBoot {
			return e.codeSecureBoot
		} else {
			return e.code
		}
	}
}

func (e *EFIEnvironment) EFIVars(secureBoot bool, vmType SecureVMType) string {
	switch vmType {
	case SEV:
		if secureBoot {
			// secure boot cannot work with SEV
			return ""
		} else {
			return e.varsSEV
		}
	case SNP, TDX:
		// Both SNP and TDX use stateless firmware
		return ""
	default:
		if secureBoot {
			return e.varsSecureBoot
		} else {
			return e.vars
		}
	}
}

func DetectEFIEnvironment(arch, ovmfPath string) *EFIEnvironment {
	if arch == "arm64" {
		codeArm64 := getEFIBinaryIfExists(ovmfPath, EFICodeAARCH64)
		varsArm64 := getEFIBinaryIfExists(ovmfPath, EFIVarsAARCH64)

		return &EFIEnvironment{
			code: codeArm64,
			vars: varsArm64,
		}
	}

	// detect EFI with SecureBoot
	codeWithSB := getEFIBinaryIfExists(ovmfPath, EFICodeSecureBoot)
	varsWithSB := getEFIBinaryIfExists(ovmfPath, EFIVarsSecureBoot)

	// detect EFI without SecureBoot
	code := getEFIBinaryIfExists(ovmfPath, EFICode)
	vars := getEFIBinaryIfExists(ovmfPath, EFIVars)
	if code == "" {
		// The combination (EFICodeSecureBoot + EFIVars) is valid
		// for booting in EFI mode with SecureBoot disabled
		code = codeWithSB
	}

	// detect EFI with SEV
	codeWithSEV := getEFIBinaryIfExists(ovmfPath, EFICodeSEV)
	varsWithSEV := getEFIBinaryIfExists(ovmfPath, EFIVarsSEV)
	codeWithSNP := getEFIBinaryIfExists(ovmfPath, EFICodeSNP)

	// detect EFI with TDX
	codeWithTDX := getEFIBinaryIfExists(ovmfPath, EFICodeTDX)
	codeWithTDXSB := getEFIBinaryIfExists(ovmfPath, EFICodeTDXSecureBoot)

	return &EFIEnvironment{
		codeSecureBoot:    codeWithSB,
		varsSecureBoot:    varsWithSB,
		code:              code,
		vars:              vars,
		codeSEV:           codeWithSEV,
		varsSEV:           varsWithSEV,
		codeSNP:           codeWithSNP,
		codeTDX:           codeWithTDX,
		codeTDXSecureBoot: codeWithTDXSB,
	}
}

func getEFIBinaryIfExists(path, binary string) string {
	fullPath := filepath.Join(path, binary)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath
	}
	return ""
}
