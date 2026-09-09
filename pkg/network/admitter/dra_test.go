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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validate network DRA", func() {
	It("should reject DRA network when feature gate is disabled", func() {
		spec := newDRASpec()
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "vmi.spec.networks contains DRA networks but NetworkDevicesWithDRA feature gate is not enabled",
			Field:   "fake.networks",
		}}
		Expect(causes).To(Equal(expectedCauses))
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
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "claimName is required for DRA network",
			Field:   "fake.networks[0].resourceClaim.claimName",
		}}
		Expect(causes).To(Equal(expectedCauses))
	})

	It("should reject DRA network with empty requestName", func() {
		spec := newDRASpec()
		spec.Networks[0].ResourceClaim.RequestName = ""
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "requestName is required for DRA network",
			Field:   "fake.networks[0].resourceClaim.requestName",
		}}
		Expect(causes).To(Equal(expectedCauses))
	})

	It("should reject DRA network with non-existent resourceClaim reference", func() {
		spec := newDRASpec()
		spec.Networks[0].ResourceClaim.ClaimName = "missing-claim"
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Message: `network references resourceClaim "missing-claim" which is not defined in spec.resourceClaims`,
			Field:   "fake.networks[0].resourceClaim.claimName",
		}}
		Expect(causes).To(Equal(expectedCauses))
	})

	It("should reject duplicate claimName/requestName across DRA networks", func() {
		spec := newDRASpec()
		spec.Domain.Devices.Interfaces = []v1.Interface{
			{
				Name:    "dra-net-1",
				Binding: &v1.PluginBinding{Name: "netbinding"},
			},
			{
				Name:    "dra-net-2",
				Binding: &v1.PluginBinding{Name: "netbinding"},
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
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueDuplicate,
			Message: `duplicate claimName/requestName combination "claim1/vf"`,
			Field:   "fake.networks[1]",
		}}
		Expect(causes).To(Equal(expectedCauses))
	})

	It("should reject mixing Multus and DRA networks", func() {
		spec := newDRASpec()
		spec.Domain.Devices.Interfaces = []v1.Interface{
			{
				Name:    "multus-net",
				Binding: &v1.PluginBinding{Name: "netbinding"},
			},
			{
				Name:    "dra-net",
				Binding: &v1.PluginBinding{Name: "netbinding"},
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
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "mixing Multus and DRA resourceClaim networks in the same VMI is not supported",
			Field:   "fake.networks",
		}}
		Expect(causes).To(Equal(expectedCauses))
	})

	DescribeTable("should reject DRA network with core interface binding",
		func(iface v1.Interface) {
			spec := newDRASpec()
			iface.Name = "dra-net"
			spec.Domain.Devices.Interfaces = []v1.Interface{iface}
			validator := admitter.NewValidator(k8sfield.NewPath("fake"), spec, stubClusterConfigChecker{networkDRAEnabled: true})
			causes := validator.Validate()
			Expect(causes).To(ContainElement(HaveField("Message", `DRA network "dra-net" requires a binding plugin interface`)))
		},
		Entry("bridge", v1.Interface{InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}),
		Entry("masquerade", v1.Interface{InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}),
		Entry("SR-IOV", v1.Interface{InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}),
		Entry("passtBinding", v1.Interface{InterfaceBindingMethod: v1.InterfaceBindingMethod{PasstBinding: &v1.InterfacePasstBinding{}}}),
	)

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
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "fake.networks[0].name 'dra-net' not found.",
			Field:   "fake.networks[0].name",
		}}
		Expect(causes).To(Equal(expectedCauses))
	})
})

func newDRASpec() *v1.VirtualMachineInstanceSpec {
	return &v1.VirtualMachineInstanceSpec{
		Domain: v1.DomainSpec{
			Devices: v1.Devices{
				Interfaces: []v1.Interface{
					{
						Name:    "dra-net",
						Binding: &v1.PluginBinding{Name: "netbinding"},
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
		ResourceClaims: []v1.VirtualMachineInstanceResourceClaim{
			{Name: "claim1", ResourceClaimName: new("claim1")},
		},
	}
}
