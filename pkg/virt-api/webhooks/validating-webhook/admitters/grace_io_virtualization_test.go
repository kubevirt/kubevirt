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

package admitters

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("GraceIOVirtualization admission", func() {
	const graceResourceName = "nvidia.com/grace-gpu"

	newGraceConfig := func(pciVendorSelector string, featureGates ...string) *virtconfig.ClusterConfig {
		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: v1.KubeVirtSpec{
				Configuration: v1.KubeVirtConfiguration{
					DeveloperConfiguration: &v1.DeveloperConfiguration{FeatureGates: featureGates},
					PermittedHostDevices: &v1.PermittedHostDevices{
						PciHostDevices: []v1.PciHostDevice{
							{
								PCIVendorSelector: pciVendorSelector,
								ResourceName:      graceResourceName,
							},
						},
					},
				},
			},
			Status: v1.KubeVirtStatus{
				Phase:               v1.KubeVirtPhaseDeploying,
				DefaultArchitecture: "arm64",
			},
		}
		config, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)
		return config
	}

	newGraceSpec := func() *v1.VirtualMachineInstanceSpec {
		return &v1.VirtualMachineInstanceSpec{
			Architecture: "arm64",
			Domain: v1.DomainSpec{
				Machine: &v1.Machine{Type: "virt"},
				CPU:     &v1.CPU{DedicatedCPUPlacement: true},
				Devices: v1.Devices{
					GPUs: []v1.GPU{
						{
							Name:       "gpu1",
							DeviceName: graceResourceName,
						},
					},
				},
			},
		}
	}

	graceCauses := func(spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
		return validateGraceIOVirtualization(k8sfield.NewPath("spec"), spec, config)
	}

	It("accepts a Grace GPU request that satisfies the static baseline", func() {
		config := newGraceConfig("10DE:2342", featuregate.GraceIOVirtualization, featuregate.PCINUMAAwareTopologyEnabled, featuregate.IOMMUFDGate)

		Expect(graceCauses(newGraceSpec(), config)).To(BeEmpty())
	})

	It("accepts a Grace host device request that satisfies the static baseline", func() {
		config := newGraceConfig("10DE:2941", featuregate.GraceIOVirtualization, featuregate.PCINUMAAwareTopologyEnabled, featuregate.IOMMUFDGate)
		spec := newGraceSpec()
		spec.Domain.Devices.GPUs = nil
		spec.Domain.Devices.HostDevices = []v1.HostDevice{{Name: "hostdev1", DeviceName: graceResourceName}}

		Expect(graceCauses(spec, config)).To(BeEmpty())
	})

	It("requires the GraceIOVirtualization feature gate for Grace GPU resources", func() {
		config := newGraceConfig("10DE:2342", featuregate.PCINUMAAwareTopologyEnabled)

		causes := graceCauses(newGraceSpec(), config)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus[0].deviceName"))
		Expect(causes[0].Message).To(ContainSubstring(featuregate.GraceIOVirtualization))
	})

	It("requires the PCI NUMA-aware topology feature gate", func() {
		config := newGraceConfig("10DE:2342", featuregate.GraceIOVirtualization, featuregate.IOMMUFDGate)

		causes := graceCauses(newGraceSpec(), config)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Field).To(Equal("spec.domain.devices"))
		Expect(causes[0].Message).To(ContainSubstring(featuregate.PCINUMAAwareTopologyEnabled))
	})

	It("requires arm64 architecture", func() {
		config := newGraceConfig("10DE:2342", featuregate.GraceIOVirtualization, featuregate.PCINUMAAwareTopologyEnabled, featuregate.IOMMUFDGate)
		spec := newGraceSpec()
		spec.Architecture = "amd64"

		causes := graceCauses(spec, config)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Field).To(Equal("spec.architecture"))
		Expect(causes[0].Message).To(ContainSubstring("arm64"))
	})

	It("uses the effective default architecture and machine type", func() {
		config := newGraceConfig("10DE:2342", featuregate.GraceIOVirtualization, featuregate.PCINUMAAwareTopologyEnabled, featuregate.IOMMUFDGate)
		spec := newGraceSpec()
		spec.Architecture = ""
		spec.Domain.Machine = nil

		Expect(graceCauses(spec, config)).To(BeEmpty())
	})

	It("requires virt machine type", func() {
		config := newGraceConfig("10DE:2342", featuregate.GraceIOVirtualization, featuregate.PCINUMAAwareTopologyEnabled, featuregate.IOMMUFDGate)
		spec := newGraceSpec()
		spec.Domain.Machine.Type = "q35"

		causes := graceCauses(spec, config)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Field).To(Equal("spec.domain.machine.type"))
		Expect(causes[0].Message).To(ContainSubstring("virt"))
	})

	It("requires dedicated CPU placement", func() {
		config := newGraceConfig("10DE:2342", featuregate.GraceIOVirtualization, featuregate.PCINUMAAwareTopologyEnabled, featuregate.IOMMUFDGate)
		spec := newGraceSpec()
		spec.Domain.CPU.DedicatedCPUPlacement = false

		causes := graceCauses(spec, config)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Field).To(Equal("spec.domain.cpu.dedicatedCpuPlacement"))
	})

	It("requires the IOMMUFD feature gate", func() {
		config := newGraceConfig("10DE:2342", featuregate.GraceIOVirtualization, featuregate.PCINUMAAwareTopologyEnabled)

		causes := graceCauses(newGraceSpec(), config)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Field).To(Equal("spec.domain.devices"))
		Expect(causes[0].Message).To(ContainSubstring(featuregate.IOMMUFDGate))
	})

	It("does not classify DRA claim requests as Grace PCI host devices", func() {
		config := newGraceConfig("10DE:2342", featuregate.GraceIOVirtualization, featuregate.PCINUMAAwareTopologyEnabled, featuregate.IOMMUFDGate)
		spec := newGraceSpec()
		spec.Domain.Devices.GPUs = []v1.GPU{
			{
				Name:         "gpu1",
				ClaimRequest: &v1.ClaimRequest{ClaimName: "claim1", RequestName: "gpu"},
			},
		}

		Expect(graceCauses(spec, config)).To(BeEmpty())
	})

	It("rejects ambiguous NVIDIA wildcard PCI selectors when GraceIOVirtualization is enabled", func() {
		config := newGraceConfig("10DE:*", featuregate.GraceIOVirtualization, featuregate.PCINUMAAwareTopologyEnabled, featuregate.IOMMUFDGate)

		causes := graceCauses(newGraceSpec(), config)
		Expect(causes).To(HaveLen(1))
		Expect(causes[0].Field).To(Equal("spec.domain.devices.gpus[0].deviceName"))
		Expect(causes[0].Message).To(ContainSubstring("exact pciVendorSelector"))
	})

	It("does not classify non-Grace NVIDIA PCI selectors", func() {
		config := newGraceConfig("10DE:2330", featuregate.GraceIOVirtualization, featuregate.PCINUMAAwareTopologyEnabled, featuregate.IOMMUFDGate)

		Expect(graceCauses(newGraceSpec(), config)).To(BeEmpty())
	})

	It("runs as part of the VMI spec validator", func() {
		config := newGraceConfig("10DE:2342")
		vmi := api.NewMinimalVMI("testvm")
		vmi.Spec.Domain.Devices.GPUs = []v1.GPU{{Name: "gpu1", DeviceName: graceResourceName}}

		causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("spec"), &vmi.Spec, config)
		var graceCause *metav1.StatusCause
		for index := range causes {
			if causes[index].Field == "spec.domain.devices.gpus[0].deviceName" {
				graceCause = &causes[index]
				break
			}
		}
		Expect(graceCause).ToNot(BeNil())
		Expect(graceCause.Message).To(ContainSubstring(featuregate.GraceIOVirtualization))
	})
})
