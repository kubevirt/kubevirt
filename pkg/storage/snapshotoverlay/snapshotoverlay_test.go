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

package snapshotoverlay

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"

	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

var _ = Describe("Snapshot overlay", func() {
	Context("ScratchDir", func() {
		const contentUID = "content-uid"
		scratch := storagetypes.ScratchPVCName(contentUID)

		vmiWith := func(withReference bool, utilityVolumes ...string) *v1.VirtualMachineInstance {
			vmi := &v1.VirtualMachineInstance{}
			if withReference {
				encoded, err := json.Marshal(storagetypes.SnapshotOverlayReference{ContentUID: contentUID, ScratchPVCUID: "scratch-uid"})
				Expect(err).ToNot(HaveOccurred())
				vmi.Annotations = map[string]string{storagetypes.OverlaySnapshotReferenceAnnotation: string(encoded)}
			}
			for _, name := range utilityVolumes {
				vmi.Spec.UtilityVolumes = append(vmi.Spec.UtilityVolumes, v1.UtilityVolume{
					Name:                              name,
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: name},
				})
			}
			return vmi
		}

		It("should return the mount directory of the attached scratch volume", func() {
			Expect(ScratchDir(vmiWith(true, scratch), scratch)).To(Equal(hotplugdisk.GetVolumeMountDir(scratch)))
		})

		DescribeTable("should refuse", func(vmi *v1.VirtualMachineInstance, volumeName, message string) {
			_, err := ScratchDir(vmi, volumeName)
			Expect(err).To(MatchError(ContainSubstring(message)))
		},
			Entry("a VMI without a reference", vmiWith(false, scratch), scratch, "no matching transaction"),
			Entry("a volume the reference does not name", vmiWith(true, "other"), "other", "no matching transaction"),
			Entry("a scratch volume that is not attached", vmiWith(true), scratch, "is not a utility volume"),
		)
	})
})
