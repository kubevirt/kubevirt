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
	EFICodeAARCH64CCA = "AAVMF_CODE.cca.fd"
	EFIVarsAARCH64CCA = "AAVMF_VARS.cca.fd"
)

type EFIEnvironment struct {
	code           string
	vars           string
	codeSecureBoot string
	varsSecureBoot string
	codeSEV        string
	varsSEV        string
	codeCCA        string
	varsCCA        string
}

func (e *EFIEnvironment) Bootable(secureBoot, sev bool, cca bool) bool {
	if secureBoot {
		return e.varsSecureBoot != "" && e.codeSecureBoot != ""
	} else if sev {
		return e.varsSEV != "" && e.codeSEV != ""
	} else if cca {
		return e.varsCCA != "" && e.codeCCA != ""
	} else {
		return e.vars != "" && e.code != ""
	}
}

func (e *EFIEnvironment) EFICode(secureBoot, sev bool, cca bool) string {
	if secureBoot {
		return e.codeSecureBoot
	} else if sev {
		return e.codeSEV
	} else if cca {
		return e.codeCCA
	} else {
		return e.code
	}
}

func (e *EFIEnvironment) EFIVars(secureBoot, sev bool, cca bool) string {
	if secureBoot {
		return e.varsSecureBoot
	} else if sev {
		return e.varsSEV
	} else if cca {
		return e.varsCCA
	} else {
		return e.vars
	}
}

func DetectEFIEnvironment(arch, ovmfPath string) *EFIEnvironment {
	if arch == "arm64" {
		codeArm64 := getEFIBinaryIfExists(ovmfPath, EFICodeAARCH64)
		varsArm64 := getEFIBinaryIfExists(ovmfPath, EFIVarsAARCH64)

		codeArm64WithCCA := getEFIBinaryIfExists(ovmfPath, EFICodeAARCH64CCA)
		varsArm64WithCCA := getEFIBinaryIfExists(ovmfPath, EFIVarsAARCH64CCA)

		return &EFIEnvironment{
			code:    codeArm64,
			vars:    varsArm64,
			codeCCA: codeArm64WithCCA,
			varsCCA: varsArm64WithCCA,
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

	return &EFIEnvironment{
		codeSecureBoot: codeWithSB,
		varsSecureBoot: varsWithSB,
		code:           code,
		vars:           vars,
		codeSEV:        codeWithSEV,
		varsSEV:        varsWithSEV,
	}
}

func getEFIBinaryIfExists(path, binary string) string {
	fullPath := filepath.Join(path, binary)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath
	}
	return ""
}
