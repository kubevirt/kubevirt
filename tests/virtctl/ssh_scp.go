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
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libssh"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("[sig-compute]SSH and SCP", decorators.SigCompute, func() {
	const randSuffixLen = 8
	var (
		keyFile string
		vmi     *v1.VirtualMachineInstance
	)

	Context("TCP port-forward", Ordered, decorators.OncePerOrderedCleanup, func() {
		BeforeAll(func() {
			vmi, keyFile = createVMWithPublicKey(
				libvmifact.NewAlpineWithTestTooling, console.LoginToAlpine,
				libssh.RenderUserDataWithKey,
			)
		})

		It("[test_id:11661]should succeed to execute a command on the VM", func() {
			runSSHCommand(vmi.Name, "root", keyFile, false)
		})

		It("[test_id:11659]should copy a local file back and forth", func() {
			remoteFile := "vmi/" + vmi.Name + ":./keyfile-" + rand.String(randSuffixLen)

			By("copying a file to the VMI")
			runSCPCommand(keyFile, remoteFile, keyFile, false, false)

			By("copying the file back")
			copyBackFile := filepath.Join(GinkgoT().TempDir(), "remote_id_rsa")
			runSCPCommand(remoteFile, copyBackFile, keyFile, false, false)

			By("comparing the two files")
			compareFile(keyFile, copyBackFile)
		})

		It("[test_id:11660]should copy a local directory back and forth", func() {
			copyDirectoryBackAndForth(vmi.Name, keyFile, false)
		})
	})

	Context("VSOCK", Serial, decorators.VSOCK, Ordered, decorators.OncePerOrderedCleanup, func() {
		BeforeAll(func() {
			if !checks.HasFeature(featuregate.VSOCKGate) {
				config.EnableFeatureGate(featuregate.VSOCKGate)
				DeferCleanup(config.DisableFeatureGate, featuregate.VSOCKGate)
			}
			vmi, keyFile = createVMWithPublicKey(
				libvmifact.NewFedora, console.LoginToFedora,
				renderUserDataWithVSOCKBridge,
				func(vmi *v1.VirtualMachineInstance) {
					vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.P(true)
				},
			)
		})

		It("[test_id:?]should succeed to ssh to the VMI via VSOCK", func() {
			runSSHCommand(vmi.Name, "root", keyFile, true)
		})

		It("[test_id:?]should succeed to scp a file to the VMI via VSOCK", func() {
			remoteFile := "vmi/" + vmi.Name + ":./keyfile-" + rand.String(randSuffixLen)

			By("copying a file to the VMI")
			runSCPCommand(keyFile, remoteFile, keyFile, false, true)

			By("copying the file back")
			copyBackFile := filepath.Join(GinkgoT().TempDir(), "remote_id_rsa")
			runSCPCommand(remoteFile, copyBackFile, keyFile, false, true)

			By("comparing the two files")
			compareFile(keyFile, copyBackFile)
		})

		It("[test_id:?]should succeed to scp a directory to the VMI via VSOCK", func() {
			copyDirectoryBackAndForth(vmi.Name, keyFile, true)
		})
	})
}))

type userDataRenderer func(ssh.PublicKey) string

func createVMWithPublicKey(
	newVMI func(...libvmi.Option) *v1.VirtualMachineInstance,
	loginTo console.LoginToFunction,
	renderUserData userDataRenderer,
	opts ...libvmi.Option,
) (vmi *v1.VirtualMachineInstance, keyFile string) {
	keyFile = filepath.Join(GinkgoT().TempDir(), "id_rsa")

	priv, pub, err := libssh.NewKeyPair()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, libssh.DumpPrivateKey(priv, keyFile)).To(Succeed())

	By("injecting a SSH public key into a VMI")
	vmiOpts := append([]libvmi.Option{
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData(renderUserData(pub))),
	}, opts...)
	vmi = newVMI(vmiOpts...)
	vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).
		Create(context.Background(), vmi, metav1.CreateOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return libwait.WaitUntilVMIReady(vmi, loginTo), keyFile
}

func renderUserDataWithVSOCKBridge(key ssh.PublicKey) string {
	return fmt.Sprintf(`#!/bin/sh
mkdir -p /root/.ssh/
echo "%s" > /root/.ssh/authorized_keys
chown -R root:root /root/.ssh
socat VSOCK-LISTEN:22,reuseaddr,fork TCP:localhost:22 &
`, string(ssh.MarshalAuthorizedKey(key)))
}

func runSSHCommand(name, user, keyFile string, vsock bool) {
	libssh.DisableSSHAgent()
	args := []string{
		"ssh",
		"--namespace", testsuite.GetTestNamespace(nil),
		"--username", user,
		"--identity-file", keyFile,
		"-t", "-o StrictHostKeyChecking=no",
		"-t", "-o UserKnownHostsFile=/dev/null",
		"--command", "true",
	}
	if vsock {
		args = append(args, "--vsock")
	}
	args = append(args, "vmi/"+name)

	runVirtctlBinary(args)
}

func runSCPCommand(src, dst, keyFile string, recursive, vsock bool) {
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
	if vsock {
		args = append(args, "--vsock")
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
	Expect(err).ToNot(HaveOccurred(), "virtctl output: %s", string(out))
	Expect(out).ToNot(BeEmpty())
}

func copyDirectoryBackAndForth(vmiName, keyFile string, vsock bool) {
	const randSuffixLen = 8

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

	remoteDir := "vmi/" + vmiName + ":./sourcedir-" + rand.String(randSuffixLen)

	By("copying a directory to the VMI")
	runSCPCommand(copyFromDir, remoteDir, keyFile, true, vsock)

	By("copying the directory back")
	runSCPCommand(remoteDir, copyToDir, keyFile, true, vsock)

	By("comparing the two directories")
	compareFile(filepath.Join(copyFromDir, "file1"), filepath.Join(copyToDir, "file1"))
	compareFile(filepath.Join(copyFromDir, "file2"), filepath.Join(copyToDir, "file2"))
}

func compareFile(file1, file2 string) {
	expected, err := os.ReadFile(file1)
	Expect(err).ToNot(HaveOccurred())
	actual, err := os.ReadFile(file2)
	Expect(err).ToNot(HaveOccurred())
	Expect(string(actual)).To(Equal(string(expected)))
}
