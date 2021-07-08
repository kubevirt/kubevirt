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

package efi

import (
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("EFI environment detection", func() {
	createEFIRoms := func(efiRoms ...string) string {
		ovmfPath, err := os.MkdirTemp("", "kubevirt-ovmf")
		Expect(err).To(BeNil())

		for i := range efiRoms {
			if efiRoms[i] != "" {
				f, err := os.Create(path.Join(ovmfPath, efiRoms[i]))
				Expect(err).To(BeNil())
				f.Close()
			}
		}
		return ovmfPath
	}

	table.DescribeTable("EFI Roms",
		func(arch, codeSB, varsSB, code, vars string, SBBootable, NoSBBootable bool) {
			ovmfPath := createEFIRoms(codeSB, varsSB, code, vars)
			defer os.RemoveAll(ovmfPath)

			efiEnv := DetectEFIEnvironment(arch, ovmfPath)
			Expect(efiEnv).ToNot(BeNil())

			Expect(efiEnv.Bootable(true)).To(Equal(SBBootable))
			Expect(efiEnv.Bootable(false)).To(Equal(NoSBBootable))

			if SBBootable {
				Expect(efiEnv.EFICode(true)).To(Equal(filepath.Join(ovmfPath, codeSB)))
				Expect(efiEnv.EFIVars(true)).To(Equal(filepath.Join(ovmfPath, varsSB)))
			}
			if NoSBBootable {
				Expect(efiEnv.EFICode(false)).To(Equal(filepath.Join(ovmfPath, code)))
				Expect(efiEnv.EFIVars(false)).To(Equal(filepath.Join(ovmfPath, vars)))
			}

		},
		table.Entry("SB and NoSB available", "x86_64", EFICodeSecureBoot, EFIVarsSecureBoot, EFICode, EFIVars, true, true),
		table.Entry("Only SB available", "x86_64", EFICodeSecureBoot, EFIVarsSecureBoot, EFICodeSecureBoot, "", true, false),
		table.Entry("Only NoSB available", "x86_64", "", "", EFICode, EFIVars, false, true),
		table.Entry("Arm64 EFI", "arm64", "", "", EFICodeAARCH64, EFIVarsAARCH64, false, true),
		table.Entry("SB and NoSB available when OVMF_CODE.fd does not exist", "x86_64", EFICodeSecureBoot, EFIVarsSecureBoot, EFICodeSecureBoot, EFIVars, true, true),
		table.Entry("Only NoSB available when OVMF_CODE.fd and OVMF_VARS.secboot.fd do not exist", "x86_64", EFICodeSecureBoot, "", EFICodeSecureBoot, EFIVars, false, true),
		table.Entry("EFI booting not available for x86_64", "x86_64", "", "", "", "", false, false),
		table.Entry("EFI booting not available for arm64", "arm64", "", "", "", "", false, false),
	)
})
