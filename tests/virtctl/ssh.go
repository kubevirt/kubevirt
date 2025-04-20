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

package virtctl

import (
	"context"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libssh"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("[sig-compute]SSH", decorators.SigCompute, Ordered, decorators.OncePerOrderedCleanup, func() {
	var (
		keyFile string
		vmi     *v1.VirtualMachineInstance
	)

	cmdNative := func(vmiName string) {
		Expect(newRepeatableVirtctlCommand(
			"ssh",
			"--local-ssh=false",
			"--namespace", testsuite.GetTestNamespace(nil),
			"--username", "root",
			"--identity-file", keyFile,
			"--known-hosts=",
			"--command", "true",
			"vmi/"+vmiName,
		)()).To(Succeed())
	}

	cmdLocal := func(appendLocalSSH bool) func(vmiName string) {
		return func(vmiName string) {
			args := []string{
				"ssh",
				"--namespace", testsuite.GetTestNamespace(nil),
				"--username", "root",
				"--identity-file", keyFile,
				"-t", "-o StrictHostKeyChecking=no",
				"-t", "-o UserKnownHostsFile=/dev/null",
				"--command", "true",
			}
			if appendLocalSSH {
				args = append(args, "--local-ssh=true")
			}
			args = append(args, "vmi/"+vmiName)

			// The virtctl binary needs to run here because of the way local SSH client wrapping works.
			// Running the command through newRepeatableVirtctlCommand does not suffice.
			_, cmd, err := clientcmd.CreateCommandWithNS(testsuite.GetTestNamespace(nil), "virtctl", args...)
			Expect(err).ToNot(HaveOccurred())
			out, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
			Expect(out).ToNot(BeEmpty())
		}
	}

	BeforeAll(func() {
		vmi, keyFile = createVMWithPublicKey()
	})

	DescribeTable("[test_id:11661]should succeed to execute a command on the VM", func(cmdFn func(string)) {
		By("ssh into the VM")
		cmdFn(vmi.Name)
	},
		Entry("using the local ssh method", cmdLocal(false)),
		Entry("using the local ssh method with --local-ssh=true flag", decorators.NativeSSH, cmdLocal(true)),
		Entry("using the native ssh method with --local-ssh=false flag", decorators.NativeSSH, cmdNative),
	)

	It("[test_id:11666]local-ssh flag should be unavailable in virtctl", decorators.ExcludeNativeSSH, func() {
		// The built virtctl binary should be tested here, therefore clientcmd.CreateCommandWithNS needs to be used.
		// Running the command through newRepeatableVirtctlCommand would test the test binary instead.
		_, cmd, err := clientcmd.CreateCommandWithNS(testsuite.NamespaceTestDefault, "virtctl", "ssh", "--local-ssh=false")
		Expect(err).ToNot(HaveOccurred())
		out, err := cmd.CombinedOutput()
		Expect(err).To(HaveOccurred(), "out[%s]", string(out))
		Expect(string(out)).To(Equal("unknown flag: --local-ssh\n"))
	})
}))

func createVMWithPublicKey() (vmi *v1.VirtualMachineInstance, keyFile string) {
	libssh.DisableSSHAgent()
	keyFile = filepath.Join(GinkgoT().TempDir(), "id_rsa")

	priv, pub, err := libssh.NewKeyPair()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, libssh.DumpPrivateKey(priv, keyFile)).To(Succeed())

	By("injecting a SSH public key into a VMI")
	vmi = libvmifact.NewAlpineWithTestTooling(
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData(libssh.RenderUserDataWithKey(pub))),
	)
	vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).
		Create(context.Background(), vmi, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine), keyFile
}
