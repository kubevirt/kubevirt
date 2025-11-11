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
 * Copyright The KubeVirt Authors
 *
 */

package launchsecurity

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libvmops"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("[sig-compute]Intel TDX", decorators.TDX, decorators.SigCompute, decorators.RequiresAMD64, Serial, func() {
	// I could use embedded data but it's not worth the effort at the moment
	const cloudInitUserData = "#cloud-config\n" +
		"user: fedora\n" +
		"password: fedora\n" +
		"chpasswd: { expire: False }\n" +
		"ssh_pwauth: true\n"

	newTDXFedora := func() *v1.VirtualMachineInstance {
		// Configure libvirt to use TDX
		// Use fedora image with support for TDX
		// NewFedora() uses fedora39 which doesn't support TDX yet
		tdxOptions := []libvmi.Option{
			libvmi.WithTDX(),
			libvmi.WithMemoryRequest("512Mi"),
			libvmi.WithContainerDisk("rootdisk1", "quay.io/containerdisks/fedora"),
			libvmi.WithUefi(false),
			libvmi.WithCPUModel("host-passthrough"),
			// Set the credentials for LoginToFedora()
			libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData(cloudInitUserData)),
		}

		vmi := libvmi.New(tdxOptions...)
		return vmi
	}

	Context("lifecycle", func() {
		var (
			virtClient kubecli.KubevirtClient
		)

		BeforeEach(func() {
			virtClient = kubevirt.Client()
			// Enable the TDX feature gate if it's not enabled
			// This should not be executed in parallel
			config.EnableFeatureGate(featuregate.WorkloadEncryptionTDX)
		})

		It("should verify TDX guest execution with proper logging", func() {
			By("Checking if we have a valid virt client")
			Expect(virtClient).ToNot(BeNil())

			By("Creating VMI definition")
			vmi := newTDXFedora()
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsXHuge)

			By("Expecting the VirtualMachineInstance console")
			// Login is sucessful, but LoginToFedora is failing due to the shell integration codes
			// appear before and after the return code, which cannot be matched by the regexp,
			// ignore the result of the login for the moment
			console.LoginToFedora(vmi)

			By("Verifying that TDX is enabled in the guest")
			err := console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "sudo dmesg | grep --color=never tdx\n"},
				&expect.BExp{R: "tdx: Guest detected"},
				&expect.BSnd{S: "ls /dev/tdx*\n"},
				&expect.BExp{R: "/dev/tdx_guest"},
			}, 30)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
