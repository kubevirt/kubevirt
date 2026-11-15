package ssh_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

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

	DescribeTable("GlobalFlagsFromCmd", func(args, expected []string) {
		root := &cobra.Command{Use: "virtctl"}
		root.PersistentFlags().String("context", "", "")
		root.PersistentFlags().String("kubeconfig", "", "")
		root.PersistentFlags().String("token", "", "")
		root.PersistentFlags().String("password", "", "")
		root.PersistentFlags().Bool("insecure-skip-tls-verify", false, "")
		root.PersistentFlags().StringArray("as-group", nil, "")
		child := &cobra.Command{Use: "ssh", Run: func(cmd *cobra.Command, _ []string) {
			Expect(ssh.GlobalFlagsFromCmd(cmd)).To(ConsistOf(expected))
		}}
		child.Flags().String("local-only", "", "")
		root.AddCommand(child)
		root.SetArgs(args)
		Expect(root.Execute()).To(Succeed())
	},
		Entry("omits unset flags", []string{"ssh"}, []string(nil)),
		Entry("context before subcommand", []string{"--context=selected", "ssh"}, []string{"--context=selected"}),
		Entry("context after subcommand", []string{"ssh", "--context=selected"}, []string{"--context=selected"}),
		Entry("kubeconfig", []string{"ssh", "--kubeconfig=config"}, []string{"--kubeconfig=config"}),
		Entry("token override", []string{"ssh", "--token=example"}, []string{"--token=example"}),
		Entry("empty password override", []string{"ssh", "--password="}, []string{"--password="}),
		Entry("boolean override", []string{"ssh", "--insecure-skip-tls-verify"}, []string{"--insecure-skip-tls-verify=true"}),
		Entry("explicit false", []string{"--insecure-skip-tls-verify=false", "ssh"}, []string{"--insecure-skip-tls-verify=false"}),
		Entry("repeated groups", []string{"ssh", "--as-group=one,two", "--as-group=three"}, []string{"--as-group=one,two", "--as-group=three"}),
		Entry("omits local flags", []string{"ssh", "--local-only=value"}, []string(nil)),
		Entry("password override", []string{"--password=example", "ssh"}, []string{"--password=example"}),
		Entry("empty token override", []string{"ssh", "--token="}, []string{"--token="}),
	)

	It("replaces a stale proxy payload when no flags are set", func() {
		GinkgoT().Setenv("VIRTCTL_PROXY_FLAGS", `["--token=stale"]`)
		cmd := ssh.LocalClientCmd("ssh", "vmi", "default", "test", &ssh.SSHOptions{}, nil)
		var payload string
		for _, entry := range cmd.Env {
			if strings.HasPrefix(entry, "VIRTCTL_PROXY_FLAGS=") {
				payload = strings.TrimPrefix(entry, "VIRTCTL_PROXY_FLAGS=")
			}
		}
		var flags []string
		Expect(json.Unmarshal([]byte(payload), &flags)).To(Succeed())
		Expect(flags).To(BeEmpty())
		Expect(flags).NotTo(BeNil())
	})

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

		const commandSSH = "ssh"
		DescribeTable("LocalClientCmd",
			func(kind string, opts ssh.SSHOptions, expectedProxyCmd string, globalFlags ...string) {
				c := ssh.NewSSH(&opts)
				clientArgs := c.BuildSSHTarget(kind, fakeNamespace, fakeName)
				cmd := ssh.LocalClientCmd(commandSSH, kind, fakeNamespace, fakeName, &opts, clientArgs, globalFlags...)
				Expect(cmd).ToNot(BeNil())
				Expect(cmd.Args).To(HaveExactElements(commandSSH, "-o", expectedProxyCmd, clientArgs[0]))
			},
			Entry("with context",
				fakeKind, ssh.SSHOptions{SSHPort: 12345},
				fmt.Sprintf("ProxyCommand=%s port-forward --stdio=true %s/%s/%s %d",
					os.Args[0], fakeKind, fakeName, fakeNamespace, 12345), "--context=selected"),
			Entry("without vsock",
				fakeKind, ssh.SSHOptions{SSHPort: 12345},
				fmt.Sprintf("ProxyCommand=%s port-forward --stdio=true %s/%s/%s %d",
					os.Args[0], fakeKind, fakeName, fakeNamespace, 12345)),
			Entry("with vsock and vmi kind",
				"vmi", ssh.SSHOptions{SSHPort: 12345, UseVsock: true},
				fmt.Sprintf("ProxyCommand=%s port-forward --stdio=true --vsock --vsock-tls=false vmi/%s/%s %d",
					os.Args[0], fakeName, fakeNamespace, 12345)),
			Entry("with vsock and vm kind",
				"vm", ssh.SSHOptions{SSHPort: 12345, UseVsock: true},
				fmt.Sprintf("ProxyCommand=%s port-forward --stdio=true --vsock --vsock-tls=false vm/%s/%s %d",
					os.Args[0], fakeName, fakeNamespace, 12345)),
		)
	})
})
