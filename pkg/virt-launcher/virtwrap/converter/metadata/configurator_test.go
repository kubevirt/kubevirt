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

package metadata_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/metadata"
)

var _ = Describe("Metadata Domain Configurator", func() {
	It("Should configure Domain Metadata", func() {
		const (
			testNamespace = "testNamespace"
			testName      = "testName"
		)

		configurator := metadata.DomainConfigurator{}

		vmi := libvmi.New(
			libvmi.WithNamespace(testNamespace),
			libvmi.WithName(testName),
		)

		var domain api.Domain
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		const expectedSpecName = "testNamespace_testName"
		expectedDomain := api.Domain{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: api.DomainSpec{
				Name: expectedSpecName,
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})
})
