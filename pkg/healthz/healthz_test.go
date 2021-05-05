package healthz

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Healthz", func() {
	Context("KubeApiHealthzVersion", func() {
		apiHealthVersion := KubeApiHealthzVersion{}
		testValue := "this is a test"

		It("Should return nil by default", func() {
			Expect(apiHealthVersion.GetVersion()).To(BeNil())
		})

		It("Should store a value", func() {
			apiHealthVersion.Update(testValue)
			Expect(apiHealthVersion.GetVersion()).To(Equal(testValue))
		})

		It("Should be clearable", func() {
			apiHealthVersion.Clear()
			Expect(apiHealthVersion.GetVersion()).To(BeNil())
		})
	})
})
