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
		// Legacy parsing
		Entry("name with dots", "vmi/testvmi.with.dots", "dots", "testvmi.with", "vmi", ""),
		Entry("kind vmi with name and namespace (legacy)", "vmi/testvmi.default", "default", "testvmi", "vmi", ""),
		Entry("kind vm with name and namespace (legacy)", "vm/testvm.default", "default", "testvm", "vm", ""),
	)
})
