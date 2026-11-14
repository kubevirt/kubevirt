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

package hibernation

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	vmName    = "testvm"
	nodeName  = "node01"
	diskClaim = "rootdisk"
	sourceUID = types.UID("6f0a1c3e-aaaa-bbbb-cccc-000000000001")
	targetUID = types.UID("6f0a1c3e-aaaa-bbbb-cccc-000000000002")
	diskUID   = types.UID("disk-uid-1")
	random    = "abcde"
)

var _ = Describe("Hibernation", func() {
	var (
		k8sClient      *k8sfake.Clientset
		virtFakeClient *fake.Clientset
		pvcStore       cache.Store
		recorder       *record.FakeRecorder
		controller     *Controller
	)

	BeforeEach(func() {
		virtClient := kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
		virtFakeClient = fake.NewSimpleClientset()
		k8sClient = k8sfake.NewSimpleClientset()
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(
			virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault),
		).AnyTimes()
		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()

		pvcInformer, _ := testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		pvcStore = pvcInformer.GetStore()
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true
		controller = NewController(virtClient, pvcStore, recorder)

		Expect(pvcStore.Add(newClaim(diskClaim, diskUID, nil))).To(Succeed())
	})

	Context("ClaimName", func() {
		It("should derive the claim name from the VM name, the VMI UID and a random part", func() {
			Expect(ClaimName(vmName, sourceUID, random)).To(Equal("testvm-saved-state-6f0a1c3e-abcde"))
		})

		It("should keep the claim name a valid object name for long VM names", func() {
			name := ClaimName(strings.Repeat("a", 300), sourceUID, random)
			Expect(name).To(HaveLen(maxClaimName))
			Expect(name).To(HaveSuffix("-saved-state-6f0a1c3e-abcde"))
		})

		It("should use a new PVC for every hibernation request of a VMI", func() {
			vm, vmi := newRunningVM()
			first := NewSavedState(vm, vmi, nil).ClaimName
			second := NewSavedState(vm, vmi, nil).ClaimName
			Expect(first).To(HavePrefix("testvm-saved-state-6f0a1c3e-"))
			Expect(first).ToNot(Equal(second))
		})

		It("should derive a distinct plain file name from every claim", func() {
			Expect(FileNameFor("a")).To(MatchRegexp(`^domain-[0-9a-f]{12}\.save$`))
			Expect(FileNameFor("a")).ToNot(Equal(FileNameFor("b")))
		})

		It("should keep the restore source and the save target apart", func() {
			Expect(SourceVolumeName).ToNot(Equal(VolumeName))
			target := v1.UtilityVolume{Name: VolumeName, Type: new(v1.SavedState)}
			source := v1.UtilityVolume{Name: SourceVolumeName, Type: new(v1.SavedState)}
			Expect(IsSaveTargetVolume(target)).To(BeTrue())
			Expect(IsSaveTargetVolume(source)).To(BeFalse())
			Expect(IsSavedStateVolume(source)).To(BeTrue())
		})

		It("should record the run strategy the hibernation replaces", func() {
			vm, vmi := newRunningVM()
			vm.Spec.RunStrategy = new(v1.RunStrategyManual)
			Expect(NewSavedState(vm, vmi, nil).RunStrategy).To(HaveValue(Equal(v1.RunStrategyManual)))
		})
	})

	Context("ValidateRequest", func() {
		It("should accept a running VMI", func() {
			vm, vmi := newRunningVM()
			Expect(ValidateRequest(vm, vmi)).To(Succeed())
		})

		It("should reject a VM without a running VMI", func() {
			vm, vmi := newRunningVM()
			vmi.Status.Phase = v1.Scheduled
			Expect(ValidateRequest(vm, vmi)).To(MatchError(ContainSubstring("is not running")))
			Expect(ValidateRequest(vm, nil)).To(MatchError(ContainSubstring("is not running")))
		})

		It("should reject a migrating VMI", func() {
			vm, vmi := newRunningVM()
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{}
			Expect(ValidateRequest(vm, vmi)).To(MatchError(ContainSubstring("is migrating")))
		})

		It("should reject a VM with changes pending a restart", func() {
			vm, vmi := newRunningVM()
			vm.Status.Conditions = []v1.VirtualMachineCondition{{Type: v1.VirtualMachineRestartRequired, Status: k8sv1.ConditionTrue}}
			Expect(ValidateRequest(vm, vmi)).To(MatchError(ContainSubstring("require a restart")))
		})

		It("should reject a VM that is already hibernating", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationSaving}
			Expect(ValidateRequest(vm, vmi)).To(MatchError(ContainSubstring("already hibernating")))
		})

		It("should reject a VMI still resuming from a saved state", func() {
			vm, vmi := newRunningVM()
			vmi.Spec.SavedStateSource = &v1.SavedStateSource{ClaimName: "c", FileName: FileNameFor("c")}
			Expect(ValidateRequest(vm, vmi)).To(MatchError(ContainSubstring("still resuming")))
		})

		It("should accept a VMI resumed from a saved state", func() {
			vm, vmi := resumedVM()
			Expect(ValidateRequest(vm, vmi)).To(Succeed())
		})

		It("should reject a VMI whose previous saved state volume is still attached", func() {
			vm, vmi := newRunningVM()
			withSavedStateVolume(vmi, "previous")
			Expect(ValidateRequest(vm, vmi)).To(MatchError(ContainSubstring("still attached")))
		})

		It("should reject a VMI whose device state cannot be migrated", func() {
			vm, vmi := newRunningVM()
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{
				Type:    v1.VirtualMachineInstanceIsMigratable,
				Status:  k8sv1.ConditionFalse,
				Reason:  v1.VirtualMachineInstanceReasonHostDeviceNotMigratable,
				Message: "has a host device",
			}}
			Expect(ValidateRequest(vm, vmi)).To(MatchError(ContainSubstring("has a host device")))
		})

		It("should accept a VMI that is only not migratable because of its disks", func() {
			vm, vmi := newRunningVM()
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{
				Type:   v1.VirtualMachineInstanceIsMigratable,
				Status: k8sv1.ConditionFalse,
				Reason: v1.VirtualMachineInstanceReasonDisksNotMigratable,
			}}
			Expect(ValidateRequest(vm, vmi)).To(Succeed())
		})

		DescribeTable("should reject a VMI whose disks do not persist", func(source v1.VolumeSource) {
			vm, vmi := newRunningVM()
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{{Name: "scratch"}}
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{Name: "scratch", VolumeSource: source})
			Expect(ValidateRequest(vm, vmi)).To(MatchError(ContainSubstring("does not persist")))
		},
			Entry("container disk", v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "image"}}),
			Entry("empty disk", v1.VolumeSource{EmptyDisk: &v1.EmptyDiskSource{}}),
			Entry("ephemeral disk", v1.VolumeSource{Ephemeral: &v1.EphemeralVolumeSource{}}),
		)

		It("should accept a container disk attached as a CD-ROM", func() {
			vm, vmi := newRunningVM()
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{{Name: "iso", DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{}}}}
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{Name: "iso", VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{Image: "image"}}})
			Expect(ValidateRequest(vm, vmi)).To(Succeed())
		})

		DescribeTable("should reject devices whose state a save does not cover", func(change func(*v1.VirtualMachineInstance), what string) {
			vm, vmi := newRunningVM()
			change(vmi)
			Expect(ValidateRequest(vm, vmi)).To(MatchError(ContainSubstring(what)))
		},
			Entry("a vTPM", func(vmi *v1.VirtualMachineInstance) { vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{} }, "vTPM"),
			Entry("persistent EFI variables", func(vmi *v1.VirtualMachineInstance) {
				vmi.Spec.Domain.Firmware = &v1.Firmware{Bootloader: &v1.Bootloader{EFI: &v1.EFI{Persistent: new(true)}}}
			}, "persistent EFI"),
			Entry("a GPU", func(vmi *v1.VirtualMachineInstance) {
				vmi.Spec.Domain.Devices.GPUs = []v1.GPU{{Name: "gpu", DeviceName: "nvidia.com/GP100"}}
			}, "GPUs"),
			Entry("a host device", func(vmi *v1.VirtualMachineInstance) {
				vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{{Name: "dev", DeviceName: "example.com/dev"}}
			}, "host devices"),
			Entry("an SR-IOV interface", func(vmi *v1.VirtualMachineInstance) {
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "sriov", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}}
			}, "SR-IOV"),
		)

		It("should accept a disabled vTPM", func() {
			vm, vmi := newRunningVM()
			vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{Enabled: new(false)}
			Expect(ValidateRequest(vm, vmi)).To(Succeed())
		})

		It("should reject a VMI with hotplugged volumes", func() {
			vm, vmi := newRunningVM()
			vmi.Status.VolumeStatus = []v1.VolumeStatus{{Name: "hp", HotplugVolume: &v1.HotplugVolumeStatus{}}}
			Expect(ValidateRequest(vm, vmi)).To(MatchError(ContainSubstring("hotplugged volume hp")))
		})
	})

	Context("SpecHash", func() {
		It("should not change with the run strategy", func() {
			vm, _ := newRunningVM()
			before, err := SpecHash(vm)
			Expect(err).ToNot(HaveOccurred())
			vm.Spec.RunStrategy = new(v1.RunStrategyHalted)
			after, err := SpecHash(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(after).To(Equal(before))
		})

		It("should change with the hardware", func() {
			vm, _ := newRunningVM()
			before, err := SpecHash(vm)
			Expect(err).ToNot(HaveOccurred())
			vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{Cores: 4}
			after, err := SpecHash(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(after).ToNot(Equal(before))
		})
	})

	Context("HostModelNodeSelector", func() {
		node := &k8sv1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
			v1.HostModelCPULabel + "Skylake-Client-IBRS": "true",
			v1.HostModelRequiredFeaturesLabel + "vmx":    "true",
			v1.CPUModelLabel + "Haswell":                 "true",
			"kubernetes.io/hostname":                     nodeName,
		}}}

		It("should require the host-model CPU and its features of the node", func() {
			_, vmi := newRunningVM()
			Expect(HostModelNodeSelector(vmi, node)).To(Equal(map[string]string{
				v1.SupportedHostModelMigrationCPU + "Skylake-Client-IBRS": "true",
				v1.CPUFeatureLabel + "vmx":                                "true",
			}))
		})

		It("should not require anything for a named CPU model", func() {
			_, vmi := newRunningVM()
			vmi.Spec.Domain.CPU = &v1.CPU{Model: "Haswell"}
			Expect(HostModelNodeSelector(vmi, node)).To(BeNil())
		})
	})

	Context("ClaimSize", func() {
		It("should hold the guest memory with some overhead", func() {
			_, vmi := newRunningVM()
			size := ClaimSize(vmi)
			Expect(size.Cmp(resource.MustParse("2Gi"))).To(BeNumerically(">", 0))
			Expect(size.Cmp(resource.MustParse("3Gi"))).To(BeNumerically("<", 0))
		})

		It("should size for the plugged memory, not the hotpluggable one", func() {
			_, vmi := newRunningVM()
			vmi.Spec.Domain.Memory.MaxGuest = new(resource.MustParse("32Gi"))
			vmi.Status.Memory = &v1.MemoryStatus{GuestCurrent: new(resource.MustParse("8Gi"))}
			size := ClaimSize(vmi)
			Expect(size.Cmp(resource.MustParse("8Gi"))).To(BeNumerically(">", 0))
			Expect(size.Cmp(resource.MustParse("10Gi"))).To(BeNumerically("<", 0))
		})

		It("should leave room for the device state and the file system overhead", func() {
			_, vmi := newRunningVM()
			vmi.Spec.Domain.Memory.Guest = new(resource.MustParse("8Gi"))
			size := ClaimSize(vmi)
			usable := float64(size.Value()) * (1 - filesystemOverheadRatio)
			Expect(usable).To(BeNumerically(">=", float64(8<<30)+stateOverhead))
		})
	})

	Context("Sync", func() {
		It("should create the saved state PVC owned by the VM", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			vm.Status.SavedState.StorageClassName = new("fast")

			Expect(controller.Sync(vm, vmi)).To(Succeed())

			claim, err := k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), vm.Status.SavedState.ClaimName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(metav1.IsControlledBy(claim, vm)).To(BeTrue())
			Expect(*claim.Spec.StorageClassName).To(Equal("fast"))
			Expect(*claim.Spec.VolumeMode).To(Equal(k8sv1.PersistentVolumeFilesystem))
			Expect(claim.Spec.Resources.Requests.Storage().Cmp(*vmi.Spec.Domain.Memory.Guest)).To(BeNumerically(">", 0))
		})

		It("should arm the save by attaching the PVC as a saved state utility volume", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			vm.Status.SavedState.SpecHash = "recorded"
			Expect(pvcStore.Add(newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm))).To(Succeed())
			_, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(controller.Sync(vm, vmi)).To(Succeed())

			updated, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Spec.UtilityVolumes).To(ConsistOf(v1.UtilityVolume{
				Name:                              VolumeName,
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: vm.Status.SavedState.ClaimName},
				Type:                              new(v1.SavedState),
			}))
		})

		It("should refuse to use a PVC it does not own", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			Expect(pvcStore.Add(newClaim(vm.Status.SavedState.ClaimName, "state-uid", nil))).To(Succeed())
			Expect(controller.Sync(vm, vmi)).To(MatchError(ContainSubstring("not owned by VirtualMachine")))
		})

		It("should detach the volume of a failed save before deleting the PVC", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			vm.Status.SavedState.Phase = v1.HibernationFailed
			claim := newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm)
			Expect(pvcStore.Add(claim)).To(Succeed())
			_, err := k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Create(context.Background(), claim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi.Spec.UtilityVolumes = []v1.UtilityVolume{{
				Name:                              VolumeName,
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: claim.Name},
				Type:                              new(v1.SavedState),
			}}
			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(controller.Sync(vm, vmi)).To(Succeed())
			updated, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Spec.UtilityVolumes).To(BeEmpty())
			_, err = k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), claim.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "the PVC must outlive the volume")

			Expect(controller.Sync(vm, updated)).To(Succeed())
			_, err = k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), claim.Name, metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
		})

		It("should delete the PVC of a discarded saved state", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vm.Status.SavedState.Discard = true
			claim := newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm)
			Expect(pvcStore.Add(claim)).To(Succeed())
			_, err := k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Create(context.Background(), claim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(controller.Sync(vm, nil)).To(Succeed())
			_, err = k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), claim.Name, metav1.GetOptions{})
			Expect(err).To(HaveOccurred())

			controller.UpdateStatus(vm, nil)
			Expect(vm.Status.SavedState).ToNot(BeNil(), "the status stays until the PVC is gone")
			Expect(pvcStore.Delete(claim)).To(Succeed())
			controller.UpdateStatus(vm, nil)
			Expect(vm.Status.SavedState).To(BeNil())
		})
	})

	Context("UpdateStatus while saving", func() {
		BeforeEach(func() {
			_, err := k8sClient.CoreV1().Nodes().Create(context.Background(), &k8sv1.Node{ObjectMeta: metav1.ObjectMeta{
				Name:   nodeName,
				Labels: map[string]string{v1.HostModelCPULabel + "EPYC": "true"},
			}}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should record what the saved state is valid with", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)

			controller.UpdateStatus(vm, vmi)

			state := vm.Status.SavedState
			Expect(state.Phase).To(Equal(v1.HibernationPending))
			hash, err := SpecHash(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.SpecHash).To(Equal(hash))
			Expect(state.VolumeClaimUIDs).To(Equal(map[string]types.UID{diskClaim: diskUID}))
			Expect(state.SourceNodeSelector).To(Equal(map[string]string{v1.SupportedHostModelMigrationCPU + "EPYC": "true"}))
		})

		It("should follow the save reported on the VMI", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			start, end := metav1.Now(), metav1.Now()
			fileName := FileNameFor(vm.Status.SavedState.ClaimName)

			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateSaving, FileName: fileName, StartTimestamp: &start}
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaving))

			vmi.Status.Phase = v1.Succeeded
			vmi.Status.SavedState.Phase = v1.SavedStateSaved
			vmi.Status.SavedState.EndTimestamp = &end
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaved))
			Expect(vm.Status.SavedState.FileName).To(Equal(fileName))
			Expect(vm.Status.SavedState.SaveEndTimestamp).To(Equal(&end))
			Expect(recorder.Events).To(Receive(ContainSubstring(HibernatedReason)))
		})

		It("should fail when saving fails", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
				Phase: v1.SavedStateSaveFailed, FileName: FileNameFor(vm.Status.SavedState.ClaimName), Message: "no space left",
			}
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("no space left"))
		})

		It("should ignore an earlier failed save of the VMI", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
				Phase: v1.SavedStateSaveFailed, FileName: FileNameFor("earlier-claim"), Message: "earlier",
			}
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationPending))
		})

		It("should cancel a pending hibernation that is discarded", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			vm.Status.SavedState.Discard = true
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			_, err := k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), vm.Status.SavedState.ClaimName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred(), "a cancelled hibernation must not provision a PVC")

			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("cancelled"))
			Expect(IsInProgress(vm, vmi)).To(BeTrue(), "held until the run strategy is restored")

			Expect(controller.Sync(vm, vmi)).To(Succeed())
			Expect(IsInProgress(vm, vmi)).To(BeFalse(), "the stop must not be held anymore")
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState).To(BeNil(), "a cancelled hibernation without a PVC is forgotten")
		})

		It("should fail a hibernation whose save does not start in time, even without a PVC", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			// e.g. a ResourceQuota rejects the PVC, which is never created.

			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationPending))

			controller.now = func() time.Time { return vm.Status.SavedState.RequestTimestamp.Add(PendingTimeout + time.Second) }
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("did not start"))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("keeps running"))
		})

		It("should report a save that takes too long, and keep its volume until virt-launcher reports", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			claim := newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm)
			Expect(pvcStore.Add(claim)).To(Succeed())
			_, err := k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Create(context.Background(), claim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			withSavedStateVolume(vmi, claim.Name)
			start := metav1.Now()
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
				Phase: v1.SavedStateSaving, FileName: FileNameFor(claim.Name), StartTimestamp: &start,
			}
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaving))

			controller.now = func() time.Time { return start.Add(SavingTimeout(vmi) + savingTimeoutGrace + time.Second) }
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaving), "QEMU may still write, or the guest be paused")
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("did not complete"))
			Expect(recorder.Events).To(Receive(ContainSubstring(HibernationStuckReason)))
			Expect(IsInProgress(vm, vmi)).To(BeTrue(), "the stop stays held")
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			_, err = k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), claim.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "the PVC stays")

			By("virt-launcher reporting the aborted save")
			vmi.Status.SavedState.Phase = v1.SavedStateSaveFailed
			vmi.Status.SavedState.Message = "aborted"
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("keeps running"))
		})

		It("should end a save when its VMI is gone, as QEMU is gone with it", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			start := metav1.Now()
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
				Phase: v1.SavedStateSaving, FileName: FileNameFor(vm.Status.SavedState.ClaimName), StartTimestamp: &start,
			}
			vmi.Status.Phase = v1.Failed
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("while its state was being saved"))
		})

		DescribeTable("should bound the time a save may take", func(annotation string, expected time.Duration) {
			_, vmi := newRunningVM()
			if annotation != "" {
				vmi.Annotations = map[string]string{v1.SavedStateTimeoutAnnotation: annotation}
			}
			vmi.Spec.Domain.Memory.Guest = new(resource.MustParse("32Gi"))
			Expect(SavingTimeout(vmi)).To(Equal(expected))
		},
			Entry("by the guest memory", "", 32*time.Minute),
			Entry("by the annotation", "90s", 90*time.Second),
			Entry("ignoring an invalid annotation", "soon", 32*time.Minute),
			Entry("ignoring a negative annotation", "-1m", 32*time.Minute),
		)

		It("should fail when the VMI stops before its state is saved", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			vmi.Status.Phase = v1.Failed
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))

			vm.Status.SavedState = pendingState(vm, vmi)
			controller.UpdateStatus(vm, nil)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
		})

		It("should hold the stop of the VMI only while its state is being saved", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			Expect(IsInProgress(vm, vmi)).To(BeTrue())
			Expect(IsHibernating(vm, vmi)).To(BeTrue())
			vm.Status.SavedState.Phase = v1.HibernationSaving
			Expect(IsInProgress(vm, vmi)).To(BeTrue())
			vm.Status.SavedState.Phase = v1.HibernationSaved
			Expect(IsInProgress(vm, vmi)).To(BeFalse())
			vm.Status.SavedState.Phase = v1.HibernationSaving
			vmi.UID = targetUID
			Expect(IsInProgress(vm, vmi)).To(BeFalse())
		})

		DescribeTable("should hold nothing when the VM is deleted", func(phase v1.VirtualMachineSavedStatePhase) {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			vm.Status.SavedState.Phase = phase
			Expect(IsInProgress(vm, vmi)).To(BeTrue())
			vm.DeletionTimestamp = new(metav1.Now())
			Expect(IsInProgress(vm, vmi)).To(BeFalse(), "a deletion must never wait for the hibernation")
		},
			Entry("Pending, before the PVC exists", v1.HibernationPending),
			Entry("Saving", v1.HibernationSaving),
			Entry("Failed, while the run strategy is still to be restored", v1.HibernationFailed),
		)
	})

	Context("PrepareRestore", func() {
		It("should make the VMI resume from a valid saved state", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vm.Status.SavedState.SourceNodeSelector = map[string]string{"cpu": "EPYC"}
			Expect(pvcStore.Add(newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm))).To(Succeed())
			vmi := newVMI(targetUID)

			Expect(controller.PrepareRestore(vm, vmi)).To(Succeed())

			Expect(vmi.Spec.SavedStateSource).To(Equal(&v1.SavedStateSource{ClaimName: vm.Status.SavedState.ClaimName, FileName: vm.Status.SavedState.FileName}))
			Expect(vmi.Spec.UtilityVolumes).To(ConsistOf(SourceVolume(vm.Status.SavedState.ClaimName)), "hot plugged from the start")
			Expect(IsSourceVolumeOf(vmi.Spec.UtilityVolumes[0], vmi.Spec.SavedStateSource)).To(BeTrue())
			Expect(vmi.Spec.NodeSelector).To(HaveKeyWithValue("cpu", "EPYC"))
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaved), "not before the VMI is created")

			vmi.UID = targetUID
			controller.RestoreCreated(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestoring))
			Expect(vm.Status.SavedState.RestoreUID).To(Equal(targetUID))
		})

		It("should keep the saved state when the creation of the VMI fails", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			Expect(pvcStore.Add(newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm))).To(Succeed())
			Expect(controller.PrepareRestore(vm, newVMI(""))).To(Succeed())
			// The creation fails: RestoreCreated is not called.

			next := newVMI("")
			Expect(controller.PrepareRestore(vm, next)).To(Succeed())
			Expect(next.Spec.SavedStateSource).ToNot(BeNil(), "the next attempt resumes")
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaved))
		})

		It("should refuse a saved state PVC the VM does not own", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			Expect(pvcStore.Add(newClaim(vm.Status.SavedState.ClaimName, "state-uid", nil))).To(Succeed())
			vmi := newVMI("")
			Expect(controller.PrepareRestore(vm, vmi)).To(Succeed())
			Expect(vmi.Spec.SavedStateSource).To(BeNil())
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("not a saved state PVC"))
		})

		It("should prefer the node the state was saved on", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			claim := newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm)
			claim.Annotations = map[string]string{SourceNodeAnnotation: nodeName}
			Expect(pvcStore.Add(claim)).To(Succeed())
			vmi := newVMI(targetUID)

			Expect(controller.PrepareRestore(vm, vmi)).To(Succeed())

			Expect(vmi.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(BeNil(), "only a preference")
			Expect(vmi.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution).To(ConsistOf(k8sv1.PreferredSchedulingTerm{
				Weight: sourceNodeWeight,
				Preference: k8sv1.NodeSelectorTerm{MatchFields: []k8sv1.NodeSelectorRequirement{{
					Key: "metadata.name", Operator: k8sv1.NodeSelectorOpIn, Values: []string{nodeName},
				}}},
			}))
		})

		DescribeTable("should discard a saved state that does not match the disks anymore", func(change func(vm *v1.VirtualMachine), reason string) {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			Expect(pvcStore.Add(newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm))).To(Succeed())
			change(vm)
			vmi := newVMI(targetUID)

			Expect(controller.PrepareRestore(vm, vmi)).To(Succeed())

			Expect(vmi.Spec.SavedStateSource).To(BeNil())
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring(reason))
			Expect(recorder.Events).To(Receive(ContainSubstring(SavedStateDiscardedReason)))
		},
			Entry("changed spec", func(vm *v1.VirtualMachine) {
				vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{Cores: 8}
			}, "spec changed"),
			Entry("replaced disk PVC", func(vm *v1.VirtualMachine) {
				Expect(pvcStore.Update(newClaim(diskClaim, "new-disk-uid", nil))).To(Succeed())
			}, "PVCs of the VirtualMachine disks changed"),
			Entry("missing state PVC", func(vm *v1.VirtualMachine) {
				Expect(pvcStore.Delete(newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm))).To(Succeed())
			}, "is gone"),
		)

		It("should discard a saved state whose previous restore started and never reported back", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vm.Status.SavedState.Phase = v1.HibernationRestoring
			vm.Status.SavedState.RestoreUID = targetUID
			vm.Status.SavedState.RestoreStartTimestamp = new(metav1.Now())
			vmi := newVMI("")
			Expect(controller.PrepareRestore(vm, vmi)).To(Succeed())
			Expect(vmi.Spec.SavedStateSource).To(BeNil())
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
		})

		It("should resume again from a saved state whose previous VMI never started resuming", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			Expect(pvcStore.Add(newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm))).To(Succeed())
			vm.Status.SavedState.Phase = v1.HibernationRestoring
			vm.Status.SavedState.RestoreUID = targetUID
			vmi := newVMI("")
			Expect(controller.PrepareRestore(vm, vmi)).To(Succeed())
			Expect(vmi.Spec.SavedStateSource).ToNot(BeNil())
		})
	})

	Context("UpdateStatus while restoring", func() {
		It("should detect the VMI resuming from the saved state", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vmi := newVMI(targetUID)
			vmi.Spec.SavedStateSource = &v1.SavedStateSource{ClaimName: vm.Status.SavedState.ClaimName, FileName: vm.Status.SavedState.FileName}

			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestoring), "even if the update recording the creation was lost")
			Expect(vm.Status.SavedState.RestoreUID).To(Equal(targetUID))

			start, end := metav1.Now(), metav1.Now()
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateRestoring, StartTimestamp: &start}
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.RestoreStartTimestamp).To(Equal(&start), "virt-handler committed to the restore")

			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateRestored, StartTimestamp: &start, EndTimestamp: &end}
			Expect(IsInProgress(vm, vmi)).To(BeTrue(), "the VMI is kept until its result is recorded")
			controller.UpdateStatus(vm, vmi)
			Expect(IsInProgress(vm, vmi)).To(BeFalse())
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestored))
			Expect(vm.Status.SavedState.RestoreEndTimestamp).To(Equal(&end))
			Expect(recorder.Events).To(Receive(ContainSubstring(ResumingReason)))
			Expect(recorder.Events).To(Receive(ContainSubstring(ResumedReason)))
		})

		It("should discard the saved state when a VMI boots without it", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			controller.UpdateStatus(vm, newVMI(targetUID))
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
		})

		It("should keep the saved state and hold the start when resuming failed", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vm.Status.SavedState.Phase = v1.HibernationRestoring
			vm.Status.SavedState.RestoreUID = targetUID
			vmi := newVMI(targetUID)
			vmi.Status.Phase = v1.Failed
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateRestoreFailed, Message: "ABI mismatch"}
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestoreFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("ABI mismatch"))
			Expect(recorder.Events).To(Receive(ContainSubstring(ResumeFailedReason)))
			message, blocked := BlocksStart(vm)
			Expect(blocked).To(BeTrue())
			Expect(message).To(ContainSubstring("start it to retry"))
			Expect(message).To(ContainSubstring("discard the saved state"))

			By("keeping the PVC")
			claim := newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm)
			Expect(pvcStore.Add(claim)).To(Succeed())
			_, err := k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Create(context.Background(), claim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(controller.Sync(vm, nil)).To(Succeed())
			_, err = k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), claim.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.UpdateStatus(vm, nil)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestoreFailed), "no retry, no boot")

			By("discarding the saved state")
			vm.Status.SavedState.Discard = true
			_, blocked = BlocksStart(vm)
			Expect(blocked).To(BeFalse())
			Expect(controller.Sync(vm, nil)).To(Succeed())
			_, err = k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), claim.Name, metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
			Expect(pvcStore.Delete(claim)).To(Succeed())
			controller.UpdateStatus(vm, nil)
			Expect(vm.Status.SavedState).To(BeNil())

			By("booting the VM")
			next := newVMI("next-uid")
			Expect(controller.PrepareRestore(vm, next)).To(Succeed())
			Expect(next.Spec.SavedStateSource).To(BeNil())
		})

		It("should wait for the VMI to show up in the cache", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vm.Status.SavedState.Phase = v1.HibernationRestoring
			controller.UpdateStatus(vm, nil)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestoring))
		})
	})

	Context("cancelling or failing a hibernation", func() {
		var (
			vm  *v1.VirtualMachine
			vmi *v1.VirtualMachineInstance
		)

		BeforeEach(func() {
			vm, vmi = newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			// What virt-api does to hibernate a VM that always runs.
			vm.Spec.RunStrategy = new(v1.RunStrategyHalted)
			_, err := k8sClient.CoreV1().Nodes().Create(context.Background(), &k8sv1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeName}}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		// reconcile runs the hibernation steps of a VM controller sync: the
		// stop of the VMI is held while IsInProgress, Sync may change the spec,
		// which the controller persists, and UpdateStatus moves the phase.
		reconcile := func() (stopHeld bool) {
			stopHeld = IsInProgress(vm, vmi)
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			controller.UpdateStatus(vm, vmi)
			return stopHeld
		}

		It("should keep the VM running with its run strategy when a pending hibernation is cancelled", func() {
			vm.Status.SavedState.Discard = true

			Expect(reconcile()).To(BeTrue())
			state := vm.Status.SavedState
			Expect(state.Phase).To(Equal(v1.HibernationFailed))
			Expect(state.Message).To(ContainSubstring("cancelled"))
			Expect(recorder.Events).To(Receive(And(HavePrefix(k8sv1.EventTypeNormal), ContainSubstring(HibernationCancelledReason))))
			Expect(vm.Spec.RunStrategy).To(HaveValue(Equal(v1.RunStrategyHalted)), "UpdateStatus never changes the spec")
			Expect(IsInProgress(vm, vmi)).To(BeTrue(), "the stop stays held until the run strategy is restored")

			Expect(reconcile()).To(BeTrue())
			Expect(vm.Spec.RunStrategy).To(HaveValue(Equal(v1.RunStrategyAlways)))
			Expect(IsInProgress(vm, vmi)).To(BeFalse())
			Expect(vm.Status.SavedState).To(BeNil(), "a cancelled hibernation without a PVC is forgotten")
		})

		It("should not undo a later stop of the VM", func() {
			vm.Status.SavedState.Discard = true
			reconcile()
			reconcile()
			vm.Status.SavedState = &v1.VirtualMachineSavedState{Phase: v1.HibernationFailed, SourceUID: vmi.UID}
			vm.Spec.RunStrategy = new(v1.RunStrategyHalted)
			Expect(reconcile()).To(BeFalse())
			Expect(vm.Spec.RunStrategy).To(HaveValue(Equal(v1.RunStrategyHalted)))
		})

		It("should restore spec.running", func() {
			vm.Spec.RunStrategy = nil
			vm.Spec.Running = new(false)
			vm.Status.SavedState.Discard = true
			reconcile()
			reconcile()
			Expect(vm.Spec.Running).To(HaveValue(BeTrue()))
		})

		It("should withdraw the stop request of the hibernation of a manually run VM, not the one of the user", func() {
			vm.Spec.RunStrategy = new(v1.RunStrategyManual)
			vm.Status.SavedState.RunStrategy = new(v1.RunStrategyManual)
			userStop := v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID}
			vm.Status.StateChangeRequests = []v1.VirtualMachineStateChangeRequest{
				StopRequest(vmi, vm.Status.SavedState.ClaimName),
				userStop,
			}
			vm.Status.SavedState.Discard = true

			Expect(reconcile()).To(BeTrue())
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.StateChangeRequests).To(ConsistOf(userStop), "the VM stops as the user asked")
			Expect(IsInProgress(vm, vmi)).To(BeFalse())
		})

		It("should go on when the cancellation came after the save started", func() {
			vm.Status.SavedState.Discard = true
			start := metav1.Now()
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
				Phase: v1.SavedStateSaving, FileName: FileNameFor(vm.Status.SavedState.ClaimName), StartTimestamp: &start,
			}
			Expect(reconcile()).To(BeTrue())
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaving))
			Expect(vm.Status.SavedState.Discard).To(BeFalse())
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("could not be cancelled"))
			Expect(vm.Spec.RunStrategy).To(HaveValue(Equal(v1.RunStrategyHalted)))
		})

		It("should keep the VM running when saving fails", func() {
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
				Phase: v1.SavedStateSaveFailed, FileName: FileNameFor(vm.Status.SavedState.ClaimName), Message: "aborted",
			}
			reconcile()
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("keeps running"))
			Expect(IsInProgress(vm, vmi)).To(BeTrue())
			reconcile()
			Expect(vm.Spec.RunStrategy).To(HaveValue(Equal(v1.RunStrategyAlways)))
			Expect(vm.Status.SavedState.RunStrategy).To(BeNil())
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed), "a failure is reported, unlike a cancel")
		})

		It("should keep the VM running when virt-launcher aborts a save that timed out", func() {
			start := metav1.Now()
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
				Phase: v1.SavedStateSaving, FileName: FileNameFor(vm.Status.SavedState.ClaimName), StartTimestamp: &start,
			}
			reconcile()
			controller.now = func() time.Time { return start.Add(SavingTimeout(vmi) + savingTimeoutGrace + time.Second) }
			reconcile()
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaving))
			vmi.Status.SavedState.Phase = v1.SavedStateSaveFailed
			reconcile()
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			reconcile()
			Expect(vm.Spec.RunStrategy).To(HaveValue(Equal(v1.RunStrategyAlways)))
		})

		It("should leave the VM stopped when its VMI stopped before the save", func() {
			vmi.Status.Phase = v1.Succeeded
			reconcile()
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(IsInProgress(vm, vmi)).To(BeFalse())
			reconcile()
			Expect(vm.Spec.RunStrategy).To(HaveValue(Equal(v1.RunStrategyHalted)))
		})
	})

	Context("resuming", func() {
		var (
			vm    *v1.VirtualMachine
			vmi   *v1.VirtualMachineInstance
			claim *k8sv1.PersistentVolumeClaim
		)

		BeforeEach(func() {
			vm, _ = newRunningVM()
			vm.Status.SavedState = savedState(vm)
			claim = newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm)
			Expect(pvcStore.Add(claim)).To(Succeed())
			_, err := k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Create(context.Background(), claim, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi = newVMI(targetUID)
			Expect(controller.PrepareRestore(vm, vmi)).To(Succeed())
			vmi.Status.Phase = v1.Pending
			vmi.CreationTimestamp = metav1.Now()
			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.RestoreCreated(vm, vmi)
		})

		committed := func() {
			vmi.Status.Phase = v1.Scheduled
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateRestoring, StartTimestamp: new(metav1.Now())}
		}

		getVMI := func() (*v1.VirtualMachineInstance, error) {
			return virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		}

		It("should hot plug the saved state into the new VMI and release it once resumed", func() {
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			updated, err := getVMI()
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Spec.UtilityVolumes).To(ConsistOf(v1.UtilityVolume{
				Name:                              SourceVolumeName,
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: claim.Name},
				Type:                              new(v1.SavedState),
			}))
			Expect(updated.Spec.UtilityVolumes).ToNot(ContainElement(Satisfy(IsSaveTargetVolume)))

			By("resuming")
			start, end := metav1.Now(), metav1.Now()
			updated.Status.Phase = v1.Running
			updated.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateRestored, StartTimestamp: &start, EndTimestamp: &end}
			updated.Status.VolumeStatus = []v1.VolumeStatus{{Name: SourceVolumeName, Phase: v1.HotplugVolumeMounted, HotplugVolume: &v1.HotplugVolumeStatus{}}}
			controller.UpdateStatus(vm, updated)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestored))

			By("unplugging the volume before deleting the PVC")
			Expect(controller.Sync(vm, updated)).To(Succeed())
			updated, err = getVMI()
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Spec.UtilityVolumes).To(BeEmpty())
			_, err = k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), claim.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			updated.Status.VolumeStatus = nil
			Expect(controller.Sync(vm, updated)).To(Succeed())
			_, err = k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), claim.Name, metav1.GetOptions{})
			Expect(err).To(HaveOccurred(), "the PVC is released while the VM runs")
		})

		It("should boot instead when the saved state is discarded before the VMI resumes", func() {
			vm.Status.SavedState.Discard = true
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			_, err := getVMI()
			Expect(err).To(HaveOccurred(), "the VMI that never ran is stopped")
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("discarded before"))
			_, blocked := BlocksStart(vm)
			Expect(blocked).To(BeFalse())
		})

		It("should let a VMI that committed to the restore finish before discarding it", func() {
			vm.Status.SavedState.Discard = true
			committed()
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			_, err := getVMI()
			Expect(err).ToNot(HaveOccurred())
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestoring))
		})

		It("should report a VMI that no node can run and give up after a while", func() {
			vmi.Status.Phase = v1.Scheduling
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{
				Type:    v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled),
				Status:  k8sv1.ConditionFalse,
				Reason:  k8sv1.PodReasonUnschedulable,
				Message: "0/2 nodes are available: 2 node(s) didn't match Pod's node affinity/selector",
			}}
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestoring))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("Waiting for a node"))

			controller.now = func() time.Time { return vmi.CreationTimestamp.Add(RestoreStartTimeout + time.Second) }
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			_, err := getVMI()
			Expect(err).To(HaveOccurred())
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestoreFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("no node can resume the saved state"))
			_, err = k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), claim.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "the saved state is kept")
		})

		It("should never give up on a VMI that committed to the restore, whatever the time it takes", func() {
			committed()
			controller.now = func() time.Time { return vmi.CreationTimestamp.Add(10 * RestoreStartTimeout) }
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			_, err := getVMI()
			Expect(err).ToNot(HaveOccurred())
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestoring))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("takes longer"))
		})

		It("should keep the saved state when the VMI stops before committing to the restore", func() {
			vmi.Status.Phase = v1.Failed
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaved))
			Expect(vm.Status.SavedState.RestoreUID).To(BeEmpty())
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaved), "the failed VMI is not adopted again")
		})

		It("should discard the saved state when the VMI stops after committing to the restore", func() {
			committed()
			vmi.Status.Phase = v1.Failed
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
		})
	})

	Context("serializing the cancel and the start of the save", func() {
		var (
			vm    *v1.VirtualMachine
			vmi   *v1.VirtualMachineInstance
			claim *k8sv1.PersistentVolumeClaim
		)

		BeforeEach(func() {
			vm, vmi = newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			vm.Status.SavedState.SpecHash = "recorded"
			claim = newClaim(vm.Status.SavedState.ClaimName, "state-uid", vm)
			Expect(pvcStore.Add(claim)).To(Succeed())
			withSavedStateVolume(vmi, claim.Name)
			vmi.ResourceVersion = "42"
			_, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should disarm an armed save before cancelling, with a patch conditional on the VMI it saw", func() {
			var patches []string
			virtFakeClient.PrependReactor("patch", "virtualmachineinstances", func(action testing.Action) (bool, runtime.Object, error) {
				patches = append(patches, string(action.(testing.PatchAction).GetPatch()))
				return false, nil, nil
			})
			vm.Status.SavedState.Discard = true

			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationPending), "not cancelled while armed")
			Expect(IsInProgress(vm, vmi)).To(BeTrue())

			Expect(controller.Sync(vm, vmi)).To(Succeed())
			Expect(patches).To(ConsistOf(And(
				ContainSubstring(`{"op":"test","path":"/metadata/resourceVersion","value":"42"}`),
				ContainSubstring(`"path":"/spec/utilityVolumes"`),
			)))
			disarmed, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(disarmed.Spec.UtilityVolumes).To(BeEmpty())

			controller.UpdateStatus(vm, disarmed)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("cancelled"))
		})

		It("should let the save go on when virt-handler committed to it first", func() {
			vm.Status.SavedState.Discard = true
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
				Phase: v1.SavedStateSaving, FileName: FileNameFor(claim.Name), StartTimestamp: new(metav1.Now()),
			}
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			current, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(current.Spec.UtilityVolumes).ToNot(BeEmpty(), "a committed save is never disarmed")
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaving))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("could not be cancelled"))
		})

		It("should fail the disarm when the VMI changed since it was seen, e.g. by the commit of virt-handler", func() {
			virtFakeClient.PrependReactor("patch", "virtualmachineinstances", func(testing.Action) (bool, runtime.Object, error) {
				return true, nil, fmt.Errorf("the server rejected our request due to an error in our request: testing jsonpatch failed")
			})
			vm.Status.SavedState.Discard = true
			Expect(controller.Sync(vm, vmi)).To(MatchError(ContainSubstring("unable to disarm")))
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationPending), "the next reconcile sees who won")
		})

		It("should time out the request, not the PVC, and disarm first", func() {
			controller.now = func() time.Time { return vm.Status.SavedState.RequestTimestamp.Add(PendingTimeout + time.Second) }
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationPending))
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			disarmed, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			controller.UpdateStatus(vm, disarmed)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("did not start"))
		})
	})

	Context("an incomplete request", func() {
		It("should neither provision nor arm before virt-api stopped the VM, and cancel the request after a while", func() {
			vm, vmi := newRunningVM()
			// virt-api recorded the hibernation, but switching to Halted failed.
			vm.Status.SavedState = NewSavedState(vm, vmi, nil)
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			_, err := k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), vm.Status.SavedState.ClaimName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationPending))
			Expect(vm.Status.SavedState.SpecHash).To(BeEmpty())

			controller.now = func() time.Time { return vm.Status.SavedState.RequestTimestamp.Add(requestGrace + time.Second) }
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("not completed"))
			Expect(vm.Spec.RunStrategy).To(HaveValue(Equal(v1.RunStrategyAlways)))
			Expect(IsInProgress(vm, vmi)).To(BeFalse())
		})
	})

	Context("forgetting a discarded saved state", func() {
		It("should keep the record while the run strategy is still to be restored", func() {
			vm, vmi := newRunningVM()
			vm.Status.SavedState = pendingState(vm, vmi)
			vm.Status.SavedState.Phase = v1.HibernationFailed
			vm.Status.SavedState.Discard = true
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState).ToNot(BeNil(), "the PVC is gone, but the VM is still Halted")
			Expect(vm.Status.SavedState.RunStrategy).To(HaveValue(Equal(v1.RunStrategyAlways)))

			restoreRunStrategy(vm, vmi)
			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState).To(BeNil())
		})
	})

	Context("a VM whose resume failed", func() {
		DescribeTable("should replace the final VMI that failed to resume once the failure is handled", func(change func(*v1.VirtualMachine, *v1.VirtualMachineInstance), gateEnabled, superseded bool) {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vm.Status.SavedState.Phase = v1.HibernationRestoreFailed
			vm.Status.SavedState.RestoreUID = targetUID
			vmi := newVMI(targetUID)
			vmi.Spec.SavedStateSource = &v1.SavedStateSource{ClaimName: vm.Status.SavedState.ClaimName, FileName: vm.Status.SavedState.FileName}
			vmi.Status.Phase = v1.Failed
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateRestoreFailed}
			change(vm, vmi)
			Expect(IsSupersededRestore(vm, vmi, gateEnabled)).To(Equal(superseded))
		},
			Entry("not while the failure waits for a retry or a discard", func(*v1.VirtualMachine, *v1.VirtualMachineInstance) {}, true, false),
			Entry("not before its result is recorded", func(vm *v1.VirtualMachine, _ *v1.VirtualMachineInstance) {
				vm.Status.SavedState.Phase = v1.HibernationRestoring
			}, true, false),
			Entry("not before its result is recorded, without its UID", func(vm *v1.VirtualMachine, _ *v1.VirtualMachineInstance) {
				vm.Status.SavedState.Phase = v1.HibernationRestoring
				vm.Status.SavedState.RestoreUID = ""
			}, true, false),
			Entry("not while Saved because the update recording its creation was lost", func(vm *v1.VirtualMachine, _ *v1.VirtualMachineInstance) {
				vm.Status.SavedState.Phase = v1.HibernationSaved
				vm.Status.SavedState.RestoreUID = ""
			}, true, false),
			Entry("not after a retry that superseded another VMI", func(vm *v1.VirtualMachine, _ *v1.VirtualMachineInstance) {
				vm.Status.SavedState.Phase = v1.HibernationSaved
				vm.Status.SavedState.RestoreUID = ""
				vm.Status.SavedState.SupersededUID = "earlier-uid"
			}, true, false),
			Entry("once a start retried the resume", func(vm *v1.VirtualMachine, _ *v1.VirtualMachineInstance) {
				vm.Status.SavedState.Phase = v1.HibernationSaved
				vm.Status.SavedState.RestoreUID = ""
				vm.Status.SavedState.SupersededUID = targetUID
			}, true, true),
			Entry("once the VM controller discarded the saved state", func(vm *v1.VirtualMachine, _ *v1.VirtualMachineInstance) {
				vm.Status.SavedState.Phase = v1.HibernationFailed
			}, true, true),
			Entry("once the saved state is discarded", func(vm *v1.VirtualMachine, _ *v1.VirtualMachineInstance) {
				vm.Status.SavedState.Discard = true
			}, true, true),
			Entry("once the saved state is gone", func(vm *v1.VirtualMachine, _ *v1.VirtualMachineInstance) {
				vm.Status.SavedState = nil
			}, true, true),
			Entry("with the feature gate off", func(*v1.VirtualMachine, *v1.VirtualMachineInstance) {}, false, true),
			Entry("never a VMI that is not final", func(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
				vm.Status.SavedState.Phase = v1.HibernationSaved
				vm.Status.SavedState.SupersededUID = targetUID
				vmi.Status.Phase = v1.Scheduled
			}, true, false),
			Entry("never a VMI that failed for another reason", func(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
				vm.Status.SavedState.Phase = v1.HibernationSaved
				vm.Status.SavedState.SupersededUID = targetUID
				vmi.Status.SavedState = nil
			}, true, false),
		)

		failedRestoreVMI := func(vm *v1.VirtualMachine) *v1.VirtualMachineInstance {
			vmi := newVMI(targetUID)
			vmi.Spec.SavedStateSource = &v1.SavedStateSource{ClaimName: vm.Status.SavedState.ClaimName, FileName: vm.Status.SavedState.FileName}
			vmi.Status.Phase = v1.Failed
			vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{Phase: v1.SavedStateRestoreFailed, Message: "corrupt"}
			return vmi
		}

		DescribeTable("should record the failed resume of a VMI whose creation the VM did not record", func(phase v1.VirtualMachineSavedStatePhase) {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vm.Status.SavedState.Phase = phase
			vmi := failedRestoreVMI(vm)
			Expect(IsInProgress(vm, vmi)).To(BeTrue(), "no run strategy deletes the VMI before its result is recorded")
			Expect(IsSupersededRestore(vm, vmi, true)).To(BeFalse())

			controller.UpdateStatus(vm, vmi)

			state := vm.Status.SavedState
			Expect(state.Phase).To(Equal(v1.HibernationRestoreFailed))
			Expect(state.RestoreUID).To(Equal(targetUID))
			Expect(state.Message).To(ContainSubstring("corrupt"))
			Expect(IsInProgress(vm, vmi)).To(BeFalse())
			Expect(IsSupersededRestore(vm, vmi, true)).To(BeFalse(), "waits for a retry or a discard")
			msg, blocked := BlocksStart(vm)
			Expect(blocked).To(BeTrue(), msg)
		},
			Entry("Saved", v1.HibernationSaved),
			Entry("Restoring", v1.HibernationRestoring),
		)

		It("should record the resume of a VMI whose creation the VM did not record", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vmi := failedRestoreVMI(vm)
			vmi.Status.Phase = v1.Running
			vmi.Status.SavedState.Phase = v1.SavedStateRestored
			Expect(IsInProgress(vm, vmi)).To(BeTrue())

			controller.UpdateStatus(vm, vmi)

			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestored))
			Expect(vm.Status.SavedState.RestoreUID).To(Equal(targetUID))
		})

		It("should keep the state Saved for a VMI that stopped before it committed to the restore", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vmi := failedRestoreVMI(vm)
			vmi.Status.SavedState = nil

			controller.UpdateStatus(vm, vmi)

			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaved))
			Expect(vm.Status.SavedState.RestoreUID).To(BeEmpty())
		})

		It("should leave the VMI a retry superseded to the VM controller, and record the next one", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vm.Status.SavedState.SupersededUID = targetUID
			vmi := failedRestoreVMI(vm)
			Expect(IsInProgress(vm, vmi)).To(BeFalse())
			Expect(IsSupersededRestore(vm, vmi, true)).To(BeTrue())

			controller.UpdateStatus(vm, vmi)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationSaved), "the failure was already handled")

			By("recording the next restore VMI, whose creation update was lost")
			next := failedRestoreVMI(vm)
			next.UID = "next-uid"
			Expect(IsSupersededRestore(vm, next, true)).To(BeFalse())
			controller.UpdateStatus(vm, next)
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationRestoreFailed))
			Expect(vm.Status.SavedState.RestoreUID).To(Equal(next.UID))
			Expect(vm.Status.SavedState.SupersededUID).To(BeEmpty())
		})

		It("should tell which VMI a retry supersedes", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vm.Status.SavedState.Phase = v1.HibernationRestoreFailed
			vmi := failedRestoreVMI(vm)
			Expect(SupersedesRestore(vm, vmi)).To(BeTrue())
			vmi.Status.Phase = v1.Running
			Expect(SupersedesRestore(vm, vmi)).To(BeTrue(), "a VMI that reported a failed resume never runs its guest")
			vmi.Spec.SavedStateSource.ClaimName = "other"
			Expect(SupersedesRestore(vm, vmi)).To(BeFalse())
			Expect(SupersedesRestore(vm, newVMI(targetUID))).To(BeFalse())
			Expect(SupersedesRestore(vm, nil)).To(BeFalse())
		})

		It("should discard the saved state when the VM boots anyway, e.g. with the feature gate off", func() {
			vm, _ := newRunningVM()
			vm.Status.SavedState = savedState(vm)
			vm.Status.SavedState.Phase = v1.HibernationRestoreFailed
			vm.Status.SavedState.RestoreUID = targetUID
			controller.UpdateStatus(vm, newVMI("booted-uid"))
			Expect(vm.Status.SavedState.Phase).To(Equal(v1.HibernationFailed))
			Expect(vm.Status.SavedState.Message).To(ContainSubstring("without resuming"))
		})
	})

	Context("hibernate, resume and hibernate again", func() {
		BeforeEach(func() {
			_, err := k8sClient.CoreV1().Nodes().Create(context.Background(), &k8sv1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeName}}, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should save the state of a VMI that was resumed from a saved state", func() {
			vm, vmi := resumedVM()
			oldClaim := vm.Status.SavedState.ClaimName
			restoreSource := newHibernationClaim(oldClaim, vm)
			Expect(pvcStore.Add(restoreSource)).To(Succeed())
			_, err := k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Create(context.Background(), restoreSource, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			_, err = virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("requesting the hibernation of the resumed VMI")
			Expect(ValidateRequest(vm, vmi)).To(Succeed())
			vm.Status.SavedState = pendingState(vm, vmi)
			state := vm.Status.SavedState
			Expect(state.ClaimName).ToNot(Equal(oldClaim))
			Expect(state.SourceUID).To(Equal(targetUID))
			Expect(IsInProgress(vm, vmi)).To(BeTrue())

			By("provisioning a new PVC and deleting the one the VMI resumed from")
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			claim, err := k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), state.ClaimName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(claim.Annotations).To(HaveKeyWithValue(SourceNodeAnnotation, nodeName))
			_, err = k8sClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Get(context.Background(), oldClaim, metav1.GetOptions{})
			Expect(err).To(HaveOccurred(), "the restore source is stale")

			By("recording what the state is valid with, then arming the save next to the restore source")
			claim.CreationTimestamp = metav1.Now()
			Expect(pvcStore.Add(claim)).To(Succeed())
			controller.UpdateStatus(vm, vmi)
			Expect(state.SpecHash).ToNot(BeEmpty())
			Expect(controller.Sync(vm, vmi)).To(Succeed())
			updated, err := virtFakeClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Spec.UtilityVolumes).To(ConsistOf(v1.UtilityVolume{
				Name:                              VolumeName,
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: state.ClaimName},
				Type:                              new(v1.SavedState),
			}))
			Expect(updated.Spec.SavedStateSource.ClaimName).To(Equal(oldClaim))

			By("waiting for the save while the VMI still reports its restore")
			controller.UpdateStatus(vm, updated)
			Expect(state.Phase).To(Equal(v1.HibernationPending))
			Expect(state.SpecHash).ToNot(BeEmpty())

			By("following the save to the new PVC")
			start, end := metav1.Now(), metav1.Now()
			updated.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
				Phase: v1.SavedStateSaving, FileName: FileNameFor(state.ClaimName), StartTimestamp: &start,
			}
			controller.UpdateStatus(vm, updated)
			Expect(state.Phase).To(Equal(v1.HibernationSaving))
			updated.Status.Phase = v1.Succeeded
			updated.Status.SavedState.Phase = v1.SavedStateSaved
			updated.Status.SavedState.EndTimestamp = &end
			controller.UpdateStatus(vm, updated)
			Expect(state.Phase).To(Equal(v1.HibernationSaved))
			Expect(state.FileName).To(Equal(FileNameFor(state.ClaimName)))
			Expect(recorder.Events).To(Receive(ContainSubstring(HibernatedReason)))
			Expect(IsInProgress(vm, updated)).To(BeFalse())

			By("resuming from the new state")
			next := newVMI("third-uid")
			Expect(controller.PrepareRestore(vm, next)).To(Succeed())
			Expect(next.Spec.SavedStateSource).To(Equal(&v1.SavedStateSource{ClaimName: state.ClaimName, FileName: FileNameFor(state.ClaimName)}))
			controller.RestoreCreated(vm, next)
			Expect(state.Phase).To(Equal(v1.HibernationRestoring))
		})
	})
})

