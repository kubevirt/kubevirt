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
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
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
	"k8s.io/client-go/kubernetes"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
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

var _ = Describe(SIG("DataVolume Integration", func() {

	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		if !libstorage.HasCDI() {
			Fail("Fail DataVolume tests when CDI is not present")
		}
	})

	getImageSize := func(vmi *v1.VirtualMachineInstance, dv *cdiv1.DataVolume) int64 {
		var imageSize int64
		var unused string
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).ToNot(HaveOccurred())

		lsOutput, err := exec.ExecuteCommandOnPod(
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

	getVirtualSize := func(vmi *v1.VirtualMachineInstance, dv *cdiv1.DataVolume) int64 {
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).ToNot(HaveOccurred())

		output, err := exec.ExecuteCommandOnPod(
			pod,
			"compute",
			[]string{"qemu-img", "info", "--output", "json", "/var/run/kubevirt-private/vmi-disks/disk0/disk.img"},
		)
		Expect(err).ToNot(HaveOccurred())

		var info struct {
			VirtualSize int64 `json:"virtual-size"`
		}
		err = json.Unmarshal([]byte(output), &info)
		Expect(err).ToNot(HaveOccurred())

		return info.VirtualSize
	}

	createAndWaitForVMIReady := func(vmi *v1.VirtualMachineInstance, dataVolume *cdiv1.DataVolume) *v1.VirtualMachineInstance {
		vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		By("Waiting until the DataVolume is ready")
		libstorage.EventuallyDV(dataVolume, 500, HaveSucceeded())
		By("Waiting until the VirtualMachineInstance starts")
		return libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Running}, libwait.WithTimeout(500))
	}

	Context("[storage-req]PVC expansion", decorators.StorageReq, decorators.RequiresVolumeExpansion, func() {
		DescribeTable("PVC expansion is detected by VM and can be fully used", func(volumeMode k8sv1.PersistentVolumeMode) {
			checks.SkipTestIfNoFeatureGate(featuregate.ExpandDisksGate)
			var sc string
			exists := false
			if volumeMode == k8sv1.PersistentVolumeBlock {
				sc, exists = libstorage.GetRWOBlockStorageClass()
				if !exists {
					Fail("Fail test when Block storage is not present")
				}
			} else {
				sc, exists = libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Fail("Fail test when Filesystem storage is not present")
				}
			}
			volumeExpansionAllowed := volumeExpansionAllowed(sc)
			if !volumeExpansionAllowed {
				Fail("Fail when volume expansion storage class not available")
			}

			imageUrl := cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)
			dataVolume := libdv.NewDataVolume(
				libdv.WithRegistryURLSourceAndPullMethod(imageUrl, cdiv1.RegistryPullNode),
				libdv.WithStorage(
					libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeSize(cd.CirrosVolumeSize),
					libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce),
					libdv.StorageWithVolumeMode(volumeMode),
				),
			)
			dataVolume, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := libstorage.RenderVMIWithDataVolume(dataVolume.Name, dataVolume.Namespace, libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()))
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 500)

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToCirros(vmi)).To(Succeed())

			By("Expanding PVC")
			pvc, err := k8s.Client().CoreV1().PersistentVolumeClaims(dataVolume.Namespace).Get(context.Background(), dataVolume.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			origSize, exists := pvc.Status.Capacity[k8sv1.ResourceStorage]
			Expect(exists).To(BeTrue())
			newSize := *resource.NewQuantity(2*origSize.Value(), origSize.Format)
			patchSet := patch.New(
				patch.WithAdd("/spec/resources/requests/storage", newSize),
			)
			patchData, err := patchSet.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			_, err = k8s.Client().CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(nil)).Patch(context.Background(), dataVolume.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for notification about size change")
			Eventually(func() error {
				err := console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: ""},
					&expect.BSnd{S: "dmesg |grep 'new size'\n"},
					&expect.BExp{R: ""},
					&expect.BSnd{S: "dmesg |grep -c 'new size: [1-9]'\n"},
					&expect.BExp{R: "1"},
				}, 10)
				return err
			}, 360).Should(Succeed())

			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "sudo /sbin/resize-filesystem /dev/root /run/resize.rootfs /dev/console && echo $?\n"},
				&expect.BExp{R: "0"},
			}, 30)).To(Succeed(), "failed to resize root")

			By("Writing a 1.5G file after expansion, should succeed")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "dd if=/dev/zero of=largefile count=1500 bs=1M; echo $?\n"},
				&expect.BExp{R: "0"},
			}, 360)).To(Succeed(), "can use more space after expansion and resize")
		},
			Entry("with Block PVC", decorators.RequiresBlockStorage, k8sv1.PersistentVolumeBlock),
			Entry("with Filesystem PVC", k8sv1.PersistentVolumeFilesystem),
		)

		It("Check disk expansion accounts for actual usable size", func() {
			checks.SkipTestIfNoFeatureGate(featuregate.ExpandDisksGate)

			sc, exists := libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Fail("Fail test when Filesystem storage is not present")
			}

			volumeExpansionAllowed := volumeExpansionAllowed(sc)
			if !volumeExpansionAllowed {
				Fail("Fail when volume expansion storage class not available")
			}
			dataVolume := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithStorage(
					libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeSize("512Mi"),
					libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce),
					libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeFilesystem),
				),
			)
			dataVolume, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(dataVolume, 100, HaveSucceeded())
			pvc, err := k8s.Client().CoreV1().PersistentVolumeClaims(dataVolume.Namespace).Get(context.Background(), dataVolume.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			executorPod := createExecutorPodWithPVC("size-detection", pvc)
			fstatOutput, err := exec.ExecuteCommandOnPod(
				executorPod,
				executorPod.Spec.Containers[0].Name,
				[]string{"stat", "-f", "-c", "%a %s", libstorage.DefaultPvcMountPath},
			)
			Expect(err).ToNot(HaveOccurred())
			var freeBlocks, ioBlockSize int64
			_, err = fmt.Sscanf(fstatOutput, "%d %d", &freeBlocks, &ioBlockSize)
			Expect(err).ToNot(HaveOccurred())
			freeSize := freeBlocks * ioBlockSize

			vmi := libstorage.RenderVMIWithDataVolume(dataVolume.Name, dataVolume.Namespace)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 500)

			// Let's wait for VMI to be ready
			Eventually(func() bool {
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, volStatus := range vmi.Status.VolumeStatus {
					if volStatus.Name == "disk0" {
						return true
					}
				}
				return false
			}, 30*time.Second, time.Second).Should(BeTrue(), "Expected VolumeStatus for 'disk0' to be available")

			Expect(getVirtualSize(vmi, dataVolume)).ToNot(BeNumerically(">", freeSize))
		})
	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachineInstance with a DataVolume as a volume source", func() {
		Context("Alpine import", func() {
			It("[test_id:3189]should be successfully started and stopped multiple times", decorators.Conformance, func() {
				sc, exists := libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Fail("Fail test when Filesystem storage is not present")
				}

				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
					libdv.WithStorage(libdv.StorageWithStorageClass(sc)),
				)

				vmi := libstorage.RenderVMIWithDataVolume(dataVolume.Name, testsuite.GetTestNamespace(nil))

				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// This will only work on storage with binding mode WaitForFirstConsumer,
				if libstorage.IsStorageClassBindingModeWaitForFirstConsumer(libstorage.Config.StorageRWOFileSystem) {
					Eventually(ThisDV(dataVolume), 40).Should(WaitForFirstConsumer())
				}
				num := 2
				By("Starting and stopping the VirtualMachineInstance a number of times")
				for i := 1; i <= num; i++ {
					vmi := createAndWaitForVMIReady(vmi, dataVolume)
					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By(checkingVMInstanceConsoleExpectedOut)
						Expect(console.LoginToAlpine(vmi)).To(Succeed())
					}

					err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				}
			})

			It("[test_id:6686]should successfully start multiple concurrent VMIs", func() {

				sc, exists := libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Fail("Fail test when Filesystem storage is not present")
				}

				imageUrl := cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)

				const numVmis = 5
				vmis := make([]*v1.VirtualMachineInstance, 0, numVmis)
				dvs := make([]*cdiv1.DataVolume, 0, numVmis)

				for idx := 0; idx < numVmis; idx++ {
					dataVolume := libdv.NewDataVolume(
						libdv.WithRegistryURLSource(imageUrl),
						libdv.WithStorage(libdv.StorageWithStorageClass(sc)),
					)

					vmi := libstorage.RenderVMIWithDataVolume(dataVolume.Name, testsuite.GetTestNamespace(nil))

					dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

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

					err := virtClient.VirtualMachineInstance(vmis[idx].Namespace).Delete(context.Background(), vmis[idx].Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("[test_id:5252]should be successfully started when using a PVC volume owned by a DataVolume", func() {
				sc, exists := libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Fail("Fail test when Filesystem storage is not present")
				}

				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
					libdv.WithStorage(libdv.StorageWithStorageClass(sc)),
				)

				vmi := libstorage.RenderVMIWithDataVolume(dataVolume.Name, testsuite.GetTestNamespace(nil))

				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				// This will only work on storage with binding mode WaitForFirstConsumer,
				if libstorage.IsStorageClassBindingModeWaitForFirstConsumer(libstorage.Config.StorageRWOFileSystem) {
					Eventually(ThisDV(dataVolume), 40).Should(WaitForFirstConsumer())
				}
				// with WFFC the run actually starts the import and then runs VM, so the timeout has to include both
				// import and start
				vmi = createAndWaitForVMIReady(vmi, dataVolume)

				By(checkingVMInstanceConsoleExpectedOut)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
			})

			It("should accurately aggregate DataVolume conditions from many DVs", func() {
				dataVolume1 := libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
					libdv.WithStorage(),
				)
				dataVolume2 := libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
					libdv.WithStorage(),
				)

				By("requiring a VM with 2 DataVolumes")
				vmi := libvmi.New(
					libvmi.WithMemoryRequest("128Mi"),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithDataVolume(dataVolume1.Name, dataVolume1.Name),
					libvmi.WithDataVolume(dataVolume2.Name, dataVolume2.Name),
					libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
				)
				vmSpec := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

				vm, err := virtClient.VirtualMachine(vmSpec.Namespace).Create(context.Background(), vmSpec, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(ThisVM(vm), 180*time.Second, 2*time.Second).Should(HavePrintableStatus(v1.VirtualMachineStatusPvcNotFound))

				By("creating the first DataVolume")
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vmSpec.Namespace).Create(context.Background(), dataVolume1, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("ensuring that VMI and VM are reporting VirtualMachineInstanceDataVolumesReady=False")
				Eventually(ThisVMI(vmi), 180*time.Second, 2*time.Second).Should(HaveConditionFalse(v1.VirtualMachineInstanceDataVolumesReady))
				Eventually(ThisVM(vm), 180*time.Second, 2*time.Second).Should(HaveConditionFalse(v1.VirtualMachineInstanceDataVolumesReady))

				By("creating the second DataVolume")
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vmSpec.Namespace).Create(context.Background(), dataVolume2, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("ensuring that VMI and VM are reporting VirtualMachineInstanceDataVolumesReady=True")
				Eventually(ThisVMI(vmi), 240*time.Second, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstanceDataVolumesReady))
				Eventually(ThisVM(vm), 240*time.Second, 2*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstanceDataVolumesReady))
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
				storageClass, err = k8s.Client().StorageV1().StorageClasses().Create(context.Background(), storageClass, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				dv = libdv.NewDataVolume(
					libdv.WithNamespace(testsuite.GetTestNamespace(nil)), // need it for deletion. Reading it from Create() does not work here
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
					libdv.WithPVC(libdv.PVCWithStorageClass(storageClass.Name)),
				)
				vmi = libstorage.RenderVMIWithDataVolume(dv.Name, dv.Namespace)
			})
			AfterEach(func() {
				if storageClass != nil && storageClass.Name != "" {
					err := k8s.Client().StorageV1().StorageClasses().Delete(context.Background(), storageClass.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("[test_id:4643]should NOT be rejected when VM template lists a DataVolume, but VM lists PVC VolumeSource", func() {
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() error {
					_, err := k8s.Client().CoreV1().PersistentVolumeClaims(testsuite.GetTestNamespace(dv)).Get(context.Background(), dv.Name, metav1.GetOptions{})
					return err
				}, 30*time.Second, 1*time.Second).Should(Succeed())

				vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
				dvt := &v1.DataVolumeTemplateSpec{
					ObjectMeta: dv.ObjectMeta,
					Spec:       dv.Spec,
				}
				vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dvt)
				_, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachine with an invalid DataVolume", func() {
		Context("using DataVolume with invalid URL", func() {
			It("should be possible to stop VM if datavolume is crashing", func() {
				sc, exists := libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Fail("Fail test when Filesystem storage is not present")
				}

				dataVolume := libdv.NewDataVolume(
					libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
					libdv.WithRegistryURLSourceAndPullMethod(InvalidDataVolumeUrl, cdiv1.RegistryPullPod),
					libdv.WithStorage(libdv.StorageWithStorageClass(sc)),
				)

				vm := libstorage.RenderVMWithDataVolumeTemplate(dataVolume, libvmi.WithRunStrategy(v1.RunStrategyAlways))

				By(creatingVMInvalidDataVolume)
				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for DV to start crashing")
				Eventually(ThisDVWith(vm.Namespace, dataVolume.Name), 60).Should(BeInPhase(cdiv1.ImportInProgress))

				By("Stop VM")
				libvmops.StopVirtualMachineWithTimeout(vm, time.Second*30)
			})

			It("[test_id:3190]should correctly handle invalid DataVolumes", func() {
				// Don't actually create the DataVolume since it's invalid.
				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSource(InvalidDataVolumeUrl),
					libdv.WithStorage(libdv.StorageWithStorageClass("fakeStorageClass")),
				)

				//  Add the invalid DataVolume to a VMI
				vmi := libstorage.RenderVMIWithDataVolume(dataVolume.Name, testsuite.GetTestNamespace(nil))
				// Create a VM for this VMI
				vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

				By(creatingVMInvalidDataVolume)
				vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to be created")
				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 100).Should(BeInPhase(v1.Pending))
			})
			It("[test_id:3190]should correctly handle eventually consistent DataVolumes", func() {
				sc, exists := libstorage.GetRWOFileSystemStorageClass()
				if !exists {
					Fail("Fail test when Filesystem storage is not present")
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
					libdv.WithStorage(libdv.StorageWithStorageClass(sc)),
				)

				By("Creating DataVolume with invalid URL")
				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By(creatingVMInvalidDataVolume)
				//  Add the invalid DataVolume to a VMI
				vmi := libstorage.RenderVMIWithDataVolume(dataVolume.Name, testsuite.GetTestNamespace(nil))
				// Create a VM for this VMI
				vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
				vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(ThisVMIWith(vm.Namespace, vm.Name), 100).Should(Or(BeInPhase(v1.Pending), BeInPhase(v1.Scheduling)))

				By("Creating a service which makes the registry reachable")
				_, err = k8s.Client().CoreV1().Services(vm.Namespace).Create(context.Background(), &k8sv1.Service{
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

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachine with a DataVolume using http import", func() {
		var sc string

		BeforeEach(func() {
			var exists bool
			sc, exists = libstorage.GetRWOFileSystemStorageClass()
			if !exists {
				Fail("Fail test when Filesystem storage is not present")
			}
		})

		It("[test_id:3191]should be successfully started and stopped multiple times", func() {
			vm := renderVMWithRegistryImportDataVolume(cd.ContainerDiskAlpine, sc)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			num := 2
			By("Starting and stopping the VirtualMachine number of times")
			for i := 0; i < num; i++ {
				By(fmt.Sprintf("Doing run: %d", i))
				vm = libvmops.StartVirtualMachine(vm)
				// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
				// after being restarted multiple times
				if i == num {
					By(checkingVMInstanceConsoleExpectedOut)
					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(console.LoginToAlpine(vmi)).To(Succeed())
				}
				vm = libvmops.StopVirtualMachine(vm)
			}
		})

		It("[test_id:837]deleting VM with background propagation policy should automatically delete DataVolumes and VMI owned by VM.", func() {
			vm := renderVMWithRegistryImportDataVolume(cd.ContainerDiskAlpine, sc)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vm = libvmops.StartVirtualMachine(vm)

			By(verifyingDataVolumeSuccess)
			libstorage.EventuallyDVWith(vm.Namespace, vm.Spec.DataVolumeTemplates[0].Name, 100, And(HaveSucceeded(), BeOwned()))

			By(verifyingPVCCreated)
			Eventually(ThisPVCWith(vm.Namespace, vm.Spec.DataVolumeTemplates[0].Name), 160).Should(Exist())

			By(verifyingVMICreated)
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 160).Should(And(BeRunning(), BeOwned()))

			By("Deleting VM with background propagation policy")
			Expect(virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{
				PropagationPolicy: pointer.P(metav1.DeletePropagationBackground),
			})).To(Succeed())

			By("Waiting for the VM to be deleted")
			Eventually(ThisVM(vm), 100).Should(BeGone())

			By("Waiting for the VMI to be deleted")
			Eventually(ThisVMIWith(vm.Namespace, vm.Name), 100).Should(BeGone())

			By("Waiting for the DataVolume to be deleted")
			Eventually(ThisDVWith(vm.Namespace, vm.Spec.DataVolumeTemplates[0].Name), 100).Should(BeGone())

			By("Waiting for the PVC to be deleted")
			Eventually(ThisPVCWith(vm.Namespace, vm.Spec.DataVolumeTemplates[0].Name), 100).Should(BeGone())
		})

		It("[test_id:3192]should remove owner references on DataVolume if VM is orphan deleted.", func() {
			vm := renderVMWithRegistryImportDataVolume(cd.ContainerDiskAlpine, sc)
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vm = libvmops.StartVirtualMachine(vm)

			// Check for owner reference
			Eventually(ThisDVWith(vm.Namespace, vm.Spec.DataVolumeTemplates[0].Name), 100).Should(BeOwned())

			// Delete the VM with orphan Propagation
			Expect(virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{
				PropagationPolicy: pointer.P(metav1.DeletePropagationOrphan),
			})).To(Succeed())

			// Wait for the VM to be deleted
			Eventually(ThisVM(vm), 100).Should(BeGone())

			// Wait for the owner reference to disappear
			libstorage.EventuallyDVWith(vm.Namespace, vm.Spec.DataVolumeTemplates[0].Name, 100, And(HaveSucceeded(), Not(BeOwned())))

			// Verify VMI still exists with owner references removed
			Consistently(ThisVMIWith(vm.Namespace, vm.Name), 60, 1).Should(And(BeRunning(), Not(BeOwned())))
		})
	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] DataVolume clone permission checking", func() {
		Context("using Alpine import/clone", decorators.RequiresSnapshotStorageClass, func() {
			var sourceDV *cdiv1.DataVolume
			var cloneRole *rbacv1.Role
			var cloneRoleBinding *rbacv1.RoleBinding
			var storageClass string
			var vm *v1.VirtualMachine

			BeforeEach(func() {
				storageClass, err = libstorage.GetSnapshotStorageClass(virtClient, k8s.Client())
				Expect(err).ToNot(HaveOccurred())
				if storageClass == "" {
					Fail("Failing test, no VolumeSnapshot support")
				}

				sourceDV = libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
					libdv.WithStorage(libdv.StorageWithStorageClass(storageClass)),
					libdv.WithForceBindAnnotation(),
				)
				sourceDV, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.NamespaceTestAlternative).Create(context.Background(), sourceDV, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libstorage.EventuallyDV(sourceDV, 90, HaveSucceeded())

				dv := libdv.NewDataVolume(
					libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
					libdv.WithPVCSource(testsuite.NamespaceTestAlternative, sourceDV.Name),
					libdv.WithStorage(
						libdv.StorageWithStorageClass(storageClass),
						libdv.StorageWithoutVolumeSize(),
					),
				)

				vm = libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
						libvmi.WithResourceMemory("1Mi"),
						libvmi.WithDataVolume("disk0", dv.Name),
						libvmi.WithServiceAccountDisk(testsuite.AdminServiceAccountName),
					),
					libvmi.WithDataVolumeTemplate(dv),
				)
			})

			AfterEach(func() {
				if cloneRole != nil {
					err := k8s.Client().RbacV1().Roles(cloneRole.Namespace).Delete(context.Background(), cloneRole.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					cloneRole = nil
				}

				if cloneRoleBinding != nil {
					err := k8s.Client().RbacV1().RoleBindings(cloneRoleBinding.Namespace).Delete(context.Background(), cloneRoleBinding.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
					cloneRoleBinding = nil
				}
			})

			createVMSuccess := func() {
				// sometimes it takes a bit for permission to actually be applied so eventually
				Eventually(func() bool {
					_, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
					if err != nil {
						fmt.Printf("command should have succeeded maybe new permissions not applied yet\nerror\n%s\n", err)
						return false
					}
					return true
				}, 90*time.Second, 1*time.Second).Should(BeTrue())

				// start vm and check dv clone succeeded
				vm = libvmops.StartVirtualMachine(vm)
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
				snapshotClassName, err := libstorage.GetSnapshotClass(storageClass, virtClient, k8s.Client())
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

				By("Waiting for snapshot to have restore size")
				Eventually(func() (*vsv1.VolumeSnapshot, error) {
					snap, err = virtClient.KubernetesSnapshotClient().
						SnapshotV1().
						VolumeSnapshots(snap.Namespace).
						Get(context.Background(), snap.Name, metav1.GetOptions{})
					return snap, err
				}).WithTimeout(90 * time.Second).WithPolling(time.Second).Should(HaveField("Status.RestoreSize", Not(BeNil())))

				// set the target DV size to the snapshot restore size if it is not zero
				if !snap.Status.RestoreSize.IsZero() {
					if dvt.Spec.Storage.Resources.Requests == nil {
						dvt.Spec.Storage.Resources.Requests = k8sv1.ResourceList{}
					}
					dvt.Spec.Storage.Resources.Requests[k8sv1.ResourceStorage] = *snap.Status.RestoreSize
				}
			}

			createSnapshotDataSource := func() *cdiv1.DataSource {
				snapshotClassName, err := libstorage.GetSnapshotClass(storageClass, virtClient, k8s.Client())
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
					k8s.Client(),
					explicitCloneRole,
					testsuite.AdminServiceAccountName,
					testsuite.GetTestNamespace(nil),
					testsuite.NamespaceTestAlternative,
				)

				createVMSuccess()

				dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.TODO(), dvt.Name, metav1.GetOptions{})
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
					k8s.Client(),
					explicitCloneRole,
					testsuite.AdminServiceAccountName,
					testsuite.GetTestNamespace(nil),
					testsuite.NamespaceTestAlternative,
				)

				// We first delete the source PVC and DataVolume to force a clone without source
				err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.NamespaceTestAlternative).Delete(context.Background(), sourceDV.Name, metav1.DeleteOptions{})
				Expect(err).To(Or(
					Not(HaveOccurred()),
					Satisfy(errors.IsNotFound),
				))
				Eventually(ThisPVCWith(testsuite.NamespaceTestAlternative, sourceDV.Name), 10*time.Second, 1*time.Second).Should(BeGone())

				// We check if the VM is successfully created
				By("Creating VM")
				Eventually(func() error {
					_, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
					return err
				}, 90*time.Second, 1*time.Second).Should(Succeed())

				// Check for owner reference
				Eventually(ThisDVWith(vm.Namespace, vm.Spec.DataVolumeTemplates[0].Name), 100).Should(BeOwned())

				// We check the expected event
				By("Expecting SourcePVCNotAvailabe event")
				Eventually(func() bool {
					events, err := k8s.Client().CoreV1().Events(vm.Namespace).List(context.Background(), metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, e := range events.Items {
						if e.Reason == "SourcePVCNotAvailabe" {
							return true
						}
					}
					return false
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			})

			DescribeTable("[storage-req] deny then allow clone request", decorators.Conformance, decorators.StorageReq, func(role *rbacv1.Role, allServiceAccounts, allServiceAccountsInNamespace bool, cloneMutateFunc func(), fail bool) {
				if cloneMutateFunc != nil {
					cloneMutateFunc()
				}
				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(ThisVM(vm), 1*time.Minute, 2*time.Second).Should(HaveConditionTrueWithMessage(v1.VirtualMachineFailure, "insufficient permissions in clone source namespace"))

				saName := testsuite.AdminServiceAccountName
				saNamespace := testsuite.GetTestNamespace(nil)

				if allServiceAccounts {
					saName = ""
					saNamespace = ""
				} else if allServiceAccountsInNamespace {
					saName = ""
				}

				// add permission
				cloneRole, cloneRoleBinding = addClonePermission(k8s.Client(), role, saName, saNamespace, testsuite.NamespaceTestAlternative)
				if fail {
					Consistently(ThisVM(vm), 10*time.Second, 1*time.Second).Should(HaveConditionTrueWithMessage(v1.VirtualMachineFailure, "insufficient permissions in clone source namespace"))
					return
				}

				libvmops.StartVirtualMachine(vm)
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

			It("should skip authorization when DataVolume already exists", func() {
				cloneRole, cloneRoleBinding = addClonePermission(
					k8s.Client(),
					explicitCloneRole,
					"",
					"",
					testsuite.NamespaceTestAlternative,
				)

				dv := &cdiv1.DataVolume{
					ObjectMeta: vm.Spec.DataVolumeTemplates[0].ObjectMeta,
					Spec:       vm.Spec.DataVolumeTemplates[0].Spec,
				}
				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libstorage.EventuallyDV(dv, 90, Or(HaveSucceeded(), WaitForFirstConsumer()))

				err := k8s.Client().RbacV1().RoleBindings(cloneRoleBinding.Namespace).Delete(context.Background(), cloneRoleBinding.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
				cloneRoleBinding = nil

				createVMSuccess()
			})
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
				Fail("Fail test when Filesystem storage is not present")
			}

			dataVolume := libdv.NewDataVolume(
				libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling)),
				libdv.WithStorage(
					libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeFilesystem),
					libdv.StorageWithVolumeSize(cd.FedoraVolumeSize),
				),
				libdv.WithForceBindAnnotation(), // So we can wait for DV to finish before starting the VMI
			)

			dataVolume = dvChange(dataVolume)
			preallocated := dataVolume.Spec.Preallocation != nil && *dataVolume.Spec.Preallocation

			vmi := libstorage.RenderVMIWithDataVolume(dataVolume.Name, testsuite.GetTestNamespace(nil),
				libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
				libvmi.WithMemoryRequest("512Mi"),
			)
			vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus = "scsi"

			dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Importing Fedora is so slow that we get "resourceVersion too old" when trying
			// to watch for events between the VMI creation and VMI starting.
			By("Making sure the slow Fedora import is complete before creating the VMI")
			libstorage.EventuallyDV(dataVolume, 500, HaveSucceeded())

			vmi = createAndWaitForVMIReady(vmi, dataVolume)

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			imageSizeAfterBoot := getImageSize(vmi, dataVolume)
			By(fmt.Sprintf("image size after boot is %d", imageSizeAfterBoot))

			By("Filling out disk space")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "dd if=/dev/urandom of=largefile bs=1M count=300 2> /dev/null\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: syncName},
				&expect.BExp{R: ""},
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
				&expect.BExp{R: ""},
				&expect.BSnd{S: syncName},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "rm -f largefile\n"},
				&expect.BExp{R: ""},
			}, 60)).To(Succeed(), "should trim within the VM")

			Eventually(func() bool {
				By("Running trim")
				err := console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo fstrim -v /\n"},
					&expect.BExp{R: ""},
					&expect.BSnd{S: syncName},
					&expect.BExp{R: ""},
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

			err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("[test_id:5894]by default, fstrim will make the image smaller", noop, true),
			Entry("[test_id:5898]with preallocation true, fstrim has no effect", addPreallocationTrue, false),
			Entry("[test_id:5897]with preallocation false, fstrim will make the image smaller", addPreallocationFalse, true),
			Entry("[test_id:5899]with thick provision true, fstrim has no effect", addThickProvisionedTrueAnnotation, false),
			Entry("[test_id:5896]with thick provision false, fstrim will make the image smaller", addThickProvisionedFalseAnnotation, true),
		)
	})
}))

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

func addClonePermission(client kubernetes.Interface, role *rbacv1.Role, sa, saNamespace, targetNamesace string) (*rbacv1.Role, *rbacv1.RoleBinding) {
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
	storageClass, err := k8s.Client().StorageV1().StorageClasses().Get(context.Background(), sc, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	return storageClass.AllowVolumeExpansion != nil &&
		*storageClass.AllowVolumeExpansion
}

func renderVMWithRegistryImportDataVolume(containerDisk cd.ContainerDisk, storageClass string) *v1.VirtualMachine {
	importUrl := cd.DataVolumeImportUrlForContainerDisk(containerDisk)
	dv := libdv.NewDataVolume(
		libdv.WithRegistryURLSource(importUrl),
		libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
		libdv.WithStorage(
			libdv.StorageWithStorageClass(storageClass),
			libdv.StorageWithVolumeSize(cd.ContainerDiskSizeBySourceURL(importUrl)),
		),
	)
	return libstorage.RenderVMWithDataVolumeTemplate(dv)
}
