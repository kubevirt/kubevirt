package components

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Webhooks", func() {
	It("should set the right namespace on the operator webhook service", func() {
		service := NewOperatorWebhookService("testnamespace")
		Expect(service.Namespace).To(Equal("testnamespace"))
	})

	It("should set the right namespace on the operator webhook configuration", func() {
		configuration := NewOpertorValidatingWebhookConfiguration("testnamespace")
		for _, webhook := range configuration.Webhooks {
			Expect(webhook.ClientConfig.Service.Namespace).To(Equal("testnamespace"))
		}
	})

	It("should set the right namespace on the virt-api mutating webhook configurations", func() {
		configuration := NewVirtAPIMutatingWebhookConfiguration("testnamespace")
		for _, webhook := range configuration.Webhooks {
			Expect(webhook.ClientConfig.Service.Namespace).To(Equal("testnamespace"))
		}
	})

	It("should set the right namespace on the virt-api validating webhook configurations", func() {
		configuration := NewVirtAPIValidatingWebhookConfiguration("testnamespace")
		for _, webhook := range configuration.Webhooks {
			Expect(webhook.ClientConfig.Service.Namespace).To(Equal("testnamespace"))
		}
	})
})
