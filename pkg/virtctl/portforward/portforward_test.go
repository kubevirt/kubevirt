package portforward_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/portforward"
)

var _ = Describe("Port forward", func() {
	DescribeTable("ParseTarget", func(arg, targetNamespace, targetName, targetKind, expectedError string) {
		kind, namespace, name, err := portforward.ParseTarget(arg)
		Expect(namespace).To(Equal(targetNamespace))
		Expect(name).To(Equal(targetName))
		Expect(kind).To(Equal(targetKind))
		if expectedError == "" {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(MatchError(expectedError))
		}
	},
		Entry("kind vmi with name", "vmi/testvmi", "", "testvmi", "vmi", ""),
		Entry("kind vmi with name and namespace", "vmi/testvmi/default", "default", "testvmi", "vmi", ""),
		Entry("kind vm with name", "vm/testvm", "", "testvm", "vm", ""),
		Entry("kind vm with name and namespace", "vm/testvm/default", "default", "testvm", "vm", ""),
		Entry("name with dots and namespace", "vmi/testvmi.with.dots/default", "default", "testvmi.with.dots", "vmi", ""),
		Entry("name and namespace with dots", "vmi/testvmi/default.with.dots", "default.with.dots", "testvmi", "vmi", ""),
		Entry("name with dots and namespace with dots", "vmi/testvmi.with.dots/default.with.dots", "default.with.dots", "testvmi.with.dots", "vmi", ""),
		Entry("no slash", "testvmi", "", "", "", "target must contain type and name separated by '/'"),
		Entry("empty namespace", "vmi/testvmi/", "", "", "", "namespace cannot be empty"),
		Entry("more than three slashes", "vmi/testvmi/default/something", "", "", "", "target is not valid with more than two '/'"),
		Entry("invalid type with name", "invalid/testvmi", "", "", "", "unsupported resource type 'invalid'"),
		Entry("invalid type with name and namespace", "invalid/testvmi/default", "", "", "", "unsupported resource type 'invalid'"),
		Entry("only valid kind", "vmi/", "", "", "", "name cannot be empty"),
		Entry("empty target", "", "", "", "", "target cannot be empty"),
		Entry("only slash", "/", "", "", "", "unsupported resource type ''"),
		Entry("two slashes", "//", "", "", "", "namespace cannot be empty"),
		Entry("only dot", ".", "", "", "", "target must contain type and name separated by '/'"),
		Entry("only separators", "/.", "", "", "", "unsupported resource type ''"),
		// Normalization of type
		Entry("kind vmi", "vmi/testvmi", "", "testvmi", "vmi", ""),
		Entry("kind vmis", "vmis/testvmi", "", "testvmi", "vmi", ""),
		Entry("kind virtualmachineinstance", "virtualmachineinstance/testvmi", "", "testvmi", "vmi", ""),
		Entry("kind virtualmachineinstances", "virtualmachineinstances/testvmi", "", "testvmi", "vmi", ""),
		Entry("kind vm", "vm/testvm", "", "testvm", "vm", ""),
		Entry("kind vms", "vms/testvm", "", "testvm", "vm", ""),
		Entry("kind virtualmachine", "virtualmachine/testvm", "", "testvm", "vm", ""),
		Entry("kind virtualmachines", "virtualmachines/testvm", "", "testvm", "vm", ""),
		// Before 1.7 these syntaxes resulted in the last part after a dot being parsed as the namespace.
		// This was changed in 1.7 to not parse as namespace anymore.
		Entry("name with dots", "vmi/testvmi.with.dots", "", "testvmi.with.dots", "vmi", ""),
		Entry("kind vmi with name with dot", "vmi/testvmi.default", "", "testvmi.default", "vmi", ""),
		Entry("kind vm with name with dot", "vm/testvm.default", "", "testvm.default", "vm", ""),
	)
})
