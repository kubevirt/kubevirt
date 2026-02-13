package naming_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/util/naming"
)

var _ = Describe("Naming", func() {
	It("should concatenate base and suffix, if maxLength is enough", func() {
		const base = "base"
		const suffix = "suffix"
		name := naming.GetName(base, suffix, 128)
		Expect(name).To(Equal(base + "-" + suffix))
	})

	It("should truncate base, if full name does not fit", func() {
		const base = "some.long.base.string.1234"
		const suffix = "some.long.suffix.string.1234"
		const fullLength = len(base) + len(suffix) + 1
		const maxLength = fullLength - 5

		name := naming.GetName(base, suffix, maxLength)
		Expect(name).To(Equal("some.long.ba-3b0abe31-" + suffix))
	})

	It("should ignore suffix, if it does not fit", func() {
		const base = "some.long.base.string.1234"
		const suffix = "some.long.suffix.string.1234"
		const fullLength = len(base) + len(suffix) + 1
		const maxLength = fullLength - 20

		name := naming.GetName(base, suffix, maxLength)
		Expect(name).To(Equal("some.long.base.string.1234-9259e0dc"))
	})
})
