package api

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Defaults", func() {

	It("should set q35 machine type", func() {
		domain := &Domain{}
		SetDefaults_OSType(&domain.Spec.OS.Type)
		Expect(domain.Spec.OS.Type.Machine).To(Equal("q35"))
	})
})
