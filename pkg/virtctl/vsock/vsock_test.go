package vsock_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/vsock"
)

var _ = Describe("Vsock", func() {
	It("NewCommand creates vsock command", func() {
		cmd := vsock.NewCommand()
		Expect(cmd).ToNot(BeNil())
		Expect(cmd.Use).To(ContainSubstring("vsock"))
	})

	It("NewCommand has tls flag", func() {
		cmd := vsock.NewCommand()
		Expect(cmd.Flags().Lookup("tls")).ToNot(BeNil())
	})

	It("NewCommand requires exactly 2 args", func() {
		cmd := vsock.NewCommand()
		Expect(cmd.Args(cmd, []string{})).To(MatchError(ContainSubstring("accepts 2 arg(s)")))
		Expect(cmd.Args(cmd, []string{"vmi/testvm"})).To(MatchError(ContainSubstring("accepts 2 arg(s)")))
		Expect(cmd.Args(cmd, []string{"vmi/testvm", "22", "extra"})).To(MatchError(ContainSubstring("accepts 2 arg(s)")))
	})
})
