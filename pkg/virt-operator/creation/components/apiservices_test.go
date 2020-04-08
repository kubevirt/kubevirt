package components

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("APIServices", func() {

	It("should load one APIService with the correct namespace", func() {
		services := NewVirtAPIAPIServices("mynamespace")
		Expect(services).To(HaveLen(1))
		Expect(services[0].Spec.Service.Namespace).To(Equal("mynamespace"))
	})
})
