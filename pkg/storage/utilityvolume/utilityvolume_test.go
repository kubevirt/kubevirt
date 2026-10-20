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

package utilityvolume

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"

	corev1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Utility volumes", func() {
	const volumeName = "scratch-volume"
	var vmi *v1.VirtualMachineInstance

	utilityVolume := func(name string, volumeType v1.UtilityVolumeType) v1.UtilityVolume {
		return v1.UtilityVolume{
			Name: name,
			PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: name + "-pvc",
			},
			Type: pointer.P(volumeType),
		}
	}

	BeforeEach(func() {
		vmi = libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("test-vmi"))
	})

	Context("find", func() {
		It("should find the volume of the expected claim and type", func() {
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{utilityVolume(volumeName, v1.SnapshotOverlay)}
			Expect(Find(vmi, volumeName, volumeName+"-pvc", v1.SnapshotOverlay)).To(BeTrue())
		})

		It("should not find a volume that is not in the spec", func() {
			Expect(Find(vmi, volumeName, volumeName+"-pvc", v1.SnapshotOverlay)).To(BeFalse())
			Expect(Find(nil, volumeName, volumeName+"-pvc", v1.SnapshotOverlay)).To(BeFalse())
		})

		DescribeTable("should refuse another source under the name", func(claimName string, volumeType v1.UtilityVolumeType) {
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{utilityVolume(volumeName, v1.SnapshotOverlay)}
			_, err := Find(vmi, volumeName, claimName, volumeType)
			Expect(err).To(MatchError(ContainSubstring("different source")))
		},
			Entry("another claim", "other-pvc", v1.SnapshotOverlay),
			Entry("another type", volumeName+"-pvc", v1.Backup),
		)
	})

	Context("add patch", func() {
		apply := func(payload []byte, target *v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error) {
			patch, err := jsonpatch.DecodePatch(payload)
			Expect(err).ToNot(HaveOccurred())
			original, err := json.Marshal(target)
			Expect(err).ToNot(HaveOccurred())
			patched, err := patch.Apply(original)
			if err != nil {
				return nil, err
			}
			updated := &v1.VirtualMachineInstance{}
			Expect(json.Unmarshal(patched, updated)).To(Succeed())
			return updated, nil
		}

		BeforeEach(func() {
			vmi.UID, vmi.ResourceVersion = "vmi-uid", "1"
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{utilityVolume("other-volume", v1.Backup)}
		})

		It("should add the volume next to the existing ones", func() {
			payload, err := AddPatch(vmi, volumeName, volumeName+"-pvc", v1.SnapshotOverlay)
			Expect(err).ToNot(HaveOccurred())
			updated, err := apply(payload, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Spec.UtilityVolumes).To(Equal([]v1.UtilityVolume{
				utilityVolume("other-volume", v1.Backup),
				utilityVolume(volumeName, v1.SnapshotOverlay),
			}))
		})

		It("should not apply to a VMI that changed in between", func() {
			payload, err := AddPatch(vmi, volumeName, volumeName+"-pvc", v1.SnapshotOverlay)
			Expect(err).ToNot(HaveOccurred())
			changed := vmi.DeepCopy()
			changed.ResourceVersion = "2"
			_, err = apply(payload, changed)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("attached", func() {
		DescribeTable("should be", func(status []v1.VolumeStatus, attached bool) {
			vmi.Status.VolumeStatus = status
			Expect(Attached(vmi, volumeName)).To(Equal(attached))
		},
			Entry("false with no volume status at all", nil, false),
			Entry("false while the volume is still being mounted",
				[]v1.VolumeStatus{{Name: volumeName, HotplugVolume: &v1.HotplugVolumeStatus{}, Phase: v1.HotplugVolumeAttachedToNode}}, false),
			Entry("false for a volume that is not hotplugged",
				[]v1.VolumeStatus{{Name: volumeName, Phase: v1.HotplugVolumeMounted}}, false),
			Entry("true once it is mounted",
				[]v1.VolumeStatus{{Name: volumeName, HotplugVolume: &v1.HotplugVolumeStatus{}, Phase: v1.HotplugVolumeMounted}}, true),
			Entry("false for somebody else's mounted volume",
				[]v1.VolumeStatus{{Name: "memory-dump", HotplugVolume: &v1.HotplugVolumeStatus{}, Phase: v1.HotplugVolumeMounted}}, false),
		)

		It("should be false without a VMI", func() {
			Expect(Attached(nil, volumeName)).To(BeFalse())
		})
	})

	Context("detached", func() {
		It("should be false while the volume is still in the spec", func() {
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{utilityVolume(volumeName, v1.SnapshotOverlay)}
			Expect(Detached(vmi, volumeName)).To(BeFalse())
		})

		It("should be false while the volume is still in the status", func() {
			// The spec patch landed but the volume is still mounted, and deleting
			// the PVC here is what hangs it on pvc-protection.
			vmi.Status.VolumeStatus = []v1.VolumeStatus{{Name: volumeName, Phase: v1.HotplugVolumeMounted}}
			Expect(Detached(vmi, volumeName)).To(BeFalse())
		})

		It("should be true once the volume is in neither", func() {
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{utilityVolume("memory-dump", v1.MemoryDump)}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{{Name: "memory-dump", Phase: v1.HotplugVolumeMounted}}
			Expect(Detached(vmi, volumeName)).To(BeTrue())
		})

		It("should be true without a VMI", func() {
			Expect(Detached(nil, volumeName)).To(BeTrue())
		})
	})
})
