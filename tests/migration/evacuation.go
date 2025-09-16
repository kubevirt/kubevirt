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
 * Copyright The KubeVirt Authors
 *
 */

package migration

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe(SIG("VM Live Migration triggered by evacuation", decorators.RequiresTwoSchedulableNodes, func() {
	Context("during evacuation", func() {
		It("should add eviction-in-progress annotation to source virt-launcher pod", func() {
			vmi := libvmifact.NewCirros(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
			)
			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

			virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())

			By("Triggering an eviction by evict API")
			ctx := context.Background()
			err = kubevirt.Client().CoreV1().Pods(virtLauncherPod.Namespace).EvictV1(ctx, &policy.Eviction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      virtLauncherPod.Name,
					Namespace: virtLauncherPod.Namespace,
				},
			})
			Expect(err).To(MatchError(ContainSubstring("Eviction triggered evacuation of VMI")))

			By("Waiting for the eviction-in-progress annotation to be added to the source pod")
			Eventually(func() map[string]string {
				virtLauncherPod, err = kubevirt.Client().CoreV1().Pods(virtLauncherPod.Namespace).Get(ctx, virtLauncherPod.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return virtLauncherPod.GetAnnotations()
			}).WithTimeout(20 * time.Second).WithPolling(1 * time.Second).Should(HaveKey("descheduler.alpha.kubernetes.io/eviction-in-progress"))

			By("Waiting for a migration to be scheduled and to succeed")
			labelSelector, err := labels.Parse(fmt.Sprintf("%s in (%s)", v1.MigrationSelectorLabel, vmi.Name))
			Expect(err).ToNot(HaveOccurred())
			listOptions := metav1.ListOptions{
				LabelSelector: labelSelector.String(),
			}

			Eventually(func(g Gomega) {
				migrations, err := kubevirt.Client().VirtualMachineInstanceMigration(vmi.Namespace).List(ctx, listOptions)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(migrations.Items).To(HaveLen(1))
				g.Expect(migrations.Items[0].Status.Phase).To(Equal(v1.MigrationSucceeded))
			}).WithTimeout(60 * time.Second).WithPolling(1 * time.Second).Should(Succeed())

			By("Ensuring for the eviction-in-progress annotation is not present on the final pod")
			vmi, err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			targetVirtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(targetVirtLauncherPod.Name).ToNot(Equal(virtLauncherPod.Name))
			Expect(targetVirtLauncherPod.GetAnnotations()).ToNot(HaveKey("descheduler.alpha.kubernetes.io/eviction-in-progress"))
		})

		Context("when evacuating fails", func() {
			var vmi *v1.VirtualMachineInstance
			BeforeEach(func() {
				vmi = libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithAnnotation(v1.FuncTestForceLauncherMigrationFailureAnnotation, ""),
					libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
				)
			})

			It("should not remove eviction-in-progress annotation from source virt-launcher pod", func() {
				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

				virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())

				By("Triggering an eviction by evict API")
				ctx := context.Background()
				err = kubevirt.Client().CoreV1().Pods(virtLauncherPod.Namespace).EvictV1(ctx, &policy.Eviction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      virtLauncherPod.Name,
						Namespace: virtLauncherPod.Namespace,
					},
				})
				Expect(err).To(MatchError(ContainSubstring("Eviction triggered evacuation of VMI")))

				By("Waiting for a migration to be scheduled and fail")
				labelSelector, err := labels.Parse(fmt.Sprintf("%s in (%s)", v1.MigrationSelectorLabel, vmi.Name))
				Expect(err).ToNot(HaveOccurred())
				listOptions := metav1.ListOptions{
					LabelSelector: labelSelector.String(),
				}
				Eventually(func(g Gomega) {
					migrations, err := kubevirt.Client().VirtualMachineInstanceMigration(vmi.Namespace).List(ctx, listOptions)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(migrations.Items).ToNot(BeEmpty())
					for _, migration := range migrations.Items {
						g.Expect(migration.Status.Phase).To(Equal(v1.MigrationFailed))
					}
				}).WithTimeout(60 * time.Second).WithPolling(1 * time.Second).Should(Succeed())

				By("Ensuring eviction-in-progress annotation is not removed from the source pod")
				Consistently(func() map[string]string {
					virtLauncherPod, err = kubevirt.Client().CoreV1().Pods(virtLauncherPod.Namespace).Get(ctx, virtLauncherPod.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					return virtLauncherPod.GetAnnotations()
				}).WithTimeout(30 * time.Second).WithPolling(1 * time.Second).Should(HaveKey("descheduler.alpha.kubernetes.io/eviction-in-progress"))

				By("Ensuring eviction-in-progress annotation is not set on the target pod")
				vmi, err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.MigrationState).ToNot(BeNil())
				Expect(vmi.Status.MigrationState.TargetPod).ToNot(Equal(""))
				targetPod, err := kubevirt.Client().CoreV1().Pods(vmi.Namespace).Get(ctx, vmi.Status.MigrationState.TargetPod, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(targetPod.GetAnnotations()).ToNot(HaveKey("descheduler.alpha.kubernetes.io/eviction-in-progress"))
			})

			It("retrying immediately should be blocked by the migration backoff", func() {
				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

				By("Waiting for the migration to fail")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				setEvacuationAnnotation(migration)
				_ = libmigration.RunMigrationAndExpectFailure(migration, libmigration.MigrationWaitTime)

				By("Try again")
				migration = libmigration.New(vmi.Name, vmi.Namespace)
				setEvacuationAnnotation(migration)
				_ = libmigration.RunMigrationAndExpectFailure(migration, libmigration.MigrationWaitTime)

				events.ExpectEvent(vmi, k8sv1.EventTypeWarning, controller.MigrationBackoffReason)
			})

			It("after a successful migration backoff should be cleared", func() {
				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsHuge)

				By("Waiting for the migration to fail")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				setEvacuationAnnotation(migration)
				_ = libmigration.RunMigrationAndExpectFailure(migration, libmigration.MigrationWaitTime)

				By("Patch VMI")
				patchBytes := []byte(fmt.Sprintf(`[{"op": "remove", "path": "/metadata/annotations/%s"}]`, patch.EscapeJSONPointer(v1.FuncTestForceLauncherMigrationFailureAnnotation)))
				_, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Try again with backoff")
				migration = libmigration.New(vmi.Name, vmi.Namespace)
				setEvacuationAnnotation(migration)
				_ = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

				// Intentionally modifying history
				events.DeleteEvents(vmi, k8sv1.EventTypeWarning, controller.MigrationBackoffReason)

				By("There should be no backoff now")
				migration = libmigration.New(vmi.Name, vmi.Namespace)
				setEvacuationAnnotation(migration)
				_ = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

				By("Checking that no backoff event occurred")
				events.ExpectNoEvent(vmi, k8sv1.EventTypeWarning, controller.MigrationBackoffReason)
			})
		})
	})
}))

func setEvacuationAnnotation(migrations ...*v1.VirtualMachineInstanceMigration) {
	for _, m := range migrations {
		m.Annotations = map[string]string{
			v1.EvacuationMigrationAnnotation: m.Name,
		}
	}
}
