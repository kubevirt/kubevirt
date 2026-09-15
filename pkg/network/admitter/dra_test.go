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

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("Validate network DRA", func() {
	It("should reject DRA network when feature gate is disabled", func() {
		vmi := newDRAVMI(libvmi.DRANetwork("dra-net", "claim1", "vf"))
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{})
		causes := validator.Validate()
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "vmi.spec.networks contains DRA networks but NetworkDevicesWithDRA feature gate is not enabled",
			Field:   "fake.networks",
		}}
		Expect(causes).To(Equal(expectedCauses))
	})

	It("should accept valid DRA network when feature gate is enabled", func() {
		vmi := newDRAVMI(libvmi.DRANetwork("dra-net", "claim1", "vf"))
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		Expect(causes).To(BeEmpty())
	})

	It("should reject DRA network with empty claimName", func() {
		vmi := newDRAVMI(libvmi.DRANetwork("dra-net", "", "vf"))
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "claimName is required for DRA network",
			Field:   "fake.networks[0].resourceClaim.claimName",
		}}
		Expect(causes).To(Equal(expectedCauses))
	})

	It("should reject DRA network with empty requestName", func() {
		vmi := newDRAVMI(libvmi.DRANetwork("dra-net", "claim1", ""))
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "requestName is required for DRA network",
			Field:   "fake.networks[0].resourceClaim.requestName",
		}}
		Expect(causes).To(Equal(expectedCauses))
	})

	It("should reject DRA network with non-existent resourceClaim reference", func() {
		vmi := newDRAVMI(libvmi.DRANetwork("dra-net", "missing-claim", "vf"))
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Message: `network references resourceClaim "missing-claim" which is not defined in spec.resourceClaims`,
			Field:   "fake.networks[0].resourceClaim.claimName",
		}}
		Expect(causes).To(Equal(expectedCauses))
	})

	It("should reject duplicate claimName/requestName across DRA networks", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.NewInterface("dra-net-1", libvmi.WithBindingPlugin(v1.PluginBinding{Name: "netbinding"}))),
			libvmi.WithInterface(libvmi.NewInterface("dra-net-2", libvmi.WithBindingPlugin(v1.PluginBinding{Name: "netbinding"}))),
			libvmi.WithNetwork(libvmi.DRANetwork("dra-net-1", "claim1", "vf")),
			libvmi.WithNetwork(libvmi.DRANetwork("dra-net-2", "claim1", "vf")),
			libvmi.WithResourceClaim(v1.VirtualMachineInstanceResourceClaim{Name: "claim1", ResourceClaimName: new("claim1")}),
		)
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueDuplicate,
			Message: `duplicate claimName/requestName combination "claim1/vf"`,
			Field:   "fake.networks[1]",
		}}
		Expect(causes).To(Equal(expectedCauses))
	})

	It("should reject mixing Multus and DRA networks", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.NewInterface("multus-net", libvmi.WithBindingPlugin(v1.PluginBinding{Name: "netbinding"}))),
			libvmi.WithInterface(libvmi.NewInterface("dra-net", libvmi.WithBindingPlugin(v1.PluginBinding{Name: "netbinding"}))),
			libvmi.WithNetwork(libvmi.MultusNetwork("multus-net", "nad1")),
			libvmi.WithNetwork(libvmi.DRANetwork("dra-net", "claim1", "vf")),
			libvmi.WithResourceClaim(v1.VirtualMachineInstanceResourceClaim{Name: "claim1", ResourceClaimName: new("claim1")}),
		)
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{networkDRAEnabled: true})
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
			vmi := newDRAVMI(libvmi.DRANetwork("dra-net", "claim1", "vf"))
			iface.Name = "dra-net"
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}
			validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{networkDRAEnabled: true})
			causes := validator.Validate()
			Expect(causes).To(ContainElement(HaveField("Message", `DRA network "dra-net" requires a binding plugin interface`)))
		},
		Entry("bridge", v1.Interface{InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}),
		Entry("masquerade", v1.Interface{InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}),
		Entry("SR-IOV", v1.Interface{InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}),
		Entry("passtBinding", v1.Interface{InterfaceBindingMethod: v1.InterfaceBindingMethod{PasstBinding: &v1.InterfacePasstBinding{}}}),
	)

	It("should accept DRA network with plugin interface binding", func() {
		vmi := newDRAVMI(libvmi.DRANetwork("dra-net", "claim1", "vf"))
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
			libvmi.NewInterface("default", libvmi.WithMasqueradeBinding()),
			libvmi.NewInterface("dra-net", libvmi.WithBindingPlugin(v1.PluginBinding{Name: "vhostuser"})),
		}
		vmi.Spec.Networks = append(vmi.Spec.Networks, *v1.DefaultPodNetwork())

		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		Expect(causes).To(BeEmpty())
	})

	It("should reject DRA network with no corresponding interface", func() {
		vmi := libvmi.New(
			libvmi.WithNetwork(libvmi.DRANetwork("dra-net", "claim1", "vf")),
			libvmi.WithResourceClaim(v1.VirtualMachineInstanceResourceClaim{Name: "claim1", ResourceClaimName: new("claim1")}),
		)
		validator := admitter.NewValidator(k8sfield.NewPath("fake"), &vmi.Spec, stubClusterConfigChecker{networkDRAEnabled: true})
		causes := validator.Validate()
		expectedCauses := []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "fake.networks[0].name 'dra-net' not found.",
			Field:   "fake.networks[0].name",
		}}
		Expect(causes).To(Equal(expectedCauses))
	})
})

func newDRAVMI(network *v1.Network, opts ...libvmi.Option) *v1.VirtualMachineInstance {
	base := []libvmi.Option{
		libvmi.WithInterface(libvmi.NewInterface("dra-net",
			libvmi.WithBindingPlugin(v1.PluginBinding{Name: "netbinding"}))),
		libvmi.WithNetwork(network),
		libvmi.WithResourceClaim(v1.VirtualMachineInstanceResourceClaim{Name: "claim1", ResourceClaimName: new("claim1")}),
	}
	return libvmi.New(append(base, opts...)...)
}
