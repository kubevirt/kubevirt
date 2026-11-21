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

package virthandler

import (
	"fmt"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/hibernation"
	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	hotplugvolume "kubevirt.io/kubevirt/pkg/virt-handler/hotplug-disk"
	launcherclients "kubevirt.io/kubevirt/pkg/virt-handler/launcher-clients"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	oldClaim = "testvm-saved-state-11111111-aaaaa"
	newClaim = "testvm-saved-state-22222222-bbbbb"
)

// withSavedStateVolume hot plugs the saved state utility volume of claim
// into vmi, mounted or not.
func withSavedStateVolume(vmi *v1.VirtualMachineInstance, claim string, mounted bool) {
	vmi.Spec.UtilityVolumes = []v1.UtilityVolume{{
		Name:                              hibernation.VolumeName,
		PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: claim},
		Type:                              new(v1.SavedState),
	}}
	phase := v1.HotplugVolumeAttachedToNode
	if mounted {
		phase = v1.HotplugVolumeMounted
	}
	vmi.Status.VolumeStatus = []v1.VolumeStatus{{
		Name:          hibernation.VolumeName,
		Phase:         phase,
		HotplugVolume: &v1.HotplugVolumeStatus{AttachPodName: "hp-volume-xyz"},
	}}
}

// resumedVMI returns a running VMI resumed from the state saved in oldClaim.
func resumedVMI() *v1.VirtualMachineInstance {
	start, end := metav1.Now(), metav1.Now()
	return &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: metav1.NamespaceDefault, UID: "resumed-uid"},
		Spec: v1.VirtualMachineInstanceSpec{
			SavedStateSource: &v1.SavedStateSource{ClaimName: oldClaim, FileName: hibernation.FileNameFor(oldClaim)},
		},
		Status: v1.VirtualMachineInstanceStatus{
			Phase: v1.Running,
			SavedState: &v1.VirtualMachineInstanceSavedStateStatus{
				Phase: v1.SavedStateRestored, FileName: hibernation.FileNameFor(oldClaim), StartTimestamp: &start, EndTimestamp: &end,
			},
		},
	}
}

// pendingSaveOf returns the save target of vmi, if a save to it did not
// start yet.
func pendingSaveOf(vmi *v1.VirtualMachineInstance) (string, string, bool) {
	path, fileName, ok := saveTarget(vmi)
	if !ok {
		return "", "", false
	}
	if status := vmi.Status.SavedState; status != nil && hibernation.IsSavePhase(status.Phase) && status.FileName == fileName {
		return "", "", false
	}
	return path, fileName, true
}

