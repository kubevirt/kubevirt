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
 *
 */

package callback_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"libvirt.org/go/libvirtxml"

	"kubevirt.io/kubevirt/cmd/sidecars/network-vhostuserdra-binding/callback"
)

var _ = Describe("vhostuser hook callback handler", func() {
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
			domainXML, err := (&libvirtxml.Domain{Name: "test"}).Marshal()
			Expect(err).ToNot(HaveOccurred())

			expectedErr := fmt.Errorf("test error")
			domSpecMutator := mutatorStub{failMutate: expectedErr}

			_, err = callback.OnDefineDomain([]byte(domainXML), domSpecMutator)
			Expect(err).To(Equal(expectedErr))
		})

		It("given no-op mutator, domain spec should not change", func() {
			domain := &libvirtxml.Domain{Name: "test"}
			domainXML, err := domain.Marshal()
			Expect(err).ToNot(HaveOccurred())

			domSpecMutator := mutatorStub{domSpec: domain}

			Expect(callback.OnDefineDomain([]byte(domainXML), domSpecMutator)).To(Equal([]byte(domainXML)))
		})

		It("domain spec should mutate successfully", func() {
			domain := &libvirtxml.Domain{Name: "test"}
			domainXML, err := domain.Marshal()
			Expect(err).ToNot(HaveOccurred())

			mutatedDomain := &libvirtxml.Domain{
				Name: "test",
				Devices: &libvirtxml.DomainDeviceList{
					Interfaces: []libvirtxml.DomainInterface{
						{Alias: &libvirtxml.DomainAlias{Name: "ua-test"}},
					},
				},
			}
			domSpecMutator := mutatorStub{domSpec: mutatedDomain}

			mutatedDomainXML, err := mutatedDomain.Marshal()
			Expect(err).ToNot(HaveOccurred())

			Expect(callback.OnDefineDomain([]byte(domainXML), domSpecMutator)).To(Equal([]byte(mutatedDomainXML)))
		})
	})
})

type mutatorStub struct {
	domSpec    *libvirtxml.Domain
	failMutate error
}

func (s mutatorStub) Mutate(_ *libvirtxml.Domain) (*libvirtxml.Domain, error) {
	return s.domSpec, s.failMutate
}
