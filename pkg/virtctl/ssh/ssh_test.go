package ssh_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/ssh"
)

var _ = Describe("SSH", func() {
	DescribeTable("ParseSSHTarget", func(arg, targetNamespace, targetName, targetKind, targetUsername, expectedError string) {
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
		Entry("username and name", "user@testvmi", "", "testvmi", "vmi", "user", ""),
		Entry("username and name and namespace", "user@testvmi.default", "default", "testvmi", "vmi", "user", ""),
		Entry("kind vmi with name and username", "user@vmi/testvmi", "", "testvmi", "vmi", "user", ""),
		Entry("kind vmi with name and namespace and username", "user@vmi/testvmi.default", "default", "testvmi", "vmi", "user", ""),
		Entry("only username", "user@", "", "", "", "", "expected target after '@'"),
		Entry("only at and target", "@testvmi", "", "", "", "", "expected username before '@'"),
		Entry("only separators", "@/.", "", "", "", "", "expected username before '@'"),
		Entry("only at", "@", "", "", "", "", "expected username before '@'"),
	)
})
