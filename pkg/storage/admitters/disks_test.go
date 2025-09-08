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
	"fmt"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Disk Validation", func() {

	Context("with ValidateDisks", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = api.NewMinimalVMI("testvmi")
			vmi.Spec.Architecture = runtime.GOARCH
		})

		DescribeTable("should accept valid disks",
			func(disk v1.Disk) {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, disk)

				causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(causes).To(BeEmpty())

			},
			Entry("with Disk target",
				v1.Disk{Name: "testdisk", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{}}},
			),
			Entry("with LUN target",
				v1.Disk{Name: "testdisk", DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{}}},
			),
			Entry("with CDRom target",
				v1.Disk{Name: "testdisk", DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{}}},
			),
		)

		It("should allow disk without a target", func() {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				// disk without a target defaults to DiskTarget
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "fake"},
				},
			})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(BeEmpty())
		})

		It("should reject disks with duplicate names ", func() {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})
			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[1].name"))
		})

		It("should reject disks with SATA and read-only set", func() {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus:      v1.DiskBusSATA,
						ReadOnly: true,
					},
				},
			})
			causes := ValidateDisks(k8sfield.NewPath("disks"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("disks[0].disk.bus"))
		})

		It("should reject disks with PCI address on a non-virtio bus ", func() {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						PciAddress: "0000:04:10.0",
						Bus:        v1.DiskBusSCSI},
				},
			})
			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks.disk[0].pciAddress"))
		})

		It("should reject disks malformed PCI addresses ", func() {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						PciAddress: "0000:81:100.a",
						Bus:        v1.DiskBusVirtio,
					},
				},
			})
			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks.disk[0].pciAddress"))
		})

		It("should reject disk with multiple targets ", func() {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
				DiskDevice: v1.DiskDevice{
					Disk:  &v1.DiskTarget{},
					CDRom: &v1.CDRomTarget{},
				},
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					ContainerDisk: &v1.ContainerDiskSource{Image: "fake"},
				},
			})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})

		It("should accept a boot order greater than '0'", func() {
			order := uint(1)

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:      "testdisk",
				BootOrder: &order,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(BeEmpty())
		})

		It("should reject a disk with a boot order of '0'", func() {
			order := uint(0)

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:      "testdisk",
				BootOrder: &order,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].bootOrder"))
		})

		It("should accept disks with supported or unspecified buses", func() {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk1",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: v1.DiskBusVirtio,
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk2",
				DiskDevice: v1.DiskDevice{
					LUN: &v1.LunTarget{
						Bus: v1.DiskBusSATA,
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk3",
				DiskDevice: v1.DiskDevice{
					CDRom: &v1.CDRomTarget{
						Bus: v1.DiskBusSCSI,
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk4",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: v1.DiskBusUSB,
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk5",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(BeEmpty())
		})

		It("should reject disks with unsupported buses", func() {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk1",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: "ide",
					},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk2",
				DiskDevice: v1.DiskDevice{
					LUN: &v1.LunTarget{
						Bus: "unsupported",
					},
				},
			})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(2))
			Expect(causes[0].Field).To(Equal("fake[0].disk.bus"))
			Expect(causes[1].Field).To(Equal("fake[1].lun.bus"))
		})

		It("should reject disks with unsupported I/O modes", func() {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk1",
				IO:   "native",
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk2",
				IO:   "unsupported",
			})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[1].io"))
		})

		It("should reject disk with invalid cache mode", func() {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk", Cache: "unspported", DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{}}})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake[0].cache"))
			Expect(causes[0].Message).To(Equal("fake[0].cache has invalid value unspported"))
		})

		DescribeTable("It should accept a disk with a valid cache mode", func(mode v1.DriverCache) {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk", Cache: mode, DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{}}})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(BeEmpty())
		},
			Entry("none", v1.CacheNone),
			Entry("writethrough", v1.CacheWriteThrough),
			Entry("writeback", v1.CacheWriteBack),
		)

		DescribeTable("should reject disk with invalid errorPolicy", func(policy string) {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk", ErrorPolicy: pointer.P(v1.DiskErrorPolicy(policy)), DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{}}})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(1))
			Expect(string(causes[0].Type)).To(Equal("FieldValueInvalid"))
			Expect(causes[0].Field).To(Equal("fake[0].errorPolicy"))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("fake[0].errorPolicy has invalid value \"%s\"", policy)))
		},
			Entry("with arbitrary string", "unsupported"),
			Entry("with empty string", ""),
		)

		DescribeTable("It should accept a disk with a valid errorPolicy", func(mode v1.DiskErrorPolicy) {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk", ErrorPolicy: pointer.P(mode), DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{}}})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(BeEmpty())
		},
			Entry("stop", v1.DiskErrorPolicyStop),
			Entry("report", v1.DiskErrorPolicyReport),
			Entry("ignore", v1.DiskErrorPolicyIgnore),
			Entry("enospace", v1.DiskErrorPolicyEnospace),
		)

		It("should reject invalid SN characters", func() {
			order := uint(1)
			sn := "$$$$"

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:      "testdisk2",
				BootOrder: &order,
				Serial:    sn,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].serial"))
		})

		It("should reject SN > maxStrLen characters", func() {
			order := uint(1)
			sn := strings.Repeat("1", maxStrLen+1)

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:      "testdisk2",
				BootOrder: &order,
				Serial:    sn,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Field).To(Equal("fake[0].serial"))
		})

		It("should accept valid SN", func() {
			order := uint(1)

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:      "testdisk2",
				BootOrder: &order,
				Serial:    "SN-1_a",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(BeEmpty())
		})

		DescribeTable("Should reject disk with DedicatedIOThread and non-virtio bus", func(bus v1.DiskBus) {
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks,
				v1.Disk{
					Name:              "disk-with-dedicated-io-thread-and-sata",
					DedicatedIOThread: pointer.P(true),
					DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{
						Bus: bus,
					}},
				},
				v1.Disk{
					Name:              "disk-with-dedicated-io-thread-and-virtio",
					DedicatedIOThread: pointer.P(true),
					DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{
						Bus: v1.DiskBusVirtio,
					}},
				},
				v1.Disk{
					Name:              "disk-without-dedicated-io-thread-and-with-sata",
					DedicatedIOThread: pointer.P(false),
					DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{
						Bus: v1.DiskBusSATA,
					}},
				},
			)

			causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(causes).To(HaveLen(1)) // Only first disk should fail
			Expect(string(causes[0].Type)).To(Equal("FieldValueNotSupported"))
			Expect(causes[0].Field).To(ContainSubstring("domain.devices.disks"))
			Expect(causes[0].Message).To(Equal(fmt.Sprintf("IOThreads are not supported for disks on a %s bus", bus)))

		},
			Entry("SATA bus", v1.DiskBusSATA),
			Entry("SCSI bus", v1.DiskBusSCSI),
			Entry("USB bus", v1.DiskBusUSB),
		)

		Context("With block size", func() {

			DescribeTable("It should accept a disk with a valid block size of", func(logicalSize, physicalSize int) {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blockdisk",
					BlockSize: &v1.BlockSize{
						Custom: &v1.CustomBlockSize{
							Logical:  uint(logicalSize),
							Physical: uint(physicalSize),
						},
					},
				})

				causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(causes).To(BeEmpty())
			},
				Entry("a 512n disk", 512, 512),
				Entry("a 512e disk", 512, 4096),
				Entry("a 4096n (4kn) disk", 4096, 4096),
				Entry("a custom 1 MiB disk", 1048576, 1048576),
			)

			DescribeTable("It should deny a disk's block size configuration when", func(logicalSize, physicalSize int) {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blockdisk",
					BlockSize: &v1.BlockSize{
						Custom: &v1.CustomBlockSize{
							Logical:  uint(logicalSize),
							Physical: uint(physicalSize),
						},
					},
				})

				causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(causes).To(HaveLen(2))
				Expect(causes[0].Field).To(Equal("fake[0].blockSize.custom.logical"))
				Expect(causes[1].Field).To(Equal("fake[0].blockSize.custom.physical"))
			},
				Entry("less than 512", 128, 128),
				Entry("greater than 2 MiB", 3000000, 3000000),
				Entry("not a power of 2", 1234, 1234),
			)

			It("Should deny a disk's block size configuration when logical > physical", func() {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blockdisk",
					BlockSize: &v1.BlockSize{
						Custom: &v1.CustomBlockSize{
							Logical:  4096,
							Physical: 512,
						},
					},
				})

				causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Field).To(Equal("fake[0].blockSize.custom.logical"))
			})

			It("Should accept disks with block size matching enabled", func() {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blockdisk",
					BlockSize: &v1.BlockSize{
						MatchVolume: &v1.FeatureState{
							Enabled: pointer.P(true),
						},
					},
				})

				causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(causes).To(BeEmpty())
			})

			It("Should reject disk with custom block size and size matching enabled", func() {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blockdisk",
					BlockSize: &v1.BlockSize{
						Custom: &v1.CustomBlockSize{
							Logical:  1234,
							Physical: 1234,
						},
						MatchVolume: &v1.FeatureState{
							Enabled: pointer.P(true),
						},
					},
				})

				causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(causes).To(HaveLen(1))
				Expect(causes[0].Field).To(Equal("fake[0].blockSize"))
			})

			It("Should accept disks with a custom block size and size matching explicitly disabled", func() {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: "blockdisk",
					BlockSize: &v1.BlockSize{
						Custom: &v1.CustomBlockSize{
							Logical:  4096,
							Physical: 4096,
						},
						MatchVolume: &v1.FeatureState{
							Enabled: pointer.P(false),
						},
					},
				})

				causes := ValidateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(causes).To(BeEmpty())
			})
		})
	})

})
