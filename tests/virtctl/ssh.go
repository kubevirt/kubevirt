package virtctl

import (
	"crypto/ecdsa"
	"fmt"
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

var _ = Describe("[sig-compute][virtctl]SSH", func() {
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
		tests.BeforeTestCleanup()
		var priv *ecdsa.PrivateKey
		priv, pub, err = NewKeyPair()
		Expect(err).ToNot(HaveOccurred())
		Expect(DumpPrivateKey(priv, keyFile)).To(Succeed())
	})

	It("should succeed to execute a command on the VM using the ssh native method", func() {
		By("injecting a SSH public key into a VMI")
		vmi := libvmi.NewCirros(
			libvmi.WithCloudInitNoCloudUserData(renderUserDataWithKey(pub), false))
		vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
		Expect(err).ToNot(HaveOccurred())

		vmi = tests.WaitUntilVMIReady(vmi, console.LoginToCirros)
		By("ssh into the VM using the native option")
		Expect(clientcmd.NewRepeatableVirtctlCommand(
			"ssh",
			"--local-ssh=false",
			"--namespace", util.NamespaceTestDefault,
			"--username", "cirros",
			"--identity-file", keyFile,
			"--known-hosts=",
			`--command='true'`,
			vmi.Name)()).To(Succeed())

	})
	It("should succeed execute a command on the VM using local ssh method", func() {
		By("injecting a SSH public key into a VMI")
		vmi := libvmi.NewCirros(
			libvmi.WithCloudInitNoCloudUserData(renderUserDataWithKey(pub), false))
		vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
		Expect(err).ToNot(HaveOccurred())

		vmi = tests.WaitUntilVMIReady(vmi, console.LoginToCirros)
		By("ssh into the VM using the native option")
		_, cmd, err := clientcmd.CreateCommandWithNS(util.NamespaceTestDefault, "virtctl",
			"ssh",
			"--local-ssh=true",
			"--namespace", util.NamespaceTestDefault,
			"--username", "cirros",
			"--identity-file", keyFile,
			"-t", "-o StrictHostKeyChecking=no",
			"-t", "-o UserKnownHostsFile=/dev/null",
			"--command", "true",
			vmi.Name)
		Expect(err).ToNot(HaveOccurred())
		cmd.Env = append(cmd.Env,
			"SSH_AUTH_SOCK=/dev/null")
		out, err := cmd.CombinedOutput()
		fmt.Println(string(out))
		Expect(err).ToNot(HaveOccurred())
	})
})
