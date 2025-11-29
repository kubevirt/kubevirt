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

package converter_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

var _ = Describe("Domain Builder", func() {
	It("Should succeed when all configurators succeed", func() {
		configurator1 := newStubConfigurator(nil)
		configurator2 := newStubConfigurator(nil)

		builder := converter.NewDomainBuilder(configurator1, configurator2)

		vmi := libvmi.New()
		var domain api.Domain

		Expect(builder.Build(vmi, &domain)).To(Succeed())
	})

	It("Should fail when a single configurator fails", func() {
		expectedError := errors.New("some error")
		configurator1 := newStubConfigurator(
			func(_ *api.Domain) error { return expectedError },
		)

		configurator2 := newStubConfigurator(nil)

		builder := converter.NewDomainBuilder(configurator1, configurator2)

		vmi := libvmi.New()
		var domain api.Domain

		Expect(builder.Build(vmi, &domain)).To(MatchError(expectedError))
	})

	It("Should allow all configurators to configure the domain", func() {
		const expectedIfaceName = "iface1"
		expectedInterfaces := []api.Interface{{Alias: api.NewUserDefinedAlias(expectedIfaceName)}}
		configurator1 := newStubConfigurator(
			func(domain *api.Domain) error {
				domain.Spec.Devices.Interfaces = expectedInterfaces
				return nil
			},
		)

		const expectedDomainSpecName = "domain"
		configurator2 := newStubConfigurator(
			func(domain *api.Domain) error {
				domain.Spec.Name = expectedDomainSpecName
				return nil
			},
		)

		vmi := libvmi.New()
		var domain api.Domain

		builder := converter.NewDomainBuilder(configurator1, configurator2)
		Expect(builder.Build(vmi, &domain)).To(Succeed())

		Expect(domain.Spec.Devices.Interfaces).To(Equal(expectedInterfaces))
		Expect(domain.Spec.Name).To(Equal(expectedDomainSpecName))
	})

	It("The last configurator should win if both set the same domain field", func() {
		const firstDomainSpecName = "first"
		configurator1 := newStubConfigurator(
			func(domain *api.Domain) error {
				domain.Spec.Name = firstDomainSpecName
				return nil
			},
		)

		const secondDomainSpecName = "second"
		configurator2 := newStubConfigurator(
			func(domain *api.Domain) error {
				domain.Spec.Name = secondDomainSpecName
				return nil
			},
		)

		vmi := libvmi.New()
		var domain api.Domain

		builder := converter.NewDomainBuilder(configurator1, configurator2)
		Expect(builder.Build(vmi, &domain)).To(Succeed())
		Expect(domain.Spec.Name).To(Equal(secondDomainSpecName))
	})
})

type stubConfigurator struct {
	f func(*api.Domain) error
}

func newStubConfigurator(f func(*api.Domain) error) stubConfigurator {
	return stubConfigurator{f: f}
}

func (sc stubConfigurator) Configure(_ *v1.VirtualMachineInstance, domain *api.Domain) error {
	if f := sc.f; f != nil {
		return f(domain)
	}

	return nil
}
