package ssh_test

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/virtctl/ssh"
)

const commandSSH = "ssh"

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
		Entry("name with dots and namespace and username",
			"user@vmi/testvmi.with.dots/default", "default", "testvmi.with.dots", "vmi", "user", ""),
		Entry("name and namespace with dots", "vmi/testvmi/default.with.dots", "default.with.dots", "testvmi", "vmi", "", ""),
		Entry("name and namespace with dots and username",
			"user@vmi/testvmi/default.with.dots", "default.with.dots", "testvmi", "vmi", "user", ""),
		Entry("name with dots and namespace with dots",
			"vmi/testvmi.with.dots/default.with.dots", "default.with.dots", "testvmi.with.dots", "vmi", "", ""),
		Entry("name with dots and namespace with dots and username",
			"user@vmi/testvmi.with.dots/default.with.dots", "default.with.dots", "testvmi.with.dots", "vmi", "user", ""),
		Entry("no slash", "testvmi", "", "", "", "", "target must contain type and name separated by '/'"),
		Entry("no slash and username", "user@testvmi", "", "", "", "", "target must contain type and name separated by '/'"),
		Entry("empty namespace", "vmi/testvmi/", "", "", "", "", "namespace cannot be empty"),
		Entry("empty namespace and username", "user@vmi/testvmi/", "", "", "", "", "namespace cannot be empty"),
		Entry("more than tree slashes", "vmi/testvmi/default/something", "", "", "", "", "target is not valid with more than two '/'"),
		Entry("more than tree slashes and username",
			"user@vmi/testvmi/default/something", "", "", "", "", "target is not valid with more than two '/'"),
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
			proxyCommand := ssh.BuildProxyCommandOption(fakeKind, fakeNamespace, fakeName, sshPort, nil)
			expected := fmt.Sprintf("ProxyCommand=%s port-forward --stdio=true %s/%s/%s %d",
				os.Args[0], fakeKind, fakeName, fakeNamespace, sshPort)
			Expect(proxyCommand).To(Equal(expected))
		})

		It("BuildProxyCommandOption with global flags", func() {
			const sshPort = 12345
			globalFlags := []string{"--context=my-context", "--kubeconfig=/path/to/config"}
			proxyCommand := ssh.BuildProxyCommandOption(fakeKind, fakeNamespace, fakeName, sshPort, globalFlags)
			expected := fmt.Sprintf("ProxyCommand=%s '--context=my-context' '--kubeconfig=/path/to/config' port-forward --stdio=true %s/%s/%s %d",
				os.Args[0], fakeKind, fakeName, fakeNamespace, sshPort)
			Expect(proxyCommand).To(Equal(expected))
		})

		It("BuildProxyCommandOption shell-escapes flag values with special characters", func() {
			const sshPort = 12345
			globalFlags := []string{
				"--context=kind-experiment",
				"--as=john doe",
				"--as-group=my group",
			}
			proxyCommand := ssh.BuildProxyCommandOption(fakeKind, fakeNamespace, fakeName, sshPort, globalFlags)
			expected := fmt.Sprintf("ProxyCommand=%s '--context=kind-experiment' '--as=john doe' '--as-group=my group' port-forward --stdio=true %s/%s/%s %d",
				os.Args[0], fakeKind, fakeName, fakeNamespace, sshPort)
			Expect(proxyCommand).To(Equal(expected))
		})

		It("LocalClientCmd", func() {
			opts := ssh.DefaultSSHOptions()
			opts.SSHPort = 12345
			c := ssh.NewSSH(opts)
			clientArgs := c.BuildSSHTarget(fakeKind, fakeNamespace, fakeName)
			cmd := ssh.LocalClientCmd(commandSSH, fakeKind, fakeNamespace, fakeName, opts, nil, clientArgs)
			Expect(cmd).ToNot(BeNil())
			Expect(cmd.Args).To(HaveLen(4))
			Expect(cmd.Args[0]).To(Equal(commandSSH))
			Expect(cmd.Args[2]).To(Equal(ssh.BuildProxyCommandOption(fakeKind, fakeNamespace, fakeName, opts.SSHPort, nil)))
			Expect(cmd.Args[3]).To(Equal(c.BuildSSHTarget(fakeKind, fakeNamespace, fakeName)[0]))
		})

		It("LocalClientCmd with global flags embeds them in ProxyCommand", func() {
			opts := ssh.DefaultSSHOptions()
			opts.SSHPort = 12345
			c := ssh.NewSSH(opts)
			clientArgs := c.BuildSSHTarget(fakeKind, fakeNamespace, fakeName)
			globalFlags := []string{"--context=my-context", "--kubeconfig=/path/to/config"}
			cmd := ssh.LocalClientCmd(commandSSH, fakeKind, fakeNamespace, fakeName, opts, globalFlags, clientArgs)
			Expect(cmd).ToNot(BeNil())
			Expect(cmd.Args).To(HaveLen(4))
			Expect(cmd.Args[0]).To(Equal(commandSSH))
			Expect(cmd.Args[1]).To(Equal("-o"))
			Expect(cmd.Args[2]).To(Equal(ssh.BuildProxyCommandOption(fakeKind, fakeNamespace, fakeName, opts.SSHPort, globalFlags)))
			Expect(cmd.Args[3]).To(Equal(c.BuildSSHTarget(fakeKind, fakeNamespace, fakeName)[0]))
		})
	})

	Describe("GlobalFlagsFromCmd", func() {
		var (
			root  *cobra.Command
			child *cobra.Command
		)

		BeforeEach(func() {
			root = &cobra.Command{Use: "root"}
			root.PersistentFlags().String("context", "", "kube context")
			root.PersistentFlags().String("kubeconfig", "", "kubeconfig path")
			root.PersistentFlags().StringSlice("as-groups", nil, "groups to impersonate")
			root.PersistentFlags().Bool("insecure-skip-tls-verify", false, "skip TLS verification")

			child = &cobra.Command{
				Use:  commandSSH,
				RunE: func(cmd *cobra.Command, args []string) error { return nil },
			}
			child.Flags().String("local-flag", "", "local only flag")
			root.AddCommand(child)
		})

		It("returns empty when no flags are set", func() {
			root.SetArgs([]string{commandSSH})
			Expect(root.Execute()).To(Succeed())
			Expect(ssh.GlobalFlagsFromCmd(child)).To(BeEmpty())
		})

		It("extracts root persistent string flags that are set", func() {
			root.SetArgs([]string{"--context=my-context", "--kubeconfig=/path/to/config", commandSSH})
			Expect(root.Execute()).To(Succeed())
			Expect(ssh.GlobalFlagsFromCmd(child)).To(ConsistOf(
				"--context=my-context",
				"--kubeconfig=/path/to/config",
			))
		})

		It("excludes command-local flags from output", func() {
			root.SetArgs([]string{commandSSH, "--local-flag=value"})
			Expect(root.Execute()).To(Succeed())
			Expect(ssh.GlobalFlagsFromCmd(child)).To(BeEmpty())
		})

		It("expands slice flags into individual entries", func() {
			root.SetArgs([]string{"--as-groups=group1", "--as-groups=group2", commandSSH})
			Expect(root.Execute()).To(Succeed())
			Expect(ssh.GlobalFlagsFromCmd(child)).To(ConsistOf(
				"--as-groups=group1",
				"--as-groups=group2",
			))
		})

		It("does not include unset boolean persistent flags", func() {
			root.SetArgs([]string{commandSSH})
			Expect(root.Execute()).To(Succeed())
			Expect(ssh.GlobalFlagsFromCmd(child)).ToNot(ContainElement("--insecure-skip-tls-verify=true"))
			Expect(ssh.GlobalFlagsFromCmd(child)).ToNot(ContainElement("--insecure-skip-tls-verify=false"))
		})

		It("includes boolean persistent flags when set to true", func() {
			root.SetArgs([]string{"--insecure-skip-tls-verify=true", commandSSH})
			Expect(root.Execute()).To(Succeed())
			Expect(ssh.GlobalFlagsFromCmd(child)).To(ContainElement("--insecure-skip-tls-verify=true"))
		})
	})
})
