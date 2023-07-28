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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package storage

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	expect "github.com/google/goexpect"
	storagev1 "k8s.io/api/storage/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	instancetypeapi "kubevirt.io/api/instancetype"
	instanceType "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	checkingVMInstanceConsoleExpectedOut = "Checking that the VirtualMachineInstance console has expected output"
	creatingVMInvalidDataVolume          = "Creating a VM with an invalid DataVolume"
	creatingVMDataVolumeTemplateEntry    = "Creating VM with DataVolumeTemplate entry with k8s client binary"
	verifyingDataVolumeSuccess           = "Verifying DataVolume succeeded and is created with VM owner reference"
	verifyingPVCCreated                  = "Verifying PVC is created"
	verifyingVMICreated                  = "Verifying VMI is created with VM owner reference"
	syncName                             = "sync\n"
)

const InvalidDataVolumeUrl = "docker://127.0.0.1/invalid:latest"

var _ = SIGDescribe("DataVolume Integration", func() {

	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		if !libstorage.HasCDI() {
			Skip("Skip DataVolume tests when CDI is not present")
		}
	})

	Context("[storage-req]PVC expansion", decorators.StorageReq, func() {
		DescribeTable("PVC expansion is detected by VM and can be fully used", func(volumeMode k8sv1.PersistentVolumeMode) {
			checks.SkipTestIfNoFeatureGate(virtconfig.ExpandDisksGate)
			if !libstorage.HasCDI() {
				Skip("Skip DataVolume tests when CDI is not present")
			}
			var sc string
			exists := false
			if volumeMode == k8sv1.PersistentVolumeBlock {
				sc, exists = libstorage.GetRWOBlockStorageClass()
				if !exists {
					Skip("Skip test when Block storage is not present")
				}
			} else {
				sc, exists = libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Skip("Skip test when Filesystem storage is not present")
				}
			}
			volumeExpansionAllowed := volumeExpansionAllowed(sc)
			if !volumeExpansionAllowed {
				Skip("Skip when volume expansion storage class not available")
			}
			vmi, dataVolume := tests.NewRandomVirtualMachineInstanceWithDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), testsuite.GetTestNamespace(nil), sc, k8sv1.ReadWriteOnce, volumeMode)
			tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
			vmi = tests.RunVMIAndExpectLaunch(vmi, 500)

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToCirros(vmi)).To(Succeed())

			By("Expanding PVC")
			patchData, err := patch.GeneratePatchPayload(patch.PatchOperation{
				Op:    patch.PatchAddOp,
				Path:  "/spec/resources/requests/storage",
				Value: resource.MustParse("2Gi"),
			})
			Expect(err).ToNot(HaveOccurred())
			_, err = virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Patch(context.Background(), dataVolume.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for notification about size change")
			Eventually(func() error {
				err := console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "dmesg |grep 'new size'\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "dmesg |grep -c 'new size: [34]'\n"},
					&expect.BExp{R: "1"},
				}, 10)
				return err
			}, 360).Should(BeNil())

			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "sudo /sbin/resize-filesystem /dev/root /run/resize.rootfs /dev/console && echo $?\n"},
				&expect.BExp{R: "0"},
			}, 30)).To(Succeed(), "failed to resize root")

			By("Writing a 1.5G file after expansion, should succeed")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "dd if=/dev/zero of=largefile count=1500 bs=1M; echo $?\n"},
				&expect.BExp{R: "0"},
			}, 360)).To(Succeed(), "can use more space after expansion and resize")
		},
			Entry("with Block PVC", k8sv1.PersistentVolumeBlock),
			Entry("with Filesystem PVC", k8sv1.PersistentVolumeFilesystem),
		)
	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachineInstance with a DataVolume as a volume source", func() {

		Context("[Serial]without fsgroup support", Serial, func() {
			size := "1Gi"

			It("should succesfully start", func() {
				// Create DV and alter permission of disk.img
				sc, foundSC := libstorage.GetRWXFileSystemStorageClass()
				if !foundSC {
					Skip("Skip test when Filesystem storage is not present")
				}

				dv := libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
					libdv.WithPVC(
						libdv.PVCWithStorageClass(sc),
						libdv.PVCWithVolumeSize(size),
						libdv.PVCWithReadWriteManyAccessMode(),
					),
					libdv.WithForceBindAnnotation(),
				)

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.NamespacePrivileged).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				var pvc *k8sv1.PersistentVolumeClaim
				Eventually(func() *k8sv1.PersistentVolumeClaim {
					pvc, err = virtClient.CoreV1().PersistentVolumeClaims(testsuite.NamespacePrivileged).Get(context.Background(), dv.Name, metav1.GetOptions{})
					if err != nil {
						return nil
					}
					return pvc
				}, 30*time.Second).Should(Not(BeNil()))
				By("waiting for the dv import to pvc to finish")
				libstorage.EventuallyDV(dv, 180, HaveSucceeded())
				tests.ChangeImgFilePermissionsToNonQEMU(pvc)

				vmi := tests.NewRandomVMIWithDataVolume(dv.Name)
				vmi.Namespace = testsuite.NamespacePrivileged

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 120)

				By(checkingVMInstanceConsoleExpectedOut)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())
			})
		})

		Context("Alpine import", func() {
			BeforeEach(func() {
				cdis, err := virtClient.CdiClient().CdiV1beta1().CDIs().List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(cdis.Items).To(HaveLen(1))
				hasWaitForFirstConsumerGate := false
				for _, feature := range cdis.Items[0].Spec.Config.FeatureGates {
					if feature == "HonorWaitForFirstConsumer" {
						hasWaitForFirstConsumerGate = true
						break
					}
				}
				if !hasWaitForFirstConsumerGate {
					Skip("HonorWaitForFirstConsumer is disabled in CDI, skipping tests relying on it")
				}
			})

			It("[test_id:3189]should be successfully started and stopped multiple times", func() {
				sc, exists := libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Skip("Skip test when Filesystem storage is not present")
				}

				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
					libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
				)

				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)

				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// This will only work on storage with binding mode WaitForFirstConsumer,
				if libstorage.IsStorageClassBindingModeWaitForFirstConsumer(libstorage.Config.StorageRWOFileSystem) {
					Eventually(ThisDV(dataVolume), 40).Should(Or(BeInPhase(cdiv1.WaitForFirstConsumer), BeInPhase(cdiv1.PendingPopulation)))
				}
				num := 2
				By("Starting and stopping the VirtualMachineInstance a number of times")
				for i := 1; i <= num; i++ {
					vmi := tests.RunVMIAndExpectLaunchWithDataVolume(vmi, dataVolume, 500)
					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By(checkingVMInstanceConsoleExpectedOut)
						Expect(console.LoginToAlpine(vmi)).To(Succeed())
					}

					err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				}
				libstorage.DeleteDataVolume(&dataVolume)
			})

			It("[test_id:6686]should successfully start multiple concurrent VMIs", func() {

				sc, exists := libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Skip("Skip test when Filesystem storage is not present")
				}

				imageUrl := cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)

				const numVmis = 5
				vmis := make([]*v1.VirtualMachineInstance, 0, numVmis)
				dvs := make([]*cdiv1.DataVolume, 0, numVmis)

				for idx := 0; idx < numVmis; idx++ {
					dataVolume := libdv.NewDataVolume(
						libdv.WithRegistryURLSource(imageUrl),
						libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
					)

					vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("128Mi")

					dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vmi = tests.RunVMI(vmi, 60)
					vmis = append(vmis, vmi)
					dvs = append(dvs, dataVolume)
				}

				for idx := 0; idx < numVmis; idx++ {
					libwait.WaitForSuccessfulVMIStart(vmis[idx],
						libwait.WithFailOnWarnings(false),
						libwait.WithTimeout(500),
					)
					By(checkingVMInstanceConsoleExpectedOut)
					Expect(console.LoginToAlpine(vmis[idx])).To(Succeed())

					err := virtClient.VirtualMachineInstance(vmis[idx].Namespace).Delete(context.Background(), vmis[idx].Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					libstorage.DeleteDataVolume(&dvs[idx])
				}
			})

			It("[test_id:5252]should be successfully started when using a PVC volume owned by a DataVolume", func() {
				sc, exists := libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Skip("Skip test when Filesystem storage is not present")
				}

				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
					libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
				)

				vmi := tests.NewRandomVMIWithPVC(dataVolume.Name)

				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				// This will only work on storage with binding mode WaitForFirstConsumer,
				if libstorage.IsStorageClassBindingModeWaitForFirstConsumer(libstorage.Config.StorageRWOFileSystem) {
					Eventually(ThisDV(dataVolume), 40).Should(Or(BeInPhase(cdiv1.WaitForFirstConsumer), BeInPhase(cdiv1.PendingPopulation)))
				}
				// with WFFC the run actually starts the import and then runs VM, so the timeout has to include both
				// import and start
				vmi = tests.RunVMIAndExpectLaunchWithDataVolume(vmi, dataVolume, 500)

				By(checkingVMInstanceConsoleExpectedOut)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				libstorage.DeleteDataVolume(&dataVolume)
			})

			It("should accurately report DataVolume provisioning", func() {
				sc, err := libstorage.GetSnapshotStorageClass(virtClient)
				if err != nil {
					Skip("no snapshot storage class configured")
				}

				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
					libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
				)

				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
				vmSpec := tests.NewRandomVirtualMachine(vmi, false)

				vm, err := virtClient.VirtualMachine(vmSpec.Namespace).Create(context.Background(), vmSpec)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() v1.VirtualMachinePrintableStatus {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.PrintableStatus
				}, 180*time.Second, 2*time.Second).Should(Equal(v1.VirtualMachineStatusStopped))

				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vmSpec.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				defer libstorage.DeleteDataVolume(&dataVolume)

				Eventually(func() v1.VirtualMachinePrintableStatus {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.PrintableStatus
				}, 180*time.Second, 1*time.Second).Should(Equal(v1.VirtualMachineStatusProvisioning))

				Eventually(func() v1.VirtualMachinePrintableStatus {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.PrintableStatus
				}, 180*time.Second, 2*time.Second).Should(Equal(v1.VirtualMachineStatusStopped))
			})
		})

		Context("with a PVC from a Datavolume", func() {
			var storageClass *storagev1.StorageClass
			var vmi *v1.VirtualMachineInstance
			var dv *cdiv1.DataVolume
			BeforeEach(func() {
				// ensure that we always use a storage class which binds immediately,
				// otherwise we will never see a PVC appear for the datavolume
				bindMode := storagev1.VolumeBindingImmediate
				storageClass = &storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "fake",
					},
					Provisioner:       "afakeone",
					VolumeBindingMode: &bindMode,
				}
				storageClass, err = virtClient.StorageV1().StorageClasses().Create(context.Background(), storageClass, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				dv = libdv.NewDataVolume(
					libdv.WithNamespace(testsuite.GetTestNamespace(nil)), // need it for deletion. Reading it from Create() does not work here
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
					libdv.WithPVC(libdv.PVCWithStorageClass(storageClass.Name)),
				)
				vmi = libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithResourceMemory("32Mi"),
					libvmi.WithPersistentVolumeClaim(diskName, dv.ObjectMeta.Name),
				)
			})
			AfterEach(func() {
				if storageClass != nil && storageClass.Name != "" {
					err := virtClient.StorageV1().StorageClasses().Delete(context.Background(), storageClass.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
				libstorage.DeleteDataVolume(&dv)
			})

			It("[test_id:4643]should NOT be rejected when VM template lists a DataVolume, but VM lists PVC VolumeSource", func() {
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() error {
					_, err := virtClient.CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(dv)).Get(context.Background(), dv.Name, metav1.GetOptions{})
					return err
				}, 30*time.Second, 1*time.Second).Should(Succeed())

				vm := tests.NewRandomVirtualMachine(vmi, true)
				dvt := &v1.DataVolumeTemplateSpec{
					ObjectMeta: dv.ObjectMeta,
					Spec:       dv.Spec,
				}
				vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dvt)
				_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachine with an invalid DataVolume", func() {
		Context("using DataVolume with invalid URL", func() {
			It("should be possible to stop VM if datavolume is crashing", func() {
				sc, exists := libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Skip("Skip test when Filesystem storage is not present")
				}

				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(InvalidDataVolumeUrl, cdiv1.RegistryPullPod),
					libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
				)

				vm := tests.NewRandomVirtualMachine(tests.NewRandomVMIWithDataVolume(dataVolume.Name), true)
				vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{
					{
						ObjectMeta: dataVolume.ObjectMeta,
						Spec:       dataVolume.Spec,
					},
				}

				By(creatingVMInvalidDataVolume)
				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for DV to start crashing")
				Eventually(ThisDVWith(vm.Namespace, dataVolume.Name), 60).Should(BeInPhase(cdiv1.ImportInProgress))

				By("Stop VM")
				tests.StopVirtualMachineWithTimeout(vm, time.Second*30)
			})

			It("[test_id:3190]should correctly handle invalid DataVolumes", func() {
				// Don't actually create the DataVolume since it's invalid.
				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSource(InvalidDataVolumeUrl),
					libdv.WithPVC(libdv.PVCWithStorageClass("fakeStorageClass")),
				)

				//  Add the invalid DataVolume to a VMI
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
				// Create a VM for this VMI
				vm := tests.NewRandomVirtualMachine(vmi, true)

				By(creatingVMInvalidDataVolume)
				vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to be created")
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 100).Should(BeInPhase(v1.Pending))
			})
			It("[test_id:3190]should correctly handle eventually consistent DataVolumes", func() {
				sc, exists := libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Skip("Skip test when Filesystem storage is not present")
				}

				realRegistryName := flags.KubeVirtUtilityRepoPrefix
				realRegistryPort := ""
				if strings.Contains(flags.KubeVirtUtilityRepoPrefix, ":") {
					realRegistryName = strings.Split(flags.KubeVirtUtilityRepoPrefix, ":")[0]
					realRegistryPort = strings.Split(flags.KubeVirtUtilityRepoPrefix, ":")[1]
				}
				if realRegistryPort == "" {
					Skip("Skip when no port, CDI will always try to reach dockerhub/fakeregistry instead of just fakeregistry")
				}

				fakeRegistryName := "fakeregistry"
				fakeRegistryWithPort := fakeRegistryName
				if realRegistryPort != "" {
					fakeRegistryWithPort = fmt.Sprintf("%s:%s", fakeRegistryName, realRegistryPort)
				}

				imageUrl := cd.DataVolumeImportUrlFromRegistryForContainerDisk(fakeRegistryWithPort, cd.ContainerDiskCirros)

				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(imageUrl, cdiv1.RegistryPullPod),
					libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
				)

				defer libstorage.DeleteDataVolume(&dataVolume)

				By("Creating DataVolume with invalid URL")
				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By(creatingVMInvalidDataVolume)
				//  Add the invalid DataVolume to a VMI
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
				// Create a VM for this VMI
				vm := tests.NewRandomVirtualMachine(vmi, true)
				vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 100).Should(BeInPhase(v1.Pending))

				By("Creating a service which makes the registry reachable")
				_, err = virtClient.CoreV1().Services(vm.Namespace).Create(context.Background(), &k8sv1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: fakeRegistryName,
					},
					Spec: k8sv1.ServiceSpec{
						Type:         k8sv1.ServiceTypeExternalName,
						ExternalName: realRegistryName,
					},
				}, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Wait for DataVolume to complete")
				libstorage.EventuallyDV(dataVolume, 160, HaveSucceeded())

				By("Waiting for VMI to be created")
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 100).Should(BeInPhase(v1.Running))
			})
		})
	})

	Describe("[rfe_id:896][crit:high][vendor:cnv-qe@redhat.com][level:system] with oc/kubectl", func() {
		var vm *v1.VirtualMachine
		var err error
		var vmJson string
		var dataVolumeName string
		var pvcName string

		k8sClient := clientcmd.GetK8sCmdClient()

		BeforeEach(func() {
			running := true

			var foundSC bool
			vm, foundSC = tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil))
			if !foundSC {
				Skip("Skip test when Filesystem storage is not present")
			}

			vm.Spec.Running = &running

			dataVolumeName = vm.Spec.DataVolumeTemplates[0].Name
			pvcName = dataVolumeName

			vmJson, err = tests.GenerateVMJson(vm, GinkgoT().TempDir())
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:836] Creating a VM with DataVolumeTemplates should succeed.", func() {
			By(creatingVMDataVolumeTemplateEntry)
			_, _, err = clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By(verifyingDataVolumeSuccess)
			libstorage.EventuallyDVWith(vm.Namespace, dataVolumeName, 100, And(HaveSucceeded(), BeOwned()))

			By(verifyingPVCCreated)
			Eventually(ThisPVCWith(vm.Namespace, pvcName), 160).Should(Exist())

			By(verifyingVMICreated)
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 160).Should(And(BeRunning(), BeOwned()))
		})

		It("[test_id:837]deleting VM with cascade=true should automatically delete DataVolumes and VMI owned by VM.", func() {
			By(creatingVMDataVolumeTemplateEntry)
			_, _, err = clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By(verifyingDataVolumeSuccess)
			libstorage.EventuallyDVWith(vm.Namespace, dataVolumeName, 100, And(HaveSucceeded(), BeOwned()))

			By(verifyingPVCCreated)
			Eventually(ThisPVCWith(vm.Namespace, pvcName), 160).Should(Exist())

			By(verifyingVMICreated)
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 160).Should(And(BeRunning(), BeOwned()))

			By("Deleting VM with cascade=true")
			_, _, err = clientcmd.RunCommand("kubectl", "delete", "vm", vm.Name, "--cascade=true")
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VM to be deleted")
			Eventually(ThisVM(vm), 100).Should(BeGone())

			By("Waiting for the VMI to be deleted")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 100).Should(BeGone())

			By("Waiting for the DataVolume to be deleted")
			Eventually(ThisDVWith(vm.Namespace, dataVolumeName), 100).Should(BeGone())

			By("Waiting for the PVC to be deleted")
			Eventually(ThisPVCWith(vm.Namespace, pvcName), 100).Should(BeGone())
		})

		It("[test_id:838]deleting VM with cascade=false should orphan DataVolumes and VMI owned by VM.", func() {
			_, _, err = clientcmd.RunCommand(k8sClient, "create", "-f", vmJson)
			By(creatingVMDataVolumeTemplateEntry)
			Expect(err).ToNot(HaveOccurred())

			By(verifyingDataVolumeSuccess)
			libstorage.EventuallyDVWith(vm.Namespace, dataVolumeName, 100, And(HaveSucceeded(), BeOwned()))

			By(verifyingPVCCreated)
			Eventually(ThisPVCWith(vm.Namespace, pvcName), 160).Should(Exist())

			By(verifyingVMICreated)
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 160).Should(And(BeRunning(), BeOwned()))

			By("Deleting VM with cascade=false")
			_, _, err = clientcmd.RunCommand("kubectl", "delete", "vm", vm.Name, "--cascade=false")
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VM to be deleted")
			Eventually(ThisVM(vm), 100).Should(BeGone())

			By("Verifying DataVolume still exists with owner references removed")
			libstorage.EventuallyDVWith(vm.Namespace, dataVolumeName, 100, And(HaveSucceeded(), Not(BeOwned())))

			By("Verifying VMI still exists with owner references removed")
			Consistently(ThisVMIWith(vm.Namespace, vm.Name), 60, 1).Should(And(BeRunning(), Not(BeOwned())))
		})

	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachine with a DataVolume", func() {
		Context("using Alpine http import", func() {
			It("a DataVolume with preallocation shouldn't have discard=unmap", func() {
				vm, foundSC := tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil))
				if !foundSC {
					Skip("Skip test when Filesystem storage is not present")
				}

				preallocation := true
				vm.Spec.DataVolumeTemplates[0].Spec.Preallocation = &preallocation

				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				vm = tests.StartVirtualMachine(vm)
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).ToNot(ContainSubstring("discard='unmap'"))
				vm = tests.StopVirtualMachine(vm)
				Expect(virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})).To(Succeed())
			})

			DescribeTable("[test_id:3191]should be successfully started and stopped multiple times", func(isHTTP bool) {
				var (
					vm      *v1.VirtualMachine
					foundSC bool
				)
				if isHTTP {
					vm, foundSC = tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil))
				} else {
					url := cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)
					vm, foundSC = tests.NewRandomVMWithDataVolume(url, testsuite.GetTestNamespace(nil))
				}
				if !foundSC {
					Skip("Skip test when Filesystem storage is not present")
				}

				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())
				num := 2
				By("Starting and stopping the VirtualMachine number of times")
				for i := 0; i < num; i++ {
					By(fmt.Sprintf("Doing run: %d", i))
					vm = tests.StartVirtualMachine(vm)
					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By(checkingVMInstanceConsoleExpectedOut)
						vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						Expect(console.LoginToAlpine(vmi)).To(Succeed())
					}
					vm = tests.StopVirtualMachine(vm)
				}
			},

				Entry("with http import", true),
				Entry("with registry import", false),
			)

			It("[test_id:3192]should remove owner references on DataVolume if VM is orphan deleted.", func() {
				vm, foundSC := tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil))
				if !foundSC {
					Skip("Skip test when Filesystem storage is not present")
				}

				vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
				Expect(err).ToNot(HaveOccurred())

				// Check for owner reference
				Eventually(ThisDVWith(vm.Namespace, vm.Spec.DataVolumeTemplates[0].Name), 100).Should(BeOwned())

				// Delete the VM with orphan Propagation
				orphanPolicy := metav1.DeletePropagationOrphan
				Expect(virtClient.VirtualMachine(vm.Namespace).
					Delete(context.Background(), vm.Name, &metav1.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())

				// Wait for the owner reference to disappear
				Eventually(ThisDVWith(vm.Namespace, vm.Spec.DataVolumeTemplates[0].Name), 100).Should(Not(BeOwned()))
			})
		})
	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] DataVolume clone permission checking", func() {
		Context("using Alpine import/clone", func() {
			var dataVolume *cdiv1.DataVolume
			var cloneRole *rbacv1.Role
			var cloneRoleBinding *rbacv1.RoleBinding
			var storageClass string
			var vm *v1.VirtualMachine

			BeforeEach(func() {
				storageClass, err = libstorage.GetSnapshotStorageClass(virtClient)
				Expect(err).ToNot(HaveOccurred())
				if storageClass == "" {
					Skip("Skiping test, no VolumeSnapshot support")
				}

				dataVolume = libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
					libdv.WithPVC(libdv.PVCWithStorageClass(storageClass)),
					libdv.WithForceBindAnnotation(),
				)

				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.NamespaceTestAlternative).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libstorage.EventuallyDV(dataVolume, 90, HaveSucceeded())

				vm = newRandomVMWithCloneDataVolume(testsuite.NamespaceTestAlternative, dataVolume.Name, testsuite.GetTestNamespace(nil), storageClass)

				const volumeName = "sa"
				saVol := v1.Volume{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						ServiceAccount: &v1.ServiceAccountVolumeSource{
							ServiceAccountName: testsuite.AdminServiceAccountName,
						},
					},
				}
				vm.Spec.DataVolumeTemplates[0].Spec.PVC.StorageClassName = pointer.String(storageClass)
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, saVol)
				vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{Name: volumeName})
			})

			AfterEach(func() {
				if cloneRole != nil {
					err := virtClient.RbacV1().Roles(cloneRole.Namespace).Delete(context.Background(), cloneRole.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					cloneRole = nil
				}

				if cloneRoleBinding != nil {
					err := virtClient.RbacV1().RoleBindings(cloneRoleBinding.Namespace).Delete(context.Background(), cloneRoleBinding.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					cloneRoleBinding = nil
				}
			})

			createVMSuccess := func() {
				// sometimes it takes a bit for permission to actually be applied so eventually
				Eventually(func() bool {
					_, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
					if err != nil {
						fmt.Printf("command should have succeeded maybe new permissions not applied yet\nerror\n%s\n", err)
						return false
					}
					return true
				}, 90*time.Second, 1*time.Second).Should(BeTrue())

				// start vm and check dv clone succeeded
				vm = tests.StartVirtualMachine(vm)
				targetDVName := vm.Spec.DataVolumeTemplates[0].Name
				libstorage.EventuallyDVWith(vm.Namespace, targetDVName, 90, HaveSucceeded())
			}

			createPVCDataSource := func() *cdiv1.DataSource {
				dvt := &vm.Spec.DataVolumeTemplates[0]
				return &cdiv1.DataSource{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ds-" + rand.String(12),
					},
					Spec: cdiv1.DataSourceSpec{
						Source: cdiv1.DataSourceSource{
							PVC: dvt.Spec.Source.PVC,
						},
					},
				}
			}

			snapshotCloneMutateFunc := func() {
				snapshotClassName, err := libstorage.GetSnapshotClass(storageClass, virtClient)
				Expect(err).ToNot(HaveOccurred())
				if snapshotClassName == "" {
					Fail("The clone permission suite uses a snapshot-capable storage class, must have associated snapshot class")
				}

				dvt := &vm.Spec.DataVolumeTemplates[0]
				snap := libstorage.NewVolumeSnapshot(dvt.Spec.Source.PVC.Name, dvt.Spec.Source.PVC.Namespace, dvt.Spec.Source.PVC.Name, &snapshotClassName)
				snap, err = virtClient.KubernetesSnapshotClient().
					SnapshotV1().
					VolumeSnapshots(snap.Namespace).
					Create(context.Background(), snap, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				dvt.Spec.Source = &cdiv1.DataVolumeSource{
					Snapshot: &cdiv1.DataVolumeSourceSnapshot{
						Namespace: snap.Namespace,
						Name:      snap.Name,
					},
				}
			}

			createSnapshotDataSource := func() *cdiv1.DataSource {
				snapshotClassName, err := libstorage.GetSnapshotClass(storageClass, virtClient)
				Expect(err).ToNot(HaveOccurred())
				if snapshotClassName == "" {
					Fail("The clone permission suite uses a snapshot-capable storage class, must have associated snapshot class")
				}

				dvt := &vm.Spec.DataVolumeTemplates[0]
				snap := libstorage.NewVolumeSnapshot(dvt.Spec.Source.PVC.Name, dvt.Spec.Source.PVC.Namespace, dvt.Spec.Source.PVC.Name, &snapshotClassName)
				snap, err = virtClient.KubernetesSnapshotClient().
					SnapshotV1().
					VolumeSnapshots(snap.Namespace).
					Create(context.Background(), snap, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				return &cdiv1.DataSource{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ds-" + rand.String(12),
					},
					Spec: cdiv1.DataSourceSpec{
						Source: cdiv1.DataSourceSource{
							Snapshot: &cdiv1.DataVolumeSourceSnapshot{
								Namespace: snap.Namespace,
								Name:      snap.Name,
							},
						},
					},
				}
			}

			DescribeTable("should resolve DataVolume sourceRef", func(createDataSourceFunc func() *cdiv1.DataSource) {
				// convert DV to use datasource
				dvt := &vm.Spec.DataVolumeTemplates[0]
				ds := createDataSourceFunc()
				ds, err := virtClient.CdiClient().CdiV1beta1().DataSources(vm.Namespace).Create(context.TODO(), ds, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				defer func() {
					err := virtClient.CdiClient().CdiV1beta1().DataSources(ds.Namespace).Delete(context.TODO(), ds.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}()

				dvt.Spec.Source = nil
				dvt.Spec.SourceRef = &cdiv1.DataVolumeSourceRef{
					Kind: "DataSource",
					Name: ds.Name,
				}

				cloneRole, cloneRoleBinding = addClonePermission(
					virtClient,
					explicitCloneRole,
					testsuite.AdminServiceAccountName,
					testsuite.GetTestNamespace(nil),
					testsuite.NamespaceTestAlternative,
				)

				createVMSuccess()

				dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.TODO(), dvt.Name, metav1.GetOptions{})
				if libstorage.IsDataVolumeGC(virtClient) {
					Expect(errors.IsNotFound(err)).To(BeTrue())
					return
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(dv.Spec.SourceRef).To(BeNil())
				switch {
				case dv.Spec.Source.PVC != nil:
					Expect(dv.Spec.Source.PVC.Namespace).To(Equal(ds.Spec.Source.PVC.Namespace))
					Expect(dv.Spec.Source.PVC.Name).To(Equal(ds.Spec.Source.PVC.Name))
				case dv.Spec.Source.Snapshot != nil:
					Expect(dv.Spec.Source.Snapshot.Namespace).To(Equal(ds.Spec.Source.Snapshot.Namespace))
					Expect(dv.Spec.Source.Snapshot.Name).To(Equal(ds.Spec.Source.Snapshot.Name))
				default:
					Fail(fmt.Sprintf("wrong dv source %+v", dv))
				}
			},
				Entry("with PVC source", createPVCDataSource),
				Entry("with Snapshot source", createSnapshotDataSource),
			)

			It("should report DataVolume without source PVC", func() {
				cloneRole, cloneRoleBinding = addClonePermission(
					virtClient,
					explicitCloneRole,
					testsuite.AdminServiceAccountName,
					testsuite.GetTestNamespace(nil),
					testsuite.NamespaceTestAlternative,
				)

				// We first delete the source PVC and DataVolume to force a clone without source
				err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.NamespaceTestAlternative).Delete(context.Background(), dataVolume.Name, metav1.DeleteOptions{})
				if !errors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}
				err = virtClient.CoreV1().PersistentVolumeClaims(testsuite.NamespaceTestAlternative).Delete(context.Background(), dataVolume.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())

				// We check if the VM is succesfully created
				By("Creating VM")
				Eventually(func() bool {
					_, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
					if err != nil {
						return false
					}
					return true
				}, 90*time.Second, 1*time.Second).Should(BeTrue())

				// Check for owner reference
				Eventually(ThisDVWith(vm.Namespace, vm.Spec.DataVolumeTemplates[0].Name), 100).Should(BeOwned())

				// We check the expected event
				By("Expecting SourcePVCNotAvailabe event")
				Eventually(func() bool {
					events, err := virtClient.CoreV1().Events(vm.Namespace).List(context.Background(), metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, e := range events.Items {
						if e.Reason == "SourcePVCNotAvailabe" {
							return true
						}
					}
					return false
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			})

			DescribeTable("[storage-req] deny then allow clone request", decorators.StorageReq, func(role *rbacv1.Role, allServiceAccounts, allServiceAccountsInNamespace bool, cloneMutateFunc func(), fail bool) {
				if cloneMutateFunc != nil {
					cloneMutateFunc()
				}
				_, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("insufficient permissions in clone source namespace"))

				saName := testsuite.AdminServiceAccountName
				saNamespace := testsuite.GetTestNamespace(nil)

				if allServiceAccounts {
					saName = ""
					saNamespace = ""
				} else if allServiceAccountsInNamespace {
					saName = ""
				}

				// add permission
				cloneRole, cloneRoleBinding = addClonePermission(virtClient, role, saName, saNamespace, testsuite.NamespaceTestAlternative)
				if fail {
					Consistently(func() error {
						_, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
						return err
					}, 5*time.Second, 1*time.Second).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("insufficient permissions in clone source namespace"))

					return
				}
				createVMSuccess()

				// stop vm
				vm = tests.StopVirtualMachine(vm)
			},
				Entry("[test_id:3193]with explicit role", explicitCloneRole, false, false, nil, false),
				Entry("[test_id:3194]with implicit role", implicitCloneRole, false, false, nil, false),
				Entry("[test_id:5253]with explicit role (all namespaces)", explicitCloneRole, true, false, nil, false),
				Entry("[test_id:5254]with explicit role (one namespace)", explicitCloneRole, false, true, nil, false),
				Entry("with explicit role snapshot clone", explicitCloneRole, false, false, snapshotCloneMutateFunc, false),
				Entry("with implicit insufficient role snapshot clone", implicitCloneRole, false, false, snapshotCloneMutateFunc, true),
				Entry("with implicit sufficient role snapshot clone", implicitSnapshotCloneRole, false, false, snapshotCloneMutateFunc, false),
				Entry("with explicit role (all namespaces) snapshot clone", explicitCloneRole, true, false, snapshotCloneMutateFunc, false),
				Entry("with explicit role (one namespace) snapshot clone", explicitCloneRole, false, true, snapshotCloneMutateFunc, false),
			)
		})
	})

	Describe("[Serial][rfe_id:8400][crit:high][vendor:cnv-qe@redhat.com][level:system] Garbage collection of succeeded DV", Serial, func() {
		var originalTTL *int32

		BeforeEach(func() {
			cdi := libstorage.GetCDI(virtClient)
			originalTTL = cdi.Spec.Config.DataVolumeTTLSeconds
		})

		AfterEach(func() {
			libstorage.SetDataVolumeGC(virtClient, originalTTL)
		})

		DescribeTable("Verify DV of VM with DataVolumeTemplates is garbage collected when", func(ttlBefore, ttlAfter *int32, gcAnnotation string) {
			libstorage.SetDataVolumeGC(virtClient, ttlBefore)

			vm, foundSC := tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil))
			if !foundSC {
				Skip("Skip test when Filesystem storage is not present")
			}

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			vm = tests.StartVirtualMachine(vm)
			By(checkingVMInstanceConsoleExpectedOut)
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			libstorage.SetDataVolumeGC(virtClient, ttlAfter)

			dvName := vm.Spec.DataVolumeTemplates[0].Name

			if gcAnnotation != "" {
				dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.TODO(), dvName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				dv.Annotations = map[string]string{"cdi.kubevirt.io/storage.deleteAfterCompletion": gcAnnotation}
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Update(context.TODO(), dv, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			libstorage.EventuallyDVWith(vm.Namespace, dvName, 100, BeNil())

			vm = tests.StopVirtualMachine(vm)
		},
			Entry("[test_id:8567]GC is enabled", pointer.Int32(0), pointer.Int32(0), ""),
			Entry("[test_id:8571]GC is disabled, and after VM creation, GC is enabled and DV is annotated", pointer.Int32(-1), pointer.Int32(0), "true"),
		)
	})

	Context("Fedora VMI tests", func() {
		imageSizeEqual := func(a, b int64) bool {
			// Our OCS image size method probe is very precise and can show a few
			// bytes of difference.
			// A VM cannot do sub-512 byte accesses anyway, so such small size
			// differences are practically equal.
			if math.Abs((float64)(a-b)) >= 512 {
				By(fmt.Sprintf("Image sizes not equal, %d - %d >= 512", a, b))
				return false
			} else {
				return true
			}
		}
		getImageSize := func(vmi *v1.VirtualMachineInstance, dv *cdiv1.DataVolume) int64 {
			var imageSize int64
			var unused string
			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(nil))
			lsOutput, err := exec.ExecuteCommandOnPod(
				virtClient,
				pod,
				"compute",
				[]string{"ls", "-s", "/var/run/kubevirt-private/vmi-disks/disk0/disk.img"},
			)
			Expect(err).ToNot(HaveOccurred())
			if _, err := fmt.Sscanf(lsOutput, "%d %s", &imageSize, &unused); err != nil {
				return 0
			}
			return imageSize
		}

		noop := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
			return dv
		}
		addPreallocationTrue := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
			preallocation := true
			dv.Spec.Preallocation = &preallocation
			return dv
		}
		addPreallocationFalse := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
			preallocation := false
			dv.Spec.Preallocation = &preallocation
			return dv
		}
		addThickProvisionedTrueAnnotation := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
			dv.Annotations["user.custom.annotation/storage.thick-provisioned"] = "true"
			return dv
		}
		addThickProvisionedFalseAnnotation := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
			dv.Annotations["user.custom.annotation/storage.thick-provisioned"] = "false"
			return dv
		}
		DescribeTable("[rfe_id:5070][crit:medium][vendor:cnv-qe@redhat.com][level:component]fstrim from the VM influences disk.img", func(dvChange func(*cdiv1.DataVolume) *cdiv1.DataVolume, expectSmaller bool) {
			sc, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Skip("Skip test when Filesystem storage is not present")
			}

			dataVolume := libdv.NewDataVolume(
				libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling)),
				libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithVolumeSize(cd.FedoraVolumeSize)),
				libdv.WithForceBindAnnotation(), // So we can wait for DV to finish before starting the VMI
			)

			dataVolume = dvChange(dataVolume)
			preallocated := dataVolume.Spec.Preallocation != nil && *dataVolume.Spec.Preallocation

			vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
			vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus = "scsi"
			tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\n echo hello\n")

			dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Importing Fedora is so slow that we get "resourceVersion too old" when trying
			// to watch for events between the VMI creation and VMI starting.
			By("Making sure the slow Fedora import is complete before creating the VMI")
			libstorage.EventuallyDV(dataVolume, 500, HaveSucceeded())

			vmi = tests.RunVMIAndExpectLaunchWithDataVolume(vmi, dataVolume, 500)

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			imageSizeAfterBoot := getImageSize(vmi, dataVolume)
			By(fmt.Sprintf("image size after boot is %d", imageSizeAfterBoot))

			By("Filling out disk space")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "dd if=/dev/urandom of=largefile bs=1M count=300 2> /dev/null\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: syncName},
				&expect.BExp{R: console.PromptExpression},
			}, 360)).To(Succeed(), "should write a large file")

			if preallocated {
				// Preallocation means no changes to disk size
				Eventually(imageSizeEqual, 120*time.Second).WithArguments(getImageSize(vmi, dataVolume), imageSizeAfterBoot).Should(BeTrue())
			} else {
				Eventually(getImageSize, 120*time.Second).WithArguments(vmi, dataVolume).Should(BeNumerically(">", imageSizeAfterBoot))
			}

			imageSizeBeforeTrim := getImageSize(vmi, dataVolume)
			By(fmt.Sprintf("image size before trim is %d", imageSizeBeforeTrim))

			By("Writing a small file so that we detect a disk space usage change.")
			By("Deleting large file and trimming disk")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				// Write a small file so that we'll have an increase in image size if trim is unsupported.
				&expect.BSnd{S: "dd if=/dev/urandom of=smallfile bs=1M count=50 2> /dev/null\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: syncName},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "rm -f largefile\n"},
				&expect.BExp{R: console.PromptExpression},
			}, 60)).To(Succeed(), "should trim within the VM")

			Eventually(func() bool {
				By("Running trim")
				err := console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo fstrim -v /\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: syncName},
					&expect.BExp{R: console.PromptExpression},
				}, 60)
				Expect(err).ToNot(HaveOccurred())

				currentImageSize := getImageSize(vmi, dataVolume)
				if expectSmaller {
					// Trim should make the space usage go down
					By(fmt.Sprintf("We expect disk usage to go down from the use of trim.\nIt is currently %d and was previously %d", currentImageSize, imageSizeBeforeTrim))
					return currentImageSize < imageSizeBeforeTrim
				} else if preallocated {
					By(fmt.Sprintf("Trim shouldn't do anything, and preallocation should mean no change to disk usage.\nIt is currently %d and was previously %d", currentImageSize, imageSizeBeforeTrim))
					return imageSizeEqual(currentImageSize, imageSizeBeforeTrim)

				} else {
					By(fmt.Sprintf("Trim shouldn't do anything, but we expect size usage to go up, because we wrote another small file.\nIt is currently %d and was previously %d", currentImageSize, imageSizeBeforeTrim))
					return currentImageSize > imageSizeBeforeTrim
				}
			}, 120*time.Second).Should(BeTrue())

			err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("[test_id:5894]by default, fstrim will make the image smaller", noop, true),
			Entry("[test_id:5898]with preallocation true, fstrim has no effect", addPreallocationTrue, false),
			Entry("[test_id:5897]with preallocation false, fstrim will make the image smaller", addPreallocationFalse, true),
			Entry("[test_id:5899]with thick provision true, fstrim has no effect", addThickProvisionedTrueAnnotation, false),
			Entry("[test_id:5896]with thick provision false, fstrim will make the image smaller", addThickProvisionedFalseAnnotation, true),
		)
	})

	Context("With VirtualMachinePreference and PreferredStorageClassName", func() {
		var vm *v1.VirtualMachine
		var storageClass *storagev1.StorageClass
		var virtualMachinePreference *instanceType.VirtualMachinePreference

		BeforeEach(func() {
			vm, _ = tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil))

			storageClass = &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "storageclass-",
				},
				Provisioner: "default",
			}
			storageClass, err = virtClient.StorageV1().StorageClasses().Create(context.Background(), storageClass, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			virtualMachinePreference = &instanceType.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "virtualmachinepreference-test",
					Namespace: vm.Namespace,
				},
				Spec: instanceType.VirtualMachinePreferenceSpec{
					Volumes: &instanceType.VolumePreferences{
						PreferredStorageClassName: storageClass.Name,
					},
				},
			}
			_, err := virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(nil)).Create(context.Background(), virtualMachinePreference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Bind VirtualMachinePreference to VirtualMachine
			vm.Spec.Preference = &v1.PreferenceMatcher{
				Name: virtualMachinePreference.Name,
				Kind: instancetypeapi.SingularPreferenceResourceName,
			}
		})

		AfterEach(func() {
			Expect(virtClient.VirtualMachinePreference(virtualMachinePreference.Namespace).Delete(context.Background(), virtualMachinePreference.Name, metav1.DeleteOptions{})).To(Succeed())
		})

		It("should use PreferredStorageClassName when storage class not provided by VM", func() {
			// Remove storage class name from VM definition
			vm.Spec.DataVolumeTemplates[0].Spec.PVC.StorageClassName = nil

			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			Expect(*vm.Spec.DataVolumeTemplates[0].Spec.PVC.StorageClassName).To(Equal(virtualMachinePreference.Spec.Volumes.PreferredStorageClassName))
		})

		It("should always use VM defined storage class over PreferredStorageClassName", func() {
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			Expect(*vm.Spec.DataVolumeTemplates[0].Spec.PVC.StorageClassName).NotTo(Equal(virtualMachinePreference.Spec.Volumes.PreferredStorageClassName))
		})
	})
})