var _ = Describe("Saving the state", func() {
	It("should save the state of a resumed VMI to the new volume", func() {
		vmi := resumedVMI()
		withSavedStateVolume(vmi, newClaim, true)

		path, fileName, ok := pendingSaveOf(vmi)
		Expect(ok).To(BeTrue(), "the restore status must not block a new save")
		Expect(fileName).To(Equal(hibernation.FileNameFor(newClaim)))
		Expect(path).To(Equal(filepath.Join(hotplugdisk.GetVolumeMountDir(hibernation.VolumeName), fileName)))
		Expect(path).ToNot(HavePrefix(hotplugdisk.GetVolumeMountDir(hibernation.SourceVolumeName)), "the save target must not be the restore source")
	})

	It("should wait for the volume to be mounted", func() {
		vmi := resumedVMI()
		withSavedStateVolume(vmi, newClaim, false)
		_, _, ok := pendingSaveOf(vmi)
		Expect(ok).To(BeFalse())
	})

	It("should never save to the volume a VMI resumes from", func() {
		vmi := resumedVMI()
		vmi.Status.SavedState = nil
		withSavedStateVolume(vmi, oldClaim, true)
		vmi.Spec.UtilityVolumes[0].Name = hibernation.SourceVolumeName
		vmi.Status.VolumeStatus[0].Name = hibernation.SourceVolumeName
		_, _, ok := pendingSaveOf(vmi)
		Expect(ok).To(BeFalse())
	})

	It("should not save without a saved state volume", func() {
		_, _, ok := pendingSaveOf(resumedVMI())
		Expect(ok).To(BeFalse())
	})

	It("should save only once to a volume", func() {
		vmi := resumedVMI()
		withSavedStateVolume(vmi, newClaim, true)
		vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateSaving, FileName: hibernation.FileNameFor(newClaim)}
		_, _, ok := pendingSaveOf(vmi)
		Expect(ok).To(BeFalse())
		vmi.Status.SavedState.Phase = v1.SavedStateSaveFailed
		_, _, ok = pendingSaveOf(vmi)
		Expect(ok).To(BeFalse(), "a failed save is not retried on the same volume")
	})

	It("should save again after an earlier save of the VMI failed", func() {
		vmi := resumedVMI()
		withSavedStateVolume(vmi, newClaim, true)
		vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateSaveFailed, FileName: hibernation.FileNameFor(oldClaim)}
		_, fileName, ok := pendingSaveOf(vmi)
		Expect(ok).To(BeTrue())
		Expect(fileName).To(Equal(hibernation.FileNameFor(newClaim)))
	})

	Context("saveState", func() {
		var (
			vmi        *v1.VirtualMachineInstance
			client     *cmdclient.MockLauncherClient
			controller *VirtualMachineController
			domains    cache.Store
			path       string
		)

		BeforeEach(func() {
			vmi = resumedVMI()
			withSavedStateVolume(vmi, newClaim, true)
			client = cmdclient.NewMockLauncherClient(gomock.NewController(GinkgoT()))
			domains = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
			controller = &VirtualMachineController{BaseController: &BaseController{
				logger:          log.Log.With("controller", "vm"),
				launcherClients: &launcherclients.MockLauncherClientManager{Client: client},
				domainStore:     domains,
			}}
			path = filepath.Join(hotplugdisk.GetVolumeMountDir(hibernation.VolumeName), hibernation.FileNameFor(newClaim))
		})

		It("should commit to the save before asking virt-launcher, and ask it once committed", func() {
			client.EXPECT().SaveVirtualMachine(gomock.Any(), gomock.Any()).Times(0)
			Expect(controller.saveState(vmi)).To(Succeed())
			Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateSaving))
			Expect(vmi.Status.SavedState.FileName).To(Equal(hibernation.FileNameFor(newClaim)))
			Expect(vmi.Status.SavedState.StartTimestamp).ToNot(BeNil())

			By("asking virt-launcher once the commit is persisted")
			client.EXPECT().SaveVirtualMachine(vmi, path).Return(nil).Times(1)
			Expect(controller.saveState(vmi)).To(Succeed())
		})

		It("should not ask again once virt-launcher took the save", func() {
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateSaving, FileName: hibernation.FileNameFor(newClaim), StartTimestamp: new(metav1.Now())}
			domain := api.NewMinimalDomainWithNS(vmi.Namespace, vmi.Name)
			domain.Spec.Metadata.KubeVirt.SavedState = &api.SavedStateMetadata{Operation: api.SavedStateOperationSave, FileName: hibernation.FileNameFor(newClaim)}
			Expect(domains.Add(domain)).To(Succeed())
			client.EXPECT().SaveVirtualMachine(gomock.Any(), gomock.Any()).Times(0)
			Expect(controller.saveState(vmi)).To(Succeed())
			Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateSaving))
		})

		It("should fail the save of a virt-launcher that cannot save", func() {
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateSaving, FileName: hibernation.FileNameFor(newClaim), StartTimestamp: new(metav1.Now())}
			client.EXPECT().SaveVirtualMachine(vmi, path).Return(status.Error(codes.Unimplemented, "unknown method")).Times(1)
			Expect(controller.saveState(vmi)).To(Succeed())
			Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateSaveFailed))
			Expect(vmi.Status.SavedState.Message).To(ContainSubstring("does not support"))
		})

		It("should retry a failed request, and give up on a save virt-launcher never took", func() {
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateSaving, FileName: hibernation.FileNameFor(newClaim), StartTimestamp: new(metav1.Now())}
			client.EXPECT().SaveVirtualMachine(vmi, path).Return(fmt.Errorf("connection refused")).Times(2)
			Expect(controller.saveState(vmi)).To(MatchError(ContainSubstring("connection refused")))
			Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateSaving))

			vmi.Status.SavedState.StartTimestamp = new(metav1.NewTime(time.Now().Add(-saveStartTimeout - time.Second)))
			Expect(controller.saveState(vmi)).To(Succeed())
			Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateSaveFailed))
			Expect(vmi.Status.SavedState.Message).To(ContainSubstring("did not start saving"))
		})

		It("should do nothing without an armed save", func() {
			vmi.Spec.UtilityVolumes = nil
			client.EXPECT().SaveVirtualMachine(gomock.Any(), gomock.Any()).Times(0)
			Expect(controller.saveState(vmi)).To(Succeed())
			Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateRestored))
		})
	})
})

