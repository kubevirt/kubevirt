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

package migration

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			)
			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

			migration := libmigration.New(vmi.Name, vmi.Namespace)
			setEvacuationAnnotation(migration)
			_ = libmigration.RunMigration(kubevirt.Client(), migration)

			Eventually(func() map[string]string {
				virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())
				return virtLauncherPod.GetAnnotations()
			}).WithTimeout(60 * time.Second).WithPolling(1 * time.Second).Should(HaveKeyWithValue("descheduler.alpha.kubernetes.io/eviction-in-progress", "kubevirt"))
		})

		Context("when evacuating fails", func() {
			var vmi *v1.VirtualMachineInstance
			BeforeEach(func() {
				vmi = libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithAnnotation(v1.FuncTestForceLauncherMigrationFailureAnnotation, ""),
				)
			})

			It("should remove eviction-in-progress annotation to source virt-launcher pod", func() {
				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				// Manually adding the eviction-in-progress annotation to the virt-launcher pod
				// to avoid flakiness between annotation addition and removal
				virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())
				patchBytes, err := patch.New(
					patch.WithAdd(fmt.Sprintf("/metadata/annotations/%s", patch.EscapeJSONPointer("descheduler.alpha.kubernetes.io/eviction-in-progress")), "kubevirt"),
				).GeneratePayload()
				Expect(err).NotTo(HaveOccurred())
				_, err = kubevirt.Client().CoreV1().Pods(virtLauncherPod.Namespace).Patch(context.Background(), virtLauncherPod.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
				Expect(err).NotTo(HaveOccurred())

				migration := libmigration.New(vmi.Name, vmi.Namespace)
				setEvacuationAnnotation(migration)
				_ = libmigration.RunMigrationAndExpectFailure(migration, libmigration.MigrationWaitTime)

				Eventually(func() map[string]string {
					virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
					Expect(err).NotTo(HaveOccurred())
					return virtLauncherPod.GetAnnotations()
				}).WithTimeout(60 * time.Second).WithPolling(1 * time.Second).ShouldNot(HaveKey("descheduler.alpha.kubernetes.io/eviction-in-progress"))
			})

			It("retrying immediately should be blocked by the migration backoff", func() {
				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

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
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

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