var explicitCloneRole = &rbacv1.Role{
	ObjectMeta: metav1.ObjectMeta{
		Name: "explicit-clone-role",
	},
	Rules: []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"cdi.kubevirt.io",
			},
			Resources: []string{
				"datavolumes/source",
			},
			Verbs: []string{
				"create",
			},
		},
	},
}

var implicitCloneRole = &rbacv1.Role{
	ObjectMeta: metav1.ObjectMeta{
		Name: "implicit-clone-role",
	},
	Rules: []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"",
			},
			Resources: []string{
				"pods",
			},
			Verbs: []string{
				"create",
			},
		},
	},
}

var implicitSnapshotCloneRole = &rbacv1.Role{
	ObjectMeta: metav1.ObjectMeta{
		Name: "implicit-clone-role",
	},
	Rules: []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"",
			},
			Resources: []string{
				"pods",
				"pvcs",
			},
			Verbs: []string{
				"create",
			},
		},
	},
}

func addClonePermission(client kubecli.KubevirtClient, role *rbacv1.Role, sa, saNamespace, targetNamesace string) (*rbacv1.Role, *rbacv1.RoleBinding) {
	role, err := client.RbacV1().Roles(targetNamesace).Create(context.Background(), role, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: role.Name,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     role.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	if sa != "" {
		rb.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: saNamespace,
			},
		}
	} else {
		g := "system:serviceaccounts"
		if saNamespace != "" {
			g += ":" + saNamespace
		}
		rb.Subjects = []rbacv1.Subject{
			{
				Kind:     "Group",
				Name:     g,
				APIGroup: "rbac.authorization.k8s.io",
			},
		}
	}

	rb, err = client.RbacV1().RoleBindings(targetNamesace).Create(context.Background(), rb, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	return role, rb
}

func volumeExpansionAllowed(sc string) bool {
	virtClient := kubevirt.Client()
	storageClass, err := virtClient.StorageV1().StorageClasses().Get(context.Background(), sc, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return storageClass.AllowVolumeExpansion != nil &&
		*storageClass.AllowVolumeExpansion
}

func newRandomVMWithCloneDataVolume(sourceNamespace, sourceName, targetNamespace, sc string) *v1.VirtualMachine {
	dataVolume := libdv.NewDataVolume(
		libdv.WithPVCSource(sourceNamespace, sourceName),
		libdv.WithPVC(libdv.PVCWithStorageClass(sc), libdv.PVCWithVolumeSize("1Gi")),
	)

	vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
	vmi.Namespace = targetNamespace
	vm := tests.NewRandomVirtualMachine(vmi, false)

	libstorage.AddDataVolumeTemplate(vm, dataVolume)
	return vm
}