var _ = Describe("Saved state status", func() {
	var (
		vmi    *v1.VirtualMachineInstance
		domain *api.Domain
		start  metav1.Time
		end    metav1.Time
	)

	BeforeEach(func() {
		vmi = &v1.VirtualMachineInstance{Status: v1.VirtualMachineInstanceStatus{Phase: v1.Running}}
		domain = api.NewMinimalDomain("testvm")
		domain.Status.Status = api.Running
		start = metav1.Now()
		end = metav1.Now()
	})

	saving := func() {
		vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateSaving, FileName: hibernation.FileNameFor(newClaim), StartTimestamp: &start}
	}

	It("should stay saving while the domain runs", func() {
		saving()
		updateSavedStateFromDomain(vmi, domain)
		Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateSaving))
	})

	It("should report the state saved once libvirt stopped the domain after saving it", func() {
		saving()
		domain.Status.Status = api.Shutoff
		domain.Status.Reason = api.ReasonSaved
		domain.Spec.Metadata.KubeVirt.SavedState = &api.SavedStateMetadata{
			Operation: api.SavedStateOperationSave, FileName: hibernation.FileNameFor(newClaim), Completed: true, StartTimestamp: &start, EndTimestamp: &end,
		}
		updateSavedStateFromDomain(vmi, domain)
		Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateSaved))
		Expect(vmi.Status.SavedState.EndTimestamp).To(Equal(&end))
	})

	It("should not report a domain stopped for another reason as saved", func() {
		saving()
		domain.Status.Status = api.Shutoff
		domain.Status.Reason = api.ReasonShutdown
		updateSavedStateFromDomain(vmi, domain)
		Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateSaving))
	})

	It("should report a failed save", func() {
		saving()
		domain.Spec.Metadata.KubeVirt.SavedState = &api.SavedStateMetadata{
			Operation: api.SavedStateOperationSave, FileName: hibernation.FileNameFor(newClaim), Completed: true, Failed: true, FailureReason: "no space left on device", EndTimestamp: &end,
		}
		updateSavedStateFromDomain(vmi, domain)
		Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateSaveFailed))
		Expect(vmi.Status.SavedState.Message).To(Equal("no space left on device"))
	})

	It("should ignore the restore a resumed VMI reported before the save started", func() {
		saving()
		domain.Spec.Metadata.KubeVirt.SavedState = &api.SavedStateMetadata{
			Operation: api.SavedStateOperationRestore, FileName: hibernation.FileNameFor(oldClaim), Completed: true, StartTimestamp: &start, EndTimestamp: &end,
		}
		updateSavedStateFromDomain(vmi, domain)
		Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateSaving))
	})

	It("should ignore an earlier failed save of the VMI", func() {
		saving()
		domain.Spec.Metadata.KubeVirt.SavedState = &api.SavedStateMetadata{
			Operation: api.SavedStateOperationSave, FileName: hibernation.FileNameFor(oldClaim), Completed: true, Failed: true, FailureReason: "earlier",
		}
		updateSavedStateFromDomain(vmi, domain)
		Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateSaving))
	})

	It("should report a restore", func() {
		vmi.Spec.SavedStateSource = &v1.SavedStateSource{ClaimName: "state", FileName: hibernation.FileNameFor(oldClaim)}
		updateSavedStateFromDomain(vmi, domain)
		Expect(vmi.Status.SavedState).To(BeNil(), "nothing to report before the restore completes")

		domain.Spec.Metadata.KubeVirt.SavedState = &api.SavedStateMetadata{
			Operation: api.SavedStateOperationRestore, FileName: hibernation.FileNameFor(oldClaim), Completed: true, StartTimestamp: &start, EndTimestamp: &end,
		}
		updateSavedStateFromDomain(vmi, domain)
		Expect(vmi.Status.SavedState).To(Equal(&v1.VirtualMachineInstanceSavedStateStatus{
			Phase: v1.SavedStateRestored, FileName: hibernation.FileNameFor(oldClaim), StartTimestamp: &start, EndTimestamp: &end,
		}))
	})

	It("should rebuild the status of a save from virt-launcher when it was lost", func() {
		vmi.Spec.UtilityVolumes = []v1.UtilityVolume{{
			Name:                              hibernation.VolumeName,
			PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: newClaim},
			Type:                              new(v1.SavedState),
		}}
		domain.Status.Status = api.Shutoff
		domain.Status.Reason = api.ReasonSaved
		domain.Spec.Metadata.KubeVirt.SavedState = &api.SavedStateMetadata{
			Operation: api.SavedStateOperationSave, FileName: hibernation.FileNameFor(newClaim), Completed: true, StartTimestamp: &start, EndTimestamp: &end,
		}
		updateSavedStateFromDomain(vmi, domain)
		Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateSaved))
		Expect(vmi.Status.SavedState.FileName).To(Equal(hibernation.FileNameFor(newClaim)))
	})

	It("should report the restore a VMI committed to", func() {
		vmi.Spec.SavedStateSource = &v1.SavedStateSource{ClaimName: "state", FileName: hibernation.FileNameFor(oldClaim)}
		vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateRestoring, FileName: hibernation.FileNameFor(oldClaim), StartTimestamp: &start}
		updateSavedStateFromDomain(vmi, domain)
		Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateRestoring))

		domain.Spec.Metadata.KubeVirt.SavedState = &api.SavedStateMetadata{
			Operation: api.SavedStateOperationRestore, FileName: hibernation.FileNameFor(oldClaim), Completed: true, StartTimestamp: &start, EndTimestamp: &end,
		}
		updateSavedStateFromDomain(vmi, domain)
		Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateRestored))
	})

	It("should report a failed restore", func() {
		vmi.Spec.SavedStateSource = &v1.SavedStateSource{ClaimName: "state", FileName: hibernation.FileNameFor(oldClaim)}
		domain.Spec.Metadata.KubeVirt.SavedState = &api.SavedStateMetadata{
			Operation: api.SavedStateOperationRestore, FileName: hibernation.FileNameFor(oldClaim), Completed: true, Failed: true, FailureReason: "unsupported configuration",
		}
		updateSavedStateFromDomain(vmi, domain)
		Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateRestoreFailed))
		Expect(vmi.Status.SavedState.Message).To(Equal("unsupported configuration"))
	})
})

