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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package dhcpd_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vmschema "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/cmd/sidecars/network-bridge-binding/dhcpd"
	"reflect"
	"strings"
)

var _ = Describe("Server", func() {
	Describe("GenerateMac", func() {
		Context("with various VMIs", func() {
			type test struct {
				name string
				vmi  *vmschema.VirtualMachineInstance
				want string
			}

			tests := []test{
				{
					name: "Test with UID 1",
					vmi:  vmschema.NewVMI("test", "123456"),
					want: "52:54:7c:4a:8d:09",
				},
				{
					name: "Test with UID 2",
					vmi:  vmschema.NewVMI("test", "abcdef"),
					want: "52:54:1f:8a:c1:0f",
				},
			}

			for _, tt := range tests {
				tt := tt
				It(tt.name, func() {
					server := &dhcpd.Server{}

					got := server.GenerateMac(tt.vmi)
					Expect(reflect.TypeOf(got).String()).To(Equal("net.HardwareAddr"), "Expected type net.HardwareAddr")
					Expect(got.String()).To(Equal(tt.want), "Expected MAC address to match")
					Expect(strings.HasPrefix(got.String(), "52:54")).To(BeTrue(), "Expected MAC address to have the correct prefix")
				})
			}
		})

		Context("idempotency check", func() {
			It("should return the same MAC address when called multiple times", func() {
				server := &dhcpd.Server{}
				vmi := vmschema.NewVMI("test", "12345-67890")

				// Generate the MAC address
				got := server.GenerateMac(vmi)
				got2nd := server.GenerateMac(vmi)

				Expect(got2nd).To(Equal(got), "Expected MAC address generation to be idempotent")
			})
		})
	})
})
