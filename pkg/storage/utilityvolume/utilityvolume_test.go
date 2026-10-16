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
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
)

type patchOp struct {
	Op    string             `json:"op"`
	Path  string             `json:"path"`
	Value []v1.UtilityVolume `json:"value"`
}

var _ = Describe("Utility volumes", func() {
	const (
		volumeName = "scratch-volume"
		pvcName    = "scratch-pvc"
	)

	var (
		ctrl         *gomock.Controller
		virtClient   *kubecli.MockKubevirtClient
		vmiInterface *kubecli.MockVirtualMachineInstanceInterface
		vmi          *v1.VirtualMachineInstance
	)

	utilityVolume := func(name string, volumeType v1.UtilityVolumeType) v1.UtilityVolume {
		return v1.UtilityVolume{
			Name: name,
			PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: name + "-pvc",
			},
			Type: pointer.P(volumeType),
		}
	}

	// Every patch is a test of the list the caller read, followed by the one
	// operation that changes it.
	expectPatch := func(assert func(test, op patchOp)) {
		vmiInterface.EXPECT().Patch(context.Background(), vmi.Name, types.JSONPatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ types.PatchType, data []byte, _ metav1.PatchOptions, _ ...string) (*v1.VirtualMachineInstance, error) {
				var ops []patchOp
				Expect(json.Unmarshal(data, &ops)).To(Succeed())
				Expect(ops).To(HaveLen(2))
				assert(ops[0], ops[1])
				return vmi, nil
			})
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		vmi = libvmi.New(libvmi.WithNamespace("default"), libvmi.WithName("test-vmi"))
		virtClient.EXPECT().VirtualMachineInstance(vmi.Namespace).Return(vmiInterface).AnyTimes()
	})

	Context("attaching", func() {
		It("should add the first one", func() {
			expectPatch(func(_, op patchOp) {
				Expect(op.Op).To(Equal("add"))
				Expect(op.Path).To(Equal("/spec/utilityVolumes"))
				Expect(op.Value).To(ConsistOf(v1.UtilityVolume{
					Name: volumeName,
					PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
					Type: pointer.P(v1.SnapshotOverlay),
				}))
			})

			Expect(Attach(virtClient, vmi, volumeName, pvcName, v1.SnapshotOverlay)).To(Succeed())
		})

		It("should replace the list when the VMI already has one", func() {
			existing := utilityVolume("memory-dump", v1.MemoryDump)
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{existing}

			expectPatch(func(_, op patchOp) {
				Expect(op.Op).To(Equal("replace"))
				// The other writer's volume survives.
				Expect(op.Value).To(HaveLen(2))
				Expect(op.Value[0]).To(Equal(existing))
				Expect(op.Value[1].Name).To(Equal(volumeName))
			})

			Expect(Attach(virtClient, vmi, volumeName, pvcName, v1.SnapshotOverlay)).To(Succeed())
		})

		It("should test the list it read, so a concurrent writer is a conflict", func() {
			existing := utilityVolume("memory-dump", v1.MemoryDump)
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{existing}

			expectPatch(func(test, _ patchOp) {
				Expect(test.Op).To(Equal("test"))
				Expect(test.Path).To(Equal("/spec/utilityVolumes"))
				Expect(test.Value).To(ConsistOf(existing))
			})

			Expect(Attach(virtClient, vmi, volumeName, pvcName, v1.SnapshotOverlay)).To(Succeed())
		})

		It("should do nothing when the volume is already there", func() {
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{utilityVolume(volumeName, v1.SnapshotOverlay)}

			// No Patch is expected: attaching twice is what every reconcile does.
			Expect(Attach(virtClient, vmi, volumeName, pvcName, v1.SnapshotOverlay)).To(Succeed())
		})

		It("should report a patch that failed", func() {
			vmiInterface.EXPECT().Patch(gomock.Any(), gomock.Any(), types.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("the VMI moved on"))

			Expect(Attach(virtClient, vmi, volumeName, pvcName, v1.SnapshotOverlay)).
				To(MatchError(ContainSubstring("the VMI moved on")))
		})
	})

	Context("detaching", func() {
		It("should remove the list when it holds nothing else", func() {
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{utilityVolume(volumeName, v1.SnapshotOverlay)}

			expectPatch(func(_, op patchOp) {
				// An empty list left behind would be a spec diff nobody asked for.
				Expect(op.Op).To(Equal("remove"))
				Expect(op.Path).To(Equal("/spec/utilityVolumes"))
			})

			Expect(Detach(virtClient, vmi, volumeName)).To(Succeed())
		})

		It("should leave the other volumes alone", func() {
			existing := utilityVolume("memory-dump", v1.MemoryDump)
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{existing, utilityVolume(volumeName, v1.SnapshotOverlay)}

			expectPatch(func(_, op patchOp) {
				Expect(op.Op).To(Equal("replace"))
				Expect(op.Value).To(ConsistOf(existing))
			})

			Expect(Detach(virtClient, vmi, volumeName)).To(Succeed())
		})

		It("should do nothing when the VMI has no utility volumes", func() {
			Expect(Detach(virtClient, vmi, volumeName)).To(Succeed())
		})

		It("should report a patch that failed", func() {
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{utilityVolume(volumeName, v1.SnapshotOverlay)}
			vmiInterface.EXPECT().Patch(gomock.Any(), gomock.Any(), types.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("the VMI moved on"))

			Expect(Detach(virtClient, vmi, volumeName)).To(MatchError(ContainSubstring("the VMI moved on")))
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