func withSavedStateVolume(vmi *v1.VirtualMachineInstance, claimName string) {
	vmi.Spec.UtilityVolumes = append(vmi.Spec.UtilityVolumes, v1.UtilityVolume{
		Name:                              VolumeName,
		PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
		Type:                              new(v1.SavedState),
	})
}

// resumedVM returns a VM whose running VMI resumed from a saved state.
func resumedVM() (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	vm, _ := newRunningVM()
	state := savedState(vm)
	start, end := metav1.Now(), metav1.Now()
	state.Phase = v1.HibernationRestored
	state.RestoreUID = targetUID
	vm.Status.SavedState = state
	vmi := newVMI(targetUID)
	vmi.Spec.SavedStateSource = &v1.SavedStateSource{ClaimName: state.ClaimName, FileName: state.FileName}
	vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
		Phase: v1.SavedStateRestored, FileName: state.FileName, StartTimestamp: &start, EndTimestamp: &end,
	}
	return vm, vmi
}

func newHibernationClaim(name string, owner *v1.VirtualMachine) *k8sv1.PersistentVolumeClaim {
	claim := newClaim(name, types.UID(name+"-uid"), owner)
	claim.Labels = map[string]string{v1.AppLabel: appLabelValue, v1.VirtualMachineLabel: owner.Name}
	return claim
}

