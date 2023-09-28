package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Instancetypes", func() {
	It("should successfully fetch and decode VirtualMachineClusterInstancetypes", func() {
		instancetypes, err := NewClusterInstancetypes()
		Expect(err).ToNot(HaveOccurred())
		Expect(instancetypes).ToNot(BeEmpty())
	})

	It("should successfully fetch and decode VirtualMachineClusterPreferences", func() {
		preferences, err := NewClusterPreferences()
		Expect(err).ToNot(HaveOccurred())
		Expect(preferences).ToNot(BeEmpty())
	})
})
