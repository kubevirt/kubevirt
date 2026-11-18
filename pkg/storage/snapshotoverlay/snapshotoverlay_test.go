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
	"encoding/xml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	clientgoapi "kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/controller"
	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
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

	Context("UpdateVMIStatus", func() {
		var (
			vmi         *v1.VirtualMachineInstance
			domain      *api.Domain
			condManager *controller.VirtualMachineInstanceConditionManager
		)

		overlayCondition := func() *v1.VirtualMachineInstanceCondition {
			return condManager.GetCondition(vmi, v1.VirtualMachineInstanceOverlaySnapshotActive)
		}

		BeforeEach(func() {
			vmi = clientgoapi.NewMinimalVMI("testvmi")
			domain = &api.Domain{}
			condManager = controller.NewVirtualMachineInstanceConditionManager()
		})

		expectedReference := `{"contentUID":"content-uid","scratchPVCUID":"scratch-uid"}`
		DescribeTable("should acknowledge only the expected snapshot overlay PVC", func(contentUID, scratchUID, annotation, expected string) {
			vmi.Annotations = map[string]string{storagetypes.OverlaySnapshotReferenceAnnotation: annotation}
			vmi.Status.SnapshotOverlayPVCUID = "previous-scratch"
			overlay := &api.SnapshotOverlayMetadata{}
			Expect(xml.Unmarshal([]byte(`<snapshotOverlay><phase>OverlayCommitted</phase><contentUID>content-uid</contentUID><scratchPVCUID>scratch-uid</scratchPVCUID></snapshotOverlay>`), overlay)).To(Succeed())
			overlay.ContentUID, overlay.ScratchPVCUID = types.UID(contentUID), types.UID(scratchUID)
			domain.Spec.Metadata.KubeVirt.SnapshotOverlay = overlay
			UpdateVMIStatus(vmi, domain, condManager)
			Expect(overlayCondition().Status).To(Equal(k8sv1.ConditionFalse))
			Expect(overlayCondition().Reason).To(Equal("OverlayCommitted"))
			encoded, err := json.Marshal(vmi.Status)
			Expect(err).ToNot(HaveOccurred())
			if expected == "" {
				Expect(string(encoded)).ToNot(ContainSubstring(`"snapshotOverlayPVCUID"`))
			} else {
				Expect(string(encoded)).To(ContainSubstring(`"snapshotOverlayPVCUID":"` + expected + `"`))
			}
			Expect(string(encoded)).ToNot(ContainSubstring(`"snapshotOverlayReference"`))
		},
			Entry("matching transaction", "content-uid", "scratch-uid", expectedReference, "scratch-uid"),
			Entry("different content on the same PVC", "previous-content", "scratch-uid", expectedReference, ""),
			Entry("recreated scratch PVC", "content-uid", "previous-scratch", expectedReference, ""),
			Entry("legacy launcher without identity", "", "", expectedReference, ""),
			Entry("missing expected reference", "content-uid", "scratch-uid", "", ""),
			Entry("invalid expected reference", "content-uid", "scratch-uid", "{", ""),
		)

		DescribeTable("should project the launcher's phase", func(phase api.SnapshotOverlayPhase, status k8sv1.ConditionStatus, reason string) {
			domain.Spec.Metadata.KubeVirt.SnapshotOverlay = &api.SnapshotOverlayMetadata{Phase: phase}

			UpdateVMIStatus(vmi, domain, condManager)

			Expect(overlayCondition()).To(PointTo(MatchFields(IgnoreExtras, Fields{
				"Status": Equal(status),
				"Reason": Equal(reason),
			})))
		},
			Entry("in progress", api.SnapshotOverlayPreparing, k8sv1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlayPreparing),
			Entry("ready", api.SnapshotOverlayReady, k8sv1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlaysReady),
			Entry("committing", api.SnapshotOverlayCommitting, k8sv1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlayCommitting),
			// A failed commit leaves the overlays live, so the window stays open.
			Entry("commit failed", api.SnapshotOverlayCommitFailed, k8sv1.ConditionTrue, v1.VirtualMachineInstanceReasonOverlayCommitFailed),
			// A failed snapshot means the disks are back on base, and only then.
			Entry("snapshot failed", api.SnapshotOverlaySnapshotFailed, k8sv1.ConditionFalse, v1.VirtualMachineInstanceReasonOverlaySnapshotFailed),
		)

		It("should set the condition to True for an unknown phase", func() {
			domain.Spec.Metadata.KubeVirt.SnapshotOverlay = &api.SnapshotOverlayMetadata{Phase: "SomethingNewer"}

			UpdateVMIStatus(vmi, domain, condManager)

			Expect(overlayCondition()).To(PointTo(MatchFields(IgnoreExtras, Fields{
				"Status": Equal(k8sv1.ConditionTrue),
				"Reason": Equal("SomethingNewer"),
			})))
		})

		It("should set the condition message from the launcher's metadata", func() {
			const message = "block commit failed on disk vda"
			domain.Spec.Metadata.KubeVirt.SnapshotOverlay = &api.SnapshotOverlayMetadata{
				Phase:   api.SnapshotOverlayCommitFailed,
				Message: message,
			}

			UpdateVMIStatus(vmi, domain, condManager)

			Expect(overlayCondition().Message).To(Equal(message))
		})

		It("should remove the condition once the launcher clears its metadata", func() {
			vmi.Status.SnapshotOverlayPVCUID = "scratch-uid"
			domain.Spec.Metadata.KubeVirt.SnapshotOverlay = &api.SnapshotOverlayMetadata{Phase: api.SnapshotOverlayReady}
			UpdateVMIStatus(vmi, domain, condManager)

			// The launcher zeroes the struct rather than dropping the element,
			// so an empty phase is what a committed snapshot looks like.
			domain.Spec.Metadata.KubeVirt.SnapshotOverlay = &api.SnapshotOverlayMetadata{}
			UpdateVMIStatus(vmi, domain, condManager)

			Expect(overlayCondition()).To(BeNil())
			Expect(vmi.Status.SnapshotOverlayPVCUID).To(BeEmpty())
		})

		It("should remove the condition when the domain never had overlays", func() {
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{
				Type:   v1.VirtualMachineInstanceOverlaySnapshotActive,
				Status: k8sv1.ConditionTrue,
			}}

			UpdateVMIStatus(vmi, domain, condManager)

			Expect(overlayCondition()).To(BeNil())
		})

		It("should leave the condition alone when the domain could not be read", func() {
			vmi.Status.SnapshotOverlayPVCUID = "scratch-uid"
			// Removing it here would report a commit that may still be running,
			// and the controller would delete the scratch volume under it.
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{
				Type:   v1.VirtualMachineInstanceOverlaySnapshotActive,
				Status: k8sv1.ConditionTrue,
				Reason: v1.VirtualMachineInstanceReasonOverlayCommitting,
			}}

			UpdateVMIStatus(vmi, nil, condManager)

			Expect(vmi.Status.SnapshotOverlayPVCUID).To(Equal(types.UID("scratch-uid")))
			Expect(overlayCondition()).To(PointTo(MatchFields(IgnoreExtras, Fields{
				"Status": Equal(k8sv1.ConditionTrue),
				"Reason": Equal(v1.VirtualMachineInstanceReasonOverlayCommitting),
			})))
		})

		It("should not add the condition to a VMI that has none when the domain could not be read", func() {
			UpdateVMIStatus(vmi, nil, condManager)
			Expect(vmi.Status.Conditions).To(BeEmpty())
		})

		It("should keep one condition across repeated syncs", func() {
			domain.Spec.Metadata.KubeVirt.SnapshotOverlay = &api.SnapshotOverlayMetadata{Phase: api.SnapshotOverlayReady}

			UpdateVMIStatus(vmi, domain, condManager)
			UpdateVMIStatus(vmi, domain, condManager)

			Expect(vmi.Status.Conditions).To(HaveLen(1))
		})

		It("should not touch the other conditions", func() {
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{
				Type:   v1.VirtualMachineInstanceReady,
				Status: k8sv1.ConditionTrue,
			}}

			UpdateVMIStatus(vmi, domain, condManager)

			Expect(condManager.HasCondition(vmi, v1.VirtualMachineInstanceReady)).To(BeTrue())
		})
	})
})
