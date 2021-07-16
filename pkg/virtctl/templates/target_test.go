package templates_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

var _ = Describe("Target", func() {

	table.DescribeTable("ParseTarget", func(arg, targetNamespace, targetName, targetKind string, success bool) {
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
		table.Entry("only name", "testvmi", "", "testvmi", "vmi", true),
		table.Entry("name and namespace", "testvmi.default", "default", "testvmi", "vmi", true),
		table.Entry("kind vmi with name", "vmi/testvmi", "", "testvmi", "vmi", true),
		table.Entry("kind vmi with name and namespace", "vmi/testvmi.default", "default", "testvmi", "vmi", true),
		table.Entry("kind vm with name", "vm/testvm", "", "testvm", "vm", true),
		table.Entry("kind vm with name and namespace", "vm/testvm.default", "default", "testvm", "vm", true),
		table.Entry("kind invalid with name and namespace", "invalid/testvm.default", "", "", "", false),
		table.Entry("name with separator but missing namespace", "testvm.", "", "", "", false),
		table.Entry("namespace with separator but missing name", ".default", "", "", "", false),
		table.Entry("only valid kind", "vmi/", "", "", "", false),
		table.Entry("only separators", "/.", "", "", "", false),
		table.Entry("only dot", ".", "", "", "", false),
		table.Entry("only slash", "/", "", "", "", false),
	)
	table.DescribeTable("ParseSSHTarget", func(arg, targetNamespace, targetName, targetKind, targetUsername string, success bool) {
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
		table.Entry("username and name", "user@testvmi", "", "testvmi", "vmi", "user", true),
		table.Entry("username and name and namespace", "user@testvmi.default", "default", "testvmi", "vmi", "user", true),
		table.Entry("kind vmi with name and username", "user@vmi/testvmi", "", "testvmi", "vmi", "user", true),
		table.Entry("kind vmi with name and namespace and username", "user@vmi/testvmi.default", "default", "testvmi", "vmi", "user", true),
		table.Entry("only username", "user@", "", "", "", "", false),
		table.Entry("only at and target", "@testvmi", "", "", "", "", false),
		table.Entry("only separators", "@/.", "", "", "", "", false),
		table.Entry("only at", "@", "", "", "", "", false),
	)
})
