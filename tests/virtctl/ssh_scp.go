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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

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

var _ = Describe(SIG("[sig-compute]SSH and SCP", decorators.SigCompute, Ordered, decorators.OncePerOrderedCleanup, func() {
	const (
		randSuffixLen = 8
	)

	var (
		keyFile string
		vmi     *v1.VirtualMachineInstance
	)

	BeforeAll(func() {
		vmi, keyFile = createVMWithPublicKey()
	})

	It("[test_id:11661]should succeed to execute a command on the VM", func() {
		runSSHCommand(vmi.Name, "root", keyFile)
	})

	It("[test_id:11659]should copy a local file back and forth", func() {
		remoteFile := "vmi/" + vmi.Name + ":./keyfile-" + rand.String(randSuffixLen)

		By("copying a file to the VMI")
		runSCPCommand(keyFile, remoteFile, keyFile, false)

		By("copying the file back")
		copyBackFile := filepath.Join(GinkgoT().TempDir(), "remote_id_rsa")
		runSCPCommand(remoteFile, copyBackFile, keyFile, false)

		By("comparing the two files")
		compareFile(keyFile, copyBackFile)
	})

	It("[test_id:11660]should copy a local directory back and forth", func() {
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

		remoteDir := "vmi/" + vmi.Name + ":./sourcedir-" + rand.String(randSuffixLen)

		By("copying a file to the VMI")
		runSCPCommand(copyFromDir, remoteDir, keyFile, true)

		By("copying the file back")
		runSCPCommand(remoteDir, copyToDir, keyFile, true)

		By("comparing the two directories")
		compareFile(filepath.Join(copyFromDir, "file1"), filepath.Join(copyToDir, "file1"))
		compareFile(filepath.Join(copyFromDir, "file2"), filepath.Join(copyToDir, "file2"))
	})
}))

func createVMWithPublicKey() (vmi *v1.VirtualMachineInstance, keyFile string) {
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

func runSSHCommand(name, user, keyFile string) {
	libssh.DisableSSHAgent()
	args := []string{
		"ssh",
		"--namespace", testsuite.GetTestNamespace(nil),
		"--username", user,
		"--identity-file", keyFile,
		"-t", "-o StrictHostKeyChecking=no",
		"-t", "-o UserKnownHostsFile=/dev/null",
		"--command", "true",
		"vmi/" + name,
	}

	runVirtctlBinary(args)
}

func runSCPCommand(src, dst, keyFile string, recursive bool) {
	libssh.DisableSSHAgent()
	args := []string{
		"scp",
		"--namespace", testsuite.GetTestNamespace(nil),
		"--username", "root",
		"--identity-file", keyFile,
		"-t", "-o StrictHostKeyChecking=no",
		"-t", "-o UserKnownHostsFile=/dev/null",
	}
	if recursive {
		args = append(args, "--recursive")
	}
	args = append(args, src, dst)

	runVirtctlBinary(args)
}

func runVirtctlBinary(args []string) {
	// The virtctl binary needs to run here because of the way local client wrapping works.
	// Running the command through newRepeatableVirtctlCommand does not suffice.
	_, cmd, err := clientcmd.CreateCommandWithNS(testsuite.GetTestNamespace(nil), "virtctl", args...)
	Expect(err).ToNot(HaveOccurred())
	out, err := cmd.CombinedOutput()
	Expect(err).ToNot(HaveOccurred())
	Expect(out).ToNot(BeEmpty())
}

func compareFile(file1, file2 string) {
	expected, err := os.ReadFile(file1)
	Expect(err).ToNot(HaveOccurred())
	actual, err := os.ReadFile(file2)
	Expect(err).ToNot(HaveOccurred())
	Expect(string(actual)).To(Equal(string(expected)))
}
