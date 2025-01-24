package ssh_test

import (
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
		Entry("name", "testvmi", "", "testvmi", "vmi", "", ""),
		Entry("name and namespace", "testvmi.default", "default", "testvmi", "vmi", "", ""),
		Entry("name with dot and namespace", "testvmi.dot.default", "default", "testvmi.dot", "vmi", "", ""),
		Entry("name with dots and namespace", "testvmi.with.dots.default", "default", "testvmi.with.dots", "vmi", "", ""),
		Entry("username and name", "user@testvmi", "", "testvmi", "vmi", "user", ""),
		Entry("username and name and namespace", "user@testvmi.default", "default", "testvmi", "vmi", "user", ""),
		Entry("username and name with dot and namespace", "user@testvmi.dot.default", "default", "testvmi.dot", "vmi", "user", ""),
		Entry("username and name with dots and namespace", "user@testvmi.with.dots.default", "default", "testvmi.with.dots", "vmi", "user", ""),
		Entry("kind vmi with name", "vmi/testvmi", "", "testvmi", "vmi", "", ""),
		Entry("kind vmi with name and namespace", "vmi/testvmi.default", "default", "testvmi", "vmi", "", ""),
		Entry("kind vmi with name and username", "user@vmi/testvmi", "", "testvmi", "vmi", "user", ""),
		Entry("kind vmi with name and namespace and username", "user@vmi/testvmi.default", "default", "testvmi", "vmi", "user", ""),
		Entry("kind vmi with name with dot and namespace and username", "user@vmi/testvmi.dot.default", "default", "testvmi.dot", "vmi", "user", ""),
		Entry("kind vmi with name with dots and namespace and username", "user@vmi/testvmi.with.dots.default", "default", "testvmi.with.dots", "vmi", "user", ""),
		Entry("kind vm with name", "vm/testvm", "", "testvm", "vm", "", ""),
		Entry("kind vm with name and namespace", "vm/testvm.default", "default", "testvm", "vm", "", ""),
		Entry("kind vm with name and username", "user@vm/testvm", "", "testvm", "vm", "user", ""),
		Entry("kind vm with name and namespace and username", "user@vm/testvm.default", "default", "testvm", "vm", "user", ""),
		Entry("kind vm with name with dot and namespace and username", "user@vm/testvm.dot.default", "default", "testvm.dot", "vm", "user", ""),
		Entry("kind vm with name with dots and namespace and username", "user@vm/testvm.with.dots.default", "default", "testvm.with.dots", "vm", "user", ""),
		Entry("only valid kind", "vmi/", "", "", "", "", "expected name after '/'"),
		Entry("only dot", ".", "", "", "", "", "expected name before '.'"),
		Entry("only slash", "/", "", "", "", "", "unsupported resource kind "),
		Entry("only separators", "/.", "", "", "", "", "unsupported resource kind "),
		Entry("only separators and at", "@/.", "", "", "", "", "expected username before '@'"),
		Entry("only username", "user@", "", "", "", "", "expected target after '@'"),
		Entry("only at", "@", "", "", "", "", "expected username before '@'"),
		Entry("only at and target", "@testvmi", "", "", "", "", "expected username before '@'"),
	)
})
