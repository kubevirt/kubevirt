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

package reservation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
)

var _ = Describe("Persistent Reservation", func() {

	Context("HasVMIPersistentReservation", func() {
		It("should return true when VMI has LUN with reservation", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name: "disk1",
									DiskDevice: v1.DiskDevice{
										LUN: &v1.LunTarget{
											Reservation: true,
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(reservation.HasVMIPersistentReservation(vmi)).To(BeTrue())
		})

		It("should return false when VMI has LUN without reservation", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name: "disk1",
									DiskDevice: v1.DiskDevice{
										LUN: &v1.LunTarget{
											Reservation: false,
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(reservation.HasVMIPersistentReservation(vmi)).To(BeFalse())
		})

		It("should return false when VMI has no LUN disks", func() {
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name: "disk1",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: v1.DiskBusVirtio,
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(reservation.HasVMIPersistentReservation(vmi)).To(BeFalse())
		})
	})

	Context("IsPersistentReservationMigratable", func() {
		It("should return true for SCSI LUN with reservation (virtio-scsi)", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name: "disk1",
									DiskDevice: v1.DiskDevice{
										LUN: &v1.LunTarget{
											Bus:         v1.DiskBusSCSI,
											Reservation: true,
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(reservation.IsPersistentReservationMigratable(vmi)).To(BeTrue())
		})

		It("should return false for LUN with reservation and SATA bus", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name: "disk1",
									DiskDevice: v1.DiskDevice{
										LUN: &v1.LunTarget{
											Bus:         v1.DiskBusSATA,
											Reservation: true,
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(reservation.IsPersistentReservationMigratable(vmi)).To(BeFalse())
		})

		It("should return true when VMI has no persistent reservations", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name: "disk1",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: v1.DiskBusVirtio,
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(reservation.IsPersistentReservationMigratable(vmi)).To(BeTrue())
		})

		It("should return false for non-SCSI bus with reservation", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name: "disk1",
									DiskDevice: v1.DiskDevice{
										LUN: &v1.LunTarget{
											Bus:         v1.DiskBusSATA,
											Reservation: true,
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(reservation.IsPersistentReservationMigratable(vmi)).To(BeFalse())
		})

		It("should return true for multiple SCSI LUNs with reservation", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "default",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name: "disk1",
									DiskDevice: v1.DiskDevice{
										LUN: &v1.LunTarget{
											Bus:         v1.DiskBusSCSI,
											Reservation: true,
										},
									},
								},
								{
									Name: "disk2",
									DiskDevice: v1.DiskDevice{
										LUN: &v1.LunTarget{
											Bus:         v1.DiskBusSCSI,
											Reservation: true,
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(reservation.IsPersistentReservationMigratable(vmi)).To(BeTrue())
		})

		It("should return true for nil VMI", func() {
			Expect(reservation.IsPersistentReservationMigratable(nil)).To(BeTrue())
		})
	})
})
