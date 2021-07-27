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
 * Copyright 2021
 *
 */

package launchsecurity_test

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/launchsecurity"
)

var _ = Describe("LaunchSecurity: AMD Secure Encrypted Virtualization (SEV)", func() {
	Context("SEV capabilities detection", func() {
		var ctrl *gomock.Controller
		var virsh *launchsecurity.MockVirsh

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virsh = launchsecurity.NewMockVirsh(ctrl)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return a valid configuration when SEV is supported", func() {
			virsh.EXPECT().Domcapabilities().DoAndReturn(func() ([]byte, error) {
				bytes, err := ioutil.ReadFile("testdata/domcapabilities_sev.xml")
				Expect(err).ToNot(HaveOccurred())
				return bytes, err
			})
			sevConfiguration, err := launchsecurity.QuerySEVConfiguration(virsh)
			Expect(err).ToNot(HaveOccurred())
			Expect(sevConfiguration).ToNot(BeNil())
			Expect(sevConfiguration.Supported).To(Equal("yes"))
			Expect(sevConfiguration.Cbitpos).To(Equal("47"))
			Expect(sevConfiguration.ReducedPhysBits).To(Equal("1"))
		})

		It("should return an empty configuration when SEV is not supported", func() {
			virsh.EXPECT().Domcapabilities().DoAndReturn(func() ([]byte, error) {
				bytes, err := ioutil.ReadFile("testdata/domcapabilities_nosev.xml")
				Expect(err).ToNot(HaveOccurred())
				return bytes, err
			})
			sevConfiguration, err := launchsecurity.QuerySEVConfiguration(virsh)
			Expect(err).ToNot(HaveOccurred())
			Expect(sevConfiguration).ToNot(BeNil())
			Expect(sevConfiguration.Supported).To(Equal("no"))
			Expect(sevConfiguration.Cbitpos).To(BeEmpty())
			Expect(sevConfiguration.ReducedPhysBits).To(BeEmpty())
		})

		It("should return an error when domain capabilities cannot be fetched", func() {
			virsh.EXPECT().Domcapabilities().DoAndReturn(func() ([]byte, error) {
				return nil, fmt.Errorf("error")
			})
			sevConfiguration, err := launchsecurity.QuerySEVConfiguration(virsh)
			Expect(err).To(HaveOccurred())
			Expect(sevConfiguration).To(BeNil())
		})
	})
})
