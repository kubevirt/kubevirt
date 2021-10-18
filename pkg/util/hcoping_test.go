package util

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("test hco ping", func() {
	Context("test hcoChecker", func() {
		It("should return no error", func() {
			Expect(GetHcoPing()(nil)).ToNot(HaveOccurred())
		})
	})
})
