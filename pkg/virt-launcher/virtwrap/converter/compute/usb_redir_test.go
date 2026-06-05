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

package compute_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("USB Redirect Device Domain Configurator", func() {
	Context("When ClientPassthrough is nil", func() {
		It("should return not configure any redirect devices", func() {
			vmi := libvmi.New()
			// ClientPassthrough is nil by default
			domain := api.Domain{}

			Expect(compute.UsbRedirectDeviceDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())

			Expect(domain).To(Equal(api.Domain{}))
		})
	})

	Context("When ClientPassthrough is set", func() {
		It("should configure the maximum number of USB redirect devices", func() {
			var vmiUID types.UID = "test-vmi-uid"
			vmi := libvmi.New(libvmi.WithUID(vmiUID))
			vmi.Spec.Domain.Devices.ClientPassthrough = &v1.ClientPassthroughDevices{}
			domain := api.Domain{}

			Expect(compute.UsbRedirectDeviceDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Redirs: make([]api.RedirectedDevice, v1.UsbClientPassthroughMaxNumberOf),
					},
				},
			}
			// Populate expected redirect devices
			for i := 0; i < v1.UsbClientPassthroughMaxNumberOf; i++ {
				path := fmt.Sprintf("/var/run/kubevirt-private/%s/virt-usbredir-%d", vmiUID, i)
				expectedDomain.Spec.Devices.Redirs[i] = api.RedirectedDevice{
					Type: "unix",
					Bus:  "usb",
					Source: api.RedirectedDeviceSource{
						Mode: "bind",
						Path: path,
					},
				}
			}

			Expect(domain).To(Equal(expectedDomain))
		})
	})
})
