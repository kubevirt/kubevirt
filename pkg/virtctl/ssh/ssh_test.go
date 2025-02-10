package ssh_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/ssh"
)

var _ = Describe("SSH", func() {
	DescribeTable("ParseTarget", func(arg, targetNamespace, targetName, targetUsername, expectedError string) {
		namespace, name, username, err := ssh.ParseTarget(arg)
		Expect(namespace).To(Equal(targetNamespace))
		Expect(name).To(Equal(targetName))
		Expect(username).To(Equal(targetUsername))
		if expectedError == "" {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(MatchError(expectedError))
		}
	},
		Entry("username and name", "user@testvmi", "", "testvmi", "user", ""),
		Entry("username and dot after name", "user@testvmi.", "", "testvmi.", "user", ""),
		Entry("username and dot before name", "user@.testvmi", "", ".testvmi", "user", ""),
		Entry("username and name and namespace", "user@default/testvmi", "default", "testvmi", "user", ""),
		Entry("username and name with dot and namespace", "user@default/testvmi.dot", "default", "testvmi.dot", "user", ""),
		Entry("username and name with dots and namespace", "user@default/testvmi.with.dots", "default", "testvmi.with.dots", "user", ""),
		Entry("username and only dot", "user@.", "", ".", "user", ""),
		Entry("empty target", "", "", "", "", "target cannot be empty or expected target after '@'"),
		Entry("only separators and at", "@/.", "", "", "", "expected username before '@'"),
		Entry("only username", "user@", "", "", "", "target cannot be empty or expected target after '@'"),
		Entry("only at", "@", "", "", "", "expected username before '@'"),
		Entry("only at and target", "@testvmi", "", "", "", "expected username before '@'"),
		// These cases should work the same as for portforward.ParseTarget
		Entry("only name", "testvmi", "", "testvmi", "", ""),
		Entry("dot after name", "testvmi.", "", "testvmi.", "", ""),
		Entry("dot before name", ".testvmi", "", ".testvmi", "", ""),
		Entry("name and namespace", "default/testvmi", "default", "testvmi", "", ""),
		Entry("name with dot and namespace", "default/testvmi.dot", "default", "testvmi.dot", "", ""),
		Entry("name with dots and namespace", "default/testvmi.with.dots", "default", "testvmi.with.dots", "", ""),
		Entry("only dot", ".", "", ".", "", ""),
		Entry("only slash", "/", "", "", "", "namespace cannot be empty"),
		Entry("empty namespace before slash", "/testvm", "", "", "", "namespace cannot be empty"),
		Entry("empty name after slash", "default/", "", "", "", "name cannot be empty or expected name after '/'"),
		Entry("more than one slash", "namespace/name/something", "", "", "", "target is not valid with more than one '/'"),
		// Legacy syntax test cases
		Entry("only reserved namespace vmi", "vmi/", "", "", "", "name cannot be empty or expected name after '/'"),
		Entry("only reserved namespace vm", "vm/", "", "", "", "name cannot be empty or expected name after '/'"),
		Entry("reserved namespace vmi with name", "vmi/testvmi", "", "testvmi", "", ""),
		Entry("reserved namespace vmi with name and namespace", "vmi/testvmi.default", "default", "testvmi", "", ""),
		Entry("reserved namespace vmi with name and username", "user@vmi/testvmi", "", "testvmi", "user", ""),
		Entry("reserved namespace vmi with name and namespace and username", "user@vmi/testvmi.default", "default", "testvmi", "user", ""),
		Entry("reserved namespace vmi with name with dot and namespace and username", "user@vmi/testvmi.dot.default", "default", "testvmi.dot", "user", ""),
		Entry("reserved namespace vmi with name with dots and namespace and username", "user@vmi/testvmi.with.dots.default", "default", "testvmi.with.dots", "user", ""),
		Entry("reserved namespace vm with name", "vm/testvm", "", "testvm", "", ""),
		Entry("reserved namespace vm with name and namespace", "vm/testvm.default", "default", "testvm", "", ""),
		Entry("reserved namespace vm with name and username", "user@vm/testvm", "", "testvm", "user", ""),
		Entry("reserved namespace vm with name and namespace and username", "user@vm/testvm.default", "default", "testvm", "user", ""),
		Entry("reserved namespace vm with name with dot and namespace and username", "user@vm/testvm.dot.default", "default", "testvm.dot", "user", ""),
		Entry("reserved namespace vm with name with dots and namespace and username", "user@vm/testvm.with.dots.default", "default", "testvm.with.dots", "user", ""),
	)
})
