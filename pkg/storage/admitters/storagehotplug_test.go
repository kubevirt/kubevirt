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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/client-go/api"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("Storage Hotplug Admitter", func() {
	const kubeVirtNamespace = "kubevirt"

	kv := &v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt",
			Namespace: "kubevirt",
		},
		Spec: v1.KubeVirtSpec{
			Configuration: v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{},
			},
		},
		Status: v1.KubeVirtStatus{
			Phase: v1.KubeVirtPhaseDeploying,
		},
	}
	config, _, kvStore := testutils.NewFakeClusterConfigUsingKV(kv)

	enableFeatureGate := func(featureGate string) {
		kvConfig := kv.DeepCopy()
		kvConfig.Spec.Configuration.DeveloperConfiguration.FeatureGates = []string{featureGate}
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvConfig)
	}
	disableFeatureGates := func() {
		testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kv)
	}

	AfterEach(func() {
		disableFeatureGates()
	})

	makeVolumes := func(indexes ...int) []v1.Volume {
		res := make([]v1.Volume, 0)
		for _, index := range indexes {
			res = append(res, v1.Volume{
				Name: fmt.Sprintf("volume-name-%d", index),
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: fmt.Sprintf("dv-name-%d", index),
					},
				},
			})
		}
		return res
	}

	makeStatus := func(statusCount, hotplugCount int) []v1.VolumeStatus {
		res := make([]v1.VolumeStatus, 0)
		for i := 0; i < statusCount; i++ {
			res = append(res, v1.VolumeStatus{
				Name:   fmt.Sprintf("volume-name-%d", i),
				Target: fmt.Sprintf("volume-target-%d", i),
			})
			if i >= statusCount-hotplugCount {
				res[i].HotplugVolume = &v1.HotplugVolumeStatus{
					AttachPodName: fmt.Sprintf("test-pod-%d", i),
				}
				res[i].Phase = v1.VolumeReady
			}
		}
		return res
	}

	makeDisksWithBus := func(bus v1.DiskBus, indexes ...int) []v1.Disk {
		res := make([]v1.Disk, 0)
		for _, index := range indexes {
			bootOrder := uint(index + 1)
			res = append(res, v1.Disk{
				Name: fmt.Sprintf("volume-name-%d", index),
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: bus,
					},
				},
				BootOrder: &bootOrder,
			})
		}
		return res
	}

	makeDisks := func(indexes ...int) []v1.Disk {
		return makeDisksWithBus(v1.DiskBusSCSI, indexes...)
	}

	makeLUNDisks := func(indexes ...int) []v1.Disk {
		res := make([]v1.Disk, 0)
		for _, index := range indexes {
			bootOrder := uint(index + 1)
			res = append(res, v1.Disk{
				Name: fmt.Sprintf("volume-name-%d", index),
				DiskDevice: v1.DiskDevice{
					LUN: &v1.LunTarget{
						Bus: "scsi",
					},
				},
				BootOrder: &bootOrder,
			})
		}
		return res
	}

	makeCDRomDisks := func(indexes ...int) []v1.Disk {
		res := make([]v1.Disk, 0)
		for _, index := range indexes {
			bootOrder := uint(index + 1)
			res = append(res, v1.Disk{
				Name: fmt.Sprintf("volume-name-%d", index),
				DiskDevice: v1.DiskDevice{
					CDRom: &v1.CDRomTarget{
						Bus: "scsi",
					},
				},
				BootOrder: &bootOrder,
			})
		}
		return res
	}

	makeFilesystems := func(indexes ...int) []v1.Filesystem {
		res := make([]v1.Filesystem, 0)
		for _, index := range indexes {
			res = append(res, v1.Filesystem{
				Name:     fmt.Sprintf("volume-name-%d", index),
				Virtiofs: &v1.FilesystemVirtiofs{},
			})
		}
		return res
	}

	emptyResult := func() map[string]v1.Volume {
		return make(map[string]v1.Volume, 0)
	}

	makeResult := func(indexes ...int) map[string]v1.Volume {
		res := emptyResult()
		for _, index := range indexes {
			res[fmt.Sprintf("volume-name-%d", index)] = v1.Volume{
				Name: fmt.Sprintf("volume-name-%d", index),
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: fmt.Sprintf("dv-name-%d", index),
					},
				},
			}
		}
		return res
	}

	makeVolumesWithMemoryDumpVol := func(total int, indexes ...int) []v1.Volume {
		res := make([]v1.Volume, 0)
		for i := 0; i < total; i++ {
			memoryDump := false
			for _, index := range indexes {
				if i == index {
					memoryDump = true
					res = append(res, v1.Volume{
						Name: fmt.Sprintf("volume-name-%d", index),
						VolumeSource: v1.VolumeSource{
							MemoryDump: testutils.NewFakeMemoryDumpSource(fmt.Sprintf("volume-name-%d", index)),
						},
					})
				}
			}
			if !memoryDump {
				res = append(res, v1.Volume{
					Name: fmt.Sprintf("volume-name-%d", i),
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: fmt.Sprintf("dv-name-%d", i),
						},
					},
				})
			}
		}
		return res
	}

	makeInvalidVolumes := func(total int, indexes ...int) []v1.Volume {
		res := make([]v1.Volume, 0)
		for i := 0; i < total; i++ {
			foundInvalid := false
			for _, index := range indexes {
				if i == index {
					foundInvalid = true
					res = append(res, v1.Volume{
						Name: fmt.Sprintf("volume-name-%d", index),
						VolumeSource: v1.VolumeSource{
							ContainerDisk: testutils.NewFakeContainerDiskSource(),
						},
					})
				}
			}
			if !foundInvalid {
				res = append(res, v1.Volume{
					Name: fmt.Sprintf("volume-name-%d", i),
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: fmt.Sprintf("dv-name-%d", i),
						},
					},
				})
			}
		}
		return res
	}

	makeDisksInvalidBusLastDisk := func(indexes ...int) []v1.Disk {
		res := makeDisks(indexes...)
		if len(res) > 0 {
			res[len(res)-1].Disk.Bus = "invalid"
		}
		return res
	}

	makeLUNDisksInvalidBusLastDisk := func(indexes ...int) []v1.Disk {
		res := makeLUNDisks(indexes...)
		if len(res) > 0 {
			res[len(res)-1].LUN.Bus = "invalid"
		}
		return res
	}

	makeDisksWithIOThreadsAndBus := func(bus v1.DiskBus, indexes ...int) []v1.Disk {
		res := makeDisksWithBus(bus, indexes...)
		if len(res) > 0 {
			res[len(res)-1].DedicatedIOThread = pointer.P(true)
		}
		return res
	}

	makeDisksWithIOThreads := func(indexes ...int) []v1.Disk {
		res := makeDisks(indexes...)
		if len(res) > 0 {
			res[len(res)-1].DedicatedIOThread = pointer.P(true)
		}
		return res
	}

	makeDisksInvalidBootOrder := func(indexes ...int) []v1.Disk {
		res := makeDisks(indexes...)
		if len(res) > 0 {
			res[len(res)-1].BootOrder = pointer.P(uint(0))
		}
		return res
	}

	makeDisksNoVolume := func(indexes ...int) []v1.Disk {
		res := make([]v1.Disk, 0)
		for _, index := range indexes {
			res = append(res, v1.Disk{
				Name: fmt.Sprintf("invalid-volume-name-%d", index),
			})
		}
		return res
	}

	makeExpected := func(message, field string) *admissionv1.AdmissionResponse {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: message,
				Field:   field,
			},
		})
	}

	DescribeTable("Should properly calculate the hotplugvolumes", func(volumes []v1.Volume, statuses []v1.VolumeStatus, expected map[string]v1.Volume) {
		result := getHotplugVolumes(volumes, statuses)
		Expect(equality.Semantic.DeepEqual(result, expected)).To(BeTrue(), "result: %v and expected: %v do not match", result, expected)
	},
		Entry("Should be empty if statuses is empty", makeVolumes(), makeStatus(0, 0), emptyResult()),
		Entry("Should be empty if statuses has multiple entries, but no hotplug", makeVolumes(), makeStatus(2, 0), emptyResult()),
		Entry("Should be empty if statuses has one entry, but no hotplug", makeVolumes(), makeStatus(1, 0), emptyResult()),
		Entry("Should have a single hotplug if status has one hotplug", makeVolumes(0, 1), makeStatus(2, 1), makeResult(1)),
		Entry("Should have a multiple hotplug if status has multiple hotplug", makeVolumes(0, 1, 2, 3), makeStatus(4, 2), makeResult(2, 3)),
	)

	DescribeTable("Should properly calculate the permanent volumes", func(volumes []v1.Volume, statusVolumes []v1.VolumeStatus, expected map[string]v1.Volume) {
		result := getPermanentVolumes(volumes, statusVolumes)
		Expect(equality.Semantic.DeepEqual(result, expected)).To(BeTrue(), "result: %v and expected: %v do not match", result, expected)
	},
		Entry("Should be empty if volume is empty", makeVolumes(), makeStatus(0, 0), emptyResult()),
		Entry("Should be empty if all volumes are hotplugged", makeVolumes(0, 1, 2, 3), makeStatus(4, 4), emptyResult()),
		Entry("Should return all volumes if hotplugged is empty with multiple volumes", makeVolumes(0, 1, 2, 3), makeStatus(4, 0), makeResult(0, 1, 2, 3)),
		Entry("Should return all volumes if hotplugged is empty with a single volume", makeVolumes(0), makeStatus(1, 0), makeResult(0)),
		Entry("Should return 3 volumes if  1 hotplugged volume", makeVolumes(0, 1, 2, 3), makeStatus(4, 1), makeResult(0, 1, 2)),
	)

	testHotplugResponse := func(newVolumes, oldVolumes []v1.Volume, newDisks, oldDisks []v1.Disk, filesystems []v1.Filesystem, volumeStatuses []v1.VolumeStatus, expected *admissionv1.AdmissionResponse, featureGates ...string) {
		newVMI := api.NewMinimalVMI("testvmi")
		newVMI.Spec.Volumes = newVolumes
		newVMI.Spec.Domain.Devices.Disks = newDisks
		newVMI.Spec.Domain.Devices.Filesystems = filesystems

		for _, featureGate := range featureGates {
			enableFeatureGate(featureGate)
		}
		result := AdmitHotplugStorage(newVolumes, oldVolumes, newDisks, oldDisks, volumeStatuses, newVMI, config)
		Expect(equality.Semantic.DeepEqual(result, expected)).To(BeTrue(), "result: %v and expected: %v do not match", result, expected)
	}

	DescribeTable("Should return proper admission response", testHotplugResponse,
		Entry("Should accept if no volumes are there or added",
			makeVolumes(),
			makeVolumes(),
			makeDisks(),
			makeDisks(),
			makeFilesystems(),
			makeStatus(0, 0),
			nil),
		Entry("Should reject if #volumes != #disks",
			makeVolumes(1, 2),
			makeVolumes(1, 2),
			makeDisks(1),
			makeDisks(1),
			makeFilesystems(),
			makeStatus(0, 0),
			makeExpected("mismatch between volumes declared (2) and required (1)", "")),
		Entry("Should reject if we remove a permanent volume",
			makeVolumes(),
			makeVolumes(0),
			makeDisks(),
			makeDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("Number of permanent volumes has changed", "")),
		Entry("Should reject if we add a disk without a matching volume",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksNoVolume(0, 1),
			makeDisksNoVolume(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("disk volume-name-1 does not exist", "")),
		Entry("Should reject if we modify existing volume to be invalid",
			makeVolumes(0, 1),
			makeVolumes(0, 1),
			makeDisksNoVolume(0, 1),
			makeDisks(0, 1),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("permanent disk volume-name-0, changed", "")),
		Entry("Should reject if a hotplug volume changed",
			makeInvalidVolumes(2, 1),
			makeVolumes(0, 1),
			makeDisks(0, 1),
			makeDisks(0, 1),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("hotplug volume volume-name-1, changed", "")),
		Entry("Should reject if we add volumes that are not PVC or DV",
			makeInvalidVolumes(2, 1),
			makeVolumes(0),
			makeDisks(0, 1),
			makeDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("volume volume-name-1 is not a PVC or DataVolume", "")),
		Entry("Should accept if we add volumes and disk properly",
			makeVolumes(0, 1),
			makeVolumes(0, 1),
			makeDisks(0, 1),
			makeDisks(0, 1),
			makeFilesystems(),
			makeStatus(2, 1),
			nil),
		Entry("Should accept if we add volumes and disk properly (virtio bus)",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksWithBus(v1.DiskBusVirtio, 0, 1),
			makeDisksWithBus(v1.DiskBusVirtio, 0),
			makeFilesystems(),
			makeStatus(1, 0),
			nil),
		Entry("Should reject if we hotplug a volume with dedicated IOThreads",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksWithIOThreads(0, 1),
			makeDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("Hotplug configuration for [volume-name-1] requires virtio bus for IOThreads.", "")),
		Entry("Should accept if we hotplug a virtio volume with dedicated IOThreads",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksWithIOThreadsAndBus(v1.DiskBusVirtio, 0, 1),
			makeDisksWithBus(v1.DiskBusVirtio, 0),
			makeFilesystems(),
			makeStatus(1, 0),
			nil),
		Entry("Should accept if we add LUN disk with valid SCSI bus",
			makeVolumes(0, 1),
			makeVolumes(0, 1),
			makeLUNDisks(0, 1),
			makeLUNDisks(0, 1),
			makeFilesystems(),
			makeStatus(2, 1),
			nil),
		Entry("Should reject if we add disk with invalid bus",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksInvalidBusLastDisk(0, 1),
			makeDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("Hotplug configuration for disk [volume-name-1] requires bus to be 'scsi' or 'virtio'. [invalid] is not permitted.", "")),
		Entry("Should reject if we add LUN disk with invalid bus",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeLUNDisksInvalidBusLastDisk(0, 1),
			makeLUNDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("Hotplug configuration for LUN [volume-name-1] requires bus to be 'scsi'. [invalid] is not permitted.", "")),
		Entry("Should reject if we add disk with neither Disk nor LUN type",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeCDRomDisks(0, 1),
			makeCDRomDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("Hotplug configuration for [volume-name-1] requires diskDevice of type 'disk' or 'lun' to be used.", "")),
		Entry("Should allow cd-rom inject",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeCDRomDisks(0, 1),
			makeCDRomDisks(0, 1),
			makeFilesystems(),
			makeStatus(1, 0),
			nil),
		Entry("Should allow cd-rom eject",
			makeVolumes(0),
			makeVolumes(0, 1),
			makeCDRomDisks(0, 1),
			makeCDRomDisks(0, 1),
			makeFilesystems(),
			makeStatus(1, 0),
			nil,
			featuregate.DeclarativeHotplugVolumesGate),
		Entry("Should reject cd-rom eject",
			makeVolumes(0),
			makeVolumes(0, 1),
			makeCDRomDisks(0, 1),
			makeCDRomDisks(0, 1),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("mismatch between volumes declared (1) and required (2)", "")),
		Entry("Should reject if we add disk with invalid boot order",
			makeVolumes(0, 1),
			makeVolumes(0),
			makeDisksInvalidBootOrder(0, 1),
			makeDisks(0),
			makeFilesystems(),
			makeStatus(1, 0),
			makeExpected("spec.domain.devices.disks[1] must have a boot order > 0, if supplied", "spec.domain.devices.disks[1].bootOrder")),
		Entry("Should accept if memory dump volume exists without matching disk",
			makeVolumesWithMemoryDumpVol(3, 2),
			makeVolumes(0, 1),
			makeDisks(0, 1),
			makeDisks(0, 1),
			makeFilesystems(),
			makeStatus(3, 1),
			nil),
		Entry("Should reject if #volumes != #disks even when there is memory dump volume",
			makeVolumesWithMemoryDumpVol(3, 2),
			makeVolumesWithMemoryDumpVol(3, 2),
			makeDisks(1),
			makeDisks(1),
			makeFilesystems(),
			makeStatus(0, 0),
			makeExpected("mismatch between volumes declared (2) and required (1)", "")),
	)

	Context("with filesystem devices", func() {
		BeforeEach(func() {
			enableFeatureGate(featuregate.VirtIOFSStorageVolumeGate)
		})

		DescribeTable("Should return proper admission response", testHotplugResponse,
			Entry("Should accept if volume without matching disk is used by filesystem",
				makeVolumes(0, 1, 2),
				makeVolumes(0, 1),
				makeDisks(0, 2),
				makeDisks(0),
				makeFilesystems(1),
				makeStatus(3, 1),
				nil),
			Entry("Should reject if #volumes != #disks even when there are volumes used by filesystems",
				makeVolumes(0, 1, 2),
				makeVolumes(0, 1, 2),
				makeDisks(0),
				makeDisks(0),
				makeFilesystems(1),
				makeStatus(2, 0),
				makeExpected("mismatch between volumes declared (3) and required (2)", "")),
		)
	})

	DescribeTable("should allow change for a persistent volume if it is a migrated volume", func(hotpluggable bool) {
		disks := []v1.Disk{
			{
				Name:       "vol0",
				DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}},
			},
			{
				Name:       "vol1",
				DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}},
			},
		}
		oldVols := []v1.Volume{
			{
				Name: "vol0",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc0"},
						Hotpluggable:                      hotpluggable,
					},
				},
			},
			{
				Name: "vol1",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"},
						Hotpluggable:                      hotpluggable,
					},
				},
			},
		}
		newVols := []v1.Volume{
			{
				Name: "vol0",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc0"},
						Hotpluggable:                      hotpluggable,
					},
				},
			},
			{
				Name: "vol1",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc2"},
						Hotpluggable:                      hotpluggable,
					},
				},
			},
		}
		volumeStatuses := []v1.VolumeStatus{
			{Name: "vol0"},
			{Name: "vol1"},
		}
		vmi := api.NewMinimalVMI("testvmi")
		vmi.Status.MigratedVolumes = []v1.StorageMigratedVolumeInfo{
			{
				VolumeName:         "vol1",
				SourcePVCInfo:      &v1.PersistentVolumeClaimInfo{ClaimName: "pvc1"},
				DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{ClaimName: "pvc1"},
			},
		}
		Expect(AdmitHotplugStorage(newVols, oldVols, disks, disks, volumeStatuses, vmi, config)).To(BeNil())
	},
		Entry("and not hotpluggable", false),
		Entry("and hotpluggable", true),
	)
})

