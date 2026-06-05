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

package admitter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validate network DRA", func() {
	It("should reject DRA network when feature gate is disabled", func() {
		spec := newDRASpec()
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("vmi.spec.networks contains DRA networks but NetworkDevicesWithDRA feature gate is not enabled"))
		Expect(causes[0].Field).To(Equal("fake.networks"))
	})

	It("should accept valid DRA network when feature gate is enabled", func() {
		spec := newDRASpec()
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		Expect(causes).To(BeEmpty())
	})

	It("should reject DRA network with empty claimName", func() {
		spec := newDRASpec()
		spec.Networks[0].ResourceClaim.ClaimName = ""
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("claimName is required for DRA network"))
		Expect(causes[0].Field).To(Equal("fake.networks[0].resourceClaim.claimName"))
	})

	It("should reject DRA network with empty requestName", func() {
		spec := newDRASpec()
		spec.Networks[0].ResourceClaim.RequestName = ""
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("requestName is required for DRA network"))
		Expect(causes[0].Field).To(Equal("fake.networks[0].resourceClaim.requestName"))
	})

	It("should reject DRA network with non-existent resourceClaim reference", func() {
		spec := newDRASpec()
		spec.Networks[0].ResourceClaim.ClaimName = "missing-claim"
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal(`network references resourceClaim "missing-claim" which is not defined in spec.resourceClaims`))
		Expect(causes[0].Field).To(Equal("fake.networks[0].resourceClaim.claimName"))
	})

	It("should reject duplicate claimName/requestName across DRA networks", func() {
		spec := newDRASpec()
		spec.Domain.Devices.Interfaces = []v1.Interface{
			{
				Name: "dra-net-1",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					SRIOV: &v1.InterfaceSRIOV{},
				},
			},
			{
				Name: "dra-net-2",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					SRIOV: &v1.InterfaceSRIOV{},
				},
			},
		}
		spec.Networks = []v1.Network{
			{
				Name: "dra-net-1",
				NetworkSource: v1.NetworkSource{
					ResourceClaim: &v1.ClaimRequest{
						ClaimName:   "claim1",
						RequestName: "vf",
					},
				},
			},
			{
				Name: "dra-net-2",
				NetworkSource: v1.NetworkSource{
					ResourceClaim: &v1.ClaimRequest{
						ClaimName:   "claim1",
						RequestName: "vf",
					},
				},
			},
		}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal(`duplicate claimName/requestName combination "claim1/vf"`))
		Expect(causes[0].Field).To(Equal("fake.networks[1]"))
	})

	It("should reject mixing Multus and DRA networks", func() {
		spec := newDRASpec()
		spec.Domain.Devices.Interfaces = []v1.Interface{
			{
				Name: "multus-net",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					SRIOV: &v1.InterfaceSRIOV{},
				},
			},
			{
				Name: "dra-net",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					SRIOV: &v1.InterfaceSRIOV{},
				},
			},
		}
		spec.Networks = []v1.Network{
			{
				Name:          "multus-net",
				NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "nad1"}},
			},
			spec.Networks[0],
		}
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("mixing Multus and DRA resourceClaim networks in the same VMI is not supported"))
		Expect(causes[0].Field).To(Equal("fake.networks"))
	})

	It("should reject DRA network with non-SRIOV interface binding", func() {
		spec := newDRASpec()
		spec.Domain.Devices.Interfaces[0] = *v1.DefaultBridgeNetworkInterface()
		spec.Domain.Devices.Interfaces[0].Name = "dra-net"
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal(`DRA network "dra-net" requires an SR-IOV or binding plugin interface`))
		Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces"))
	})

	It("should accept DRA network with plugin interface binding", func() {
		spec := newDRASpec()
		spec.Domain.Devices.Interfaces = []v1.Interface{
			{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Masquerade: &v1.InterfaceMasquerade{},
				},
			},
			{
				Name: "dra-net",
				Binding: &v1.PluginBinding{
					Name: "vhostuser",
				},
			},
		}
		spec.Networks = append(spec.Networks, v1.Network{
			Name:          "default",
			NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
		})

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		Expect(causes).To(BeEmpty())
	})

	It("should reject DRA network with no corresponding interface", func() {
		spec := newDRASpec()
		spec.Domain.Devices.Interfaces = nil
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Message).To(Equal("fake.networks[0].name 'dra-net' not found."))
		Expect(causes[0].Field).To(Equal("fake.networks[0].name"))
	})
})

func newDRASpec() *v1.VirtualMachineInstanceSpec {
	return &v1.VirtualMachineInstanceSpec{
		Domain: v1.DomainSpec{
			Devices: v1.Devices{
				Interfaces: []v1.Interface{
					{
						Name: "dra-net",
						InterfaceBindingMethod: v1.InterfaceBindingMethod{
							SRIOV: &v1.InterfaceSRIOV{},
						},
					},
				},
			},
		},
		Networks: []v1.Network{
			{
				Name: "dra-net",
				NetworkSource: v1.NetworkSource{
					ResourceClaim: &v1.ClaimRequest{
						ClaimName:   "claim1",
						RequestName: "vf",
					},
				},
			},
		},
		ResourceClaims: []k8sv1.PodResourceClaim{
			{Name: "claim1", ResourceClaimName: ptr.To("claim1")},
		},
	}
}
