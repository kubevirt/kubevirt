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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe("[sig-compute]Cross-architecture software emulation", Serial, decorators.SigCompute, func() {

	BeforeEach(func() {
		config.EnableFeatureGate(featuregate.CrossArchitectureVirtualization)
	})

	DescribeTable("should boot a guest using QEMU TCG emulation on a cross-architecture host",
		func(guestArch, expectedUnameArch, containerDiskImage, expectedEmulator string) {
			vmi := libvmi.New(
				libvmi.WithArchitecture(guestArch),
				libvmi.WithContainerDiskAndPullPolicy("disk0", containerDiskImage, k8sv1.PullIfNotPresent),
				libvmi.WithMemoryRequest("1Gi"),
				libvmi.WithRng(),
			)

			By("Creating a VMI with " + guestArch + " architecture on a cross-architecture host")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsXHuge)

			By("Verifying the libvirt domain uses QEMU TCG emulation with the correct emulator")
			domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domSpec.Type).To(Equal("qemu"))
			Expect(domSpec.Devices.Emulator).To(Equal(expectedEmulator))

			By("Verifying the VMI has the SoftwareEmulation condition")
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceSoftwareEmulation))

			By("Verifying the virt-launcher pod has the cross-arch emulation flag")
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

			for _, container := range pod.Spec.Containers {
				if container.Name == "compute" {
					Expect(container.Command).To(ContainElement("--allow-cross-arch-emulation"))
					break
				}
			}

			By("Logging into the guest and verifying the architecture via uname -m")
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			output, err := console.RunCommandAndStoreOutput(vmi, "uname -m", 30*time.Second)
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal(expectedUnameArch))
		},
		Entry("arm64 guest on amd64 host",
			"arm64",
			"aarch64",
			"quay.io/containerdisks/fedora@sha256:eb87a994833d82feb2279575aafa42a06c5feb124ea72cae6efc0ee46bb729d8",
			"/usr/bin/qemu-system-aarch64",
			decorators.RequiresAMD64,
		),
		Entry("amd64 guest on arm64 host",
			"amd64",
			"x86_64",
			cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling),
			"/usr/bin/qemu-system-x86_64",
			decorators.RequiresARM64,
		),
	)
})
