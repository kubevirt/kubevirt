package templates_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

var _ = Describe("Target", func() {

	DescribeTable("ParseTarget", func(arg, targetNamespace, targetName, targetKind string, success bool) {
		kind, namespace, name, err := templates.ParseTarget(arg)
		Expect(namespace).To(Equal(targetNamespace))
		Expect(name).To(Equal(targetName))
		Expect(kind).To(Equal(targetKind))
		if success {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
		}
	},
		Entry("only name", "testvmi", "", "testvmi", "vmi", true),
		Entry("name and namespace", "testvmi.default", "default", "testvmi", "vmi", true),
		Entry("kind vmi with name", "vmi/testvmi", "", "testvmi", "vmi", true),
		Entry("kind vmi with name and namespace", "vmi/testvmi.default", "default", "testvmi", "vmi", true),
		Entry("kind vm with name", "vm/testvm", "", "testvm", "vm", true),
		Entry("kind vm with name and namespace", "vm/testvm.default", "default", "testvm", "vm", true),
		Entry("kind invalid with name and namespace", "invalid/testvm.default", "", "", "", false),
		Entry("name with separator but missing namespace", "testvm.", "", "", "", false),
		Entry("namespace with separator but missing name", ".default", "", "", "", false),
		Entry("only valid kind", "vmi/", "", "", "", false),
		Entry("only separators", "/.", "", "", "", false),
		Entry("only dot", ".", "", "", "", false),
		Entry("only slash", "/", "", "", "", false),
	)
	DescribeTable("ParseSSHTarget", func(arg, targetNamespace, targetName, targetKind, targetUsername string, success bool) {
		kind, namespace, name, username, err := templates.ParseSSHTarget(arg)
		Expect(namespace).To(Equal(targetNamespace))
		Expect(name).To(Equal(targetName))
		Expect(kind).To(Equal(targetKind))
		Expect(username).To(Equal(targetUsername))
		if success {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
		}
	},
		Entry("username and name", "user@testvmi", "", "testvmi", "vmi", "user", true),
		Entry("username and name and namespace", "user@testvmi.default", "default", "testvmi", "vmi", "user", true),
		Entry("kind vmi with name and username", "user@vmi/testvmi", "", "testvmi", "vmi", "user", true),
		Entry("kind vmi with name and namespace and username", "user@vmi/testvmi.default", "default", "testvmi", "vmi", "user", true),
		Entry("only username", "user@", "", "", "", "", false),
		Entry("only at and target", "@testvmi", "", "", "", "", false),
		Entry("only separators", "@/.", "", "", "", "", false),
		Entry("only at", "@", "", "", "", "", false),
	)
})
