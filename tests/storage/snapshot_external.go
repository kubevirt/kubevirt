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

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

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
			if sc == "" {
				Skip("No snapshot storage class available")
			}
			snapshotStorageClass = sc
		})

		// Enough disks that the commit is a sequence rather than an instant,
		// which is what the mid-commit cases need to land inside.
		const dataDisks = 6

		createRunningVM := func(disks int) (*v1.VirtualMachine, *v1.VirtualMachineInstance) {
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

			created := libvmi.NewVirtualMachine(libvmifact.NewFedora(vmiOpts...), vmOpts...)
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

		overlaysLeft := func(vmi *v1.VirtualMachineInstance, scratch string) int {
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			if err != nil {
				return -1
			}
			out, err := exec.ExecuteCommandOnPod(pod, "compute",
				[]string{"/bin/sh", "-c", fmt.Sprintf("ls /var/run/kubevirt/hotplug-disks/%s/ 2>/dev/null | grep -c qcow2 || true", scratch)})
			if err != nil {
				return -1
			}
			var count int
			if _, err := fmt.Sscanf(strings.TrimSpace(out), "%d", &count); err != nil {
				return -1
			}
			return count
		}

		killLauncher := func(vmi *v1.VirtualMachineInstance) {
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(virtClient.CoreV1().Pods(vmi.Namespace).Delete(
				context.Background(), pod.Name,
				metav1.DeleteOptions{GracePeriodSeconds: pointer.P(int64(0))},
			)).To(Succeed())
		}

		overlayCondition := func(namespace, name string) func() *v1.VirtualMachineInstanceCondition {
			return func() *v1.VirtualMachineInstanceCondition {
				vmi, err := virtClient.VirtualMachineInstance(namespace).Get(
					context.Background(), name, metav1.GetOptions{})
				if err != nil {
					return nil
				}
				for i := range vmi.Status.Conditions {
					if vmi.Status.Conditions[i].Type == v1.VirtualMachineInstanceOverlaySnapshotActive {
						return &vmi.Status.Conditions[i]
					}
				}
				return nil
			}
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
			Eventually(overlayCondition(vm.Namespace, vm.Name), 10*time.Minute, 5*time.Second).Should(And(Not(BeNil()), HaveField("Status", k8sv1.ConditionFalse), HaveField("Reason", v1.VirtualMachineInstanceReasonOverlayCommitted)))
			Eventually(func() bool {
				_, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(
					context.Background(), scratchPVCName(), metav1.GetOptions{})
				return k8serrors.IsNotFound(err)
			}, 5*time.Minute, 5*time.Second).Should(BeTrue())
		})

		It("should keep the guest responsive while it is snapshotted under load", func() {
			var vmi *v1.VirtualMachineInstance
			vm, vmi = createRunningVM(dataDisks)

			Expect(console.LoginToFedora(vmi)).To(Succeed())
			By("writing continuously in the guest")
			Expect(console.RunCommand(vmi,
				"nohup dd if=/dev/urandom of=/tmp/load bs=1M count=4096 oflag=direct >/dev/null 2>&1 &", 20*time.Second)).
				To(Succeed())

			snapshot = externalSnapshotOf(vm.Name, vm.Namespace)
			_, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(
				context.Background(), snapshot, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			snapshot = libstorage.WaitSnapshotSucceeded(virtClient, vm.Namespace, snapshot.Name)

			By("the guest still answers")
			Expect(console.RunCommand(vmi, "echo alive", 20*time.Second)).To(Succeed())
		})

		It("should refuse to migrate a VM while its disks are on overlays", decorators.RequiresTwoSchedulableNodes, func() {
			vm, _ = createRunningVM(dataDisks)

			snapshot = externalSnapshotOf(vm.Name, vm.Namespace)
			_, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(
				context.Background(), snapshot, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the disks to be on overlays")
			Eventually(overlayCondition(vm.Namespace, vm.Name), 5*time.Minute, 2*time.Second).
				ShouldNot(BeNil())

			By("the VMI advertises that it cannot be migrated")
			Eventually(func() string {
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(
					context.Background(), vm.Name, metav1.GetOptions{})
				if err != nil {
					return ""
				}
				for _, c := range vmi.Status.Conditions {
					if c.Type == v1.VirtualMachineInstanceIsMigratable && c.Status == k8sv1.ConditionFalse {
						return c.Reason
					}
				}
				return ""
			}, 2*time.Minute, 2*time.Second).Should(Equal(v1.VirtualMachineInstanceReasonSnapshotOverlaysNotMigratable))
		})

		writeDiskMarkers := func(vmi *v1.VirtualMachineInstance) {
			Expect(console.LoginToFedora(vmi)).To(Succeed())
			for i := 0; i < dataDisks; i++ {
				Expect(console.RunCommand(vmi, fmt.Sprintf("printf 'snapshot-disk-%d-after-overlay' | dd of=/dev/disk/by-id/virtio-snapshot-data-disk-%d bs=512 count=1 conv=notrunc,fsync", i, i), 20*time.Second)).To(Succeed())
			}
		}
		verifyRecoveredDiskMarkers := func(original *v1.VirtualMachineInstance) {
			Eventually(func() bool {
				current, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				return err == nil && current.UID != original.UID && current.IsRunning()
			}, 6*time.Minute, 2*time.Second).Should(BeTrue())
			recovered, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(console.LoginToFedora(recovered)).To(Succeed())
			for i := 0; i < dataDisks; i++ {
				marker := fmt.Sprintf("snapshot-disk-%d-after-overlay", i)
				Expect(console.RunCommand(recovered, fmt.Sprintf("test \"$(head -c %d /dev/disk/by-id/virtio-snapshot-data-disk-%d)\" = '%s'", len(marker), i, marker), 20*time.Second)).To(Succeed())
			}
		}

		Context("crash recovery", Serial, decorators.RequiresSnapshotStorageClass, func() {
			DescribeTable("should preserve disk writes across a launcher crash", func(duringCommit bool) {
				var vmi *v1.VirtualMachineInstance
				vm, vmi = createRunningVM(dataDisks)

				snapshot = externalSnapshotOf(vm.Name, vm.Namespace)
				_, err := virtClient.VirtualMachineSnapshot(snapshot.Namespace).Create(
					context.Background(), snapshot, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				scratch := scratchPVCName()
				By("waiting until every disk is on an overlay")
				Eventually(func() int { return overlaysLeft(vmi, scratch) },
					5*time.Minute, 2*time.Second).Should(Equal(dataDisks))
				writeDiskMarkers(vmi)

				if duringCommit {
					// Each pivot unlinks its overlay: a partial count identifies the
					// window where recovery must combine bases and remaining overlays.
					By("waiting until some disks have committed")
					Eventually(func() bool {
						left := overlaysLeft(vmi, scratch)
						return left > 0 && left < dataDisks
					}, 10*time.Minute, time.Second).Should(BeTrue(),
						"the commit never reported a partial state to interrupt")
				}

				By("killing the launcher and verifying every disk marker after recovery")
				killLauncher(vmi)
				verifyRecoveredDiskMarkers(vmi)
				Eventually(overlayCondition(vm.Namespace, vm.Name), 10*time.Minute, 5*time.Second).
					Should(And(Not(BeNil()), HaveField("Status", k8sv1.ConditionFalse), HaveField("Reason", v1.VirtualMachineInstanceReasonOverlayCommitted)))

				By("the scratch volume is released")
				Eventually(func() bool {
					_, err := virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(
						context.Background(), scratch, metav1.GetOptions{})
					return k8serrors.IsNotFound(err)
				}, 5*time.Minute, 5*time.Second).Should(BeTrue())
			},
				Entry("before commit", false),
				Entry("during a partial commit", true),
			)
		})
	})
}))
