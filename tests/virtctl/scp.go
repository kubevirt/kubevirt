package virtctl

import (
	"context"
	"crypto/ecdsa"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"

	"kubevirt.io/kubevirt/tests/libssh"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[sig-compute][virtctl]SCP", func() {
	var pub ssh.PublicKey
	var keyFile string
	var virtClient kubecli.KubevirtClient

	copyNative := func(src, dst string, recursive bool) {
		args := []string{
			"scp",
			"--namespace", util.NamespaceTestDefault,
			"--username", "root",
			"--identity-file", keyFile,
			"--known-hosts=",
		}
		if recursive {
			args = append(args, "--recursive")
		}
		args = append(args, src, dst)

		Expect(clientcmd.NewRepeatableVirtctlCommand(args...)()).To(Succeed())
	}

	copyLocal := func(src, dst string, recursive bool) {
		args := []string{
			"scp",
			"--local-ssh=true",
			"--namespace", util.NamespaceTestDefault,
			"--username", "root",
			"--identity-file", keyFile,
			"-t", "-o StrictHostKeyChecking=no",
			"-t", "-o UserKnownHostsFile=/dev/null",
		}
		if recursive {
			args = append(args, "--recursive")
		}
		args = append(args, src, dst)

		_, cmd, err := clientcmd.CreateCommandWithNS(util.NamespaceTestDefault, "virtctl", args...)
		Expect(err).ToNot(HaveOccurred())

		out, err := cmd.CombinedOutput()
		Expect(err).ToNot(HaveOccurred(), "out[%s]", string(out))
		Expect(out).ToNot(BeEmpty())
	}

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		// Disable SSH_AGENT to not influence test results
		Expect(os.Setenv("SSH_AUTH_SOCK", "/dev/null")).To(Succeed())
		keyFile = filepath.Join(GinkgoT().TempDir(), "id_rsa")
		var priv *ecdsa.PrivateKey
		priv, pub, err = libssh.NewKeyPair()
		Expect(err).ToNot(HaveOccurred())
		Expect(libssh.DumpPrivateKey(priv, keyFile)).To(Succeed())
	})

	DescribeTable("should copy a local file back and forth", func(copyFn func(string, string, bool)) {
		By("injecting a SSH public key into a VMI")
		vmi := libvmi.NewAlpineWithTestTooling(
			libvmi.WithCloudInitNoCloudUserData(libssh.RenderUserDataWithKey(pub), false))
		vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
		Expect(err).ToNot(HaveOccurred())

		vmi = tests.WaitUntilVMIReady(vmi, console.LoginToAlpine)

		By("copying a file to the VMI")
		copyFn(keyFile, vmi.Name+":"+"./keyfile", false)

		By("copying the file back")
		copyBackFile := filepath.Join(GinkgoT().TempDir(), "remote_id_rsa")
		copyFn(vmi.Name+":"+"./keyfile", copyBackFile, false)

		By("comparing the two files")
		compareFile(keyFile, copyBackFile)
	},
		Entry("using the native scp method", copyNative),
		Entry("using the local scp method", copyLocal),
	)

	DescribeTable("should copy a local directory back and forth", func(copyFn func(string, string, bool)) {
		By("injecting a SSH public key into a VMI")
		vmi := libvmi.NewAlpineWithTestTooling(
			libvmi.WithCloudInitNoCloudUserData(libssh.RenderUserDataWithKey(pub), false))
		vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
		Expect(err).ToNot(HaveOccurred())

		vmi = tests.WaitUntilVMIReady(vmi, console.LoginToAlpine)

		By("creating a few random files")
		copyFromDir := filepath.Join(GinkgoT().TempDir(), "sourcedir")
		copyToDir := filepath.Join(GinkgoT().TempDir(), "targetdir")

		Expect(os.Mkdir(copyFromDir, 0777)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(copyFromDir, "file1"), []byte("test"), 0777)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(copyFromDir, "file2"), []byte("test1"), 0777)).To(Succeed())

		By("copying a file to the VMI")
		copyFn(copyFromDir, vmi.Name+":"+"./sourcedir", true)

		By("copying the file back")
		copyFn(vmi.Name+":"+"./sourcedir", copyToDir, true)

		By("comparing the two directories")
		compareFile(filepath.Join(copyFromDir, "file1"), filepath.Join(copyToDir, "file1"))
		compareFile(filepath.Join(copyFromDir, "file2"), filepath.Join(copyToDir, "file2"))
	},
		Entry("using the native scp method", copyNative),
		Entry("using the local scp method", copyLocal),
	)
})

func compareFile(file1 string, file2 string) {
	expected, err := ioutil.ReadFile(file1)
	Expect(err).ToNot(HaveOccurred())
	actual, err := ioutil.ReadFile(file2)
	Expect(err).ToNot(HaveOccurred())
	Expect(string(actual)).To(Equal(string(expected)))
}
