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
	EFICode           = "OVMF_CODE.fd"
	EFIVars           = "OVMF_VARS.fd"
	EFICodeAARCH64    = "AAVMF_CODE.fd"
	EFIVarsAARCH64    = "AAVMF_VARS.fd"
	EFICodeSecureBoot = "OVMF_CODE.secboot.fd"
	EFIVarsSecureBoot = "OVMF_VARS.secboot.fd"
	EFICodeSEV        = "OVMF_CODE.cc.fd"
	EFIVarsSEV        = EFIVars
	EFICodeSNP        = "OVMF.amdsev.fd"
)

type EFIEnvironment struct {
	code           string
	vars           string
	codeSecureBoot string
	varsSecureBoot string
	codeSEV        string
	varsSEV        string
	codeSNP        string
}

func (e *EFIEnvironment) Bootable(secureBoot, sev bool, snp bool) bool {
	if secureBoot {
		return e.varsSecureBoot != "" && e.codeSecureBoot != ""
	} else if snp {
		// SEV-SNP only
		return e.codeSNP != ""
	} else if sev {
		// SEV and SEV-ES
		return e.varsSEV != "" && e.codeSEV != ""
	} else {
		return e.vars != "" && e.code != ""
	}
}

func (e *EFIEnvironment) EFICode(secureBoot, sev bool, snp bool) string {
	if secureBoot {
		return e.codeSecureBoot
	} else if snp {
		// SEV-SNP only
		return e.codeSNP
	} else if sev {
		// SEV and SEV-ES
		return e.codeSEV
	} else {
		return e.code
	}
}

func (e *EFIEnvironment) EFIVars(secureBoot, sev bool, snp bool) string {
	if secureBoot {
		return e.varsSecureBoot
	} else if snp {
		return ""
	} else if sev {
		return e.varsSEV
	} else {
		return e.vars
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

	return &EFIEnvironment{
		codeSecureBoot: codeWithSB,
		varsSecureBoot: varsWithSB,
		code:           code,
		vars:           vars,
		codeSEV:        codeWithSEV,
		varsSEV:        varsWithSEV,
		codeSNP:        codeWithSNP,
	}
}

func getEFIBinaryIfExists(path, binary string) string {
	fullPath := filepath.Join(path, binary)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath
	}
	return ""
}
