package portforward_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virtctl/portforward"
)

var _ = Describe("Port forward", func() {
	DescribeTable("ParseTarget", func(arg, targetNamespace, targetName, expectedError string) {
		namespace, name, err := portforward.ParseTarget(arg)
		Expect(namespace).To(Equal(targetNamespace))
		Expect(name).To(Equal(targetName))
		if expectedError == "" {
			Expect(err).NotTo(HaveOccurred())
		} else {
			Expect(err).To(MatchError(expectedError))
		}
	},
		Entry("only name", "testvmi", "", "testvmi", ""),
		Entry("dot after name", "testvm.", "", "testvm.", ""),
		Entry("dot before name", ".testvm", "", ".testvm", ""),
		Entry("name and namespace", "default/testvmi", "default", "testvmi", ""),
		Entry("name with dot and namespace", "default/testvmi.dot", "default", "testvmi.dot", ""),
		Entry("name with dots and namespace", "default/testvmi.with.dots", "default", "testvmi.with.dots", ""),
		Entry("only dot", ".", "", ".", ""),
		Entry("only slash", "/", "", "", "namespace cannot be empty"),
		Entry("empty namespace before slash", "/testvm", "", "", "namespace cannot be empty"),
		Entry("empty target", "", "", "", "name cannot be empty or expected name after '/'"),
		Entry("empty name after slash", "default/", "", "", "name cannot be empty or expected name after '/'"),
		Entry("more than one slash", "namespace/name/something", "", "", "target is not valid with more than one '/'"),
		// Legacy syntax test cases
		Entry("only reserved namespace vmi", "vmi/", "", "", "name cannot be empty or expected name after '/'"),
		Entry("only reserved namespace vm", "vm/", "", "", "name cannot be empty or expected name after '/'"),
		Entry("reserved namespace vmi with name", "vmi/testvmi", "", "testvmi", ""),
		Entry("reserved namespace vmi with name and namespace", "vmi/testvmi.default", "default", "testvmi", ""),
		Entry("reserved namespace vmi with name with dot and namespace", "vmi/testvmi.dot.default", "default", "testvmi.dot", ""),
		Entry("reserved namespace vmi with name with dots and namespace", "vmi/testvmi.with.dots.default", "default", "testvmi.with.dots", ""),
		Entry("reserved namespace vm with name", "vm/testvm", "", "testvm", ""),
		Entry("reserved namespace vm with name and namespace", "vm/testvm.default", "default", "testvm", ""),
		Entry("reserved namespace vm with name with dot and namespace", "vmi/testvm.dot.default", "default", "testvm.dot", ""),
		Entry("reserved namespace vm with name with dots and namespace", "vmi/testvm.with.dots.default", "default", "testvm.with.dots", ""),
	)
})
