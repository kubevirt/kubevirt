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

package tests_test

import (
	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = FDescribe("[sig-compute]IOMMU", decorators.WgArm64, decorators.SigCompute, func() {
	Context("SMMUv3 on ARM64", func() {
		It("should expose an SMMUv3 IOMMU device", func() {
			By("Creating a VMI with IOMMU enabled")
			vmi := libvmifact.NewFedora(
				libvmi.WithIOMMU(&v1.IOMMUDevice{
					Model: "smmuv3",
				}),
			)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsMedium)

			By("Verifying the libvirt domain XML contains the IOMMU device")
			domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domSpec.Devices.IOMMU).ToNot(BeNil())
			Expect(domSpec.Devices.IOMMU.Model).To(Equal("smmuv3"))
		})

		It("should expose an SMMUv3 IOMMU device with default model", func() {
			By("Creating a VMI with IOMMU enabled without explicit model")
			vmi := libvmifact.NewFedora(
				libvmi.WithIOMMU(&v1.IOMMUDevice{}),
			)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsMedium)

			By("Verifying the libvirt domain XML contains the IOMMU device with smmuv3 model")
			domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domSpec.Devices.IOMMU).ToNot(BeNil())
			Expect(domSpec.Devices.IOMMU.Model).To(Equal("smmuv3"))
		})

		It("should not have arm-smmu-v3 messages in dmesg when IOMMU is not enabled", func() {
			By("Creating a VMI without IOMMU enabled")
			vmi := libvmifact.NewFedora()
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsMedium)

			By("Logging into the guest")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Verifying no arm-smmu-v3 messages appear in dmesg")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "dmesg | grep -c arm-smmu-v3 || true\n"},
				&expect.BExp{R: console.RetValue("0")},
			}, 30)).To(Succeed(), "should not find arm-smmu-v3 messages in dmesg")

			By("Verifying the libvirt domain XML does not contain an IOMMU device")
			domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domSpec.Devices.IOMMU).To(BeNil())
		})
	})
})
