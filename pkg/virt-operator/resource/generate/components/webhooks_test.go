/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

package components

import (
	. "github.com/onsi/ginkgo/v2"
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

	It("should make all virt-api validating webhook required", func() {
		configuration := NewVirtAPIValidatingWebhookConfiguration("testnamespace")
		for _, webhook := range configuration.Webhooks {
			Expect(*webhook.FailurePolicy).To(Equal(v1.Fail))
		}
	})
})