var _ = Describe("Resuming from a saved state", func() {
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		vmi = &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				SavedStateSource: &v1.SavedStateSource{ClaimName: oldClaim, FileName: hibernation.FileNameFor(oldClaim)},
			},
			Status: v1.VirtualMachineInstanceStatus{Phase: v1.Scheduled},
		}
	})

	It("should commit to the restore once the saved state source is mounted, and start once committed", func() {
		controller := &VirtualMachineController{BaseController: &BaseController{logger: log.Log.With("controller", "vm")}}
		Expect(controller.restoreReady(vmi)).To(BeFalse(), "the volume is hot plugged after the VMI is created")
		withSavedStateVolume(vmi, oldClaim, false)
		vmi.Spec.UtilityVolumes[0].Name = hibernation.SourceVolumeName
		vmi.Status.VolumeStatus[0].Name = hibernation.SourceVolumeName
		Expect(controller.restoreReady(vmi)).To(BeFalse())
		Expect(vmi.Status.SavedState).To(BeNil(), "no commit before the volume is mounted")

		vmi.Status.VolumeStatus[0].Phase = v1.HotplugVolumeMounted
		Expect(controller.restoreReady(vmi)).To(BeFalse(), "the commit is persisted first")
		Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateRestoring))
		Expect(vmi.Status.SavedState.FileName).To(Equal(hibernation.FileNameFor(oldClaim)))

		Expect(controller.restoreReady(vmi)).To(BeTrue())
	})

	It("should commit as soon as this sync mounted the source", func() {
		mounter := hotplugvolume.NewMockVolumeMounter(gomock.NewController(GinkgoT()))
		controller := &VirtualMachineController{BaseController: &BaseController{logger: log.Log.With("controller", "vm")}, hotplugVolumeMounter: mounter}
		withSavedStateVolume(vmi, oldClaim, false)
		vmi.Spec.UtilityVolumes[0].Name = hibernation.SourceVolumeName
		vmi.Status.VolumeStatus[0].Name = hibernation.SourceVolumeName
		vmi.Status.VolumeStatus[0].HotplugVolume.AttachPodUID = "attach-pod-uid"
		mounter.EXPECT().IsMounted(vmi, hibernation.SourceVolumeName, types.UID("attach-pod-uid")).Return(true, nil)
		Expect(controller.restoreReady(vmi)).To(BeFalse())
		Expect(vmi.Status.SavedState.Phase).To(Equal(v1.SavedStateRestoring))
	})

	It("should not wait for anything when booting", func() {
		vmi.Spec.SavedStateSource = nil
		controller := &VirtualMachineController{BaseController: &BaseController{logger: log.Log.With("controller", "vm")}}
		Expect(controller.restoreReady(vmi)).To(BeTrue())
	})

	It("should fail the VMI once virt-launcher reports the restore failed", func() {
		domain := api.NewMinimalDomain("testvm")
		domain.Status.Status = api.Shutoff
		Expect(restoreFailed(vmi, domain)).To(BeFalse())

		domain.Spec.Metadata.KubeVirt.SavedState = &api.SavedStateMetadata{
			Operation: api.SavedStateOperationRestore, FileName: hibernation.FileNameFor(oldClaim), Completed: true, Failed: true,
		}
		Expect(restoreFailed(vmi, domain)).To(BeTrue())

		controller := &VirtualMachineController{}
		phase, err := controller.calculateVmPhaseForStatusReason(domain, vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(phase).To(Equal(v1.Failed))
	})

	It("should fail the VMI that reported a failed restore once its domain is gone", func() {
		vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateRestoreFailed}
		Expect(restoreFailed(vmi, nil)).To(BeTrue())
	})

	It("should not fail a running domain or a restore of another file", func() {
		domain := api.NewMinimalDomain("testvm")
		domain.Status.Status = api.Shutoff
		domain.Spec.Metadata.KubeVirt.SavedState = &api.SavedStateMetadata{
			Operation: api.SavedStateOperationRestore, FileName: hibernation.FileNameFor(newClaim), Completed: true, Failed: true,
		}
		Expect(restoreFailed(vmi, domain)).To(BeFalse())

		domain.Spec.Metadata.KubeVirt.SavedState.FileName = hibernation.FileNameFor(oldClaim)
		domain.Status.Status = api.Running
		Expect(restoreFailed(vmi, domain)).To(BeFalse())
	})
})
