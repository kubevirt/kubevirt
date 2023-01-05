package util

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("test hco ping", func() {
	Context("test hcoChecker", func() {
		It("should return no error", func() {
			Expect(GetHcoPing()(nil)).To(Succeed())
		})
	})
})
