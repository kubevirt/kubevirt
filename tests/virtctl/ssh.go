package virtctl

import (
	"context"
	"crypto/ecdsa"
	"os"
	"path/filepath"

	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"

	"kubevirt.io/kubevirt/tests/libssh"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[sig-compute][virtctl]SSH", decorators.SigCompute, func() {
	var pub ssh.PublicKey
	var keyFile string
	var virtClient kubecli.KubevirtClient

	cmdNative := func(vmiName string) {
		Expect(clientcmd.NewRepeatableVirtctlCommand(
			"ssh",
			"--local-ssh=false",
			"--namespace", util.NamespaceTestDefault,
			"--username", "root",
			"--identity-file", keyFile,
			"--known-hosts=",
			`--command='true'`,
			vmiName)()).To(Succeed())
	}

	cmdLocal := func(appendLocalSSH bool) func(vmiName string) {
		return func(vmiName string) {
			args := []string{
				"ssh",
				"--namespace", util.NamespaceTestDefault,
				"--username", "root",
				"--identity-file", keyFile,
				"-t", "-o StrictHostKeyChecking=no",
				"-t", "-o UserKnownHostsFile=/dev/null",
				"--command", "true",
			}
			if appendLocalSSH {
				args = append(args, "--local-ssh=true")
			}
			args = append(args, vmiName)

			_, cmd, err := clientcmd.CreateCommandWithNS(util.NamespaceTestDefault, "virtctl", args...)
			Expect(err).ToNot(HaveOccurred())

			out, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred(), "out[%s]", string(out))
			Expect(out).ToNot(BeEmpty())
		}
	}

	BeforeEach(func() {
		var err error
		virtClient = kubevirt.Client()
		// Disable SSH_AGENT to not influence test results
		Expect(os.Setenv("SSH_AUTH_SOCK", "/dev/null")).To(Succeed())
		keyFile = filepath.Join(GinkgoT().TempDir(), "id_rsa")
		var priv *ecdsa.PrivateKey
		priv, pub, err = libssh.NewKeyPair()
		Expect(err).ToNot(HaveOccurred())
		Expect(libssh.DumpPrivateKey(priv, keyFile)).To(Succeed())
	})

	DescribeTable("should succeed to execute a command on the VM", func(cmdFn func(string)) {
		By("injecting a SSH public key into a VMI")
		vmi := libvmi.New(
			libvmi.WithContainerImage(cd.ContainerDiskFor(cd.ContainerDiskAlpineTestTooling)),
			libvmi.WithAlpineResourceMemory(),
			libvmi.WithRng(),
			libvmi.WithCloudInitNoCloudUserData(libssh.RenderUserDataWithKey(pub)),
		)
		vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
		Expect(err).ToNot(HaveOccurred())

		vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

		By("ssh into the VM")
		cmdFn(vmi.Name)
	},
		Entry("using the native ssh method", decorators.NativeSsh, cmdNative),
		Entry("using the local ssh method with --local-ssh flag", decorators.NativeSsh, cmdLocal(true)),
		Entry("using the local ssh method without --local-ssh flag", decorators.ExcludeNativeSsh, cmdLocal(false)),
	)

	It("local-ssh flag should be unavailable in virtctl binary", decorators.ExcludeNativeSsh, func() {
		_, cmd, err := clientcmd.CreateCommandWithNS(util.NamespaceTestDefault, "virtctl", "ssh", "--local-ssh=false")
		Expect(err).ToNot(HaveOccurred())
		out, err := cmd.CombinedOutput()
		Expect(err).To(HaveOccurred(), "out[%s]", string(out))
		Expect(string(out)).To(Equal("unknown flag: --local-ssh\n"))
	})
})
