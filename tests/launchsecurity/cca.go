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
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("[sig-compute]ARM CCA", decorators.CCA, decorators.SigCompute, decorators.RequiresARM64, Serial, func() {
	newCCAFedora := func() *v1.VirtualMachineInstance {
		const secureBoot = false
		ccaOptions := []libvmi.Option{
			libvmi.WithCCA(),
			libvmi.WithUefi(secureBoot),
			libvmi.WithCPUModel("host-passthrough"),
		}
		return libvmifact.NewFedora(ccaOptions...)
	}

	Context("lifecycle", func() {
		var (
			virtClient kubecli.KubevirtClient
		)

		BeforeEach(func() {
			virtClient = kubevirt.Client()
			config.EnableFeatureGate(featuregate.WorkloadEncryptionCCA)
		})

		It("should verify CCA guest execution with proper logging", func() {
			By("Checking if we have a valid virt client")
			Expect(virtClient).ToNot(BeNil())

			By("Creating VMI definition")
			vmi := newCCAFedora()
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsXHuge)

			By("Querying virsh nodeccainfo")
			nodeCCAInfo := libpod.RunCommandOnVmiPod(vmi, []string{"virsh", "domcapabilities"})
			Expect(nodeCCAInfo).ToNot(BeEmpty())
			Expect(nodeCCAInfo).To(ContainSubstring("<cca supported='yes'/>"))
			Expect(nodeCCAInfo).To(ContainSubstring("measurement-algo"))

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Verifying that CCA is enabled in the guest os")
			err := console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "dmesg | grep RME\n"},
				&expect.BExp{R: "RME: Using RSI version"},
			}, 60)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
