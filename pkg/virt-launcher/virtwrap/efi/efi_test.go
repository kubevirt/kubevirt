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
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EFI environment detection", func() {
	const (
		secureBootEnabled = true
	)

	createEFIRoms := func(efiRoms ...string) string {
		ovmfPath, err := os.MkdirTemp("", "kubevirt-ovmf")
		Expect(err).ToNot(HaveOccurred())

		for i := range efiRoms {
			if efiRoms[i] != "" {
				f, err := os.Create(path.Join(ovmfPath, efiRoms[i]))
				Expect(err).ToNot(HaveOccurred())
				f.Close()
			}
		}
		return ovmfPath
	}

	DescribeTable("EFI Roms",
		func(arch, codeSB, varsSB, code, vars string, SBBootable, NoSBBootable bool) {
			ovmfPath := createEFIRoms(codeSB, varsSB, code, vars)
			defer os.RemoveAll(ovmfPath)

			efiEnv := DetectEFIEnvironment(arch, ovmfPath)
			Expect(efiEnv).ToNot(BeNil())

			Expect(efiEnv.Bootable(secureBootEnabled, None)).To(Equal(SBBootable))
			Expect(efiEnv.Bootable(secureBootEnabled, SEV)).To(BeFalse())
			Expect(efiEnv.Bootable(secureBootEnabled, SNP)).To(BeFalse())
			Expect(efiEnv.Bootable(!secureBootEnabled, None)).To(Equal(NoSBBootable))

			if SBBootable {
				Expect(efiEnv.EFICode(secureBootEnabled, None)).To(Equal(filepath.Join(ovmfPath, codeSB)))
				Expect(efiEnv.EFICode(secureBootEnabled, SEV)).ToNot(Equal(filepath.Join(ovmfPath, codeSB)))
				Expect(efiEnv.EFICode(secureBootEnabled, SNP)).ToNot(Equal(filepath.Join(ovmfPath, codeSB)))
				Expect(efiEnv.EFIVars(secureBootEnabled, None)).To(Equal(filepath.Join(ovmfPath, varsSB)))
			}
			if NoSBBootable {
				Expect(efiEnv.EFICode(!secureBootEnabled, None)).To(Equal(filepath.Join(ovmfPath, code)))
				Expect(efiEnv.EFIVars(!secureBootEnabled, None)).To(Equal(filepath.Join(ovmfPath, vars)))
			}
		},
		Entry("SB and NoSB available", "x86_64", EFICodeSecureBoot, EFIVarsSecureBoot, EFICode, EFIVars, true, true),
		Entry("Only SB available", "x86_64", EFICodeSecureBoot, EFIVarsSecureBoot, EFICodeSecureBoot, "", true, false),
		Entry("Only NoSB available", "x86_64", "", "", EFICode, EFIVars, false, true),
		Entry("Arm64 EFI", "arm64", "", "", EFICodeAARCH64, EFIVarsAARCH64, false, true),
		Entry("SB and NoSB available when OVMF_CODE.fd does not exist", "x86_64", EFICodeSecureBoot, EFIVarsSecureBoot, EFICodeSecureBoot, EFIVars, true, true),
		Entry("Only NoSB available when OVMF_CODE.fd and OVMF_VARS.secboot.fd do not exist", "x86_64", EFICodeSecureBoot, "", EFICodeSecureBoot, EFIVars, false, true),
		Entry("EFI booting not available for x86_64", "x86_64", "", "", "", "", false, false),
		Entry("EFI booting not available for arm64", "arm64", "", "", "", "", false, false),
	)

	It("SEV and SEV-SNP EFI Roms", func() {
		ovmfPath := createEFIRoms(EFICodeSEV, EFIVars, EFICodeSNP)
		defer os.RemoveAll(ovmfPath)

		efiEnv := DetectEFIEnvironment("x86_64", ovmfPath)
		Expect(efiEnv).ToNot(BeNil())

		Expect(efiEnv.Bootable(secureBootEnabled, None)).To(BeFalse())
		Expect(efiEnv.Bootable(secureBootEnabled, SEV)).To(BeFalse())
		Expect(efiEnv.Bootable(secureBootEnabled, SNP)).To(BeFalse())
		Expect(efiEnv.Bootable(!secureBootEnabled, SEV)).To(BeTrue())
		Expect(efiEnv.Bootable(!secureBootEnabled, SNP)).To(BeTrue())
		Expect(efiEnv.Bootable(!secureBootEnabled, None)).To(BeFalse())

		codeSEV := filepath.Join(ovmfPath, EFICodeSEV)
		Expect(efiEnv.EFICode(secureBootEnabled, SEV)).ToNot(Equal(codeSEV))
		Expect(efiEnv.EFICode(secureBootEnabled, None)).ToNot(Equal(codeSEV))
		Expect(efiEnv.EFICode(!secureBootEnabled, SEV)).To(Equal(codeSEV))
		Expect(efiEnv.EFICode(!secureBootEnabled, None)).ToNot(Equal(codeSEV))

		codeSNP := filepath.Join(ovmfPath, EFICodeSNP)
		Expect(efiEnv.EFICode(secureBootEnabled, SNP)).ToNot(Equal(codeSNP))
		Expect(efiEnv.EFICode(secureBootEnabled, None)).ToNot(Equal(codeSNP))
		Expect(efiEnv.EFICode(!secureBootEnabled, SNP)).To(Equal(codeSNP))
		Expect(efiEnv.EFICode(!secureBootEnabled, None)).ToNot(Equal(codeSNP))

		varsSEV := filepath.Join(ovmfPath, EFIVars)
		Expect(efiEnv.EFIVars(!secureBootEnabled, SEV)).To(Equal(varsSEV))
		Expect(efiEnv.EFIVars(!secureBootEnabled, SNP)).To(Equal(""))
		Expect(efiEnv.EFIVars(!secureBootEnabled, None)).To(Equal(varsSEV))
	})

	It("TDX EFI Roms", func() {
		ovmfPath := createEFIRoms(EFICodeTDX, EFICodeTDXSecureBoot)
		defer os.RemoveAll(ovmfPath)

		efiEnv := DetectEFIEnvironment("x86_64", ovmfPath)
		Expect(efiEnv).ToNot(BeNil())

		Expect(efiEnv.Bootable(secureBootEnabled, TDX)).To(BeTrue())
		Expect(efiEnv.Bootable(!secureBootEnabled, TDX)).To(BeTrue())
		Expect(efiEnv.Bootable(secureBootEnabled, None)).To(BeFalse())
		Expect(efiEnv.Bootable(!secureBootEnabled, None)).To(BeFalse())

		codeTDX := filepath.Join(ovmfPath, EFICodeTDX)
		codeTDXSB := filepath.Join(ovmfPath, EFICodeTDXSecureBoot)
		Expect(efiEnv.EFICode(secureBootEnabled, TDX)).To(Equal(codeTDXSB))
		Expect(efiEnv.EFICode(!secureBootEnabled, TDX)).To(Equal(codeTDX))
		Expect(efiEnv.EFICode(secureBootEnabled, None)).ToNot(Equal(codeTDX))
		Expect(efiEnv.EFICode(!secureBootEnabled, None)).ToNot(Equal(codeTDX))
		Expect(efiEnv.EFICode(secureBootEnabled, None)).ToNot(Equal(codeTDXSB))
		Expect(efiEnv.EFICode(!secureBootEnabled, None)).ToNot(Equal(codeTDXSB))

		// vars should always be ""
		Expect(efiEnv.EFIVars(secureBootEnabled, TDX)).To(Equal(""))
		Expect(efiEnv.EFIVars(!secureBootEnabled, TDX)).To(Equal(""))
	})
})
