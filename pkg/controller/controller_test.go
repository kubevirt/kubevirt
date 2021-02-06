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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package controller

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kubev1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("controller", func() {
	var vmiSpec *v1.VirtualMachineInstanceSpec
	var request *v1.VirtualMachineVolumeRequest

	Context("apply volume request on vmi spec", func() {
		Context("Add volume Request", func() {
			BeforeEach(func() {
				testVolume := &v1.Volume{
					Name: "testVol",
				}

				vmiSpec = createVMISpec(testVolume, v1.Disk{})
			})

			It("requests to add a new volume which was not already added", func() {
				request = &v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name: "newTestVol",
					},
				}

				Expect(volumeAlreadyAdded(vmiSpec, request)).To(BeFalse())
			})

			It("requests to add a volume which was already added", func() {
				request = &v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name: "testVol",
					},
				}

				Expect(volumeAlreadyAdded(vmiSpec, request)).To(BeTrue())

				vmiSpec = handleAddVolumeRequest(vmiSpec, request)
				Expect(vmiSpec.Volumes).To(HaveLen(1))
			})

			It("requests to add a new volume with PVC source", func() {
				request = &v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name: "newTestVol",
						VolumeSource: &v1.HotplugVolumeSource{
							PersistentVolumeClaim: &kubev1.PersistentVolumeClaimVolumeSource{},
						},
					},
				}

				vmiSpec = addVolumeToSpec(vmiSpec, request)
				Expect(vmiSpec.Volumes[1].Name).ToNot(BeNil())
				Expect(vmiSpec.Volumes[1].Name).To(Equal("newTestVol"))
			})

			It("requests to add a new volume with DataVolume source", func() {
				request = &v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name: "newTestVol",
						VolumeSource: &v1.HotplugVolumeSource{
							DataVolume: &v1.DataVolumeSource{},
						},
					},
				}

				vmiSpec = addVolumeToSpec(vmiSpec, request)
				Expect(vmiSpec.Volumes[1].Name).ToNot(BeNil())
				Expect(vmiSpec.Volumes[1].Name).To(Equal("newTestVol"))
			})

			It("requests to add a new volume as disk", func() {
				request = &v1.VirtualMachineVolumeRequest{
					AddVolumeOptions: &v1.AddVolumeOptions{
						Name: "newTestVol",
						Disk: &v1.Disk{},
					},
				}

				vmiSpec = addVolumeAsDisk(vmiSpec, request)
				Expect(vmiSpec.Domain.Devices.Disks[1]).ToNot(BeNil())
				Expect(vmiSpec.Domain.Devices.Disks[1].Name).To(Equal("newTestVol"))
			})
		})

		Context("Remove volume options", func() {
			BeforeEach(func() {
				testVolume := &v1.Volume{
					Name: "testVol",
				}

				disk := &v1.Disk{
					Name: "testDisk",
				}

				vmiSpec = createVMISpec(testVolume, *disk)
			})

			It("requests to remove a volume", func() {
				request = &v1.VirtualMachineVolumeRequest{
					RemoveVolumeOptions: &v1.RemoveVolumeOptions{
						Name: "testVol",
					},
				}
				vmiSpec = removeVolumeFromSpec(vmiSpec, request)
				Expect(vmiSpec.Volumes).To(HaveLen(0))

			})

			It("requests to remove a disk", func() {
				request = &v1.VirtualMachineVolumeRequest{
					RemoveVolumeOptions: &v1.RemoveVolumeOptions{
						Name: "testDisk",
					},
				}

				vmiSpec = removeVolumeAsDisk(vmiSpec, request)
				Expect(vmiSpec.Domain.Devices.Disks).To(HaveLen(0))
			})
		})
	})
})

func createVMISpec(volume *v1.Volume, disk v1.Disk) *v1.VirtualMachineInstanceSpec {
	spec := &v1.VirtualMachineInstanceSpec{
		Volumes: []v1.Volume{*volume},
		Domain: v1.DomainSpec{
			Devices: v1.Devices{
				Disks: []v1.Disk{disk},
			},
		},
	}

	return spec
}
