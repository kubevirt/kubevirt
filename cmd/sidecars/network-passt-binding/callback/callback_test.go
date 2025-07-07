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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package callback_test

import (
	"encoding/xml"
	"fmt"

	"kubevirt.io/kubevirt/cmd/sidecars/network-passt-binding/callback"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("passt hook callback handler", func() {
	Context("on define domain", func() {
		It("should fail given empty byte slice stream", func() {
			_, err := callback.OnDefineDomain([]byte{}, mutatorStub{})
			Expect(err).To(HaveOccurred())
		})

		It("should fail given invalid domain XML", func() {
			_, err := callback.OnDefineDomain([]byte("invalid-domain-xml"), mutatorStub{})
			Expect(err).To(HaveOccurred())
		})

		It("should fail when domain spec mutator fails", func() {
			domain := domainschema.NewMinimalDomain("test")
			domainXML, err := xml.Marshal(domain.Spec)
			Expect(err).ToNot(HaveOccurred())

			expectedErr := fmt.Errorf("test error")
			domSpecMutator := mutatorStub{failMutate: expectedErr}

			_, err = callback.OnDefineDomain(domainXML, domSpecMutator)
			Expect(err).To(Equal(expectedErr))
		})

		It("given no-op mutator, domain spec should not change", func() {
			domain := domainschema.NewMinimalDomain("test")
			domainSpecXML, err := xml.Marshal(domain.Spec)
			Expect(err).ToNot(HaveOccurred())

			domSpecMutator := mutatorStub{domSpec: &domain.Spec}

			Expect(callback.OnDefineDomain(domainSpecXML, domSpecMutator)).To(Equal(domainSpecXML))
		})

		It("domain spec should mutate successfully", func() {
			domain := domainschema.NewMinimalDomain("test")
			domainSpecXML, err := xml.Marshal(domain.Spec)
			Expect(err).ToNot(HaveOccurred())

			mutatedDomainSpec := domain.Spec.DeepCopy()
			mutatedDomainSpec.Devices.Interfaces = append(mutatedDomainSpec.Devices.Interfaces,
				domainschema.Interface{Alias: domainschema.NewUserDefinedAlias("test")})
			domSpecMutator := mutatorStub{domSpec: mutatedDomainSpec}

			mutatedDomainSpecXML, err := xml.Marshal(mutatedDomainSpec)
			Expect(err).ToNot(HaveOccurred())

			Expect(callback.OnDefineDomain(domainSpecXML, domSpecMutator)).To(Equal(mutatedDomainSpecXML))
		})
	})
})

type mutatorStub struct {
	domSpec    *domainschema.DomainSpec
	failMutate error
}

func (s mutatorStub) Mutate(_ *domainschema.DomainSpec) (*domainschema.DomainSpec, error) {
	return s.domSpec, s.failMutate
}