var _ = Describe("Utility Volumes Admitter", func() {
	Context("with feature gate enabled", func() {
		var config *virtconfig.ClusterConfig

		BeforeEach(func() {
			kv := &v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{
					FeatureGates: []string{featuregate.UtilityVolumesGate},
				},
			}
			config, _, _ = testutils.NewFakeClusterConfigUsingKVConfig(kv)
		})

		It("should accept valid utility volumes on creation", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				Volumes: []v1.Volume{{Name: "vol1", VolumeSource: v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}}}},
				UtilityVolumes: []v1.UtilityVolume{
					{Name: "util-vol1", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"}},
				},
			}

			result := AdmitUtilityVolumes(newSpec, nil, nil, config)
			Expect(result).To(BeNil())
		})

		It("should accept empty utility volumes list", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				Volumes:        []v1.Volume{{Name: "vol1", VolumeSource: v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}}}},
				UtilityVolumes: []v1.UtilityVolume{},
			}

			result := AdmitUtilityVolumes(newSpec, nil, nil, config)
			Expect(result).To(BeNil())
		})

		It("should reject duplicate utility volume names", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{
					{Name: "dup-vol", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"}},
					{Name: "dup-vol", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc2"}},
				},
			}

			result := AdmitUtilityVolumes(newSpec, nil, nil, config)
			Expect(result).ToNot(BeNil())
			Expect(result.Result.Message).To(ContainSubstring("must not have the same Name"))
			Expect(result.Result.Details.Causes).To(HaveLen(1))
			Expect(result.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueDuplicate))
			Expect(result.Result.Details.Causes[0].Field).To(Equal("spec.utilityVolumes[1].name"))
		})

		It("should reject utility volume name conflicting with regular volume", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				Volumes:        []v1.Volume{{Name: "conflict-vol", VolumeSource: v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}}}},
				UtilityVolumes: []v1.UtilityVolume{{Name: "conflict-vol", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"}}},
			}

			result := AdmitUtilityVolumes(newSpec, nil, nil, config)
			Expect(result).ToNot(BeNil())
			Expect(result.Result.Message).To(ContainSubstring("conflicts with"))
			Expect(result.Result.Details.Causes).To(HaveLen(1))
			Expect(result.Result.Details.Causes[0].Type).To(Equal(metav1.CauseTypeFieldValueDuplicate))
			Expect(result.Result.Details.Causes[0].Field).To(Equal("spec.utilityVolumes[0].name"))
		})

		It("should reject utility volume with empty PVC name", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{{Name: "util-vol", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: ""}}},
			}

			result := AdmitUtilityVolumes(newSpec, nil, nil, config)
			Expect(result).ToNot(BeNil())
			Expect(result.Result.Message).To(ContainSubstring("claimName' must be set"))
		})

		It("should reject utility volume with empty name", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{{Name: "", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"}}},
			}

			result := AdmitUtilityVolumes(newSpec, nil, nil, config)
			Expect(result).ToNot(BeNil())
			Expect(result.Result.Message).To(ContainSubstring("'name' must be set"))
		})

		It("should handle multiple validation errors", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				Volumes: []v1.Volume{{Name: "conflict-vol", VolumeSource: v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}}}},
				UtilityVolumes: []v1.UtilityVolume{
					{Name: "conflict-vol", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"}},
					{Name: "empty-pvc", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: ""}},
				},
			}

			result := AdmitUtilityVolumes(newSpec, nil, nil, config)
			Expect(result).ToNot(BeNil())
			// Should fail on first validation error found
			Expect(result.Result.Message).To(SatisfyAny(
				ContainSubstring("conflicts with"),
				ContainSubstring("claimName' must be set"),
			))
		})

		It("should reject utility volumes marked as permanent (must be hotplug)", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{
					{Name: "util-vol", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"}},
				},
			}
			oldSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{},
			}

			volumeStatuses := []v1.VolumeStatus{
				{Name: "util-vol", HotplugVolume: nil},
			}

			result := AdmitUtilityVolumes(newSpec, oldSpec, volumeStatuses, config)
			Expect(result).ToNot(BeNil())
			Expect(result.Result.Message).To(ContainSubstring("cannot be a permanent volume"))
		})

		It("should accept new utility volumes being added on update", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{
					{Name: "existing-util", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"}},
					{Name: "new-util", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc2"}},
				},
			}
			oldSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{
					{Name: "existing-util", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"}},
				},
			}

			volumeStatuses := []v1.VolumeStatus{
				{Name: "existing-util", HotplugVolume: &v1.HotplugVolumeStatus{}},
			}

			result := AdmitUtilityVolumes(newSpec, oldSpec, volumeStatuses, config)
			Expect(result).To(BeNil())
		})

		It("should accept utility volumes being removed on update", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{
					{Name: "remaining-util", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"}},
				},
			}
			oldSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{
					{Name: "remaining-util", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"}},
					{Name: "removed-util", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc2"}},
				},
			}

			volumeStatuses := []v1.VolumeStatus{
				{Name: "remaining-util", HotplugVolume: &v1.HotplugVolumeStatus{}},
				{Name: "removed-util", HotplugVolume: &v1.HotplugVolumeStatus{}},
			}

			result := AdmitUtilityVolumes(newSpec, oldSpec, volumeStatuses, config)
			Expect(result).To(BeNil())
		})

		It("should reject changes to existing utility volumes", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{
					{Name: "util-vol", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc2"}}, // changed claim
				},
			}
			oldSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{
					{Name: "util-vol", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"}}, // original claim
				},
			}

			volumeStatuses := []v1.VolumeStatus{
				{Name: "util-vol", HotplugVolume: &v1.HotplugVolumeStatus{}},
			}

			result := AdmitUtilityVolumes(newSpec, oldSpec, volumeStatuses, config)
			Expect(result).ToNot(BeNil())
			Expect(result.Result.Message).To(ContainSubstring("has changed"))
		})
	})

	Context("with feature gate disabled", func() {
		var config *virtconfig.ClusterConfig

		BeforeEach(func() {
			kv := &v1.KubeVirtConfiguration{}
			config, _, _ = testutils.NewFakeClusterConfigUsingKVConfig(kv)
		})

		It("should reject utility volumes when feature gate is disabled", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{
					{Name: "util-vol", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc1"}},
				},
			}

			result := AdmitUtilityVolumes(newSpec, nil, nil, config)
			Expect(result).ToNot(BeNil())
			Expect(result.Result.Message).To(Equal("UtilityVolumes feature gate is not enabled"))
		})

		It("should accept empty utility volumes when feature gate is disabled", func() {
			newSpec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{},
			}

			result := AdmitUtilityVolumes(newSpec, nil, nil, config)
			Expect(result).To(BeNil())
		})
	})

	Context("ValidateUtilityVolumesNotPresentOnCreation", func() {
		It("should reject VMI spec with utility volumes on creation", func() {
			spec := &v1.VirtualMachineInstanceSpec{
				UtilityVolumes: []v1.UtilityVolume{
					{
						Name: "test-utility-volume",
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-pvc",
						},
					},
				},
			}

			causes := ValidateUtilityVolumesNotPresentOnCreation(field.NewPath("spec"), spec)
			Expect(causes).To(HaveLen(1))
			Expect(causes[0].Type).To(Equal(metav1.CauseTypeFieldValueInvalid))
			Expect(causes[0].Message).To(ContainSubstring("cannot create VMI with utility volumes in spec"))
			Expect(causes[0].Field).To(Equal("spec.utilityVolumes"))
		})
	})
})
