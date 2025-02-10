package ssh

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/gomega"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Wrapped SSH", func() {

	var (
		fakeNamespace string
		fakeName      string

		ssh SSH
	)

	BeforeEach(func() {
		fakeNamespace = "fake-ns"
		fakeName = "fake-name"
		ssh = SSH{}
	})

	Context("buildSSHTarget", func() {

		It("with SSH username", func() {
			ssh.options = SSHOptions{SSHUsername: "testuser"}
			sshTarget := ssh.buildSSHTarget(fakeNamespace, fakeName)
			Expect(sshTarget[0]).To(Equal("testuser@fake-ns/fake-name"))
		})

		It("without SSH username", func() {
			sshTarget := ssh.buildSSHTarget(fakeNamespace, fakeName)
			Expect(sshTarget[0]).To(Equal("fake-ns/fake-name"))
		})

	})

	It("buildProxyCommandOption", func() {
		const sshPort = 12345
		proxyCommand := buildProxyCommandOption(fakeNamespace, fakeName, sshPort)
		expected := fmt.Sprintf("port-forward --stdio=true fake-ns/fake-name %d", sshPort)
		Expect(proxyCommand).To(ContainSubstring(expected))
	})

	It("RunLocalClient", func() {
		runCommand = func(cmd *exec.Cmd) error {
			Expect(cmd).ToNot(BeNil())
			Expect(cmd.Args).To(HaveLen(4))
			Expect(cmd.Args[0]).To(Equal("ssh"))
			Expect(cmd.Args[2]).To(Equal(buildProxyCommandOption(fakeNamespace, fakeName, ssh.options.SSHPort)))
			Expect(cmd.Args[3]).To(Equal(ssh.buildSSHTarget(fakeNamespace, fakeName)[0]))

			return nil
		}

		ssh.options = DefaultSSHOptions()
		ssh.options.SSHPort = 12345
		clientArgs := ssh.buildSSHTarget(fakeNamespace, fakeName)
		err := RunLocalClient(fakeNamespace, fakeName, &ssh.options, clientArgs)
		Expect(err).ShouldNot(HaveOccurred())
	})
})
