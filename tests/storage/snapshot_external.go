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

package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("External VMSnapshot", func() {
	var (
		virtClient kubecli.KubevirtClient
		vm         *v1.VirtualMachine
		snapshot   *snapshotv1.VirtualMachineSnapshot
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	AfterEach(func() {
		if snapshot != nil {
			_ = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Delete(
				context.Background(), snapshot.Name, metav1.DeleteOptions{})
			snapshot = nil
		}
	})

	externalSnapshotOf := func(vmName, namespace string) *snapshotv1.VirtualMachineSnapshot {
		s := libstorage.NewSnapshot(vmName, namespace)
		s.Spec.SnapshotMode = pointer.P(snapshotv1.SnapshotModeExternal)
		return s
	}

	// The API rejections do not need a storage class or a running VM, and the
	// CEL rules in particular cannot be reached from an admitter unit test.
	Context("validation", func() {
		var namespace string

		BeforeEach(func() {
			namespace = testsuite.GetTestNamespace(nil)
		})

		create := func(s *snapshotv1.VirtualMachineSnapshot) error {
			_, err := virtClient.VirtualMachineSnapshot(namespace).Create(
				context.Background(), s, metav1.CreateOptions{})
			return err
		}

		DescribeTable("should reject an overlayScratchSize that cannot hold anything", func(size string) {
			s := externalSnapshotOf("novm", namespace)
			s.Spec.OverlayScratchSize = pointer.P(resource.MustParse(size))

			Expect(create(s)).To(MatchError(ContainSubstring("overlayScratchSize")))
		},
			Entry("zero", "0"),
			Entry("negative", "-1Gi"),
		)

		It("should reject overlayScratchSize on a Direct mode snapshot", func() {
			s := libstorage.NewSnapshot("novm", namespace)
			s.Spec.SnapshotMode = pointer.P(snapshotv1.SnapshotModeDirect)
			s.Spec.OverlayScratchSize = pointer.P(resource.MustParse("1Gi"))

			Expect(create(s)).To(MatchError(ContainSubstring("overlayScratchSize")))
		})

		It("should reject External mode when the VM is not running", func() {
			stopped := libvmi.NewVirtualMachine(libvmifact.NewAlpine(libvmi.WithNamespace(namespace)))
			stopped, err := virtClient.VirtualMachine(namespace).Create(
				context.Background(), stopped, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(create(externalSnapshotOf(stopped.Name, namespace))).
				To(MatchError(ContainSubstring("running VirtualMachine")))
		})
	})

	Context("[storage-req]", decorators.StorageReq, decorators.RequiresSnapshotStorageClass, func() {
		var snapshotStorageClass string

		BeforeEach(func() {
			sc, err := libstorage.GetSnapshotStorageClass(virtClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(sc).ToNot(BeEmpty(), "No snapshot storage class available")
			snapshotStorageClass = sc
		})

		// Exercise a multi-disk snapshot and sequential block commits.
		const dataDisks = 6

		createRunningVM := func(disks int, options ...libvmi.Option) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
			var dvs []*cdiv1.DataVolume
			var vmiOpts []libvmi.Option
			for i := 0; i < disks; i++ {
				dv := libdv.NewDataVolume(
					libdv.WithBlankImageSource(),
					libdv.WithStorage(libdv.StorageWithStorageClass(snapshotStorageClass)),
				)
				dvs = append(dvs, dv)
				vmiOpts = append(vmiOpts, libvmi.WithDataVolume(fmt.Sprintf("data-disk-%d", i), dv.Name))
			}

			vmiOpts = append([]libvmi.Option{
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
				libnet.WithMasqueradeNetworking(),
			}, vmiOpts...)

			var vmOpts []libvmi.VMOption
			for _, dv := range dvs {
				vmOpts = append(vmOpts, libvmi.WithDataVolumeTemplate(dv))
			}

			created := libvmi.NewVirtualMachine(libvmifact.NewFedora(append(vmiOpts, options...)...), vmOpts...)
			created.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)
			for i := range created.Spec.Template.Spec.Domain.Devices.Disks {
				disk := &created.Spec.Template.Spec.Domain.Devices.Disks[i]
				if strings.HasPrefix(disk.Name, "data-disk-") {
					disk.Serial = "snapshot-" + disk.Name
				}
			}

			created, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(
				context.Background(), created, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(ThisVMIWith(created.Namespace, created.Name), 360*time.Second).Should(BeInPhase(v1.Running))
			vmi, err := virtClient.VirtualMachineInstance(created.Namespace).Get(
				context.Background(), created.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			libwait.WaitForSuccessfulVMIStart(vmi, libwait.WithTimeout(300))
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(
				matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			return created, vmi
		}

		// The scratch volume is named from the content UID, so the content has
		// to exist before the name can be derived.
		scratchPVCName := func() string {
			var name string
			Eventually(func() error {
				s, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Get(
					context.Background(), snapshot.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if s.Status == nil || s.Status.VirtualMachineSnapshotContentName == nil {
					return fmt.Errorf("content not created yet")
				}
				content, err := virtClient.VirtualMachineSnapshotContent(snapshot.Namespace).Get(
					context.Background(), *s.Status.VirtualMachineSnapshotContentName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				name = storagetypes.ScratchPVCName(content.UID)
				return nil
			}, 120*time.Second, 2*time.Second).Should(Succeed())
			return name
		}

		It("should snapshot a running multi-disk VM without stopping it", func() {
			vm, _ = createRunningVM(dataDisks)

			snapshot = externalSnapshotOf(vm.Name, vm.Namespace)
			_, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(
				context.Background(), snapshot, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			snapshot = libstorage.WaitSnapshotSucceeded(virtClient, vm.Namespace, snapshot.Name)
			Expect(snapshot.Status.CreationTime).ToNot(BeNil())

			By("the VM never left Running")
			Consistently(ThisVMIWith(vm.Namespace, vm.Name), 30*time.Second, 5*time.Second).
				Should(BeInPhase(v1.Running))

			By("the overlays are gone and the scratch volume with them")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 10*time.Minute, 5*time.Second).Should(
				HaveConditionFalseWithReason(v1.VirtualMachineInstanceOverlaySnapshotActive, v1.VirtualMachineInstanceReasonOverlayCommitted))
			Eventually(ThisPVCWith(vm.Namespace, scratchPVCName()), 5*time.Minute, 5*time.Second).Should(BeGone())
		})

		It("should keep the guest responsive while it is snapshotted under load", func() {
			var vmi *v1.VirtualMachineInstance
			vm, vmi = createRunningVM(dataDisks)

			Expect(console.LoginToFedora(vmi)).To(Succeed())
			By("writing continuously to a data disk")
			Expect(console.RunCommand(vmi,
				"sudo setsid -f sh -c 'for i in $(seq 8); do dd if=/dev/urandom of=/dev/disk/by-id/virtio-snapshot-data-disk-0 bs=1M oflag=direct; done' >/dev/null 2>&1",
				20*time.Second)).To(Succeed())

			snapshot = externalSnapshotOf(vm.Name, vm.Namespace)
			_, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(
				context.Background(), snapshot, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			snapshot = libstorage.WaitSnapshotSucceeded(virtClient, vm.Namespace, snapshot.Name)

			By("the guest still answers")
			Expect(console.RunCommand(vmi, "echo alive", 20*time.Second)).To(Succeed())
		})

		It("should commit active overlays and release scratch when the VM is stopped", func() {
			vm, _ = createRunningVM(2, libvmi.WithTerminationGracePeriod(180))
			payload, err := patch.New(patch.WithReplace("/spec/runStrategy", v1.RunStrategyManual)).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(
				context.Background(), vm.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			snapshot = externalSnapshotOf(vm.Name, vm.Namespace)
			_, err = virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(
				context.Background(), snapshot, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			scratch := scratchPVCName()

			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 5*time.Minute, 100*time.Millisecond).Should(Or(
				HaveConditionTrueWithReason(v1.VirtualMachineInstanceOverlaySnapshotActive, v1.VirtualMachineInstanceReasonOverlaysReady),
				HaveConditionTrueWithReason(v1.VirtualMachineInstanceOverlaySnapshotActive, v1.VirtualMachineInstanceReasonOverlayCommitting),
			))
			Expect(virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{})).To(Succeed())

			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 4*time.Minute, time.Second).Should(BeGone())
			Eventually(ThisPVCWith(vm.Namespace, scratch), 5*time.Minute, time.Second).Should(BeGone())
			Eventually(matcher.ThisVM(vm), time.Minute, time.Second).Should(And(
				HaveConditionMissingOrFalse(v1.VirtualMachineManualRecoveryRequired),
				HaveField("Status.SnapshotInProgress", BeNil()),
			))
		})

		It("should block startup and keep scratch after a launcher crash until the overlays are committed", func() {
			var vmi *v1.VirtualMachineInstance
			vm, vmi = createRunningVM(2)

			By("holding the capture so the overlays stay active")
			webhook := createDenyVolumeSnapshotCreateWebhook(virtClient, vm.Name)
			DeferCleanup(func() {
				err := virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(
					context.Background(), webhook.Name, metav1.DeleteOptions{})
				if !k8serrors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}
			})

			snapshot = externalSnapshotOf(vm.Name, vm.Namespace)
			_, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(
				context.Background(), snapshot, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			scratch := scratchPVCName()

			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 5*time.Minute, 2*time.Second).Should(
				HaveConditionTrueWithReason(v1.VirtualMachineInstanceOverlaySnapshotActive, v1.VirtualMachineInstanceReasonOverlaysReady))
			scratchPVC, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(
				context.Background(), scratch, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("killing virt-launcher without a graceful shutdown")
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())
			_, err = exec.ExecuteCommandOnPod(pod, "compute", []string{"pkill", "-9", "-x", "virt-launcher"})
			Expect(err).ToNot(HaveOccurred())

			By("the VM is blocked for manual recovery")
			Eventually(matcher.ThisVM(vm), 4*time.Minute, 2*time.Second).Should(
				matcher.HaveConditionTrueWithReason(v1.VirtualMachineManualRecoveryRequired, storagetypes.UncommittedOverlaysReason))

			By("an explicit start is refused")
			Expect(virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})).
				To(MatchError(ContainSubstring(scratch)))

			By("no replacement VMI starts and the scratch volume stays protected")
			Consistently(func(g Gomega) {
				current, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(
					context.Background(), vm.Name, metav1.GetOptions{})
				if err == nil {
					g.Expect(current.UID).To(Equal(vmi.UID))
				} else {
					g.Expect(k8serrors.IsNotFound(err)).To(BeTrue())
				}

				pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(
					context.Background(), scratch, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(pvc.UID).To(Equal(scratchPVC.UID))
				g.Expect(pvc.Finalizers).To(ContainElement(storagetypes.OverlayProtectionFinalizer))
			}, 30*time.Second, 2*time.Second).Should(Succeed())

			By("marking the overlays as committed releases scratch and starts the VM")
			payload, err := patch.New(patch.WithAdd(
				"/metadata/annotations/"+patch.EscapeJSONPointer(storagetypes.OverlaysCommittedAnnotation), string(scratchPVC.UID),
			)).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			_, err = virtClient.VirtualMachine(vm.Namespace).Patch(
				context.Background(), vm.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(ThisPVCWith(vm.Namespace, scratch), 5*time.Minute, 2*time.Second).Should(BeGone())
			Eventually(matcher.ThisVM(vm), 2*time.Minute, 2*time.Second).Should(And(
				HaveConditionMissingOrFalse(v1.VirtualMachineManualRecoveryRequired),
				HaveField("Annotations", Not(HaveKey(storagetypes.OverlaysCommittedAnnotation))),
			))
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 5*time.Minute, 2*time.Second).Should(BeInPhase(v1.Running))
		})

		It("should refuse to migrate a VM while its disks are on overlays", decorators.RequiresTwoSchedulableNodes, func() {
			vm, _ = createRunningVM(dataDisks)

			snapshot = externalSnapshotOf(vm.Name, vm.Namespace)
			_, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(
				context.Background(), snapshot, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the disks to be on overlays")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 5*time.Minute, 2*time.Second).Should(
				HaveConditionTrue(v1.VirtualMachineInstanceOverlaySnapshotActive))

			By("the VMI advertises that it cannot be migrated")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 2*time.Minute, 2*time.Second).Should(
				HaveConditionFalseWithReason(v1.VirtualMachineInstanceIsMigratable, v1.VirtualMachineInstanceReasonSnapshotOverlaysNotMigratable))
		})
	})
}))
