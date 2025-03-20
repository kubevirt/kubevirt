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

package subresources

import (
	"context"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(compute.SIG("Reset subresource", func() {

	Describe("Reset a VirtualMachineInstance", func() {
		const vmiLaunchTimeout = 360

		It("should succeed", func() {
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewAlpineWithTestTooling(), vmiLaunchTimeout)
			oldUID := vmi.UID

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			By("Create a file that is not expected to survive the reset request")
			err := console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "touch /tmp/non-persistent-file\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: console.EchoLastReturnValue},
				&expect.BExp{R: console.ShellSuccess},
			}, 120)
			Expect(err).ToNot(HaveOccurred())

			err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Reset(context.Background(), vmi.Name)
			Expect(err).ToNot(HaveOccurred())

			Expect(console.LoginToAlpine(vmi)).To(Succeed())
			err = console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "ls /tmp/non-persistent-file\n"},
				&expect.BExp{R: `non-persistent-file: No such file or director`},
			}, 20)
			Expect(err).ToNot(HaveOccurred())

			vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.UID).To(Equal(oldUID))
		})
	})
}))
