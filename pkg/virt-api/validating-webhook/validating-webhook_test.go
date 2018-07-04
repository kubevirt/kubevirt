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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package validating_webhook

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	v1beta1 "k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Validating Webhook", func() {
	Context("with VirtualMachineInstance admission review", func() {
		It("reject invalid VirtualMachineInstance spec", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			vmiBytes, _ := json.Marshal(&vmi)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{Group: v1.VirtualMachineInstanceGroupVersionKind.Group, Version: v1.VirtualMachineInstanceGroupVersionKind.Version, Resource: "virtualmachineinstances"},
					Object: runtime.RawExtension{
						Raw: vmiBytes,
					},
				},
			}

			resp := admitVMIs(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[0].volumeName"))
		})
		It("should accept valid vmi spec", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{},
				},
			})
			vmiBytes, _ := json.Marshal(&vmi)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{Group: v1.VirtualMachineInstanceGroupVersionKind.Group, Version: v1.VirtualMachineInstanceGroupVersionKind.Version, Resource: "virtualmachineinstances"},
					Object: runtime.RawExtension{
						Raw: vmiBytes,
					},
				},
			}
			resp := admitVMIs(ar)
			Expect(resp.Allowed).To(Equal(true))
		})
	})
	Context("with VMIRS admission review", func() {
		table.DescribeTable("reject invalid VirtualMachineInstance spec", func(vmirs *v1.VirtualMachineInstanceReplicaSet, causes []string) {
			vmirsBytes, _ := json.Marshal(&vmirs)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group,
						Version:  v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Version,
						Resource: "virtualmachineinstancereplicasets",
					},
					Object: runtime.RawExtension{
						Raw: vmirsBytes,
					},
				},
			}

			resp := admitVMIRS(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(resp.Result.Details.Causes).To(HaveLen(len(causes)))
			for i, cause := range causes {
				Expect(resp.Result.Details.Causes[i].Field).To(Equal(cause))
			}
		},
			table.Entry("with missing volume and missing labels", &v1.VirtualMachineInstanceReplicaSet{
				Spec: v1.VirtualMachineInstanceReplicaSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"match": "this"},
					},
					Template: newVirtualMachineBuilder().WithDisk(v1.Disk{
						Name:       "testdisk",
						VolumeName: "testvolume",
					}).BuildTemplate(),
				},
			}, []string{
				"spec.template.spec.domain.devices.disks[0].volumeName",
				"spec.selector",
			}),
			table.Entry("with mismatching label selectors", &v1.VirtualMachineInstanceReplicaSet{
				Spec: v1.VirtualMachineInstanceReplicaSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"match": "not"},
					},
					Template: newVirtualMachineBuilder().WithLabel("match", "this").BuildTemplate(),
				},
			}, []string{
				"spec.selector",
			}),
		)
		It("should accept valid vmi spec", func() {
			vmirs := &v1.VirtualMachineInstanceReplicaSet{
				Spec: v1.VirtualMachineInstanceReplicaSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"match": "me"},
					},
					Template: newVirtualMachineBuilder().
						WithDisk(v1.Disk{
							Name:       "testdisk",
							VolumeName: "testvolume",
						}).
						WithVolume(v1.Volume{
							Name: "testvolume",
							VolumeSource: v1.VolumeSource{
								RegistryDisk: &v1.RegistryDiskSource{},
							},
						}).
						WithLabel("match", "me").
						BuildTemplate(),
				},
			}
			vmirsBytes, _ := json.Marshal(&vmirs)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Group,
						Version:  v1.VirtualMachineInstanceReplicaSetGroupVersionKind.Version,
						Resource: "virtualmachineinstancereplicasets",
					},
					Object: runtime.RawExtension{
						Raw: vmirsBytes,
					},
				},
			}

			resp := admitVMIRS(ar)
			Expect(resp.Allowed).To(Equal(true))
		})
	})

	Context("with VM admission review", func() {
		It("reject invalid VirtualMachineInstance spec", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			vm := &v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Running: false,
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: vmi.Spec,
					},
				},
			}
			vmBytes, _ := json.Marshal(&vm)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    v1.VirtualMachineGroupVersionKind.Group,
						Version:  v1.VirtualMachineGroupVersionKind.Version,
						Resource: "virtualmachines",
					},
					Object: runtime.RawExtension{
						Raw: vmBytes,
					},
				},
			}

			resp := admitVMs(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.devices.disks[0].volumeName"))
		})
		It("should accept valid vmi spec", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{},
				},
			})

			vm := &v1.VirtualMachine{
				Spec: v1.VirtualMachineSpec{
					Running: false,
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: vmi.Spec,
					},
				},
			}
			vmBytes, _ := json.Marshal(&vm)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    v1.VirtualMachineGroupVersionKind.Group,
						Version:  v1.VirtualMachineGroupVersionKind.Version,
						Resource: "virtualmachines",
					},
					Object: runtime.RawExtension{
						Raw: vmBytes,
					},
				},
			}

			resp := admitVMs(ar)
			Expect(resp.Allowed).To(Equal(true))
		})
	})
	Context("with VMIPreset admission review", func() {
		It("reject invalid VirtualMachineInstance spec", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmiPDomain := &v1.DomainPresetSpec{}
			vmiDomainByte, _ := json.Marshal(vmi.Spec.Domain)
			Expect(json.Unmarshal(vmiDomainByte, &vmiPDomain)).To(BeNil())

			vmiPDomain.Devices.Disks = append(vmiPDomain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
				DiskDevice: v1.DiskDevice{
					Disk:   &v1.DiskTarget{},
					Floppy: &v1.FloppyTarget{},
				},
			})
			vmiPreset := &v1.VirtualMachineInstancePreset{
				Spec: v1.VirtualMachineInstancePresetSpec{
					Domain: vmiPDomain,
				},
			}
			vmiPresetBytes, _ := json.Marshal(vmiPreset)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    v1.VirtualMachineInstancePresetGroupVersionKind.Group,
						Version:  v1.VirtualMachineInstancePresetGroupVersionKind.Version,
						Resource: "virtualmachineinstancepresets",
					},
					Object: runtime.RawExtension{
						Raw: vmiPresetBytes,
					},
				},
			}

			resp := admitVMIPreset(ar)
			Expect(resp.Allowed).To(Equal(false))
			Expect(len(resp.Result.Details.Causes)).To(Equal(1))
			Expect(resp.Result.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[0]"))
		})
		It("should accept valid vmi spec", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})

			vmiPreset := &v1.VirtualMachineInstancePreset{
				Spec: v1.VirtualMachineInstancePresetSpec{
					Domain: &v1.DomainPresetSpec{},
				},
			}
			vmiPresetBytes, _ := json.Marshal(&vmiPreset)

			ar := &v1beta1.AdmissionReview{
				Request: &v1beta1.AdmissionRequest{
					Resource: metav1.GroupVersionResource{
						Group:    v1.VirtualMachineInstancePresetGroupVersionKind.Group,
						Version:  v1.VirtualMachineInstancePresetGroupVersionKind.Version,
						Resource: "virtualmachineinstancepresets",
					},
					Object: runtime.RawExtension{
						Raw: vmiPresetBytes,
					},
				},
			}

			resp := admitVMIPreset(ar)
			Expect(resp.Allowed).To(Equal(true))
		})
	})

	Context("with VirtualMachineInstance spec", func() {
		It("should accept disk and volume lists equal to max element length", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			for i := 0; i < arrayLenMax; i++ {
				diskName := fmt.Sprintf("testDisk%d", i)
				volumeName := fmt.Sprintf("testVolume%d", i)
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name:       diskName,
					VolumeName: volumeName,
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						RegistryDisk: &v1.RegistryDiskSource{},
					},
				})
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject disk lists greater than max element length", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			for i := 0; i <= arrayLenMax; i++ {
				diskName := "testDisk"
				volumeName := "testVolume"
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name:       diskName,
					VolumeName: volumeName,
				})
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			// if this is processed correctly, it should result in a single error
			// If multiple causes occurred, then the spec was processed too far.
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks"))
		})
		It("should reject volume lists greater than max element length", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			for i := 0; i <= arrayLenMax; i++ {
				volumeName := "testVolume"
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						RegistryDisk: &v1.RegistryDiskSource{},
					},
				})
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			// if this is processed correctly, it should result in a single error
			// If multiple causes occurred, then the spec was processed too far.
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.volumes"))
		})

		It("should reject disk with missing volume", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[0].volumeName"))
		})
		It("should reject multiple disks referencing same volume", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			// verify two disks referencing the same volume are rejected
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk2",
				VolumeName: "testvolume",
			})

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{},
				},
			})
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[1].volumeName"))
		})
		It("should generate multiple causes", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
				DiskDevice: v1.DiskDevice{
					Disk:   &v1.DiskTarget{},
					Floppy: &v1.FloppyTarget{},
				},
			})

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			// missing volume and multiple targets set. should result in 2 causes
			Expect(len(causes)).To(Equal(2))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.disks[0].volumeName"))
			Expect(causes[1].Field).To(Equal("fake.domain.devices.disks[0]"))
		})
		It("should reject negative requests.memory size value", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("-64Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should reject negative limits.memory size value", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("-65Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.limits.memory"))
		})
		It("should reject greater requests.memory than limits.memory", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("128Mi"),
			}
			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should accept correct memory size values", func() {
			vm := v1.NewMinimalVMI("testvm")

			vm.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vm.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("65Mi"),
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject incorrect hugepages size format", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2ab"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.hugepages.size"))
		})
		It("should reject greater hugepages.size than requests.memory", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "1Gi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should reject not divisable by hugepages.size requests.memory", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("65Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2Gi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.resources.requests.memory"))
		})
		It("should accept correct memory and hugepages size values", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("64Mi"),
			}
			vmi.Spec.Domain.Memory = &v1.Memory{Hugepages: &v1.Hugepages{}}
			vmi.Spec.Domain.Memory.Hugepages.PageSize = "2Mi"

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		table.DescribeTable("should verify LUN is mapped to PVC volume",
			func(volume *v1.Volume, expectedErrors int) {
				vmi := v1.NewMinimalVMI("testvmi")
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name:       "testdisk",
					VolumeName: "testvolume",
					DiskDevice: v1.DiskDevice{
						LUN: &v1.LunTarget{},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, *volume)

				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
				Expect(len(causes)).To(Equal(expectedErrors))
			},
			table.Entry("and reject non PVC sources",
				&v1.Volume{
					Name: "testvolume",
					VolumeSource: v1.VolumeSource{
						RegistryDisk: &v1.RegistryDiskSource{},
					},
				}, 1),
			table.Entry("and accept PVC sources",
				&v1.Volume{
					Name: "testvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{},
					},
				}, 0),
		)
		It("should accept a single interface and network", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should accept interface lists with more than one element", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			// if this is processed correctly, it should not result in any error
			Expect(len(causes)).To(Equal(0))
		})
		It("should accept network lists with more than one element", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Networks = []v1.Network{
				*v1.DefaultPodNetwork(),
				*v1.DefaultPodNetwork()}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			// if this is processed correctly, it should not result in any error
			Expect(len(causes)).To(Equal(0))
		})

		It("should accept valid interface models", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			for _, model := range validInterfaceModels {
				vmi.Spec.Domain.Devices.Interfaces[0].Model = model
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
				// if this is processed correctly, it should not result in any error
				Expect(len(causes)).To(Equal(0))
			}
		})

		It("should reject invalid interface model", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces[0].Model = "invalid_model"
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
			Expect(len(causes)).To(Equal(1))
		})

		It("should reject interfaces with missing network", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "redtest",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].name"))
		})
		It("should only accept networks with a pod network source ", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{},
				},
			}
			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(2))
			Expect(causes[0].Field).To(Equal("fake.networks[0].pod"))
		})
		It("should accept networks with a pod network source and bridge interface", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should accept networks with a pod network source and slirp interface", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should accept networks with a pod network source and slirp interface with port", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject networks with a pod network source and slirp interface without specific port", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Name: "test"}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[0]"))
		})
		It("should reject networks with a pod network source and slirp interface with bad protocol type", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Protocol: "bad", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[0].protocol"))
		})
		It("should accept networks with a pod network source and slirp interface with multiple Ports", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Port: 80}, {Protocol: "UDP", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject port out of range", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Port: 80000}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[0]"))
		})
		It("should reject interface with two ports with the same name", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Name: "testPort", Port: 80}, {Name: "testPort", Protocol: "UDP", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].ports[1].name"))
		})
		It("should reject two interfaces with same port name", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Name: "testPort", Port: 80}}},
				v1.Interface{
					Name: "default",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Slirp: &v1.InterfaceSlirp{},
					},
					Ports: []v1.Port{{Name: "testPort", Protocol: "UDP", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[1].ports[0].name"))
		})
		It("should allow interface with two same ports and protocol", func() {
			vm := v1.NewMinimalVMI("testvm")
			vm.Spec.Domain.Devices.Interfaces = []v1.Interface{v1.Interface{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Slirp: &v1.InterfaceSlirp{},
				},
				Ports: []v1.Port{{Port: 80}, {Protocol: "UDP", Port: 80}, {Protocol: "TCP", Port: 80}}}}

			vm.Spec.Networks = []v1.Network{
				v1.Network{
					Name:          "default",
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject specs with multiple pod interfaces", func() {
			vm := v1.NewMinimalVMI("testvm")
			for i := 1; i < 3; i++ {
				iface := v1.DefaultNetworkInterface()
				net := v1.DefaultPodNetwork()

				// make sure whatever the error we receive is not related to duplicate names
				name := fmt.Sprintf("podnet%d", i)
				iface.Name = name
				net.Name = name

				vm.Spec.Domain.Devices.Interfaces = append(vm.Spec.Domain.Devices.Interfaces, *iface)
				vm.Spec.Networks = append(vm.Spec.Networks, *net)
			}

			causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vm.Spec)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake.interfaces"))
		})

		It("should accept valid MAC address", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for _, macAddress := range []string{"de:ad:00:00:be:af", "de-ad-00-00-be-af"} {
				vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = macAddress // missing octet
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
				// if this is processed correctly, it should not result in any error
				Expect(len(causes)).To(Equal(0))
			}
		})

		It("should reject invalid MAC addresses", func() {
			vmi := v1.NewMinimalVMI("testvm")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			for _, macAddress := range []string{"de:ad:00:00:be", "de-ad-00-00-be", "de:ad:00:00:be:af:be:af"} {
				vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = macAddress
				causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("fake"), &vmi.Spec)
				Expect(len(causes)).To(Equal(1))
				Expect(causes[0].Field).To(Equal("fake.domain.devices.interfaces[0].macAddress"))
			}
		})
	})
	Context("with Volume", func() {
		table.DescribeTable("should accept valid volumes",
			func(volumeSource v1.VolumeSource) {
				vmi := v1.NewMinimalVMI("testvmi")
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name:         "testvolume",
					VolumeSource: volumeSource,
				})

				causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
				Expect(len(causes)).To(Equal(0))
			},
			table.Entry("with pvc volume source", v1.VolumeSource{PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{}}),
			table.Entry("with cloud-init volume source", v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: "fake"}}),
			table.Entry("with registryDisk volume source", v1.VolumeSource{RegistryDisk: &v1.RegistryDiskSource{}}),
			table.Entry("with ephemeral volume source", v1.VolumeSource{Ephemeral: &v1.EphemeralVolumeSource{}}),
			table.Entry("with emptyDisk volume source", v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}}),
		)
		It("should reject volume with no volume source set", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})
		It("should reject volume with multiple volume sources set", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk:          &v1.RegistryDiskSource{},
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})
		It("should reject volumes with duplicate names", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{},
				},
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[1].name"))
		})
		table.DescribeTable("should verify cloud-init userdata length", func(userDataLen int, expectedErrors int, base64Encode bool) {
			vmi := v1.NewMinimalVMI("testvmi")

			// generate fake userdata
			userdata := ""
			for i := 0; i < userDataLen; i++ {
				userdata = fmt.Sprintf("%sa", userdata)
			}

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{VolumeSource: v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{}}})

			if base64Encode {
				vmi.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserDataBase64 = base64.StdEncoding.EncodeToString([]byte(userdata))
			} else {
				vmi.Spec.Volumes[0].VolumeSource.CloudInitNoCloud.UserData = userdata
			}

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
			Expect(len(causes)).To(Equal(expectedErrors))
			for _, cause := range causes {
				Expect(cause.Field).To(ContainSubstring("fake[0].cloudInitNoCloud"))
			}
		},
			table.Entry("should accept userdata under max limit", 10, 0, false),
			table.Entry("should accept userdata equal max limit", cloudInitMaxLen, 0, false),
			table.Entry("should reject userdata greater than max limit", cloudInitMaxLen+1, 1, false),
			table.Entry("should accept userdata base64 under max limit", 10, 0, true),
			table.Entry("should accept userdata base64 equal max limit", cloudInitMaxLen, 0, true),
			table.Entry("should reject userdata base64 greater than max limit", cloudInitMaxLen+1, 1, true),
		)

		It("should reject cloud-init with invalid base64 data", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				VolumeSource: v1.VolumeSource{
					CloudInitNoCloud: &v1.CloudInitNoCloudSource{
						UserDataBase64: "#######garbage******",
					},
				},
			})

			causes := validateVolumes(k8sfield.NewPath("fake"), vmi.Spec.Volumes)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].cloudInitNoCloud.userDataBase64"))
		})
	})
	Context("with Disk", func() {
		table.DescribeTable("should accept valid disks",
			func(disk v1.Disk) {
				vmi := v1.NewMinimalVMI("testvmi")

				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, disk)

				causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
				Expect(len(causes)).To(Equal(0))

			},
			table.Entry("with Disk target",
				v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{}}},
			),
			table.Entry("with LUN target",
				v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{}}},
			),
			table.Entry("with Floppy target",
				v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{Floppy: &v1.FloppyTarget{}}},
			),
			table.Entry("with CDRom target",
				v1.Disk{Name: "testdisk", VolumeName: "testvolume", DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{}}},
			),
		)
		It("should allow disk without a target", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
				// disk without a target defaults to DiskTarget
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{Image: "fake"},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(0))
		})
		It("should reject disks with duplicate names ", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume1",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume2",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})
			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[1].name"))
		})

		It("should reject disk with multiple targets ", func() {
			vmi := v1.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
				DiskDevice: v1.DiskDevice{
					Disk:   &v1.DiskTarget{},
					Floppy: &v1.FloppyTarget{},
				},
			})
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "testvolume",
				VolumeSource: v1.VolumeSource{
					RegistryDisk: &v1.RegistryDiskSource{Image: "fake"},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0]"))
		})

		It("should accept a boot order greater than '0'", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			order := uint(1)

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume1",
				BootOrder:  &order,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(0))
		})

		It("should reject a disk with a boot order of '0'", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			order := uint(0)

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume1",
				BootOrder:  &order,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].bootorder"))
		})

		It("should reject invalid SN characters", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			order := uint(1)
			sn := "$$$$"

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk2",
				VolumeName: "testvolume2",
				BootOrder:  &order,
				Serial:     sn,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].serial"))
		})

		It("should reject SN > maxStrLen characters", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			order := uint(1)
			sn := strings.Repeat("1", maxStrLen+1)

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk2",
				VolumeName: "testvolume2",
				BootOrder:  &order,
				Serial:     sn,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(1))
			Expect(causes[0].Field).To(Equal("fake[0].serial"))
		})

		It("should accept valid SN", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			order := uint(1)

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk2",
				VolumeName: "testvolume2",
				BootOrder:  &order,
				Serial:     "SN-1_a",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			})

			causes := validateDisks(k8sfield.NewPath("fake"), vmi.Spec.Domain.Devices.Disks)
			Expect(len(causes)).To(Equal(0))
		})

	})
})