func newVMI(uid types.UID) *v1.VirtualMachineInstance {
	vmi := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{Name: vmName, Namespace: metav1.NamespaceDefault, UID: uid},
		Spec: v1.VirtualMachineInstanceSpec{
			Domain: v1.DomainSpec{Memory: &v1.Memory{Guest: new(resource.MustParse("2Gi"))}},
			Volumes: []v1.Volume{{
				Name: "root",
				VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: diskClaim},
				}},
			}},
		},
		Status: v1.VirtualMachineInstanceStatus{Phase: v1.Running, NodeName: nodeName},
	}
	return vmi
}

func newRunningVM() (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
	vmi := newVMI(sourceUID)
	vm := &v1.VirtualMachine{
		TypeMeta:   metav1.TypeMeta{Kind: "VirtualMachine", APIVersion: v1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Name: vmName, Namespace: metav1.NamespaceDefault, UID: "vm-uid"},
		Spec: v1.VirtualMachineSpec{
			RunStrategy: new(v1.RunStrategyAlways),
			Template:    &v1.VirtualMachineInstanceTemplateSpec{Spec: *vmi.Spec.DeepCopy()},
		},
	}
	return vm, vmi
}

// pendingState returns the state of a complete hibernation request for vmi:
// like virt-api, it switches a VM that always runs to Halted.
func pendingState(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) *v1.VirtualMachineSavedState {
	state := NewSavedState(vm, vmi, nil)
	if replacesRunStrategy(*state.RunStrategy) {
		vm.Spec.RunStrategy = new(v1.RunStrategyHalted)
	}
	return state
}

func savedState(vm *v1.VirtualMachine) *v1.VirtualMachineSavedState {
	hash, err := SpecHash(vm)
	Expect(err).ToNot(HaveOccurred())
	claimName := ClaimName(vm.Name, sourceUID, random)
	return &v1.VirtualMachineSavedState{
		Phase:           v1.HibernationSaved,
		ClaimName:       claimName,
		FileName:        FileNameFor(claimName),
		SourceUID:       sourceUID,
		SpecHash:        hash,
		VolumeClaimUIDs: map[string]types.UID{diskClaim: diskUID},
	}
}

func newClaim(name string, uid types.UID, owner *v1.VirtualMachine) *k8sv1.PersistentVolumeClaim {
	claim := &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: metav1.NamespaceDefault, UID: uid},
	}
	if owner != nil {
		claim.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(owner, v1.VirtualMachineGroupVersionKind)}
		claim.Labels = map[string]string{v1.AppLabel: appLabelValue, v1.VirtualMachineLabel: owner.Name}
	}
	return claim
}
