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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tools/vms-generator/utils"
)

var _ = Describe("[sig-compute]VMI with external kernel boot", func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		tests.BeforeTestCleanup()
	})

	Context("with external alpine-based kernel & initrd images", func() {
		It("ensure successful boot", func() {
			vmi := utils.GetVMIKernelBoot()
			obj, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(obj)
		})
	})

	Context("with illegal definition ensure rejection of", func() {

		It("VMI defined without an image", func() {
			vmi := utils.GetVMIKernelBoot()
			kernelBoot := vmi.Spec.Domain.Firmware.KernelBoot
			kernelBoot.Container.Image = ""
			_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("denied the request: spec.domain.firmware.kernelBoot.container must be defined with an image"))
		})

		It("VMI defined with image but without initrd & kernel paths", func() {
			vmi := utils.GetVMIKernelBoot()
			kernelBoot := vmi.Spec.Domain.Firmware.KernelBoot
			kernelBoot.Container.KernelPath = ""
			kernelBoot.Container.InitrdPath = ""
			_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("denied the request: spec.domain.firmware.kernelBoot.container must be defined with at least one of the following: kernelPath, initrdPath"))
		})
	})
})
