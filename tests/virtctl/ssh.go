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

package virtctl

import (
	"context"
	"crypto/ecdsa"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

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

var _ = VirtctlDescribe("[sig-compute]SSH", decorators.SigCompute, func() {
	var pub ssh.PublicKey
	var keyFile string
	var virtClient kubecli.KubevirtClient

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

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		libssh.DisableSSHAgent()
		keyFile = filepath.Join(GinkgoT().TempDir(), "id_rsa")
		var err error
		var priv *ecdsa.PrivateKey
		priv, pub, err = libssh.NewKeyPair()
		Expect(err).ToNot(HaveOccurred())
		Expect(libssh.DumpPrivateKey(priv, keyFile)).To(Succeed())
	})

	DescribeTable("[test_id:11661]should succeed to execute a command on the VM", func(cmdFn func(string)) {
		By("injecting a SSH public key into a VMI")
		vmi := libvmifact.NewAlpineWithTestTooling(
			libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData(libssh.RenderUserDataWithKey(pub))),
		)
		vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

		By("ssh into the VM")
		cmdFn(vmi.Name)
	},
		Entry("using the native ssh method", decorators.NativeSSH, cmdNative),
		Entry("using the local ssh method with --local-ssh flag", decorators.NativeSSH, cmdLocal(true)),
		Entry("using the local ssh method without --local-ssh flag", decorators.ExcludeNativeSSH, cmdLocal(false)),
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
})
