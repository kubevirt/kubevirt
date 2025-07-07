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
	"os"
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

var _ = Describe(SIG("[sig-compute]SCP", decorators.SigCompute, func() {
	var pub ssh.PublicKey
	var keyFile string
	var virtClient kubecli.KubevirtClient

	copyNative := func(src, dst string, recursive bool) {
		args := []string{
			"scp",
			"--local-ssh=false",
			"--namespace", testsuite.GetTestNamespace(nil),
			"--username", "root",
			"--identity-file", keyFile,
			"--known-hosts=",
		}
		if recursive {
			args = append(args, "--recursive")
		}
		args = append(args, src, dst)
		Expect(newRepeatableVirtctlCommand(args...)()).To(Succeed())
	}

	copyLocal := func(appendLocalSSH bool) func(src, dst string, recursive bool) {
		return func(src, dst string, recursive bool) {
			args := []string{
				"scp",
				"--namespace", testsuite.GetTestNamespace(nil),
				"--username", "root",
				"--identity-file", keyFile,
				"-t", "-o StrictHostKeyChecking=no",
				"-t", "-o UserKnownHostsFile=/dev/null",
			}
			if appendLocalSSH {
				args = append(args, "--local-ssh=true")
			}
			if recursive {
				args = append(args, "--recursive")
			}
			args = append(args, src, dst)

			// The virtctl binary needs to run here because of the way local SCP client wrapping works.
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
		// Disable SSH_AGENT to not influence test results
		Expect(os.Setenv("SSH_AUTH_SOCK", "/dev/null")).To(Succeed())
		keyFile = filepath.Join(GinkgoT().TempDir(), "id_rsa")
		var err error
		var priv *ecdsa.PrivateKey
		priv, pub, err = libssh.NewKeyPair()
		Expect(err).ToNot(HaveOccurred())
		Expect(libssh.DumpPrivateKey(priv, keyFile)).To(Succeed())
	})

	DescribeTable("[test_id:11659]should copy a local file back and forth", func(copyFn func(string, string, bool)) {
		By("injecting a SSH public key into a VMI")
		vmi := libvmifact.NewAlpineWithTestTooling(
			libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData(libssh.RenderUserDataWithKey(pub))),
		)
		vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

		remoteFile := "vmi/" + vmi.Name + ":./keyfile"

		By("copying a file to the VMI")
		copyFn(keyFile, remoteFile, false)

		By("copying the file back")
		copyBackFile := filepath.Join(GinkgoT().TempDir(), "remote_id_rsa")
		copyFn(remoteFile, copyBackFile, false)

		By("comparing the two files")
		compareFile(keyFile, copyBackFile)
	},
		Entry("using the local scp method", copyLocal(false)),
		Entry("using the local scp method with --local-ssh=true flag", decorators.NativeSSH, copyLocal(true)),
		Entry("using the native scp method with --local-ssh=false flag", decorators.NativeSSH, copyNative),
	)

	DescribeTable("[test_id:11660]should copy a local directory back and forth", func(copyFn func(string, string, bool)) {
		By("injecting a SSH public key into a VMI")
		vmi := libvmifact.NewAlpineWithTestTooling(
			libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData(libssh.RenderUserDataWithKey(pub))),
		)
		vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

		By("creating a few random files")
		copyFromDir := filepath.Join(GinkgoT().TempDir(), "sourcedir")
		copyToDir := filepath.Join(GinkgoT().TempDir(), "targetdir")

		const (
			permRWX = 0o700
			permRW  = 0o600
		)
		Expect(os.Mkdir(copyFromDir, permRWX)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(copyFromDir, "file1"), []byte("test"), permRW)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(copyFromDir, "file2"), []byte("test1"), permRW)).To(Succeed())

		By("copying a file to the VMI")
		copyFn(copyFromDir, "vmi/"+vmi.Name+":"+"./sourcedir", true)

		By("copying the file back")
		copyFn("vmi/"+vmi.Name+":"+"./sourcedir", copyToDir, true)

		By("comparing the two directories")
		compareFile(filepath.Join(copyFromDir, "file1"), filepath.Join(copyToDir, "file1"))
		compareFile(filepath.Join(copyFromDir, "file2"), filepath.Join(copyToDir, "file2"))
	},
		Entry("using the local scp method", copyLocal(false)),
		Entry("using the local scp method with --local-ssh=true flag", decorators.NativeSSH, copyLocal(true)),
		Entry("using the native scp method with --local-ssh=false flag", decorators.NativeSSH, copyNative),
	)

	It("[test_id:11665]local-ssh flag should be unavailable in virtctl", decorators.ExcludeNativeSSH, func() {
		// The built virtctl binary should be tested here, therefore clientcmd.CreateCommandWithNS needs to be used.
		// Running the command through newRepeatableVirtctlCommand would test the test binary instead.
		_, cmd, err := clientcmd.CreateCommandWithNS(testsuite.NamespaceTestDefault, "virtctl", "scp", "--local-ssh=false")
		Expect(err).ToNot(HaveOccurred())
		out, err := cmd.CombinedOutput()
		Expect(err).To(HaveOccurred(), "out[%s]", string(out))
		Expect(string(out)).To(Equal("unknown flag: --local-ssh\n"))
	})
}))

func compareFile(file1, file2 string) {
	expected, err := os.ReadFile(file1)
	Expect(err).ToNot(HaveOccurred())
	actual, err := os.ReadFile(file2)
	Expect(err).ToNot(HaveOccurred())
	Expect(string(actual)).To(Equal(string(expected)))
}
