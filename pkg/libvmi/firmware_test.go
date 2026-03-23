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

package libvmi

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("WithUefi and WithArchitecture order independence",
	func(opts []Option, expectSMM bool) {
		vmi := New(opts...)
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).ToNot(BeNil())
		Expect(*vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(BeTrue())
		if expectSMM {
			Expect(vmi.Spec.Domain.Features).ToNot(BeNil())
			Expect(vmi.Spec.Domain.Features.SMM).ToNot(BeNil())
			Expect(*vmi.Spec.Domain.Features.SMM.Enabled).To(BeTrue())
		} else {
			Expect(vmi.Spec.Domain.Features == nil || vmi.Spec.Domain.Features.SMM == nil).To(BeTrue())
		}
	},
	Entry("arm64 arch set before WithUefi: no SMM",
		[]Option{WithArchitecture("arm64"), WithUefi(true)}, false),
	Entry("arm64 arch set after WithUefi: no SMM",
		[]Option{WithUefi(true), WithArchitecture("arm64")}, false),
	Entry("amd64 arch set before WithUefi: SMM enabled",
		[]Option{WithArchitecture("amd64"), WithUefi(true)}, true),
	Entry("amd64 arch set after WithUefi: SMM enabled",
		[]Option{WithUefi(true), WithArchitecture("amd64")}, true),
)