var _ = Describe("Function getNumberOfPodInterfaces()", func() {

	It("should work for empty network list", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		Expect(getNumberOfPodInterfaces(spec)).To(Equal(0))
	})

	It("should work for non-empty network list without pod network", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net := v1.Network{}
		spec.Networks = []v1.Network{net}
		Expect(getNumberOfPodInterfaces(spec)).To(Equal(0))
	})

	It("should work for pod network with missing pod interface", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
		}
		spec.Networks = []v1.Network{net}
		Expect(getNumberOfPodInterfaces(spec)).To(Equal(0))
	})

	It("should work for valid pod network / interface combination", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
			Name: "testnet",
		}
		iface := v1.Interface{Name: net.Name}
		spec.Networks = []v1.Network{net}
		spec.Domain.Devices.Interfaces = []v1.Interface{iface}
		Expect(getNumberOfPodInterfaces(spec)).To(Equal(1))
	})

	It("should work for multiple pod network / interface combinations", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		net1 := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
			Name: "testnet1",
		}
		iface1 := v1.Interface{Name: net1.Name}
		net2 := v1.Network{
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
			Name: "testnet2",
		}
		iface2 := v1.Interface{Name: net2.Name}
		spec.Networks = []v1.Network{net1, net2}
		spec.Domain.Devices.Interfaces = []v1.Interface{iface1, iface2}
		Expect(getNumberOfPodInterfaces(spec)).To(Equal(2))
	})
})

