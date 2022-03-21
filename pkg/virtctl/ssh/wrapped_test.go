package ssh

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/gomega"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Wrapped SSH", func() {

	var fakeKind, fakeNamespace, fakeName string
	var ssh SSH

	BeforeEach(func() {
		fakeKind = "fake-kind"
		fakeNamespace = "fake-ns"
		fakeName = "fake-name"
		ssh = SSH{}
	})

	Context("buildSSHTarget", func() {

		It("with SSH username", func() {
			ssh.options = SSHOptions{SshUsername: "testuser"}
			sshTarget := ssh.buildSSHTarget(fakeKind, fakeNamespace, fakeName)
			Expect(sshTarget).To(Equal("testuser@fake-kind/fake-name.fake-ns"))
		})

		It("without SSH username", func() {
			sshTarget := ssh.buildSSHTarget(fakeKind, fakeNamespace, fakeName)
			Expect(sshTarget).To(Equal("fake-kind/fake-name.fake-ns"))
		})

	})

	It("buildProxyCommandOption", func() {
		const sshPort = 12345
		ssh.options = SSHOptions{SshPort: sshPort}
		proxyCommand := ssh.buildProxyCommandOption(fakeKind, fakeNamespace, fakeName)
		expected := fmt.Sprintf("port-forward --stdio=true fake-kind/fake-name.fake-ns %d", sshPort)
		Expect(proxyCommand).To(ContainSubstring(expected))
	})

	It("runLocalCommandClient", func() {
		runCommand = func(cmd *exec.Cmd) error {
			Expect(cmd).ToNot(BeNil())
			Expect(cmd.Args).To(HaveLen(3))
			Expect(cmd.Args[0]).To(Equal("ssh"))
			Expect(cmd.Args[1]).To(Equal(ssh.buildProxyCommandOption(fakeKind, fakeNamespace, fakeName)))
			Expect(cmd.Args[2]).To(Equal(ssh.buildSSHTarget(fakeKind, fakeNamespace, fakeName)))

			return nil
		}

		err := ssh.runLocalCommandClient(fakeKind, fakeNamespace, fakeName)
		Expect(err).ShouldNot(HaveOccurred())
	})
})
