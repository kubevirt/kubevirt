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

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Live Migration across namespaces", decorators.RequiresDecentralizedLiveMigration, func() {
	var (
		virtClient    kubecli.KubevirtClient
		connectionURL string
		err           error
	)

	BeforeEach(func() {
		if !libstorage.HasCDI() {
			Fail("Fail DataVolume tests when CDI is not present")
		}
		virtClient = kubevirt.Client()
		connectionURL, err = getKubevirtSynchronizationSyncAddress(virtClient)
		Expect(err).ToNot(HaveOccurred())
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
			libvmi.WithAnnotations(map[string]string{
				virtv1.RestoreRunStrategy: string(virtv1.RunStrategyAlways),
			}),
		)
		By(fmt.Sprintf("creating receiverVM %s/%s", vmi.Namespace, vmi.Name))
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
		By(fmt.Sprintf("Verifying VM %s/%s is stopped before deletion", vm.Namespace, vm.Name))
		Eventually(func() virtv1.VirtualMachineRunStrategy {
			vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			if k8serrors.IsNotFound(err) {
				return virtv1.RunStrategyHalted
			}
			if vm.Spec.RunStrategy == nil {
				return virtv1.RunStrategyUnknown
			}
			return *vm.Spec.RunStrategy
		}, 210*time.Second, 1*time.Second).Should(Equal(virtv1.RunStrategyHalted), "runStrategy not halted in time")
		err := virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{})
		if k8serrors.IsNotFound(err) {
			return
		}
		Expect(err).ToNot(HaveOccurred())
		By("Verifying VM is gone")
		Eventually(matcher.ThisVMWith(vm.Namespace, vm.Name), 30*time.Second, 1*time.Second).Should(matcher.BeGone(), "VM should disappear")
		By("Verifying VMI is gone")
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 30*time.Second, 1*time.Second).Should(matcher.BeGone(), "VMI should disappear")
	}

	deleteDV := func(dv *cdiv1.DataVolume) {
		err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Delete(context.Background(), dv.Name, metav1.DeleteOptions{})
		if k8serrors.IsNotFound(err) {
			return
		}
		Expect(err).ToNot(HaveOccurred())
		By("Verifying DV is gone")
		Eventually(matcher.ThisDVWith(dv.Namespace, dv.Name), 30*time.Second, 1*time.Second).Should(matcher.BeGone(), "DV should disappear")
		By("Verifying PVC is gone")
		Eventually(matcher.ThisPVCWith(dv.Namespace, dv.Name), 30*time.Second, 1*time.Second).Should(matcher.BeGone(), "PVC should disappear")
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

		It("should live migrate a container disk vm, several times", func() {
			var targetVM *virtv1.VirtualMachine

			sourceVMI := libvmifact.NewAlpine(
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			targetVMI := sourceVMI.DeepCopy()
			targetVMI.Namespace = testsuite.NamespaceTestAlternative
			sourceVM := createAndStartVMFromVMISpec(sourceVMI)
			num := 4
			for i := 0; i < num; i++ {
				migrationID := fmt.Sprintf("mig-%s", rand.String(5))
				By(fmt.Sprintf("generated migrationID %s", migrationID))
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
				Expect(console.LoginToAlpine(expectedVMI)).To(Succeed())

				By(fmt.Sprintf("deleting source VM %s/%s", sourceVM.Namespace, sourceVM.Name))
				deleteVM(sourceVM)
				sourceVM = targetVM
			}
		})

		It("should live migrate a container disk vm, with an additional PVC mounted, should stay mounted after migration", func() {
			migrationID := fmt.Sprintf("mig-%s", rand.String(5))
			sourceDV := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithStorage(),
			)

			sourceVMI := libvmifact.NewCirros(
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithDataVolume("disk1", sourceDV.Name),
			)
			targetVMI := sourceVMI.DeepCopy()
			targetVMI.Namespace = testsuite.NamespaceTestAlternative
			targetDV := sourceDV.DeepCopy()
			targetDV.Namespace = targetVMI.Namespace
			sourceDV, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(sourceDV)).Create(context.Background(), sourceDV, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(sourceDV, 240, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))

			createAndStartVMFromVMISpec(sourceVMI)
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

			targetVM := createReceiverVMFromVMISpec(targetVMI)
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
			By("verifying the runStrategy is properly updated to be what the annotation is")
			Eventually(func() virtv1.VirtualMachineRunStrategy {
				targetVM, err = virtClient.VirtualMachine(targetVM.Namespace).Get(context.Background(), targetVM.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if targetVM.Spec.RunStrategy == nil {
					return virtv1.RunStrategyUnknown
				}
				return *targetVM.Spec.RunStrategy
			}).WithTimeout(time.Second * 20).WithPolling(500 * time.Millisecond).Should(Equal(virtv1.RunStrategyAlways))
		})

		createDVBlock := func(name, namespace, sc string) *cdiv1.DataVolume {
			dvBlock := libdv.NewDataVolume(
				libdv.WithName(name),
				libdv.WithBlankImageSource(),
				libdv.WithStorage(
					libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeSize(cd.BlankVolumeSize),
					libdv.StorageWithAccessMode(k8sv1.ReadWriteMany),
					libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeBlock),
				),
			)
			dvBlock, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), dvBlock, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(dvBlock, 240, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))
			return dvBlock
		}

		addDVVolume := func(name, namespace, volumeName, claimName string, bus v1.DiskBus) {
			opts := &v1.AddVolumeOptions{
				Name: volumeName,
				Disk: &v1.Disk{
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: bus,
						},
					},
					Serial: volumeName,
				},
				VolumeSource: &v1.HotplugVolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: claimName,
					},
				},
			}

			Eventually(func() error {
				return virtClient.VirtualMachine(namespace).AddVolume(context.Background(), name, opts)
			}, 3*time.Second, 1*time.Second).Should(Succeed())
		}

		It("should live migrate a container disk vm, with an additional hotpluggedPVC mounted, should stay mounted after migration", decorators.RequiresRWXBlock, func() {
			sc, exists := libstorage.GetRWXBlockStorageClass()
			if !exists {
				Fail("Fail test when RWXBlock storage class is not present")
			}
			migrationID := fmt.Sprintf("mig-%s", rand.String(5))
			hotpluggedDV := createDVBlock(fmt.Sprintf("dv-%s", migrationID), testsuite.NamespaceTestDefault, sc)

			sourceVMI := libvmifact.NewCirros(
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithCPURequest("1"), libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithCPULimit("1"), libvmi.WithMemoryLimit("128Mi"),
				libvmi.WithAnnotation("kubevirt.io/libvirt-log-filters", "3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access 1:*"),
			)

			sourceVM := createAndStartVMFromVMISpec(sourceVMI)
			By("Adding volume to running VM")
			volumeName := "testvolume"
			addDVVolume(sourceVM.Name, sourceVM.Namespace, volumeName, hotpluggedDV.Name, virtv1.DiskBusSCSI)

			By("Verifying the volume and disk are in the VMI")
			Eventually(func() virtv1.VolumePhase {
				sourceVMI, err = virtClient.VirtualMachineInstance(sourceVMI.Namespace).Get(context.Background(), sourceVMI.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, v := range sourceVMI.Status.VolumeStatus {
					if v.Name == volumeName {
						return v.Phase
					}
				}
				return ""
			}).WithTimeout(time.Minute).WithPolling(2 * time.Second).Should(Equal(virtv1.VolumeReady))
			sourceVMI, err = virtClient.VirtualMachineInstance(sourceVMI.Namespace).Get(context.Background(), sourceVMI.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			deviceName := ""
			for _, v := range sourceVMI.Status.VolumeStatus {
				if v.Name == volumeName {
					deviceName = v.Target
					break
				}
			}

			By("Writing data to extra disk")
			Expect(console.LoginToCirros(sourceVMI)).To(Succeed())
			// I am aware I should not use the device name since it is not guaranteed to be the same as the one in the VMI
			// I should be using the serial number, but not sure how to access that in cirros.
			Expect(console.RunCommand(sourceVMI, fmt.Sprintf("sudo mkfs.ext4 /dev/%s", deviceName), 30*time.Second)).To(Succeed())
			Expect(console.RunCommand(sourceVMI, "mkdir test", 30*time.Second)).To(Succeed())
			Expect(console.RunCommand(sourceVMI, fmt.Sprintf("sudo mount -t ext4 /dev/%s /home/cirros/test", deviceName), 30*time.Second)).To(Succeed())
			Expect(console.RunCommand(sourceVMI, "sudo chmod 777 /home/cirros/test", 30*time.Second)).To(Succeed())
			Expect(console.RunCommand(sourceVMI, "sudo chown cirros:cirros /home/cirros/test", 30*time.Second)).To(Succeed())
			Expect(console.RunCommand(sourceVMI, "printf 'important data' &> /home/cirros/test/data.txt", 30*time.Second)).To(Succeed())

			By("Creating the target VM and disk")
			targetVMI := sourceVMI.DeepCopy()
			targetVMI.Namespace = testsuite.NamespaceTestAlternative
			targetVMI.Labels = map[string]string{}
			targetVMI.Spec.Domain.Devices.Interfaces[0].MacAddress = ""
			targetDV := createDVBlock(hotpluggedDV.Name, testsuite.NamespaceTestAlternative, sc)
			Expect(targetDV.Namespace).To(Equal(targetVMI.Namespace))
			createReceiverVMFromVMISpec(targetVMI)

			By("Running a migration")
			sourceMigration := libmigration.NewSource(sourceVMI.Name, sourceVMI.Namespace, migrationID, connectionURL)
			targetMigration := libmigration.NewTarget(targetVMI.Name, targetVMI.Namespace, migrationID)
			sourceMigration, targetMigration = libmigration.RunDecentralizedMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, sourceMigration, targetMigration)
			Expect(targetMigration).ToNot(BeNil())
			if sourceMigration == nil {
				// Source migration was already removed due to migration completed and VMI being deleted.
				// Verify that the VMI doesn't exist, and that the VM is in the correct state.
				By("Verifying the VMI doesn't exist")
				_, err := virtClient.VirtualMachineInstance(sourceVMI.Namespace).Get(context.Background(), sourceVMI.Name, metav1.GetOptions{})
				Expect(err).To(MatchError(k8serrors.IsNotFound, "k8serrors.IsNotFound"))

				By("Verifying the VM is in the correct state")
				Eventually(func() virtv1.VirtualMachineRunStrategy {
					vm, err := virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), sourceVM.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vm.Spec.RunStrategy).ToNot(BeNil())
					return *vm.Spec.RunStrategy
				}).WithTimeout(time.Second * 20).WithPolling(500 * time.Millisecond).Should(Equal(virtv1.RunStrategyHalted))
			}
			libmigration.ConfirmVMIPostMigration(virtClient, targetVMI, targetMigration)
			By("Verifying data on extra disk")
			Eventually(func() string {
				targetVMI, err := virtClient.VirtualMachineInstance(targetVMI.Namespace).Get(context.Background(), targetVMI.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, v := range targetVMI.Status.VolumeStatus {
					if v.Name == volumeName {
						deviceName = v.Target
						return v.Target
					}
				}
				return ""
			}).WithTimeout(time.Minute).WithPolling(2 * time.Second).ShouldNot(BeEmpty())
			Expect(console.LoginToCirros(targetVMI)).To(Succeed())
			Expect(console.RunCommand(targetVMI, "cat /home/cirros/test/data.txt", 30*time.Second)).To(Succeed())
		})

		Context("with RWOFs backend storage class", func() {
			checkTPM := func(vmi *v1.VirtualMachineInstance) {
				By("Ensuring the TPM is still functional and its state carried over")
				ExpectWithOffset(1, console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "tpm2_unseal -Q --object-context=0x81010002\n"},
					&expect.BExp{R: "MYSECRET"},
				}, 300)).To(Succeed(), "the state of the TPM did not persist")
			}

			checkEFI := func(vmi *v1.VirtualMachineInstance) {
				By("Ensuring the efivar is present")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "hexdump /sys/firmware/efi/efivars/kvtest-12345678-1234-1234-1234-123456789abc\n"},
					&expect.BExp{R: "0042"},
				}, 10)).To(Succeed(), "expected efivar is missing")
			}

			addDataToTPM := func(vmi *v1.VirtualMachineInstance) {
				By("Storing a secret into the TPM")
				// https://www.intel.com/content/www/us/en/developer/articles/code-sample/protecting-secret-data-and-keys-using-intel-platform-trust-technology.html
				// Not sealing against a set of PCRs, out of scope here, but should work with a carefully selected set (at least PCR1 was seen changing across reboots)
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "tpm2_createprimary -Q --hierarchy=o --key-context=prim.ctx\n"},
					&expect.BExp{R: ""},
					&expect.BSnd{S: "echo MYSECRET | tpm2_create --hash-algorithm=sha256 --public=seal.pub --private=seal.priv --sealing-input=- --parent-context=prim.ctx\n"},
					&expect.BExp{R: ""},
					&expect.BSnd{S: "tpm2_load -Q --parent-context=prim.ctx --public=seal.pub --private=seal.priv --name=seal.name --key-context=seal.ctx\n"},
					&expect.BExp{R: ""},
					&expect.BSnd{S: "tpm2_evictcontrol --hierarchy=o --object-context=seal.ctx 0x81010002\n"},
					&expect.BExp{R: ""},
				}, 300)).To(Succeed(), "failed to store secret into the TPM")
				checkTPM(vmi)
			}

			addDataToEFI := func(vmi *v1.VirtualMachineInstance) {
				By("Creating an efivar")
				cmd := `printf "\x07\x00\x00\x00\x42" > /sys/firmware/efi/efivars/kvtest-12345678-1234-1234-1234-123456789abc`
				err := console.RunCommand(vmi, cmd, 10*time.Second)
				Expect(err).NotTo(HaveOccurred())
				checkEFI(vmi)
			}

			var currentBackendStorageClass string
			BeforeEach(func() {
				config := getCurrentKvConfig(virtClient)
				currentBackendStorageClass = config.VMStateStorageClass
				sc, exist := libstorage.GetRWOFileSystemStorageClass()
				Expect(exist).To(BeTrue())
				By(fmt.Sprintf("Changing the backend storage class from %s to %s", currentBackendStorageClass, sc))
				config.VMStateStorageClass = sc
				kvconfig.UpdateKubeVirtConfigValueAndWait(config)
			})

			AfterEach(func() {
				By(fmt.Sprintf("Restoring the backend storage class to %s", currentBackendStorageClass))
				config := getCurrentKvConfig(virtClient)
				config.VMStateStorageClass = currentBackendStorageClass
				kvconfig.UpdateKubeVirtConfigValueAndWait(config)
			})

			// TODO: Remove the RequiresRWOFsVMStateStorageClass once libvirt allows us to tell it to ignore the check
			// for shared storage.
			It("should decentralized migrate a VMI with persistent TPM+EFI enabled", decorators.RequiresDecentralizedLiveMigration, decorators.RequiresRWOFsVMStateStorageClass, Serial, func() {
				migrationID := fmt.Sprintf("mig-%s", rand.String(5))
				By("Creating a VMI with TPM+EFI enabled")
				sourceVMI := libvmifact.NewFedora(
					libvmi.WithNamespace(testsuite.NamespaceTestDefault),
					libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(cloudinit.CreateDefaultCloudInitNetworkData())), libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithTPM(true),
					libvmi.WithUefi(false),
				)
				sourceVMI.Spec.Domain.Firmware = &v1.Firmware{
					Bootloader: &v1.Bootloader{
						EFI: &v1.EFI{SecureBoot: pointer.P(false), Persistent: pointer.P(true)},
					},
				}
				sourceVM := createAndStartVMFromVMISpec(sourceVMI)
				Expect(sourceVM.Spec.Template.Spec.Domain.Firmware.UUID).ToNot(BeEmpty())
				targetVMI := sourceVMI.DeepCopy()
				targetVMI.Namespace = testsuite.NamespaceTestAlternative
				targetVMI.Spec.Domain.Firmware = sourceVM.Spec.Template.Spec.Domain.Firmware.DeepCopy()

				By("Waiting for agent to connect")
				Eventually(matcher.ThisVMI(sourceVMI)).WithTimeout(4 * time.Minute).WithPolling(2 * time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				sourceVMI = libwait.WaitUntilVMIReady(sourceVMI, console.LoginToFedora)

				addDataToTPM(sourceVMI)
				addDataToEFI(sourceVMI)

				By("Migrating the VMI")
				targetVM := createReceiverVMFromVMISpec(targetVMI)
				sourceMigration := libmigration.NewSource(sourceVMI.Name, sourceVMI.Namespace, migrationID, connectionURL)
				targetMigration := libmigration.NewTarget(targetVMI.Name, targetVMI.Namespace, migrationID)
				sourceMigration, targetMigration = libmigration.RunDecentralizedMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, sourceMigration, targetMigration)
				libmigration.ConfirmVMIPostMigration(virtClient, targetVMI, targetMigration)

				By("Ensuring the TPM is still functional and its state and EFI vars are carried over")
				checkTPM(targetVMI)
				checkEFI(targetVMI)
				By("Stopping the VM")
				libvmops.StopVirtualMachine(targetVM)
				By("Starting the VM")
				targetVM = libvmops.StartVirtualMachine(targetVM)
				By("Logging in")
				Expect(console.LoginToFedora(targetVMI)).To(Succeed())
				By("Ensuring the TPM and EFI vars contain the same data after stop and start")
				checkTPM(targetVMI)
				checkEFI(targetVMI)
			})
		})
	})

	Context("with migration policy", func() {
		var (
			sourceVMI, targetVMI *v1.VirtualMachineInstance
		)

		BeforeEach(func() {
			sourceVMI = libvmifact.NewCirros(
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			By("limiting the bandwidth of migrations")
			Expect(CreateMigrationPolicy(virtClient, PreparePolicyAndVMIWithBandwidthLimitation(sourceVMI, resource.MustParse("1Ki")))).ToNot(BeNil())

			targetVMI = sourceVMI.DeepCopy()
			targetVMI.Namespace = testsuite.NamespaceTestAlternative
		})

		DescribeTable("should be able to cancel a migration by deleting the migration resource", decorators.SigStorage, Serial, func(deleteSource bool) {
			const timeout = 180
			migrationID := fmt.Sprintf("mig-%s", rand.String(5))

			By("starting the VirtualMachine")
			createAndStartVMFromVMISpec(sourceVMI)
			By("creating a receiver VM")
			createReceiverVMFromVMISpec(targetVMI)
			By("creating the migration")
			sourceMigration := libmigration.NewSource(sourceVMI.Name, sourceVMI.Namespace, migrationID, connectionURL)
			targetMigration := libmigration.NewTarget(targetVMI.Name, targetVMI.Namespace, migrationID)

			By("starting a migration")
			sourceMigration = libmigration.RunMigration(virtClient, sourceMigration)
			targetMigration = libmigration.RunMigration(virtClient, targetMigration)
			migrationToDelete := sourceMigration
			if !deleteSource {
				migrationToDelete = targetMigration
			}

			By("waiting until the migration is Running")
			Eventually(func() bool {
				sourceMigration, err := virtClient.VirtualMachineInstanceMigration(sourceMigration.Namespace).Get(context.Background(), sourceMigration.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(sourceMigration.Status.Phase).ToNot(Equal(v1.MigrationFailed))
				if sourceMigration.Status.Phase == v1.MigrationRunning {
					sourceVMI, err = virtClient.VirtualMachineInstance(sourceVMI.Namespace).Get(context.Background(), sourceVMI.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if sourceVMI.Status.MigrationState.Completed != true {
						return true
					}
				}
				return false
			}).WithTimeout(timeout * time.Second).WithPolling(500 * time.Millisecond).Should(BeTrue())

			By("cancelling a migration")
			Expect(virtClient.VirtualMachineInstanceMigration(migrationToDelete.Namespace).Delete(context.Background(), migrationToDelete.Name, metav1.DeleteOptions{})).To(Succeed())

			By("checking VMI, confirm migration state")
			libmigration.ConfirmVMIPostMigrationAborted(sourceVMI, string(sourceMigration.UID), timeout)

			By("Waiting for the source migration object to disappear")
			libwait.WaitForMigrationToDisappearWithTimeout(sourceMigration, timeout)
			By("Waiting for the target migration object to disappear")
			libwait.WaitForMigrationToDisappearWithTimeout(targetMigration, timeout)
			By("Logging in and ensuring the source VM is still running")
			Expect(console.LoginToCirros(sourceVMI)).To(Succeed())
			By("Checking that the receiving VM is in WaitingAsReceiver phase")
			Eventually(func() virtv1.VirtualMachineInstancePhase {
				targetVMI, err := virtClient.VirtualMachineInstance(targetVMI.Namespace).Get(context.Background(), targetVMI.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return targetVMI.Status.Phase
			}).WithTimeout(time.Minute).WithPolling(2 * time.Second).Should(Equal(virtv1.WaitingForSync))
		},
			Entry("delete source migration", true),
			Entry("delete target migration", false),
		)

		It("should properly propagate failure from target to source", func() {
			const timeout = 180
			migrationID := fmt.Sprintf("mig-%s", rand.String(5))
			By("starting the VirtualMachine")
			sourceVM := createAndStartVMFromVMISpec(sourceVMI)
			By("creating a receiver VM")
			targetVM := createReceiverVMFromVMISpec(targetVMI)
			By("creating the migration")
			sourceMigration := libmigration.NewSource(sourceVMI.Name, sourceVMI.Namespace, migrationID, connectionURL)
			targetMigration := libmigration.NewTarget(targetVMI.Name, targetVMI.Namespace, migrationID)

			By("starting a migration")
			sourceMigration = libmigration.RunMigration(virtClient, sourceMigration)
			targetMigration = libmigration.RunMigration(virtClient, targetMigration)

			By("waiting until the migration is Running")
			Eventually(func() bool {
				sourceMigration, err := virtClient.VirtualMachineInstanceMigration(sourceMigration.Namespace).Get(context.Background(), sourceMigration.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(sourceMigration.Status.Phase).ToNot(Equal(v1.MigrationFailed))
				if sourceMigration.Status.Phase == v1.MigrationRunning {
					sourceVMI, err = virtClient.VirtualMachineInstance(sourceVMI.Namespace).Get(context.Background(), sourceVMI.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if sourceVMI.Status.MigrationState.Completed != true {
						return true
					}
				}
				return false
			}).WithTimeout(timeout * time.Second).WithPolling(500 * time.Millisecond).Should(BeTrue())

			By("force stopping the source pod")
			sourcePod, err := libpod.GetPodByVirtualMachineInstance(sourceVMI, sourceVMI.Namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(sourcePod.Status.Phase).To(Equal(k8sv1.PodRunning))
			Expect(virtClient.CoreV1().Pods(sourceVMI.Namespace).Delete(context.Background(), sourcePod.Name, metav1.DeleteOptions{GracePeriodSeconds: pointer.P(int64(0))})).To(Succeed())

			By("waiting for the source migration to fail")
			Eventually(func() virtv1.VirtualMachineInstanceMigrationPhase {
				sourceMigration, err = virtClient.VirtualMachineInstanceMigration(sourceMigration.Namespace).Get(context.Background(), sourceMigration.Name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					// Migration is already deleted, this means the VMI is gone, and thus the migration failed
					return v1.MigrationFailed
				}
				Expect(err).ToNot(HaveOccurred())
				return sourceMigration.Status.Phase
			}).WithTimeout(timeout * time.Second).WithPolling(500 * time.Millisecond).Should(Equal(v1.MigrationFailed))

			By("waiting for the target migration to fail")
			Eventually(func() virtv1.VirtualMachineInstanceMigrationPhase {
				targetMigration, err = virtClient.VirtualMachineInstanceMigration(targetMigration.Namespace).Get(context.Background(), targetMigration.Name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					// Migration is already deleted, this means the VMI is gone, and thus the migration failed
					return v1.MigrationFailed
				}
				Expect(err).ToNot(HaveOccurred())
				return targetMigration.Status.Phase
			}).WithTimeout(timeout * time.Second).WithPolling(500 * time.Millisecond).Should(Equal(v1.MigrationFailed))

			By("ensuring the source VM is stopped")
			Eventually(func() virtv1.VirtualMachinePrintableStatus {
				sourceVM, err = virtClient.VirtualMachine(sourceVM.Namespace).Get(context.Background(), sourceVM.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return sourceVM.Status.PrintableStatus
			}).WithTimeout(timeout * time.Second).WithPolling(500 * time.Millisecond).Should(Equal(virtv1.VirtualMachineStatusStopped))

			By("ensuring the target VM is WaitingForReceiver")
			Eventually(func() virtv1.VirtualMachinePrintableStatus {
				targetVM, err = virtClient.VirtualMachine(targetVM.Namespace).Get(context.Background(), targetVM.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return targetVM.Status.PrintableStatus
			}).WithTimeout(timeout * time.Second).WithPolling(500 * time.Millisecond).Should(Equal(virtv1.VirtualMachineStatusWaitingForReceiver))
		})
	})
	Context("datavolume disk", func() {
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

		It("should live migrate regular disk several times", func() {
			var targetVM *virtv1.VirtualMachine
			sourceDV := libdv.NewDataVolume(
				libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
				libdv.WithStorage(
					libdv.StorageWithVolumeSize(cd.AlpineVolumeSize),
				),
			)
			sourceDV, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(sourceDV)).Create(context.Background(), sourceDV, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(sourceDV, 240, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))
			sourceVMI := libvmi.New(
				libvmi.WithNamespace(testsuite.NamespaceTestDefault),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithDataVolume("disk0", sourceDV.Name),
				libvmi.WithResourceMemory("128Mi"),
			)
			targetVMI := sourceVMI.DeepCopy()
			targetVMI.Namespace = testsuite.NamespaceTestAlternative

			sourceVM := createAndStartVMFromVMISpec(sourceVMI)
			Expect(sourceVM).ToNot(BeNil())
			Expect(console.LoginToAlpine(sourceVMI)).To(Succeed())
			var targetDV *cdiv1.DataVolume
			num := 4
			for i := 0; i < num; i++ {
				migrationID := fmt.Sprintf("mig-%s", rand.String(5))
				var sourceMigration, targetMigration *virtv1.VirtualMachineInstanceMigration
				var expectedVMI *virtv1.VirtualMachineInstance
				sourceRunStrategy := sourceVM.Spec.RunStrategy
				By(fmt.Sprintf("executing a migration, and waiting for finalized state, run %d", i))
				if i%2 == 0 {
					// source -> target
					targetDV = createBlankFromName(sourceDV.Name, testsuite.NamespaceTestAlternative)
					targetVM = createReceiverVMFromVMISpec(targetVMI)
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
