package components

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/admissionregistration/v1"
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

	It("should make all virt-api mutating webhook required", func() {
		configuration := NewVirtAPIMutatingWebhookConfiguration("testnamespace")
		for _, webhook := range configuration.Webhooks {
			Expect(*webhook.FailurePolicy).To(Equal(v1.Fail))
		}
	})

	It("should make all virt-api validating webhook required, except for the eviction validator", func() {
		configuration := NewVirtAPIValidatingWebhookConfiguration("testnamespace")
		for _, webhook := range configuration.Webhooks {
			if webhook.Name == "virt-launcher-eviction-interceptor.kubevirt.io" {
				Expect(*webhook.FailurePolicy).To(Equal(v1.Ignore))
			} else {
				Expect(*webhook.FailurePolicy).To(Equal(v1.Fail))
			}
		}
	})
})
