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

package compute_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

// These specs exercise the +kubebuilder:validation:XValidation (CEL) rules on the
// firmware types (staging/src/kubevirt.io/api/core/v1/schema.go), asserting them
// directly against the generated CRD schema via a real, in-process API server.
// No kubevirt webhook runs in this environment, so a rejection here can only
// come from the CRD-embedded CEL rule itself.
var _ = Describe("Firmware CEL validation", Ordered, ContinueOnFailure, func() {
	const (
		bootloaderExclusivityMessage    = "has both EFI and BIOS configured, but they are mutually exclusive"
		kernelArgsWithoutKernelMessage  = "kernel arguments cannot be provided without an external kernel"
		kernelBootContainerImageMessage = "kernel boot container must be defined with an image"
		kernelBootContainerPathsMessage = "kernel boot container must be defined with at least one of the following: kernelPath, initrdPath"

		kernelArgs  = "console=ttyS0"
		kernelImage = "kernel-image"
		kernelPath  = "/boot/vmlinuz"
		initrdPath  = "/boot/initrd"
	)

	var (
		f  *framework.Framework
		ns string
	)

	BeforeAll(func() {
		f = framework.New()
		f.Start(context.Background())
	})

	BeforeEach(func(ctx SpecContext) {
		ns = f.CreateNamespace(ctx)
	})

	newVMIWithFirmware := func(firmware *virtv1.Firmware) *virtv1.VirtualMachineInstance {
		vmi := libvmi.New()
		vmi.Spec.Domain.Firmware = firmware
		return vmi
	}

	newBootloaderFirmware := func(bios *virtv1.BIOS, efi *virtv1.EFI) *virtv1.Firmware {
		return &virtv1.Firmware{Bootloader: &virtv1.Bootloader{BIOS: bios, EFI: efi}}
	}

	// The container is only set when at least one of its fields is set.
	newKernelBootFirmware := func(args, image, kernel, initrd string) *virtv1.Firmware {
		kernelBoot := &virtv1.KernelBoot{KernelArgs: args}
		if image != "" || kernel != "" || initrd != "" {
			kernelBoot.Container = &virtv1.KernelBootContainer{Image: image, KernelPath: kernel, InitrdPath: initrd}
		}
		return &virtv1.Firmware{KernelBoot: kernelBoot}
	}

	DescribeTable("should accept a VMI", func(ctx SpecContext, firmware *virtv1.Firmware) {
		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, newVMIWithFirmware(firmware), metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	},
		Entry("with a BIOS bootloader only", newBootloaderFirmware(&virtv1.BIOS{}, nil)),
		Entry("with an EFI bootloader only", newBootloaderFirmware(nil, &virtv1.EFI{})),
		Entry("with an empty kernel boot", newKernelBootFirmware("", "", "", "")),
		Entry("with kernel args and a kernel boot container with image, kernelPath and initrdPath",
			newKernelBootFirmware(kernelArgs, kernelImage, kernelPath, initrdPath)),
		Entry("with a kernel boot container with image and only kernelPath", newKernelBootFirmware("", kernelImage, kernelPath, "")),
		Entry("with a kernel boot container with image and only initrdPath", newKernelBootFirmware("", kernelImage, "", initrdPath)),
	)

	DescribeTable("should reject a VMI", func(ctx SpecContext, firmware *virtv1.Firmware, expectedMessage string) {
		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, newVMIWithFirmware(firmware), metav1.CreateOptions{})
		Expect(err).To(MatchError(ContainSubstring(expectedMessage)))
	},
		Entry("with both BIOS and EFI bootloaders", newBootloaderFirmware(&virtv1.BIOS{}, &virtv1.EFI{}), bootloaderExclusivityMessage),
		Entry("with kernel args but without a kernel boot container",
			newKernelBootFirmware(kernelArgs, "", "", ""), kernelArgsWithoutKernelMessage),
		Entry("with a kernel boot container without an image",
			newKernelBootFirmware("", "", kernelPath, initrdPath), kernelBootContainerImageMessage),
		Entry("with a kernel boot container without kernelPath and initrdPath",
			newKernelBootFirmware("", kernelImage, "", ""), kernelBootContainerPathsMessage),
	)

	DescribeTable("should reject a VirtualMachine template", func(ctx SpecContext, firmware *virtv1.Firmware, expectedMessage string) {
		vm := libvmi.NewVirtualMachine(newVMIWithFirmware(firmware))

		_, err := f.VirtClient().VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).To(MatchError(ContainSubstring(expectedMessage)))
	},
		Entry("with both BIOS and EFI bootloaders", newBootloaderFirmware(&virtv1.BIOS{}, &virtv1.EFI{}), bootloaderExclusivityMessage),
		Entry("with a kernel boot container without an image",
			newKernelBootFirmware("", "", kernelPath, initrdPath), kernelBootContainerImageMessage),
	)
})
