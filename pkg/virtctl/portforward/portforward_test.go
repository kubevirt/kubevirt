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
		Entry("only name", "testvmi", "", "testvmi", "vmi", ""),
		Entry("name and namespace", "testvmi.default", "default", "testvmi", "vmi", ""),
		Entry("kind vmi with name", "vmi/testvmi", "", "testvmi", "vmi", ""),
		Entry("kind vmi with name and namespace", "vmi/testvmi.default", "default", "testvmi", "vmi", ""),
		Entry("kind vm with name", "vm/testvm", "", "testvm", "vm", ""),
		Entry("kind vm with name and namespace", "vm/testvm.default", "default", "testvm", "vm", ""),
		Entry("kind invalid with name and namespace", "invalid/testvm.default", "", "", "", "unsupported resource kind invalid"),
		Entry("name with separator but missing namespace", "testvm.", "", "", "", "expected namespace after '.'"),
		Entry("namespace with separator but missing name", ".default", "", "", "", "expected name before '.'"),
		Entry("only valid kind", "vmi/", "", "", "", "expected name after '/'"),
		Entry("only separators", "/.", "", "", "", "unsupported resource kind "),
		Entry("only dot", ".", "", "", "", "expected name before '.'"),
		Entry("only slash", "/", "", "", "", "unsupported resource kind "),
	)
})
