package components

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("APIServices", func() {

	It("should load one APIService with the correct namespace", func() {
		services := NewVirtAPIAPIServices("mynamespace")
		// a subresource aggregated api endpoint should be registered for
		// each vm/vmi api version
		Expect(services).To(HaveLen(len(v1.SubresourceGroupVersions)))
		Expect(services[0].Spec.Service.Namespace).To(Equal("mynamespace"))
	})
})
