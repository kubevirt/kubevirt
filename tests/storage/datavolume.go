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
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	storagev1 "k8s.io/api/storage/v1"

	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/util"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"

	. "kubevirt.io/kubevirt/tests/framework/matcher"
	storageframework "kubevirt.io/kubevirt/tests/framework/storage"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
)

const (
	checkingVMInstanceConsoleExpectedOut = "Checking that the VirtualMachineInstance console has expected output"
	deletingDataVolume                   = "Deleting the DataVolume"
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
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()
		if !tests.HasCDI() {
			Skip("Skip DataVolume tests when CDI is not present")
		}
	})

	Context("[storage-req]PVC expansion", func() {
		table.DescribeTable("PVC expansion is detected by VM and can be fully used", func(volumeMode k8sv1.PersistentVolumeMode) {
			checks.SkipTestIfNoFeatureGate(virtconfig.ExpandDisksGate)
			if !tests.HasCDI() {
				Skip("Skip DataVolume tests when CDI is not present")
			}
			var sc string
			exists := false
			if volumeMode == k8sv1.PersistentVolumeBlock {
				sc, exists = tests.GetRWOBlockStorageClass()
				if !exists {
					Skip("Skip test when Block storage is not present")
				}
			} else {
				sc, exists = tests.GetRWOFileSystemStorageClass()
				if !exists {
					Skip("Skip test when Filesystem storage is not present")
				}
			}
			volumeExpansionAllowed := tests.VolumeExpansionAllowed(sc)
			if !volumeExpansionAllowed {
				Skip("Skip when volume expansion storage class not available")
			}
			vmi, dataVolume := tests.NewRandomVirtualMachineInstanceWithDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), util.NamespaceTestDefault, sc, k8sv1.ReadWriteOnce, volumeMode)
			tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
			vmi = tests.RunVMIAndExpectLaunch(vmi, 500)

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToCirros(vmi)).To(Succeed())

			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Get(context.Background(), dataVolume.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Expanding PVC")
			pvc.Spec.Resources.Requests[k8sv1.ResourceStorage] = resource.MustParse("2Gi")
			_, err = virtClient.CoreV1().PersistentVolumeClaims(util.NamespaceTestDefault).Update(context.Background(), pvc, metav1.UpdateOptions{})
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
			table.Entry("with Block PVC", k8sv1.PersistentVolumeBlock),
			table.Entry("with Filesystem PVC", k8sv1.PersistentVolumeFilesystem),
		)
	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachineInstance with a DataVolume as a volume source", func() {

		Context("[Serial]without fsgroup support", func() {

			ipProtocol := k8sv1.IPv4Protocol
			os := string(cd.ContainerDiskAlpine)
			size := "1Gi"

			AfterEach(func() {
				tests.DeleteAlpineWithNonQEMUPermissions()
			})

			createNFSPvAndPvc := func(ipFamily k8sv1.IPFamily, nfsPod *k8sv1.Pod) string {
				pvName := fmt.Sprintf("test-nfs%s", rand.String(48))

				// create a new PV and PVC (PVs can't be reused)
				By("create a new NFS PV and PVC")
				nfsIP := libnet.GetPodIpByFamily(nfsPod, ipFamily)
				ExpectWithOffset(1, nfsIP).NotTo(BeEmpty())

				tests.CreateNFSPvAndPvc(pvName, util.NamespaceTestDefault, size, nfsIP, os)
				return pvName
			}

			It("should succesfully start", func() {

				targetImage, nodeName := tests.CopyAlpineWithNonQEMUPermissions()

				By("Starting an NFS POD")
				nfsPod := storageframework.InitNFS(targetImage, nodeName)
				pvName := createNFSPvAndPvc(ipProtocol, nfsPod)

				// Create fake DV and new PV&PVC of that name. Otherwise VM can't be created
				dv := tests.NewRandomDataVolumeWithPVCSource(util.NamespaceTestDefault, pvName, util.NamespaceTestDefault, k8sv1.ReadWriteMany)
				dv.Spec.PVC.Resources.Requests["storage"] = resource.MustParse(size)
				_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).To(BeNil())
				tests.CreateNFSPvAndPvc(dv.Name, util.NamespaceTestDefault, size, libnet.GetPodIpByFamily(nfsPod, ipProtocol), os)

				vmi := tests.NewRandomVMIWithDataVolume(dv.Name)

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

				dataVolume := tests.NewRandomDataVolumeWithRegistryImport(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)

				_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dataVolume.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).To(BeNil())

				// This will only work on storage with binding mode WaitForFirstConsumer,
				if tests.IsStorageClassBindingModeWaitForFirstConsumer(tests.Config.StorageRWOFileSystem) {
					Eventually(ThisDV(dataVolume), 30).Should(BeInPhase(cdiv1.WaitForFirstConsumer))
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

					err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
					Expect(err).To(BeNil())
					tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				}
				err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dataVolume.Namespace).Delete(context.Background(), dataVolume.Name, metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})

			It("[test_id:6686]should successfully start multiple concurrent VMIs", func() {

				numVmis := 5
				vmis := make([]*v1.VirtualMachineInstance, 0, numVmis)
				dvs := make([]*cdiv1.DataVolume, 0, numVmis)

				for idx := 0; idx < numVmis; idx++ {
					dataVolume := tests.NewRandomDataVolumeWithRegistryImport(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, k8sv1.ReadWriteOnce)
					vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)

					_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dataVolume.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
					Expect(err).To(BeNil())

					vmi = tests.RunVMI(vmi, 60)
					vmis = append(vmis, vmi)
					dvs = append(dvs, dataVolume)
				}

				for idx := 0; idx < numVmis; idx++ {
					tests.WaitForSuccessfulVMIStartWithTimeoutIgnoreWarnings(vmis[idx], 500)
					By(checkingVMInstanceConsoleExpectedOut)
					Expect(console.LoginToAlpine(vmis[idx])).To(Succeed())

					err := virtClient.VirtualMachineInstance(vmis[idx].Namespace).Delete(vmis[idx].Name, &metav1.DeleteOptions{})
					Expect(err).To(BeNil())
					err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dvs[idx].Namespace).Delete(context.Background(), dvs[idx].Name, metav1.DeleteOptions{})
					Expect(err).To(BeNil())
				}
			})

			It("[test_id:5252]should be successfully started when using a PVC volume owned by a DataVolume", func() {
				dataVolume := tests.NewRandomDataVolumeWithRegistryImport(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				vmi := tests.NewRandomVMIWithPVC(dataVolume.Name)

				_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dataVolume.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).To(BeNil())
				// This will only work on storage with binding mode WaitForFirstConsumer,
				if tests.IsStorageClassBindingModeWaitForFirstConsumer(tests.Config.StorageRWOFileSystem) {
					Eventually(ThisDV(dataVolume), 60).Should(BeInPhase(cdiv1.WaitForFirstConsumer))
				}
				// with WFFC the run actually starts the import and then runs VM, so the timeout has to include both
				// import and start
				vmi = tests.RunVMIAndExpectLaunchWithDataVolume(vmi, dataVolume, 500)

				By(checkingVMInstanceConsoleExpectedOut)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

				err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dataVolume.Namespace).Delete(context.Background(), dataVolume.Name, metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})
		})

		Context("with a PVC from a Datavolume", func() {
			var storageClass *storagev1.StorageClass
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
			})
			AfterEach(func() {
				if storageClass != nil && storageClass.Name != "" {
					err := virtClient.StorageV1().StorageClasses().Delete(context.Background(), storageClass.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("[test_id:4643]should NOT be rejected when VM template lists a DataVolume, but VM lists PVC VolumeSource", func() {

				dv := tests.NewRandomDataVolumeWithRegistryImportInStorageClass(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, storageClass.Name, k8sv1.ReadWriteOnce, k8sv1.PersistentVolumeFilesystem)
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).To(BeNil())

				defer func(dv *cdiv1.DataVolume) {
					By(deletingDataVolume)
					ExpectWithOffset(1, virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Delete(context.Background(), dv.Name, metav1.DeleteOptions{})).To(Succeed(), metav1.DeleteOptions{})
				}(dv)

				Eventually(func() (*k8sv1.PersistentVolumeClaim, error) {
					return virtClient.CoreV1().PersistentVolumeClaims(dv.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
				}, 30).Should(Not(BeNil()))

				vmi := tests.NewRandomVMI()

				diskName := "disk0"
				bus := "virtio"
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: diskName,
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: bus,
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: diskName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: dv.ObjectMeta.Name,
						}},
					},
				})

				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")

				vm := tests.NewRandomVirtualMachine(vmi, true)
				dvt := &v1.DataVolumeTemplateSpec{
					ObjectMeta: dv.ObjectMeta,
					Spec:       dv.Spec,
				}
				vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dvt)
				_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
			})
			It("[Serial][test_id:4644]should fail to start when a volume is backed by PVC created by DataVolume instead of the DataVolume itself", func() {
				dv := tests.NewRandomDataVolumeWithRegistryImportInStorageClass(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault, storageClass.Name, k8sv1.ReadWriteOnce, k8sv1.PersistentVolumeFilesystem)
				_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).To(BeNil())

				defer func(dv *cdiv1.DataVolume) {
					By(deletingDataVolume)
					ExpectWithOffset(1, virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Delete(context.Background(), dv.Name, metav1.DeleteOptions{})).To(Succeed(), metav1.DeleteOptions{})
				}(dv)
				Eventually(func() error {
					_, err := virtClient.CoreV1().PersistentVolumeClaims(dv.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
					return err
				}, 30*time.Second, 1*time.Second).Should(BeNil())

				vmi := tests.NewRandomVMI()

				diskName := "disk0"
				bus := "virtio"
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: diskName,
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: bus,
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: diskName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: dv.ObjectMeta.Name,
						}},
					},
				})

				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")

				vm := tests.NewRandomVirtualMachine(vmi, true)
				_, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ShouldNot(HaveOccurred())

				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return vm.Status.Created
				}, 30*time.Second, 1*time.Second).Should(Equal(false))
			})
		})
	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachine with an invalid DataVolume", func() {
		Context("using DataVolume with invalid URL", func() {
			deleteDataVolume := func(dv *cdiv1.DataVolume) {
				By(deletingDataVolume)
				ExpectWithOffset(1, virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Delete(context.Background(), dv.Name, metav1.DeleteOptions{})).To(Succeed(), metav1.DeleteOptions{})
			}

			It("shold be possible to stop VM if datavolume is crashing", func() {
				dataVolume := tests.NewRandomDataVolumeWithRegistryImport(InvalidDataVolumeUrl, util.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				vm := tests.NewRandomVirtualMachine(tests.NewRandomVMIWithDataVolume(dataVolume.Name), true)
				vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{
					{
						ObjectMeta: dataVolume.ObjectMeta,
						Spec:       dataVolume.Spec,
					},
				}

				By(creatingVMInvalidDataVolume)
				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for DV to start crashing")
				Eventually(ThisDV(dataVolume), 60).Should(BeInPhase(cdiv1.ImportInProgress))

				By("Stop VM")
				tests.StopVirtualMachineWithTimeout(vm, time.Second*30)
			})

			It("[test_id:3190]should correctly handle invalid DataVolumes", func() {
				// Don't actually create the DataVolume since it's invalid.
				dataVolume := tests.NewRandomDataVolumeWithRegistryImport(InvalidDataVolumeUrl, util.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				//  Add the invalid DataVolume to a VMI
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
				// Create a VM for this VMI
				vm := tests.NewRandomVirtualMachine(vmi, true)

				By(creatingVMInvalidDataVolume)
				vm, err = virtClient.VirtualMachine(vm.Namespace).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to be created")
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 100).Should(BeInPhase(v1.Pending))
			})
			It("[test_id:3190]should correctly handle eventually consistent DataVolumes", func() {
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

				dataVolume := tests.NewRandomDataVolumeWithRegistryImport(cd.DataVolumeImportUrlFromRegistryForContainerDisk(fakeRegistryWithPort, cd.ContainerDiskCirros),
					util.NamespaceTestDefault,
					k8sv1.ReadWriteOnce,
				)
				defer deleteDataVolume(dataVolume)

				By("Creating DataVolume with invalid URL")
				dataVolume, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dataVolume.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).To(BeNil())

				By(creatingVMInvalidDataVolume)
				//  Add the invalid DataVolume to a VMI
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
				// Create a VM for this VMI
				vm := tests.NewRandomVirtualMachine(vmi, true)
				vm, err = virtClient.VirtualMachine(vm.Namespace).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 100).Should(BeInPhase(v1.Pending))

				By("Creating a service which makes the registry reachable")
				virtClient.CoreV1().Services(vm.Namespace).Create(context.Background(), &k8sv1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: fakeRegistryName,
					},
					Spec: k8sv1.ServiceSpec{
						Type:         k8sv1.ServiceTypeExternalName,
						ExternalName: realRegistryName,
					},
				}, metav1.CreateOptions{})

				By("Wait for DataVolume to complete")
				Eventually(ThisDV(dataVolume), 160).Should(HaveSucceeded())

				By("Waiting for VMI to be created")
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 100).Should(BeInPhase(v1.Running))
			})
		})
	})

	Describe("[rfe_id:896][crit:high][vendor:cnv-qe@redhat.com][level:system] with oc/kubectl", func() {
		var vm *v1.VirtualMachine
		var err error
		var workDir string
		var vmJson string
		var dataVolumeName string
		var pvcName string

		k8sClient := tests.GetK8sCmdClient()

		BeforeEach(func() {
			running := true

			vm = tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault)
			vm.Spec.Running = &running

			dataVolumeName = vm.Spec.DataVolumeTemplates[0].Name
			pvcName = dataVolumeName

			workDir, err := ioutil.TempDir("", tests.TempDirPrefix+"-")
			Expect(err).ToNot(HaveOccurred())
			vmJson, err = tests.GenerateVMJson(vm, workDir)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			if workDir != "" {
				err = os.RemoveAll(workDir)
				Expect(err).ToNot(HaveOccurred())
				workDir = ""
			}
		})

		It("[test_id:836] Creating a VM with DataVolumeTemplates should succeed.", func() {
			By(creatingVMDataVolumeTemplateEntry)
			_, _, err = tests.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By(verifyingDataVolumeSuccess)
			Eventually(ThisDVWith(vm.Namespace, dataVolumeName), 100).Should(And(HaveSucceeded(), BeOwned()))

			By(verifyingPVCCreated)
			Eventually(ThisPVCWith(vm.Namespace, pvcName), 160).Should(Exist())

			By(verifyingVMICreated)
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 160).Should(And(BeRunning(), BeOwned()))
		})

		It("[test_id:837]deleting VM with cascade=true should automatically delete DataVolumes and VMI owned by VM.", func() {
			By(creatingVMDataVolumeTemplateEntry)
			_, _, err = tests.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By(verifyingDataVolumeSuccess)
			Eventually(ThisDVWith(vm.Namespace, dataVolumeName), 100).Should(And(HaveSucceeded(), BeOwned()))

			By(verifyingPVCCreated)
			Eventually(ThisPVCWith(vm.Namespace, pvcName), 160).Should(Exist())

			By(verifyingVMICreated)
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 160).Should(And(BeRunning(), BeOwned()))

			By("Deleting VM with cascade=true")
			_, _, err = tests.RunCommand("kubectl", "delete", "vm", vm.Name, "--cascade=true")
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
			By(creatingVMDataVolumeTemplateEntry)
			_, _, err = tests.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By(verifyingDataVolumeSuccess)
			Eventually(ThisDVWith(vm.Namespace, dataVolumeName), 100).Should(And(HaveSucceeded(), BeOwned()))

			By(verifyingPVCCreated)
			Eventually(ThisPVCWith(vm.Namespace, pvcName), 160).Should(Exist())

			By(verifyingVMICreated)
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 160).Should(And(BeRunning(), BeOwned()))

			By("Deleting VM with cascade=false")
			_, _, err = tests.RunCommand("kubectl", "delete", "vm", vm.Name, "--cascade=false")
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VM to be deleted")
			Eventually(ThisVM(vm), 100).Should(BeGone())

			By("Verifying DataVolume still exists with owner references removed")
			Eventually(ThisDVWith(vm.Namespace, dataVolumeName), 100).Should(And(HaveSucceeded(), Not(BeOwned())))

			By("Verifying VMI still exists with owner references removed")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 100).Should(And(BeRunning(), Not(BeOwned())))
		})

	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachine with a DataVolume", func() {
		Context("using Alpine http import", func() {
			It("a DataVolume with preallocation shouldn't have discard=unmap", func() {
				var vm *v1.VirtualMachine
				vm = tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault)
				preallocation := true
				vm.Spec.DataVolumeTemplates[0].Spec.Preallocation = &preallocation

				vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				vm = tests.StartVirtualMachine(vm)
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).ToNot(ContainSubstring("discard='unmap'"))
				vm = tests.StopVirtualMachine(vm)
				Expect(virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})).To(Succeed())
			})

			table.DescribeTable("[test_id:3191]should be successfully started and stopped multiple times", func(isHTTP bool) {
				var vm *v1.VirtualMachine
				if isHTTP {
					vm = tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault)
				} else {
					url := cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)
					vm = tests.NewRandomVMWithDataVolume(url, util.NamespaceTestDefault)
				}
				vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
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
						vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						Expect(console.LoginToAlpine(vmi)).To(Succeed())
					}
					vm = tests.StopVirtualMachine(vm)
				}
			},

				table.Entry("with http import", true),
				table.Entry("with registry import", false),
			)

			It("[test_id:3192]should remove owner references on DataVolume if VM is orphan deleted.", func() {
				vm := tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), util.NamespaceTestDefault)
				vm, err = virtClient.VirtualMachine(util.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				// Check for owner reference
				Eventually(ThisDVWith(vm.Namespace, vm.Spec.DataVolumeTemplates[0].Name), 100).Should(BeOwned())

				// Delete the VM with orphan Propagation
				orphanPolicy := metav1.DeletePropagationOrphan
				Expect(virtClient.VirtualMachine(vm.Namespace).
					Delete(vm.Name, &metav1.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())

				// Wait for the owner reference to disappear
				Eventually(ThisDVWith(vm.Namespace, vm.Spec.DataVolumeTemplates[0].Name), 100).Should(Not(BeOwned()))
			})
		})
	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] DataVolume clone permission checking", func() {
		Context("using Alpine import/clone", func() {
			var dataVolume *cdiv1.DataVolume
			var createdVirtualMachine *v1.VirtualMachine
			var cloneRole *rbacv1.Role
			var cloneRoleBinding *rbacv1.RoleBinding
			var storageClass string
			var vm *v1.VirtualMachine

			BeforeEach(func() {
				var exists bool
				storageClass, exists = tests.GetRWOFileSystemStorageClass()
				if !exists {
					Skip("Skip test when RWOFileSystem storage class is not present")
				}
				var err error
				dv := tests.NewRandomDataVolumeWithRegistryImportInStorageClass(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), tests.NamespaceTestAlternative, storageClass, k8sv1.ReadWriteOnce, k8sv1.PersistentVolumeFilesystem)
				tests.SetDataVolumeForceBindAnnotation(dv)
				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(ThisDV(dataVolume), 90).Should(HaveSucceeded())

				vm = tests.NewRandomVMWithCloneDataVolume(dataVolume.Namespace, dataVolume.Name, util.NamespaceTestDefault)
				const volumeName = "sa"
				saVol := v1.Volume{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						ServiceAccount: &v1.ServiceAccountVolumeSource{
							ServiceAccountName: tests.AdminServiceAccountName,
						},
					},
				}
				vm.Spec.DataVolumeTemplates[0].Spec.PVC.StorageClassName = pointer.StringPtr(storageClass)
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, saVol)
				vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{Name: volumeName})
			})

			AfterEach(func() {
				if cloneRole != nil {
					err := virtClient.RbacV1().Roles(cloneRole.Namespace).Delete(context.Background(), cloneRole.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}

				if cloneRoleBinding != nil {
					err := virtClient.RbacV1().RoleBindings(cloneRoleBinding.Namespace).Delete(context.Background(), cloneRoleBinding.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}

				if createdVirtualMachine != nil {
					err := virtClient.VirtualMachine(createdVirtualMachine.Namespace).Delete(createdVirtualMachine.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			createVmSuccess := func() {
				// sometimes it takes a bit for permission to actually be applied so eventually
				Eventually(func() bool {
					_, err = virtClient.VirtualMachine(vm.Namespace).Create(vm)
					if err != nil {
						fmt.Printf("command should have succeeded maybe new permissions not applied yet\nerror\n%s\n", err)
						return false
					}
					return true
				}, 90*time.Second, 1*time.Second).Should(BeTrue())

				createdVirtualMachine = vm

				// start vm and check dv clone succeeded
				createdVirtualMachine = tests.StartVirtualMachine(createdVirtualMachine)
				targetDVName := vm.Spec.DataVolumeTemplates[0].Name
				Eventually(ThisDVWith(createdVirtualMachine.Namespace, targetDVName), 90).Should(HaveSucceeded())
			}

			It("should resolve DataVolume sourceRef", func() {
				// convert DV to use datasource
				dvt := &vm.Spec.DataVolumeTemplates[0]
				ds := &cdiv1.DataSource{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ds-" + rand.String(12),
					},
					Spec: cdiv1.DataSourceSpec{
						Source: cdiv1.DataSourceSource{
							PVC: dvt.Spec.Source.PVC,
						},
					},
				}
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
					tests.AdminServiceAccountName,
					util.NamespaceTestDefault,
					tests.NamespaceTestAlternative,
				)

				createVmSuccess()

				dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.TODO(), dvt.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(dv.Spec.SourceRef).To(BeNil())
				Expect(dv.Spec.Source.PVC.Namespace).To(Equal(ds.Spec.Source.PVC.Namespace))
				Expect(dv.Spec.Source.PVC.Name).To(Equal(ds.Spec.Source.PVC.Name))
			})

			table.DescribeTable("[storage-req] deny then allow clone request", func(role *rbacv1.Role, allServiceAccounts, allServiceAccountsInNamespace bool) {
				_, err := virtClient.VirtualMachine(vm.Namespace).Create(vm)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Authorization failed, message is:"))

				saName := tests.AdminServiceAccountName
				saNamespace := util.NamespaceTestDefault

				if allServiceAccounts {
					saName = ""
					saNamespace = ""
				} else if allServiceAccountsInNamespace {
					saName = ""
				}

				// add permission
				cloneRole, cloneRoleBinding = addClonePermission(virtClient, role, saName, saNamespace, tests.NamespaceTestAlternative)

				createVmSuccess()

				// stop vm
				createdVirtualMachine = tests.StopVirtualMachine(createdVirtualMachine)
			},
				table.Entry("[test_id:3193]with explicit role", explicitCloneRole, false, false),
				table.Entry("[test_id:3194]with implicit role", implicitCloneRole, false, false),
				table.Entry("[test_id:5253]with explicit role (all namespaces)", explicitCloneRole, true, false),
				table.Entry("[test_id:5254]with explicit role (one namespace)", explicitCloneRole, false, true),
			)
		})
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
			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
			lsOutput, err := tests.ExecuteCommandOnPod(
				virtClient,
				pod,
				"compute",
				[]string{"ls", "-s", "/var/run/kubevirt-private/vmi-disks/disk0/disk.img"},
			)
			Expect(err).ToNot(HaveOccurred())
			fmt.Sscanf(lsOutput, "%d %s", &imageSize, &unused)
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
			dv.Annotations = map[string]string{"user.custom.annotation/storage.thick-provisioned": "true"}
			return dv
		}
		addThickProvisionedFalseAnnotation := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
			dv.Annotations = map[string]string{"user.custom.annotation/storage.thick-provisioned": "false"}
			return dv
		}
		table.DescribeTable("[QUARANTINE][rfe_id:5070][crit:medium][vendor:cnv-qe@redhat.com][level:component]fstrim from the VM influences disk.img", func(dvChange func(*cdiv1.DataVolume) *cdiv1.DataVolume, expectSmaller bool) {
			dataVolume := tests.NewRandomDataVolumeWithRegistryImport(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling), util.NamespaceTestDefault, k8sv1.ReadWriteOnce)
			dataVolume.Spec.PVC.Resources.Requests[k8sv1.ResourceStorage] = resource.MustParse("5Gi")
			dataVolume = dvChange(dataVolume)
			preallocated := dataVolume.Spec.Preallocation != nil && *dataVolume.Spec.Preallocation

			vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
			vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus = "scsi"
			tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\n echo hello\n")

			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dataVolume.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi = tests.RunVMIAndExpectLaunchWithDataVolume(vmi, dataVolume, 500)

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			imageSizeAfterBoot := getImageSize(vmi, dataVolume)
			By(fmt.Sprintf("image size after boot is %d", imageSizeAfterBoot))

			By("Filling out disk space")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "dd if=/dev/urandom of=largefile bs=1M count=100 2> /dev/null\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: syncName},
				&expect.BExp{R: console.PromptExpression},
			}, 360)).To(Succeed(), "should write a large file")

			if preallocated {
				// Preallocation means no changes to disk size
				Eventually(imageSizeEqual(getImageSize(vmi, dataVolume), imageSizeAfterBoot), 120*time.Second).Should(BeTrue())
			} else {
				Eventually(getImageSize(vmi, dataVolume), 120*time.Second).Should(BeNumerically(">", imageSizeAfterBoot))
			}

			imageSizeBeforeTrim := getImageSize(vmi, dataVolume)
			By(fmt.Sprintf("image size before trim is %d", imageSizeBeforeTrim))

			By("Writing a small file so that we detect a disk space usage change.")
			By("Deleting large file and trimming disk")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				// Write a small file so that we'll have an increase in image size if trim is unsupported.
				&expect.BSnd{S: "dd if=/dev/urandom of=smallfile bs=1M count=20 2> /dev/null\n"},
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

			err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})
			Expect(err).To(BeNil())
		},
			table.Entry("[test_id:5894]by default, fstrim will make the image smaller", noop, true),
			table.Entry("[test_id:5898]with preallocation true, fstrim has no effect", addPreallocationTrue, false),
			table.Entry("[test_id:5897]with preallocation false, fstrim will make the image smaller", addPreallocationFalse, true),
			table.Entry("[test_id:5899]with thick provision true, fstrim has no effect", addThickProvisionedTrueAnnotation, false),
			table.Entry("[test_id:5896]with thick provision false, fstrim will make the image smaller", addThickProvisionedFalseAnnotation, true),
		)
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