type virtualMachineBuilder struct {
	disks   []v1.Disk
	volumes []v1.Volume
	labels  map[string]string
}

func (b *virtualMachineBuilder) WithDisk(disk v1.Disk) *virtualMachineBuilder {
	b.disks = append(b.disks, disk)
	return b
}

func (b *virtualMachineBuilder) WithLabel(key string, value string) *virtualMachineBuilder {
	b.labels[key] = value
	return b
}

func (b *virtualMachineBuilder) WithVolume(volume v1.Volume) *virtualMachineBuilder {
	b.volumes = append(b.volumes, volume)
	return b
}

func (b *virtualMachineBuilder) Build() *v1.VirtualMachineInstance {

	vmi := v1.NewMinimalVMI("testvmi")
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, b.disks...)
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, b.volumes...)
	vmi.Labels = b.labels

	return vmi
}

func (b *virtualMachineBuilder) BuildTemplate() *v1.VirtualMachineInstanceTemplateSpec {
	vmi := b.Build()

	return &v1.VirtualMachineInstanceTemplateSpec{
		ObjectMeta: vmi.ObjectMeta,
		Spec:       vmi.Spec,
	}

}

func newVirtualMachineBuilder() *virtualMachineBuilder {
	return &virtualMachineBuilder{
		labels: map[string]string{},
	}
}
