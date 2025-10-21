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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

// getKubevirtControllerClient creates a client that impersonates the kubevirt-controller service account
// This is necessary because only kubevirt internal service accounts can modify VMI specs with utility volumes
func getKubevirtControllerClient(virtCli kubecli.KubevirtClient, namespace string) kubecli.KubevirtClient {
	config := virtCli.Config()

	// Create a new config that impersonates the kubevirt-controller service account
	impersonationConfig := rest.CopyConfig(config)
	impersonationConfig.Impersonate = rest.ImpersonationConfig{
		UserName: fmt.Sprintf("system:serviceaccount:%s:kubevirt-controller", namespace),
	}

	client, err := kubecli.GetKubevirtClientFromRESTConfig(impersonationConfig)
	Expect(err).ToNot(HaveOccurred())
	return client
}

var _ = Describe(SIG("Utility Volumes", func() {
	var (
		virtClient        kubecli.KubevirtClient
		controllerClient  kubecli.KubevirtClient
		testNamespace     string
		vmi               *v1.VirtualMachineInstance
		pvcName           string
		utilityVolumeName string
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		testNamespace = testsuite.GetTestNamespace(nil)

		// Create a client that impersonates the kubevirt-controller service account
		// which has the necessary privileges to patch VMI resources
		controllerClient = getKubevirtControllerClient(virtClient, flags.KubeVirtInstallNamespace)
	})

	addUtilityVolume := func(vmiName, namespace, utilityVolumeName, pvcName string) {
		utilityVolume := v1.UtilityVolume{
			Name: utilityVolumeName,
			PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
			},
		}

		Eventually(func() error {
			vmi, err := virtClient.VirtualMachineInstance(namespace).Get(context.Background(), vmiName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// Create a patch to add the utility volume
			patchSet := patch.New(
				patch.WithTest("/spec/utilityVolumes", vmi.Spec.UtilityVolumes),
			)

			newUtilityVolumes := append(vmi.Spec.UtilityVolumes, utilityVolume)
			if len(vmi.Spec.UtilityVolumes) > 0 {
				patchSet.AddOption(patch.WithReplace("/spec/utilityVolumes", newUtilityVolumes))
			} else {
				patchSet.AddOption(patch.WithAdd("/spec/utilityVolumes", newUtilityVolumes))
			}

			patchBytes, err := patchSet.GeneratePayload()
			if err != nil {
				return err
			}

			_, err = controllerClient.VirtualMachineInstance(namespace).Patch(context.Background(), vmiName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			return err
		}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	removeUtilityVolume := func(vmiName, namespace string) {
		Eventually(func() error {
			patchSet := patch.New(
				patch.WithRemove("/spec/utilityVolumes"),
			)
			patchBytes, err := patchSet.GeneratePayload()
			if err != nil {
				return err
			}

			_, err = controllerClient.VirtualMachineInstance(namespace).Patch(context.Background(), vmiName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			return err
		}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	Context("Basic utility volume hotplug", func() {
		BeforeEach(func() {
			pvcName = "test-utility-volume-pvc" + rand.String(5)
			utilityVolumeName = "test-utility-volume"

			vm := libvmi.NewVirtualMachine(libvmifact.NewCirros(), libvmi.WithRunStrategy(v1.RunStrategyAlways))
			vm, err := virtClient.VirtualMachine(testsuite.NamespaceTestDefault).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())

			vmi, err = virtClient.VirtualMachineInstance(testNamespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should successfully hotplug and unhotplug a utility volume", func() {
			libstorage.CreateFSPVC(pvcName, testNamespace, "500Mi", libstorage.WithStorageProfile())
			addUtilityVolume(vmi.Name, testNamespace, utilityVolumeName, pvcName)
			verifyUtilityVolumeInVMISpec(virtClient, vmi, utilityVolumeName)
			libstorage.VerifyVolumeStatus(virtClient, vmi, v1.HotplugVolumeMounted, "", false, utilityVolumeName)
			vmi, err := virtClient.VirtualMachineInstance(testNamespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			attachmentPodName := libstorage.AttachmentPodName(vmi)
			Expect(attachmentPodName).ToNot(BeEmpty())
			removeUtilityVolume(vmi.Name, testNamespace)
			verifyUtilityVolumeRemovedFromVMI(virtClient, vmi, utilityVolumeName)
			Eventually(matcher.ThisPodWith(vmi.Namespace, attachmentPodName), 90*time.Second, 1*time.Second).Should(matcher.BeGone())
		})
	})

	Context("Migration with utility volumes", decorators.RequiresTwoSchedulableNodes, func() {
		BeforeEach(func() {
			pvcName = "test-utility-migration-pvc" + rand.String(5)
			utilityVolumeName = "test-utility-migration"

			// Create a VM with masquerade networking to make it migratable
			vmiSpec := libvmifact.NewCirros()
			vmiSpec.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					Masquerade: &v1.InterfaceMasquerade{},
				},
			}}
			vmiSpec.Spec.Networks = []v1.Network{{
				Name: "default",
				NetworkSource: v1.NetworkSource{
					Pod: &v1.PodNetwork{},
				},
			}}

			vm := libvmi.NewVirtualMachine(vmiSpec, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			vm, err := virtClient.VirtualMachine(testsuite.NamespaceTestDefault).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())

			vmi, err = virtClient.VirtualMachineInstance(testNamespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should wait utility volumes detach before scheduling migration", func() {
			sourceNode := vmi.Status.NodeName

			libstorage.CreateFSPVC(pvcName, testNamespace, "500Mi", libstorage.WithStorageProfile())
			addUtilityVolume(vmi.Name, testNamespace, utilityVolumeName, pvcName)
			verifyUtilityVolumeInVMISpec(virtClient, vmi, utilityVolumeName)
			libstorage.VerifyVolumeStatus(virtClient, vmi, v1.HotplugVolumeMounted, "", false, utilityVolumeName)

			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "migration creation should succeed")

			Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
				migration, err = virtClient.VirtualMachineInstanceMigration(testNamespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return migration.Status.Phase
			}, 30*time.Second, 1*time.Second).Should(Equal(v1.MigrationPending))

			// Verify condition is set to indicate utility volumes are blocking
			Eventually(func() bool {
				migration, err = virtClient.VirtualMachineInstanceMigration(testNamespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, condition := range migration.Status.Conditions {
					if condition.Type == v1.VirtualMachineInstanceMigrationBlockedByUtilityVolumes &&
						condition.Status == k8sv1.ConditionTrue {
						return true
					}
				}
				return false
			}, 30*time.Second, 1*time.Second).Should(BeTrue(), "Should have condition indicating utility volumes are blocking")

			events.ExpectEvent(migration, k8sv1.EventTypeWarning, controller.UtilityVolumeMigrationPendingReason)

			// Remove utility volume to allow migration
			removeUtilityVolume(vmi.Name, testNamespace)
			verifyUtilityVolumeRemovedFromVMI(virtClient, vmi, utilityVolumeName)

			// Verify condition is removed after utility volumes are detached
			Eventually(func() bool {
				migration, err = virtClient.VirtualMachineInstanceMigration(testNamespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, condition := range migration.Status.Conditions {
					if condition.Type == v1.VirtualMachineInstanceMigrationBlockedByUtilityVolumes {
						return false
					}
				}
				return true
			}, 30*time.Second, 1*time.Second).Should(BeTrue(), "Utility volumes condition should be removed after volumes are detached")

			// Wait for migration to succeed
			Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
				migration, err := virtClient.VirtualMachineInstanceMigration(testNamespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return migration.Status.Phase
			}, 240*time.Second, 1*time.Second).Should(Equal(v1.MigrationSucceeded))

			// Verify VM migrated to a different node
			vmi, err = virtClient.VirtualMachineInstance(testNamespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			targetNode := vmi.Status.NodeName
			Expect(targetNode).ToNot(Equal(sourceNode))

			// Verify no attachment pods were created for utility volumes on the target node
			pods, err := virtClient.CoreV1().Pods(testNamespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("kubevirt.io/created-by=%s", vmi.UID),
			})
			Expect(err).ToNot(HaveOccurred())

			for _, pod := range pods.Items {
				if pod.Spec.NodeName == targetNode {
					// Should only be the virt-launcher pod
					Expect(pod.Labels).To(HaveKey("kubevirt.io"))
					Expect(pod.Labels).ToNot(HaveKey("kubevirt.io/domain"))
					// Verify it's not an attachment pod
					for _, volume := range pod.Spec.Volumes {
						Expect(volume.Name).ToNot(Equal(utilityVolumeName))
					}
				}
			}
		})
	})
}))

func verifyUtilityVolumeInVMISpec(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, utilityVolumeName string) {
	Eventually(func() error {
		updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		found := false
		for _, utilityVolume := range updatedVMI.Spec.UtilityVolumes {
			if utilityVolume.Name == utilityVolumeName {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("utility volume %s not found in VMI spec", utilityVolumeName)
		}

		return nil
	}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}

func verifyUtilityVolumeRemovedFromVMI(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, utilityVolumeName string) {
	Eventually(func() bool {
		currentVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		for _, utilityVolume := range currentVMI.Spec.UtilityVolumes {
			if utilityVolume.Name == utilityVolumeName {
				return false
			}
		}
		return true
	}, 30*time.Second, 1*time.Second).Should(BeTrue())

	Eventually(func() bool {
		currentVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		for _, volumeStatus := range currentVMI.Status.VolumeStatus {
			if volumeStatus.Name == utilityVolumeName {
				return false
			}
		}
		return true
	}, 30*time.Second, 1*time.Second).Should(BeTrue())
}
