package virtctl

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"

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
	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		// Disable SSH_AGENT to not influence test results
		Expect(os.Setenv("SSH_AUTH_SOCK", "/dev/null")).To(Succeed())
		keyFile = filepath.Join(GinkgoT().TempDir(), "id_rsa")
		var priv *ecdsa.PrivateKey
		priv, pub, err = NewKeyPair()
		Expect(err).ToNot(HaveOccurred())
		Expect(DumpPrivateKey(priv, keyFile)).To(Succeed())
	})

	It("should copy a local file back and forth", func() {
		By("injecting a SSH public key into a VMI")
		vmi := libvmi.NewCirros(
			libvmi.WithCloudInitNoCloudUserData(renderUserDataWithKey(pub), false))
		vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
		Expect(err).ToNot(HaveOccurred())

		vmi = tests.WaitUntilVMIReady(vmi, console.LoginToCirros)

		By("copying a file to the VMI")
		Expect(clientcmd.NewRepeatableVirtctlCommand(
			"scp",
			"--namespace", util.NamespaceTestDefault,
			"--username", "cirros",
			"--identity-file", keyFile,
			"--known-hosts=",
			keyFile,
			vmi.Name+":"+"./keyfile",
		)()).To(Succeed())

		By("copying the file back")
		copyBackFile := filepath.Join(GinkgoT().TempDir(), "remote_id_rsa")
		Expect(clientcmd.NewRepeatableVirtctlCommand(
			"scp",
			"--namespace", util.NamespaceTestDefault,
			"--username", "cirros",
			"--identity-file", keyFile,
			"--known-hosts=",
			vmi.Name+":"+"./keyfile",
			copyBackFile,
		)()).To(Succeed())

		By("comparing the two files")
		compareFile(keyFile, copyBackFile)
	})

	It("should copy a local directory back and forth", func() {
		By("injecting a SSH public key into a VMI")
		vmi := libvmi.NewCirros(
			libvmi.WithCloudInitNoCloudUserData(renderUserDataWithKey(pub), false))
		vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
		Expect(err).ToNot(HaveOccurred())

		vmi = tests.WaitUntilVMIReady(vmi, console.LoginToCirros)

		By("creating a few random files")
		copyFromDir := filepath.Join(GinkgoT().TempDir(), "sourcedir")
		copyToDir := filepath.Join(GinkgoT().TempDir(), "targetdir")

		Expect(os.Mkdir(copyFromDir, 0777)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(copyFromDir, "file1"), []byte("test"), 0777)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(copyFromDir, "file2"), []byte("test1"), 0777)).To(Succeed())

		By("copying a file to the VMI")
		Expect(clientcmd.NewRepeatableVirtctlCommand(
			"scp",
			"--namespace", util.NamespaceTestDefault,
			"--username", "cirros",
			"--identity-file", keyFile,
			"--recursive",
			"--known-hosts=",
			copyFromDir,
			vmi.Name+":"+"./sourcedir",
		)()).To(Succeed())

		By("copying the file back")
		Expect(clientcmd.NewRepeatableVirtctlCommand(
			"scp",
			"--namespace", util.NamespaceTestDefault,
			"--username", "cirros",
			"--recursive",
			"--identity-file", keyFile,
			"--known-hosts=",
			vmi.Name+":"+"./sourcedir",
			copyToDir,
		)()).To(Succeed())

		By("comparing the two directories")
		compareFile(filepath.Join(copyFromDir, "file1"), filepath.Join(copyToDir, "file1"))
		compareFile(filepath.Join(copyFromDir, "file2"), filepath.Join(copyToDir, "file2"))
	})
})

func renderUserDataWithKey(key ssh.PublicKey) string {
	return fmt.Sprintf(`#!/bin/sh
mkdir -p /home/cirros/.ssh/
echo "%s" > /home/cirros/.ssh/authorized_keys
chown -R cirros:cirros /home/cirros/.ssh
`, string(ssh.MarshalAuthorizedKey(key)))
}

func compareFile(file1 string, file2 string) {
	expected, err := ioutil.ReadFile(file1)
	Expect(err).ToNot(HaveOccurred())
	actual, err := ioutil.ReadFile(file2)
	Expect(err).ToNot(HaveOccurred())
	Expect(string(actual)).To(Equal(string(expected)))
}
