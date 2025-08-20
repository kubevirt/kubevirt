package ssh_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/ssh"
)

var _ = Describe("SSH", func() {
	DescribeTable("ParseTarget", func(arg, targetNamespace, targetName, targetKind, targetUsername, expectedError string) {
		kind, namespace, name, username, err := ssh.ParseTarget(arg)
		Expect(namespace).To(Equal(targetNamespace))
		Expect(name).To(Equal(targetName))
		Expect(kind).To(Equal(targetKind))
		Expect(username).To(Equal(targetUsername))
		if expectedError == "" {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(MatchError(expectedError))
		}
	},
		Entry("kind vmi with name", "vmi/testvmi", "", "testvmi", "vmi", "", ""),
		Entry("kind vmi with name and namespace", "vmi/testvmi/default", "default", "testvmi", "vmi", "", ""),
		Entry("kind vmi with name and username", "user@vmi/testvmi", "", "testvmi", "vmi", "user", ""),
		Entry("kind vmi with name and namespace and username", "user@vmi/testvmi/default", "default", "testvmi", "vmi", "user", ""),
		Entry("kind vm with name", "vm/testvm", "", "testvm", "vm", "", ""),
		Entry("kind vm with name and namespace", "vm/testvm/default", "default", "testvm", "vm", "", ""),
		Entry("kind vm with name and username", "user@vm/testvm", "", "testvm", "vm", "user", ""),
		Entry("kind vm with name and namespace and username", "user@vm/testvm/default", "default", "testvm", "vm", "user", ""),
		Entry("name with dots and namespace", "vmi/testvmi.with.dots/default", "default", "testvmi.with.dots", "vmi", "", ""),
		Entry("name with dots and namespace and username", "user@vmi/testvmi.with.dots/default", "default", "testvmi.with.dots", "vmi", "user", ""),
		Entry("name and namespace with dots", "vmi/testvmi/default.with.dots", "default.with.dots", "testvmi", "vmi", "", ""),
		Entry("name and namespace with dots and username", "user@vmi/testvmi/default.with.dots", "default.with.dots", "testvmi", "vmi", "user", ""),
		Entry("name with dots and namespace with dots", "vmi/testvmi.with.dots/default.with.dots", "default.with.dots", "testvmi.with.dots", "vmi", "", ""),
		Entry("name with dots and namespace with dots and username", "user@vmi/testvmi.with.dots/default.with.dots", "default.with.dots", "testvmi.with.dots", "vmi", "user", ""),
		Entry("no slash", "testvmi", "", "", "", "", "target must contain type and name separated by '/'"),
		Entry("no slash and username", "user@testvmi", "", "", "", "", "target must contain type and name separated by '/'"),
		Entry("empty namespace", "vmi/testvmi/", "", "", "", "", "namespace cannot be empty"),
		Entry("empty namespace and username", "user@vmi/testvmi/", "", "", "", "", "namespace cannot be empty"),
		Entry("more than tree slashes", "vmi/testvmi/default/something", "", "", "", "", "target is not valid with more than two '/'"),
		Entry("more than tree slashes and username", "user@vmi/testvmi/default/something", "", "", "", "", "target is not valid with more than two '/'"),
		Entry("invalid type with name", "invalid/testvmi", "", "", "", "", "unsupported resource type 'invalid'"),
		Entry("invalid type with name and username", "user@invalid/testvmi", "", "", "", "", "unsupported resource type 'invalid'"),
		Entry("only valid kind", "vmi/", "", "", "", "", "name cannot be empty"),
		Entry("only valid kind and username", "user@vmi/", "", "", "", "", "name cannot be empty"),
		Entry("empty target", "", "", "", "", "", "expected target after '@'"),
		Entry("only slash", "/", "", "", "", "", "unsupported resource type ''"),
		Entry("two slashes", "//", "", "", "", "", "namespace cannot be empty"),
		Entry("two slashes and username", "user@//", "", "", "", "", "namespace cannot be empty"),
		Entry("only dot", ".", "", "", "", "", "target must contain type and name separated by '/'"),
		Entry("only separators", "/.", "", "", "", "", "unsupported resource type ''"),
		Entry("only separators and at", "@/.", "", "", "", "", "expected username before '@'"),
		Entry("only username", "user@", "", "", "", "", "expected target after '@'"),
		Entry("only at", "@", "", "", "", "", "expected username before '@'"),
		Entry("only at and target", "@vmi/testvmi", "", "", "", "", "expected username before '@'"),
		// Before 1.7 these syntaxes resulted in the last part after a dot being parsed as the namespace.
		// This was changed in 1.7 to not parse as namespace anymore.
		Entry("name with dots", "vmi/testvmi.with.dots", "", "testvmi.with.dots", "vmi", "", ""),
		Entry("name with dots and username", "user@vmi/testvmi.with.dots", "", "testvmi.with.dots", "vmi", "user", ""),
		Entry("kind vmi with name with dot", "vmi/testvmi.default", "", "testvmi.default", "vmi", "", ""),
		Entry("kind vmi with name with dot and username", "user@vmi/testvmi.default", "", "testvmi.default", "vmi", "user", ""),
		Entry("kind vm with name with dot", "vm/testvm.default", "", "testvm.default", "vm", "", ""),
		Entry("kind vm with name with dot and username", "user@vm/testvm.default", "", "testvm.default", "vm", "user", ""),
	)

	Context("Wrapped SSH", func() {
		const (
			fakeKind      = "fake-kind"
			fakeNamespace = "fake-ns"
			fakeName      = "fake-name"
		)

		DescribeTable("BuildSSHTarget", func(opts ssh.SSHOptions, expected string) {
			c := ssh.NewSSH(&opts)
			sshTarget := c.BuildSSHTarget(fakeKind, fakeNamespace, fakeName)
			Expect(sshTarget[0]).To(Equal(expected))
		},
			Entry("with SSH username", ssh.SSHOptions{SSHUsername: "testuser"}, "testuser@fake-kind.fake-name.fake-ns"),
			Entry("without SSH username", ssh.SSHOptions{}, "fake-kind.fake-name.fake-ns"),
		)

		It("BuildProxyCommandOption", func() {
			const sshPort = 12345
			proxyCommand := ssh.BuildProxyCommandOption(fakeKind, fakeNamespace, fakeName, sshPort)
			expected := fmt.Sprintf("port-forward --stdio=true fake-kind/fake-name/fake-ns %d", sshPort)
			Expect(proxyCommand).To(ContainSubstring(expected))
		})

		It("LocalClientCmd", func() {
			opts := ssh.DefaultSSHOptions()
			opts.SSHPort = 12345
			c := ssh.NewSSH(opts)
			clientArgs := c.BuildSSHTarget(fakeKind, fakeNamespace, fakeName)
			cmd := ssh.LocalClientCmd(fakeKind, fakeNamespace, fakeName, opts, clientArgs)
			Expect(cmd).ToNot(BeNil())
			Expect(cmd.Args).To(HaveLen(4))
			Expect(cmd.Args[0]).To(Equal("ssh"))
			Expect(cmd.Args[2]).To(Equal(ssh.BuildProxyCommandOption(fakeKind, fakeNamespace, fakeName, opts.SSHPort)))
			Expect(cmd.Args[3]).To(Equal(c.BuildSSHTarget(fakeKind, fakeNamespace, fakeName)[0]))
		})
	})
})
