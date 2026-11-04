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
	"github.com/onsi/gomega/types"
)

var _ = DescribeTable("WithUefi configures Secure Boot for the selected architecture",
	func(opts []Option, secureBoot bool, expectedFeatures types.GomegaMatcher) {
		vmi := New(opts...)
		Expect(vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot).To(HaveValue(Equal(secureBoot)))
		Expect(vmi.Spec.Domain.Features).To(expectedFeatures)
	},
	Entry("ARM64 Secure Boot does not enable SMM",
		[]Option{WithArchitecture("arm64"), WithUefi(true)}, true, Or(BeNil(), HaveField("SMM", BeNil()))),
	Entry("amd64 Secure Boot enables SMM",
		[]Option{WithArchitecture("amd64"), WithUefi(true)}, true, HaveField("SMM.Enabled", HaveValue(BeTrue()))),
	Entry("Secure Boot without an explicit architecture enables SMM",
		[]Option{WithUefi(true)}, true, HaveField("SMM.Enabled", HaveValue(BeTrue()))),
	Entry("ARM64 EFI without Secure Boot does not enable SMM",
		[]Option{WithArchitecture("arm64"), WithUefi(false)}, false, Or(BeNil(), HaveField("SMM", BeNil()))),
)
