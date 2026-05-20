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

package domainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("host device metrics", func() {
	Context("on Collect", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi-1",
				Namespace: "test-ns-1",
			},
		}

		It("should return empty when Domain is nil", func() {
			vmiStats := &VirtualMachineInstanceStats{Domain: nil}
			vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

			crs := hostDeviceMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})

		It("should return empty when there are no host devices", func() {
			vmiStats := &VirtualMachineInstanceStats{
				Domain: &api.Domain{},
			}
			vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

			crs := hostDeviceMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})

		It("should collect metric with alias and pci_bus_id labels", func() {
			vmiStats := &VirtualMachineInstanceStats{
				Domain: &api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							HostDevices: []api.HostDevice{
								{
									Alias: api.NewUserDefinedAlias("gpu-0"),
									Address: &api.Address{
										Type:     api.AddressPCI,
										Domain:   "0000",
										Bus:      "09",
										Slot:     "00",
										Function: "0",
									},
								},
							},
						},
					},
				},
			}
			vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

			crs := hostDeviceMetrics{}.Collect(vmiReport)
			Expect(crs).To(HaveLen(1))
			Expect(crs[0].Value).To(Equal(float64(1)))
			Expect(crs[0].ConstLabels).To(HaveKeyWithValue("alias", "gpu-0"))
			Expect(crs[0].ConstLabels).To(HaveKeyWithValue("pci_bus_id", "00000000:09:00.0"))
		})

		It("should omit alias label when Alias is nil", func() {
			vmiStats := &VirtualMachineInstanceStats{
				Domain: &api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							HostDevices: []api.HostDevice{
								{
									Address: &api.Address{
										Type:     api.AddressPCI,
										Domain:   "0000",
										Bus:      "81",
										Slot:     "01",
										Function: "0",
									},
								},
							},
						},
					},
				},
			}
			vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

			crs := hostDeviceMetrics{}.Collect(vmiReport)
			Expect(crs).To(HaveLen(1))
			Expect(crs[0].ConstLabels).NotTo(HaveKey("alias"))
			Expect(crs[0].ConstLabels).To(HaveKeyWithValue("pci_bus_id", "00000000:81:01.0"))
		})

		It("should omit pci_bus_id label when Address is nil", func() {
			vmiStats := &VirtualMachineInstanceStats{
				Domain: &api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							HostDevices: []api.HostDevice{
								{
									Alias: api.NewUserDefinedAlias("dev-0"),
								},
							},
						},
					},
				},
			}
			vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

			crs := hostDeviceMetrics{}.Collect(vmiReport)
			Expect(crs).To(HaveLen(1))
			Expect(crs[0].ConstLabels).To(HaveKeyWithValue("alias", "dev-0"))
			Expect(crs[0].ConstLabels).NotTo(HaveKey("pci_bus_id"))
		})

		It("should omit pci_bus_id label when Address type is not PCI", func() {
			vmiStats := &VirtualMachineInstanceStats{
				Domain: &api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							HostDevices: []api.HostDevice{
								{
									Alias: api.NewUserDefinedAlias("mdev-0"),
									Address: &api.Address{
										Type:   "mdev",
										Domain: "0000",
										Bus:    "00",
									},
								},
							},
						},
					},
				},
			}
			vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

			crs := hostDeviceMetrics{}.Collect(vmiReport)
			Expect(crs).To(HaveLen(1))
			Expect(crs[0].ConstLabels).To(HaveKeyWithValue("alias", "mdev-0"))
			Expect(crs[0].ConstLabels).NotTo(HaveKey("pci_bus_id"))
		})

		It("should collect metrics for multiple host devices", func() {
			vmiStats := &VirtualMachineInstanceStats{
				Domain: &api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							HostDevices: []api.HostDevice{
								{
									Alias: api.NewUserDefinedAlias("gpu-0"),
									Address: &api.Address{
										Type:     api.AddressPCI,
										Domain:   "0000",
										Bus:      "09",
										Slot:     "00",
										Function: "0",
									},
								},
								{
									Alias: api.NewUserDefinedAlias("gpu-1"),
									Address: &api.Address{
										Type:     api.AddressPCI,
										Domain:   "0000",
										Bus:      "0a",
										Slot:     "00",
										Function: "0",
									},
								},
							},
						},
					},
				},
			}
			vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

			crs := hostDeviceMetrics{}.Collect(vmiReport)
			Expect(crs).To(HaveLen(2))
			Expect(crs[0].ConstLabels).To(HaveKeyWithValue("pci_bus_id", "00000000:09:00.0"))
			Expect(crs[1].ConstLabels).To(HaveKeyWithValue("pci_bus_id", "00000000:0a:00.0"))
		})
	})

	Context("formatPCIBusID", func() {
		DescribeTable("should format address to DCGM-compatible PCI bus ID",
			func(domain, bus, slot, function, expected string) {
				addr := &api.Address{
					Type:     api.AddressPCI,
					Domain:   domain,
					Bus:      bus,
					Slot:     slot,
					Function: function,
				}
				Expect(formatPCIBusID(addr)).To(Equal(expected))
			},
			Entry("standard GPU address", "0000", "09", "00", "0", "00000000:09:00.0"),
			Entry("with 0x prefix", "0x0000", "0x09", "0x00", "0x0", "00000000:09:00.0"),
			Entry("uppercase hex", "0000", "0A", "1F", "0", "00000000:0a:1f.0"),
			Entry("higher bus number", "0000", "81", "01", "0", "00000000:81:01.0"),
			Entry("multi-function device", "0000", "09", "00", "1", "00000000:09:00.1"),
			Entry("malformed values default to zero", "zz", "zz", "zz", "zz", "00000000:00:00.0"),
			Entry("empty strings default to zero", "", "", "", "", "00000000:00:00.0"),
		)
	})
})
