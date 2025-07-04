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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Live Migration across namespaces", Serial, decorators.RequiresDecentralizedLiveMigration, func() {
	var (
		virtClient         kubecli.KubevirtClient
		migrationID        string
		connectionURL      string
		err                error
		featureGateEnabled bool
	)

	BeforeEach(func() {
		featureGateEnabled = checks.HasFeature(featuregate.DecentralizedLiveMigration)
		if !featureGateEnabled {
			kvconfig.EnableFeatureGate(featuregate.DecentralizedLiveMigration)
		}
		if !libstorage.HasCDI() {
			Fail("Fail DataVolume tests when CDI is not present")
		}
		virtClient = kubevirt.Client()
		migrationID = fmt.Sprintf("mig-%s", rand.String(5))
		connectionURL, err = getKubevirtSynchronizationSyncAddress(virtClient)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if !featureGateEnabled {
			kvconfig.DisableFeatureGate(featuregate.DecentralizedLiveMigration)
		}
	})

	createAndStartVMFromVMISpec := func(vmi *virtv1.VirtualMachineInstance) *virtv1.VirtualMachine {
		vm := libvmi.NewVirtualMachine(vmi)
		vm, err := virtClient.VirtualMachine(vmi.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Starting the VM")
		vm = libvmops.StartVirtualMachine(vm)
		vmi = libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Running})
		_, err = libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).NotTo(HaveOccurred())

		return vm
	}

	createReceiverVMFromVMISpec := func(vmi *virtv1.VirtualMachineInstance) *virtv1.VirtualMachine {
		vm := libvmi.NewVirtualMachine(vmi,
			libvmi.WithRunStrategy(virtv1.RunStrategyWaitAsReceiver),
		)
		By(fmt.Sprintf("creating VM %s/%s", vmi.Namespace, vmi.Name))
		vm, err := virtClient.VirtualMachine(vmi.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Checking the VMI exists in receiving phase")
		Eventually(func() virtv1.VirtualMachineInstancePhase {
			receiver, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if err != nil {
				return ""
			}
			return receiver.Status.Phase
		}, 30*time.Second, 1*time.Second).Should(Equal(virtv1.WaitingForSync))

		return vm
	}

	deleteMigration := func(migration *virtv1.VirtualMachineInstanceMigration) error {
		err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Delete(context.Background(), migration.Name, metav1.DeleteOptions{})
		if k8serrors.IsNotFound(err) {
			return nil
		}
		// Verify migration is gone
		Eventually(func() *virtv1.VirtualMachineInstanceMigration {
			migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
			if k8serrors.IsNotFound(err) {
				return nil
			}
			return migration
		}, 30*time.Second, 1*time.Second).Should(BeNil())
		return nil
	}

	deleteVM := func(vm *v1.VirtualMachine) {
		err := virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
		if k8serrors.IsNotFound(err) {
			return
		}
		Expect(err).ToNot(HaveOccurred())
		// Verify VM is gone
		Eventually(func() *virtv1.VirtualMachine {
			vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if k8serrors.IsNotFound(err) {
				return nil
			}
			return vm
		}, 30*time.Second, 1*time.Second).Should(BeNil())
	}

	deleteDV := func(dv *cdiv1.DataVolume) {
		err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Delete(context.Background(), dv.Name, metav1.DeleteOptions{})
		if k8serrors.IsNotFound(err) {
			return
		}
		Expect(err).ToNot(HaveOccurred())
		// Verify DV is gone
		Eventually(func() *cdiv1.DataVolume {
			dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
			if k8serrors.IsNotFound(err) {
				return nil
			}
			return dv
		}, 30*time.Second, 1*time.Second).Should(BeNil())
	}

	updateRunStrategy := func(vm *virtv1.VirtualMachine, strategy *virtv1.VirtualMachineRunStrategy) {
		Eventually(func() error {
			vm.Spec.RunStrategy = strategy
			_, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{})
			if err != nil {
				// Ignore the error from the get.
				vm, _ = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			}
			return err
		}, 60*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	Context("container disk", func() {
		var (
			sourceVMI, targetVMI *virtv1.VirtualMachineInstance
			sourceVM, targetVM   *virtv1.VirtualMachine
		)

		It("[QUARANTINE] should live migrate a container disk vm, several times", decorators.Quarantine, func() {
			sourceVMI = libvmifact.NewCirros(
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			targetVMI = sourceVMI.DeepCopy()
			targetVMI.Namespace = testsuite.NamespaceTestAlternative
			sourceVM = createAndStartVMFromVMISpec(sourceVMI)
			num := 4
			for i := 0; i < num; i++ {
				var sourceMigration, targetMigration *virtv1.VirtualMachineInstanceMigration
				var expectedVMI *virtv1.VirtualMachineInstance
				sourceRunStrategy := sourceVM.Spec.RunStrategy
				By(fmt.Sprintf("executing a migration, and waiting for finalized state, run %d", i))
				if i%2 == 0 {
					// source -> target
					targetVM = createReceiverVMFromVMISpec(targetVMI)
					sourceMigration = libmigration.NewSource(sourceVMI.Name, sourceVMI.Namespace, migrationID, connectionURL)
					targetMigration = libmigration.NewTarget(targetVMI.Name, targetVMI.Namespace, migrationID)
					expectedVMI = targetVMI
				} else {
					// target -> source
					targetVM = createReceiverVMFromVMISpec(sourceVMI)
					sourceMigration = libmigration.NewSource(targetVMI.Name, targetVMI.Namespace, migrationID, connectionURL)
					targetMigration = libmigration.NewTarget(sourceVMI.Name, sourceVMI.Namespace, migrationID)
					expectedVMI = sourceVMI
				}
				sourceMigration, targetMigration = libmigration.RunDecentralizedMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, sourceMigration, targetMigration)
				libmigration.ConfirmVMIPostMigration(virtClient, expectedVMI, targetMigration)
				updateRunStrategy(targetVM, sourceRunStrategy)
				err = deleteMigration(sourceMigration)
				Expect(err).ToNot(HaveOccurred())
				err = deleteMigration(targetMigration)
				Expect(err).ToNot(HaveOccurred())
				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToCirros(expectedVMI)).To(Succeed())

				By(fmt.Sprintf("deleting source VM %s/%s", sourceVM.Namespace, sourceVM.Name))
				deleteVM(sourceVM)
				sourceVM = targetVM
			}
		})

		It("should live migrate a container disk vm, with an additional PVC mounted, should stay mounted after migration", func() {
			sourceDV := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithStorage(),
			)

			sourceVMI = libvmifact.NewCirros(
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithDataVolume("disk1", sourceDV.Name),
			)
			targetVMI = sourceVMI.DeepCopy()
			targetVMI.Namespace = testsuite.NamespaceTestAlternative
			targetDV := sourceDV.DeepCopy()
			targetDV.Namespace = targetVMI.Namespace
			sourceDV, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(sourceDV)).Create(context.Background(), sourceDV, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(sourceDV, 240, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))

			sourceVM = createAndStartVMFromVMISpec(sourceVMI)
			deviceName := ""
			Eventually(func() string {
				sourceVMI, err := virtClient.VirtualMachineInstance(sourceVMI.Namespace).Get(context.Background(), sourceVMI.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, v := range sourceVMI.Status.VolumeStatus {
					if v.Name == "disk1" {
						deviceName = v.Target
						return v.Target
					}
				}
				return ""
			}).WithTimeout(time.Minute).WithPolling(2 * time.Second).ShouldNot(BeEmpty())

			for _, volume := range sourceVMI.Status.VolumeStatus {
				if volume.Name == "disk1" {
					deviceName = volume.Target
				}
			}
			By("Writing data to extra disk")
			Expect(console.LoginToCirros(sourceVMI)).To(Succeed())
			Expect(console.RunCommand(sourceVMI, fmt.Sprintf("sudo mkfs.ext4 /dev/%s", deviceName), 30*time.Second)).To(Succeed())
			Expect(console.RunCommand(sourceVMI, "mkdir test", 30*time.Second)).To(Succeed())
			Expect(console.RunCommand(sourceVMI, fmt.Sprintf("sudo mount -t ext4 /dev/%s /home/cirros/test", deviceName), 30*time.Second)).To(Succeed())
			Expect(console.RunCommand(sourceVMI, "sudo chmod 777 /home/cirros/test", 30*time.Second)).To(Succeed())
			Expect(console.RunCommand(sourceVMI, "sudo chown cirros:cirros /home/cirros/test", 30*time.Second)).To(Succeed())
			Expect(console.RunCommand(sourceVMI, "printf 'important data' &> /home/cirros/test/data.txt", 30*time.Second)).To(Succeed())

			targetDV, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(targetDV)).Create(context.Background(), targetDV, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(targetDV, 240, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))

			targetVM = createReceiverVMFromVMISpec(targetVMI)
			sourceMigration := libmigration.NewSource(sourceVMI.Name, sourceVMI.Namespace, migrationID, connectionURL)
			targetMigration := libmigration.NewTarget(targetVMI.Name, targetVMI.Namespace, migrationID)
			sourceMigration, targetMigration = libmigration.RunDecentralizedMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, sourceMigration, targetMigration)
			libmigration.ConfirmVMIPostMigration(virtClient, targetVMI, targetMigration)
			By("Verifying data on extra disk")
			Eventually(func() string {
				targetVMI, err := virtClient.VirtualMachineInstance(targetVMI.Namespace).Get(context.Background(), targetVMI.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, v := range targetVMI.Status.VolumeStatus {
					if v.Name == "disk1" {
						deviceName = v.Target
						return v.Target
					}
				}
				return ""
			}).WithTimeout(time.Minute).WithPolling(2 * time.Second).ShouldNot(BeEmpty())
			Expect(console.LoginToCirros(targetVMI)).To(Succeed())
			Expect(console.RunCommand(targetVMI, "cat /home/cirros/test/data.txt", 30*time.Second)).To(Succeed())
		})
	})

	Context("datavolume disk", func() {
		var (
			sourceVMI, targetVMI *virtv1.VirtualMachineInstance
			sourceVM, targetVM   *virtv1.VirtualMachine
		)

		createBlankFromName := func(name, namespace string) *cdiv1.DataVolume {
			targetDV := libdv.NewDataVolume(
				libdv.WithName(name),
				libdv.WithBlankImageSource(),
				libdv.WithStorage(
					libdv.StorageWithVolumeSize(cd.AlpineVolumeSize),
				),
			)
			targetDV, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), targetDV, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(targetDV, 240, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))
			return targetDV
		}

		It("[QUARANTINE] should live migration regular disk several times", decorators.Quarantine, func() {
			sourceDV := libdv.NewDataVolume(
				libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
				libdv.WithStorage(
					libdv.StorageWithVolumeSize(cd.AlpineVolumeSize),
				),
			)
			sourceDV, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(sourceDV)).Create(context.Background(), sourceDV, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(sourceDV, 240, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))
			sourceVMI = libvmi.New(
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithDataVolume("disk0", sourceDV.Name),
				libvmi.WithResourceMemory("128Mi"),
			)
			targetVMI = sourceVMI.DeepCopy()
			targetVMI.Namespace = testsuite.NamespaceTestAlternative

			sourceVM = createAndStartVMFromVMISpec(sourceVMI)
			Expect(sourceVM).ToNot(BeNil())
			Expect(console.LoginToAlpine(sourceVMI)).To(Succeed())
			var targetDV *cdiv1.DataVolume
			num := 4
			for i := 0; i < num; i++ {
				var sourceMigration, targetMigration *virtv1.VirtualMachineInstanceMigration
				var expectedVMI *virtv1.VirtualMachineInstance
				sourceRunStrategy := sourceVM.Spec.RunStrategy
				By(fmt.Sprintf("executing a migration, and waiting for finalized state, run %d", i))
				if i%2 == 0 {
					// source -> target
					targetDV = createBlankFromName(sourceDV.Name, testsuite.NamespaceTestAlternative)
					targetVM = createReceiverVMFromVMISpec(targetVMI)
					time.Sleep(time.Minute)
					sourceMigration = libmigration.NewSource(sourceVMI.Name, sourceVMI.Namespace, migrationID, connectionURL)
					targetMigration = libmigration.NewTarget(targetVMI.Name, targetVMI.Namespace, migrationID)
					expectedVMI = targetVMI
				} else {
					// target -> source
					targetDV = createBlankFromName(sourceDV.Name, testsuite.NamespaceTestDefault)
					targetVM = createReceiverVMFromVMISpec(sourceVMI)
					sourceMigration = libmigration.NewSource(targetVMI.Name, targetVMI.Namespace, migrationID, connectionURL)
					targetMigration = libmigration.NewTarget(sourceVMI.Name, sourceVMI.Namespace, migrationID)
					expectedVMI = sourceVMI
				}
				sourceMigration, targetMigration = libmigration.RunDecentralizedMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, sourceMigration, targetMigration)
				libmigration.ConfirmVMIPostMigration(virtClient, expectedVMI, targetMigration)
				Expect(console.LoginToAlpine(expectedVMI)).To(Succeed())
				By("ensuring the runStrategy is properly updated to be what the source was")
				updateRunStrategy(targetVM, sourceRunStrategy)
				By("cleaning up migration resources")
				err = deleteMigration(sourceMigration)
				Expect(err).ToNot(HaveOccurred())
				err = deleteMigration(targetMigration)
				Expect(err).ToNot(HaveOccurred())
				By("checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(expectedVMI)).To(Succeed())

				By(fmt.Sprintf("deleting source VM %s/%s", sourceVM.Namespace, sourceVM.Name))
				deleteVM(sourceVM)
				sourceVM = targetVM
				By(fmt.Sprintf("deleting source DV %s/%s", sourceDV.Namespace, sourceDV.Name))
				deleteDV(sourceDV)
				sourceDV = targetDV
			}
		})
	})
}))

func getKubevirtSynchronizationSyncAddress(virtClient kubecli.KubevirtClient) (string, error) {
	kv := libkubevirt.GetCurrentKv(virtClient)
	if kv == nil {
		return "", fmt.Errorf("unable to retrieve kubevirt CR")
	}
	if kv.Status.SynchronizationAddresses == nil {
		return "", fmt.Errorf("sync address not found")
	}
	return kv.Status.SynchronizationAddresses[0], nil
}
