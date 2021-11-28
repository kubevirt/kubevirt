package ssh

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/gomega"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Wrapped SSH", func() {

	var fakeKind, fakeNamespace, fakeName string

	BeforeEach(func() {
		fakeKind = "fake-kind"
		fakeNamespace = "fake-ns"
		fakeName = "fake-name"
	})

	Context("buildSSHTarget", func() {

		It("with SSH username", func() {
			sshUsername = "testuser"

			sshTarget := buildSSHTarget(fakeKind, fakeNamespace, fakeName)
			Expect(sshTarget).To(Equal("testuser@fake-kind/fake-name.fake-ns"))
		})

		It("without SSH username", func() {
			sshUsername = ""

			sshTarget := buildSSHTarget(fakeKind, fakeNamespace, fakeName)
			Expect(sshTarget).To(Equal("fake-kind/fake-name.fake-ns"))
		})

	})

	It("buildProxyCommandOption", func() {
		sshPort = 12345
		proxyCommand := buildProxyCommandOption(fakeKind, fakeNamespace, fakeName)
		expected := fmt.Sprintf("port-forward --stdio=true fake-kind/fake-name.fake-ns %d", sshPort)
		Expect(proxyCommand).To(ContainSubstring(expected))
	})

	It("runLocalCommandClient", func() {
		runCommand = func(cmd *exec.Cmd) error {
			Expect(cmd).ToNot(BeNil())
			Expect(cmd.Args).To(HaveLen(3))
			Expect(cmd.Args[0]).To(Equal("ssh"))
			Expect(cmd.Args[1]).To(Equal(buildProxyCommandOption(fakeKind, fakeNamespace, fakeName)))
			Expect(cmd.Args[2]).To(Equal(buildSSHTarget(fakeKind, fakeNamespace, fakeName)))

			return nil
		}

		err := runLocalCommandClient(fakeKind, fakeNamespace, fakeName)
		Expect(err).ShouldNot(HaveOccurred())
	})
})
