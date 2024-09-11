package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("test", func() {
	It("dummy test", func() {
		Expect(1).To(Equal(1))
	})
})
