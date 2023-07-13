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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package apply

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"sigs.k8s.io/yaml"
)

const (
	validatingWebookYaml = `
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: test
webhooks:
- admissionReviewVersions:
  - v1
  - v1beta1
clientConfig:
  service:
    name: fake-validation-service
    namespace: kubevirt
    path: /fake-path/virtualmachineinstances.kubevirt.io
    port: 443
failurePolicy: Fail
matchPolicy: Equivalent
name: virtualmachineinstances.kubevirt.io-tmp-validator
namespaceSelector: {}
objectSelector: {}
rules:
- apiGroups:
  - kubevirt.io
  apiVersions:
  - v1alpha3
  - v1
  operations:
  - CREATE
  resources:
  - virtualmachineinstances
  scope: '*'
sideEffects: None
timeoutSeconds: 10
`
	mutatingWebhookYaml = `
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: virt-api-mutator
webhooks:
- admissionReviewVersions:
  - v1
  - v1beta1
  clientConfig:
    caBundle: deadbeef
    service:
      name: virt-api
      namespace: kubevirt
      path: /virtualmachines-mutate
      port: 443
  failurePolicy: Fail
  matchPolicy: Equivalent
  name: virtualmachines-mutator.kubevirt.io
  namespaceSelector: {}
  objectSelector: {}
  reinvocationPolicy: Never
  rules:
  - apiGroups:
    - kubevirt.io
    apiVersions:
    - v1alpha3
    - v1
    operations:
    - CREATE
    - UPDATE
    resources:
    - virtualmachines
    scope: '*'
  sideEffects: None
  timeoutSeconds: 10
`
)

var _ = Describe("WebhookConfiguration types", func() {

	Context("ValidatingWebhookConfiguration", func() {
		It("conversion from v1 to v1beta should match", func() {
			webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
			err := yaml.Unmarshal([]byte(validatingWebookYaml), &webhook)
			Expect(err).ToNot(HaveOccurred())
			binv1, err := webhook.Marshal()
			Expect(err).ToNot(HaveOccurred())
			webhookv1beta1 := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}
			err = webhookv1beta1.Unmarshal(binv1)
			Expect(err).ToNot(HaveOccurred())
			Expect(webhookv1beta1.String()).To(Equal(webhook.String()))
			binv1beta1, err := webhookv1beta1.Marshal()
			Expect(err).ToNot(HaveOccurred())
			Expect(binv1beta1).To(Equal(binv1))

			webhookv1beta1, err = convertV1ValidatingWebhookToV1beta1(webhook)
			Expect(err).ToNot(HaveOccurred())
			Expect(webhookv1beta1.String()).To(Equal(webhook.String()))
		})
	})

	Context("MutatingWebhookConfiguration", func() {
		It("conversion from v1 to v1beta should match", func() {
			webhook := &admissionregistrationv1.MutatingWebhookConfiguration{}

			webhookv1beta1, err := convertV1MutatingWebhookToV1beta1(webhook)
			Expect(err).ToNot(HaveOccurred())
			Expect(webhookv1beta1.String()).To(Equal(webhook.String()))
		})
	})
})
