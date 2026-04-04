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
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("[sig-compute]Intel TDX", decorators.TDX, decorators.SigCompute, decorators.RequiresAMD64, Serial, func() {
	newTDXFedora := func() *v1.VirtualMachineInstance {
		const secureBoot = false
		tdxOptions := []libvmi.Option{
			libvmi.WithTDX(),
			libvmi.WithUefi(secureBoot),
			libvmi.WithCPUModel("host-passthrough"),
		}
		return libvmifact.NewFedora(tdxOptions...)
	}

	Context("lifecycle", func() {
		var (
			virtClient kubecli.KubevirtClient
		)

		BeforeEach(func() {
			virtClient = kubevirt.Client()
			nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			hasTdxNode := false
			for _, node := range nodes.Items {
				for label := range node.Labels {
					if strings.Contains(label, v1.TDXLabel) {
						hasTdxNode = true
						break
					}
				}
				if hasTdxNode {
					break
				}
			}

			if !hasTdxNode {
				Skip("Skipped TDX test: No TDX capable nodes found.")
			}

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
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Verifying that TDX is enabled in the guest os")
			err := console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "sudo dmesg | grep --color=never tdx\n"},
				&expect.BExp{R: "tdx: Guest detected"},
				&expect.BSnd{S: "grep tdx_guest /proc/cpuinfo\n"},
				&expect.BExp{R: "tdx_guest"},
			}, 30)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
