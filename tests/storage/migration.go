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

package storage

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	storagev1 "k8s.io/api/storage/v1"
	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	testsmig "kubevirt.io/kubevirt/tests/migration"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Volumes update with migration", decorators.RequiresTwoSchedulableNodes, decorators.VMLiveUpdateRolloutStrategy, func() {
	var virtClient kubecli.KubevirtClient
	var testSc string
	getCSIStorageClass := libstorage.GetSnapshotStorageClass
	createBlankDV := func(virtClient kubecli.KubevirtClient, ns, size string) *cdiv1.DataVolume {
		dv := libdv.NewDataVolume(
			libdv.WithBlankImageSource(),
			libdv.WithStorage(libdv.StorageWithStorageClass(testSc),
				libdv.StorageWithVolumeSize(size),
				libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeFilesystem),
				libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce),
			),
		)
		_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(ns).Create(context.Background(),
			dv, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return dv
	}

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		originalKv := libkubevirt.GetCurrentKv(virtClient)
		updateStrategy := &virtv1.KubeVirtWorkloadUpdateStrategy{
			WorkloadUpdateMethods: []virtv1.WorkloadUpdateMethod{virtv1.WorkloadUpdateMethodLiveMigrate},
		}
		rolloutStrategy := pointer.P(virtv1.VMRolloutStrategyLiveUpdate)
		patchWorkload, err := patch.New(
			patch.WithAdd("/spec/workloadUpdateStrategy", updateStrategy),
			patch.WithAdd("/spec/configuration/vmRolloutStrategy", rolloutStrategy),
		).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		_, err = virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(
			context.Background(), originalKv.Name, types.JSONPatchType,
			patchWorkload, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		currentKv := libkubevirt.GetCurrentKv(virtClient)
		config.WaitForConfigToBePropagatedToComponent(
			"kubevirt.io=virt-controller",
			currentKv.ResourceVersion,
			config.ExpectResourceVersionToBeLessEqualThanConfigVersion,
			time.Minute)
		scName, err := getCSIStorageClass(virtClient)
		Expect(err).ToNot(HaveOccurred())
		if scName == "" {
			Fail("Fail test when a CSI storage class is not present")
		}

		sc, err := virtClient.StorageV1().StorageClasses().Get(context.Background(), scName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		wffcSc := sc.DeepCopy()
		wffcSc.ObjectMeta = metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-wffc", scName),
			Labels: map[string]string{
				cleanup.TestLabelForNamespace(testsuite.GetTestNamespace(nil)): "",
			},
		}
		wffcSc.VolumeBindingMode = pointer.P(storagev1.VolumeBindingWaitForFirstConsumer)
		sc, err = virtClient.StorageV1().StorageClasses().Create(context.Background(), wffcSc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		testSc = sc.Name
	})
	AfterEach(func() {
		virtClient.StorageV1().StorageClasses().Delete(context.Background(), testSc, metav1.DeleteOptions{})
	})

	Describe("Update volumes with the migration updateVolumesStrategy", func() {
		var (
			ns      string
			destPVC string
		)
		const (
			fsPVC            = "filesystem"
			blockPVC         = "block"
			size             = "1Gi"
			sizeWithOverhead = "1.2Gi"
		)

		waitMigrationToNotExist := func(vmiName, ns string) {
			Eventually(func() bool {
				ls := labels.Set{
					virtv1.VolumesUpdateMigration: vmiName,
				}
				migList, err := virtClient.VirtualMachineInstanceMigration(ns).List(context.Background(),
					metav1.ListOptions{
						LabelSelector: ls.String(),
					})
				Expect(err).ToNot(HaveOccurred())
				if len(migList.Items) == 0 {
					return true
				}
				return false

			}, 120*time.Second, time.Second).Should(BeTrue())
		}
		waitVMIToHaveVolumeChangeCond := func(vmiName, ns string) {

			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vmiName,
					metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				conditionManager := controller.NewVirtualMachineInstanceConditionManager()
				return conditionManager.HasCondition(vmi, virtv1.VirtualMachineInstanceVolumesChange)
			}, 120*time.Second, time.Second).Should(BeTrue())
		}

		createDV := func() *cdiv1.DataVolume {
			sc, exist := libstorage.GetRWOFileSystemStorageClass()
			Expect(exist).To(BeTrue())
			dv := libdv.NewDataVolume(
				libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
				libdv.WithStorage(libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeSize(size),
					libdv.StorageWithFilesystemVolumeMode(),
				),
			)
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(ns).Create(context.Background(),
				dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			return dv
		}

		createVMWithDV := func(dv *cdiv1.DataVolume, volName string) *virtv1.VirtualMachine {
			vmi := libvmi.New(
				libvmi.WithNamespace(ns),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
				libvmi.WithResourceMemory("128Mi"),
				libvmi.WithDataVolume(volName, dv.Name),
				libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
			)
			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
				libvmi.WithDataVolumeTemplate(dv),
			)
			vm, err := virtClient.VirtualMachine(ns).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
			libwait.WaitForSuccessfulVMIStart(vmi)

			return vm
		}
		updateVMWithPVC := func(vm *virtv1.VirtualMachine, volName, claim string) {
			// Replace dst pvc
			i := slices.IndexFunc(vm.Spec.Template.Spec.Volumes, func(volume virtv1.Volume) bool {
				return volume.Name == volName
			})
			Expect(i).To(BeNumerically(">", -1))
			By(fmt.Sprintf("Replacing volume %s with PVC %s", volName, claim))

			updatedVolume := virtv1.Volume{
				Name: volName,
				VolumeSource: virtv1.VolumeSource{PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: claim,
					}}}}

			p, err := patch.New(
				patch.WithReplace("/spec/dataVolumeTemplates", []virtv1.DataVolumeTemplateSpec{}),
				patch.WithReplace(fmt.Sprintf("/spec/template/spec/volumes/%d", i), updatedVolume),
				patch.WithReplace("/spec/updateVolumesStrategy", virtv1.UpdateVolumesStrategyMigration),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, p, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes[i].VolumeSource.PersistentVolumeClaim.
				PersistentVolumeClaimVolumeSource.ClaimName).To(Equal(claim))
		}
		// TODO: right now, for simplicity, this function assumes the DV in the first position in the datavolumes templata list. Otherwise, we need
		// to pass the old name of the DV to be replaces.
		updateVMWithDV := func(vm *virtv1.VirtualMachine, volName, name string) {
			i := slices.IndexFunc(vm.Spec.Template.Spec.Volumes, func(volume virtv1.Volume) bool {
				return volume.Name == volName
			})
			Expect(i).To(BeNumerically(">", -1))
			By(fmt.Sprintf("Replacing volume %s with DV %s", volName, name))

			updatedVolume := virtv1.Volume{
				Name: volName,
				VolumeSource: virtv1.VolumeSource{DataVolume: &virtv1.DataVolumeSource{
					Name: name,
				}}}

			p, err := patch.New(
				patch.WithReplace("/spec/dataVolumeTemplates/0/metadata/name", name),
				patch.WithReplace(fmt.Sprintf("/spec/template/spec/volumes/%d", i), updatedVolume),
				patch.WithReplace("/spec/updateVolumesStrategy", virtv1.UpdateVolumesStrategyMigration),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, p, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(vm.Spec.Template.Spec.Volumes[i].VolumeSource.DataVolume.Name).To(Equal(name))
		}

		checkVolumeMigrationOnVM := func(vm *virtv1.VirtualMachine, volName, src, dst string) {
			Eventually(func() []virtv1.StorageMigratedVolumeInfo {
				vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if vm.Status.VolumeUpdateState == nil || vm.Status.VolumeUpdateState.VolumeMigrationState == nil {
					return nil
				}
				return vm.Status.VolumeUpdateState.VolumeMigrationState.MigratedVolumes
			}).WithTimeout(120*time.Second).WithPolling(time.Second).Should(
				ContainElement(virtv1.StorageMigratedVolumeInfo{
					VolumeName: volName,
					SourcePVCInfo: &virtv1.PersistentVolumeClaimInfo{
						ClaimName:  src,
						VolumeMode: pointer.P(k8sv1.PersistentVolumeFilesystem),
					},
					DestinationPVCInfo: &virtv1.PersistentVolumeClaimInfo{
						ClaimName:  dst,
						VolumeMode: pointer.P(k8sv1.PersistentVolumeFilesystem),
					},
				}), "The volumes migrated should be set",
			)
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return controller.NewVirtualMachineConditionManager().HasCondition(
					vm, virtv1.VirtualMachineManualRecoveryRequired)
			}).WithTimeout(120 * time.Second).WithPolling(time.Second).Should(BeTrue())
		}

		BeforeEach(func() {
			ns = testsuite.GetTestNamespace(nil)
			destPVC = "dest-" + rand.String(5)

		})
		Context(" destination PVC expansion", decorators.StorageReq, decorators.RequiresVolumeExpansion, func() {
			DescribeTable("should migrate the source volume from a source DV to a destination PVC", func(mode string) {
				volName := "disk0"
				dv := createDV()
				vmi := libvmi.New(
					libvmi.WithNamespace(ns),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
					libvmi.WithResourceMemory("128Mi"),
					libvmi.WithDataVolume(volName, dv.Name),
					libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
				)
				vm := libvmi.NewVirtualMachine(vmi,
					libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
					libvmi.WithDataVolumeTemplate(dv),
				)
				vm, err := virtClient.VirtualMachine(ns).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
				libwait.WaitForSuccessfulVMIStart(vmi)

				// Create dest PVC
				var dstPVC *k8sv1.PersistentVolumeClaim
				switch mode {
				case fsPVC:
					// Add some overhead to the target PVC for filesystem.
					dstPVC = libstorage.CreateFSPVC(destPVC, ns, sizeWithOverhead, nil)
				case blockPVC:
					dstPVC = libstorage.CreateBlockPVC(destPVC, ns, size)
				default:
					Fail("Unrecognized mode")
				}
				Expect(dstPVC.Spec.StorageClassName).ToNot(BeNil())
				if !volumeExpansionAllowed(*dstPVC.Spec.StorageClassName) {
					Fail("Fail when volume expansion storage class not available")
				}

				By("Update volumes")
				updateVMWithPVC(vm, volName, destPVC)
				Eventually(func() bool {
					vmi, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vm.Name,
						metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					claim := storagetypes.PVCNameFromVirtVolume(&vmi.Spec.Volumes[0])
					return claim == destPVC
				}, 120*time.Second, time.Second).Should(BeTrue())
				waitForMigrationToSucceed(virtClient, vm.Name, ns)

				By("Expanding the destination PVC")
				p, err := patch.New(
					patch.WithAdd("/spec/resources/requests/storage", resource.MustParse("4Gi")),
				).GeneratePayload()
				Expect(err).ToNot(HaveOccurred())
				_, err = virtClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Patch(context.Background(),
					destPVC, types.JSONPatchType, p, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Expecting the VirtualMachineInstance console")
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				By("Waiting for notification about size change")
				Eventually(func() error {
					err := console.SafeExpectBatch(vmi, []expect.Batcher{
						&expect.BSnd{S: "\n"},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: "[ $(lsblk /dev/vda -o SIZE -n |sed -e \"s/ //g\") == \"4G\" ] && true\n"},
						&expect.BExp{R: "0"},
					}, 10)
					return err
				}, 120).Should(BeNil())
			},
				Entry("to a filesystem volume", fsPVC),
				Entry("to a block volume", decorators.RequiresBlockStorage, blockPVC),
			)
		})
		It("should migrate the source volume from a source DV to a destination DV", func() {
			volName := "disk0"
			srcDV := createDV()
			vm := createVMWithDV(srcDV, volName)
			destDV := createBlankDV(virtClient, ns, "2Gi")
			By("Update volumes")
			updateVMWithDV(vm, volName, destDV.Name)
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vm.Name,
					metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				claim := storagetypes.PVCNameFromVirtVolume(&vmi.Spec.Volumes[0])
				return claim == destDV.Name
			}, 120*time.Second, time.Second).Should(BeTrue())
			waitForMigrationToSucceed(virtClient, vm.Name, ns)

			By("Rollback to the original volumes")
			updateVMWithDV(vm, volName, srcDV.Name)
			waitForMigrationToSucceed(virtClient, vm.Name, ns)
		})

		It("should trigger the migration once the destination DV exists", func() {
			volName := "disk0"
			srcDV := createDV()
			vm := createVMWithDV(srcDV, volName)
			destDV := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithStorage(libdv.StorageWithStorageClass(testSc),
					libdv.StorageWithVolumeSize(size),
					libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeFilesystem),
					libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce),
				),
			)
			By("Update volumes")
			i := slices.IndexFunc(vm.Spec.Template.Spec.Volumes, func(volume virtv1.Volume) bool {
				return volume.Name == volName
			})
			Expect(i).To(BeNumerically(">", -1))
			By(fmt.Sprintf("Replacing volume %s with DV %s", volName, destDV.Name))

			updatedVolume := virtv1.Volume{
				Name: volName,
				VolumeSource: virtv1.VolumeSource{DataVolume: &virtv1.DataVolumeSource{
					Name: destDV.Name,
				}}}

			p, err := patch.New(
				patch.WithRemove("/spec/dataVolumeTemplates"),
				patch.WithReplace(fmt.Sprintf("/spec/template/spec/volumes/%d", i), updatedVolume),
				patch.WithReplace("/spec/updateVolumesStrategy", virtv1.UpdateVolumesStrategyMigration),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, p, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() []virtv1.VirtualMachineInstanceCondition {
				vmi, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.Conditions
			}, 30*time.Second, time.Second).Should(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Type":    Equal(virtv1.VirtualMachineInstanceVolumesChange),
				"Status":  Equal(k8sv1.ConditionFalse),
				"Message": ContainSubstring("One of the destination volumes doesn't exist"),
			})))

			By("Create the destination DV")
			_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(ns).Create(context.Background(),
				destDV, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			waitForMigrationToSucceed(virtClient, vm.Name, ns)
		})

		It("should trigger the migration once the destination PVC exists", func() {
			volName := "disk0"
			srcDV := createDV()
			vm := createVMWithDV(srcDV, volName)
			By("Update volumes")
			updateVMWithPVC(vm, volName, destPVC)

			By("Create the destination PVC")
			libstorage.CreateFSPVC(destPVC, ns, "2Gi", nil)

			waitForMigrationToSucceed(virtClient, vm.Name, ns)
		})

		It("should migrate the source volume from a source and destination block RWX DVs", decorators.StorageCritical, decorators.RequiresRWXBlock, func() {
			volName := "disk0"
			sc, exist := libstorage.GetRWXBlockStorageClass()
			Expect(exist).To(BeTrue())
			srcDV := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithStorage(libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeSize(size),
					libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeBlock),
				),
			)
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(ns).Create(context.Background(),
				srcDV, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			destDV := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithStorage(libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeSize(size),
					libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeBlock),
				),
			)
			_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(ns).Create(context.Background(),
				destDV, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vm := createVMWithDV(srcDV, volName)
			By("Update volumes")
			updateVMWithDV(vm, volName, destDV.Name)
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vm.Name,
					metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				claim := storagetypes.PVCNameFromVirtVolume(&vmi.Spec.Volumes[0])
				return claim == destDV.Name
			}, 120*time.Second, time.Second).Should(BeTrue())
			waitForMigrationToSucceed(virtClient, vm.Name, ns)
		})

		It("should migrate the source volume from a block source and filesystem destination DVs", decorators.RequiresBlockStorage, func() {
			volName := "disk0"
			sc, exist := libstorage.GetRWOBlockStorageClass()
			Expect(exist).To(BeTrue())
			srcDV := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithStorage(libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeSize(size),
					libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeBlock),
				),
			)
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(ns).Create(context.Background(),
				srcDV, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(err).ToNot(HaveOccurred())
			destDV := createBlankDV(virtClient, ns, size)
			vm := createVMWithDV(srcDV, volName)
			By("Update volumes")
			updateVMWithDV(vm, volName, destDV.Name)
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vm.Name,
					metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				claim := storagetypes.PVCNameFromVirtVolume(&vmi.Spec.Volumes[0])
				return claim == destDV.Name
			}, 120*time.Second, time.Second).Should(BeTrue())
			waitForMigrationToSucceed(virtClient, vm.Name, ns)
		})

		It("should migrate a PVC with a VM using a containerdisk", func() {
			volName := "volume"
			srcPVC := "src-" + rand.String(5)
			libstorage.CreateFSPVC(srcPVC, ns, size, nil)
			libstorage.CreateFSPVC(destPVC, ns, size, nil)
			vmi := libvmifact.NewCirros(
				libvmi.WithNamespace(ns),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
				libvmi.WithResourceMemory("128Mi"),
				libvmi.WithPersistentVolumeClaim(volName, srcPVC),
			)
			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
			)
			vm, err := virtClient.VirtualMachine(ns).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Update volumes")
			updateVMWithPVC(vm, volName, destPVC)
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vm.Name,
					metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, v := range vmi.Spec.Volumes {
					if v.PersistentVolumeClaim != nil {
						if v.PersistentVolumeClaim.ClaimName == destPVC {
							return true
						}
					}
				}
				return false
			}, 120*time.Second, time.Second).Should(BeTrue())
			waitForMigrationToSucceed(virtClient, vm.Name, ns)
		})

		It("should cancel the migration by the reverting to the source volume", func() {
			volName := "volume"
			dv := createDV()
			vm := createVMWithDV(dv, volName)
			// Create dest PVC
			createUnschedulablePVC(destPVC, ns, size)
			By("Update volumes")
			updateVMWithPVC(vm, volName, destPVC)
			waitMigrationToExist(virtClient, vm.Name, ns)
			waitVMIToHaveVolumeChangeCond(vm.Name, ns)
			By("Cancel the volume migration")
			updateVMWithPVC(vm, volName, dv.Name)
			// After the volume migration abortion the VMI should have:
			// 1. the source volume restored
			// 2. condition VolumesChange set to false
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vm.Name,
					metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				claim := storagetypes.PVCNameFromVirtVolume(&vmi.Spec.Volumes[0])
				if claim != dv.Name {
					return false
				}
				conditionManager := controller.NewVirtualMachineInstanceConditionManager()
				c := conditionManager.GetCondition(vmi, virtv1.VirtualMachineInstanceVolumesChange)
				if c == nil {
					return false
				}
				return c.Status == k8sv1.ConditionFalse
			}, 120*time.Second, time.Second).Should(BeTrue())
			waitMigrationToNotExist(vm.Name, ns)
		})

		It("should fail to migrate when the destination image is smaller", func() {
			const volName = "disk0"
			vm := createVMWithDV(createDV(), volName)
			createSmallImageForDestinationMigration(vm, destPVC, size)
			By("Update volume")
			updateVMWithPVC(vm, volName, destPVC)
			// let the workload updater creates some migration
			time.Sleep(2 * time.Minute)
			ls := labels.Set{virtv1.VolumesUpdateMigration: vm.Name}
			migList, err := virtClient.VirtualMachineInstanceMigration(ns).List(context.Background(),
				metav1.ListOptions{LabelSelector: ls.String()})
			Expect(err).ShouldNot(HaveOccurred())
			// It should have create some migrations, but the time between the migration creations should incrementally
			// increasing. Therefore, after 2 minutes we don't expect more then 6 mgration objects.
			Expect(len(migList.Items)).Should(BeNumerically(">", 1))
			Expect(len(migList.Items)).Should(BeNumerically("<", 56))
		})

		It("should set the restart condition since the second volume is RWO and not part of the migration", func() {
			const volName = "vol0"
			dv1 := createDV()
			dv2 := createBlankDV(virtClient, ns, size)
			destDV := createBlankDV(virtClient, ns, size)
			vmi := libvmi.New(
				libvmi.WithNamespace(ns),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
				libvmi.WithResourceMemory("128Mi"),
				libvmi.WithDataVolume(volName, dv1.Name),
				libvmi.WithDataVolume("vol1", dv2.Name),
				libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
			)
			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
				libvmi.WithDataVolumeTemplate(dv1),
			)
			vm, err := virtClient.VirtualMachine(ns).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
			libwait.WaitForSuccessfulVMIStart(vmi)
			By("Update volumes")
			updateVMWithDV(vm, volName, destDV.Name)
			Eventually(func() []virtv1.VirtualMachineCondition {
				vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Conditions
			}).WithTimeout(120*time.Second).WithPolling(time.Second).Should(
				ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Type":    Equal(virtv1.VirtualMachineRestartRequired),
					"Status":  Equal(k8sv1.ConditionTrue),
					"Message": Equal("cannot migrate the VM. The volume vol1 is RWO and not included in the migration volumes"),
				})), "The RestartRequired condition should be false",
			)
		})

		It("should refuse to restart the VM and set the ManualRecoveryRequired at VM shutdown", func() {
			volName := "volume"
			dv := createDV()
			vm := createVMWithDV(dv, volName)
			// Create dest PVC
			createUnschedulablePVC(destPVC, ns, size)
			By("Update volumes")
			updateVMWithPVC(vm, volName, destPVC)
			waitMigrationToExist(virtClient, vm.Name, ns)
			waitVMIToHaveVolumeChangeCond(vm.Name, ns)

			By("Restarting the VM during the volume migration")
			restartOptions := &virtv1.RestartOptions{GracePeriodSeconds: pointer.P(int64(0))}
			err := virtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, restartOptions)
			Expect(err).To(MatchError(ContainSubstring("VM recovery required")))

			By("Stopping the VM during the volume migration")
			stopOptions := &virtv1.StopOptions{GracePeriod: pointer.P(int64(0))}
			err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, stopOptions)
			Expect(err).ToNot(HaveOccurred())
			checkVolumeMigrationOnVM(vm, volName, dv.Name, destPVC)

			By("Starting the VM after a failed volume migration")
			err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Start(context.Background(), vm.Name, &virtv1.StartOptions{Paused: false})
			Expect(err).To(MatchError(ContainSubstring("VM recovery required")))

			By("Reverting the original volumes")
			updatedVolume := virtv1.Volume{
				Name: volName,
				VolumeSource: virtv1.VolumeSource{DataVolume: &virtv1.DataVolumeSource{
					Name: dv.Name,
				}}}
			p, err := patch.New(
				patch.WithReplace(fmt.Sprintf("/spec/template/spec/volumes/0"), updatedVolume),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, p, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return controller.NewVirtualMachineConditionManager().HasCondition(
					vm, virtv1.VirtualMachineManualRecoveryRequired)
			}).WithTimeout(120 * time.Second).WithPolling(time.Second).Should(BeFalse())

			By("Starting the VM after the volume set correction")
			err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Start(context.Background(), vm.Name, &virtv1.StartOptions{Paused: false})
			Expect(err).NotTo(HaveOccurred())
			Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 120*time.Second, 1*time.Second).Should(matcher.BeRunning())
		})

		It("should cancel the migration and clear the volume migration state", func() {
			volName := "volume"
			dv := createDV()
			vm := createVMWithDV(dv, volName)
			createUnschedulablePVC(destPVC, ns, size)
			By("Update volumes")
			updateVMWithPVC(vm, volName, destPVC)
			waitMigrationToExist(virtClient, vm.Name, ns)
			waitVMIToHaveVolumeChangeCond(vm.Name, ns)
			Eventually(func() []virtv1.StorageMigratedVolumeInfo {
				vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if vm.Status.VolumeUpdateState == nil {
					return nil
				}
				return vm.Status.VolumeUpdateState.VolumeMigrationState.MigratedVolumes
			}).WithTimeout(120*time.Second).WithPolling(time.Second).Should(
				ContainElement(virtv1.StorageMigratedVolumeInfo{
					VolumeName: volName,
					SourcePVCInfo: &virtv1.PersistentVolumeClaimInfo{
						ClaimName:  dv.Name,
						VolumeMode: pointer.P(k8sv1.PersistentVolumeFilesystem),
					},
					DestinationPVCInfo: &virtv1.PersistentVolumeClaimInfo{
						ClaimName:  destPVC,
						VolumeMode: pointer.P(k8sv1.PersistentVolumeFilesystem),
					},
				}), "The volumes migrated should be set",
			)
			By("Cancel the volume migration")
			updateVMWithPVC(vm, volName, dv.Name)
			Eventually(func() *virtv1.VolumeMigrationState {
				vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if vm.Status.VolumeUpdateState == nil {
					return nil
				}
				return vm.Status.VolumeUpdateState.VolumeMigrationState
			}).WithTimeout(120 * time.Second).WithPolling(time.Second).Should(BeNil())
		})

		Context("should be able to recover from an interrupted volume migration", func() {
			const volName = "volume0"

			createMigpolicyWithLimitedBandwidth := func(vmi *virtv1.VirtualMachineInstance) {
				policy := testsmig.PreparePolicyAndVMIWithBandwidthLimitation(vmi, resource.MustParse("1Ki"))
				testsmig.CreateMigrationPolicy(virtClient, policy)
				Eventually(func() *migrationsv1.MigrationPolicy {
					policy, err := virtClient.MigrationPolicy().Get(context.Background(), policy.Name, metav1.GetOptions{})
					if err != nil {
						return nil
					}
					return policy
				}, 30*time.Second, time.Second).ShouldNot(BeNil())
			}

			createAndStartVM := func(dv *cdiv1.DataVolume) *virtv1.VirtualMachine {
				vmi := libvmi.New(
					libvmi.WithNamespace(ns),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
					libvmi.WithResourceMemory("128Mi"),
					libvmi.WithDataVolume(volName, dv.Name),
					libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
				)
				createMigpolicyWithLimitedBandwidth(vmi)
				vm := libvmi.NewVirtualMachine(vmi,
					libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
					libvmi.WithDataVolumeTemplate(dv),
					libvmi.WithRunStrategy(virtv1.RunStrategyManual),
				)
				vm, err := virtClient.VirtualMachine(ns).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Starting the VM")
				vm = libvmops.StartVirtualMachine(vm)
				vmi = libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Running})
				_, err = libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())

				return vm
			}

			It("when the copy of the destination volumes was successful", func() {
				dv := createDV()
				libstorage.CreateFSPVC(destPVC, ns, sizeWithOverhead, nil)
				vm := createAndStartVM(dv)

				By("Update volumes")
				updateVMWithPVC(vm, volName, destPVC)

				var migration *v1.VirtualMachineInstanceMigration
				Eventually(func() int {
					ls := labels.Set{
						virtv1.VolumesUpdateMigration: vm.Name,
					}
					migList, err := virtClient.VirtualMachineInstanceMigration(ns).List(context.Background(),
						metav1.ListOptions{
							LabelSelector: ls.String(),
						})
					Expect(err).ToNot(HaveOccurred())
					if len(migList.Items) < 1 {
						return 0
					}
					Expect(migList.Items).To(HaveLen(1))
					migration = &migList.Items[0]
					return 1
				}).WithTimeout(time.Minute).WithPolling(time.Second).Should(Equal(1))
				Eventually(matcher.ThisMigration(migration), 3*time.Minute, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationRunning))

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name,
					metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Killing the source pod")
				pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())
				err = virtClient.CoreV1().Pods(vmi.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{
					GracePeriodSeconds: pointer.P(int64(0)),
				})
				Expect(err).NotTo(HaveOccurred())

				By("Making sure that post-copy migration failed")
				Eventually(matcher.ThisMigration(migration), 3*time.Minute, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationFailed))

				virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Start(context.Background(), vm.Name, &virtv1.StartOptions{Paused: false})
				checkVolumeMigrationOnVM(vm, volName, dv.Name, destPVC)

				By("By removing the update volume strategy in order to remove the manual recovery condition and start the VM")
				p, err := patch.New(
					patch.WithRemove("/spec/updateVolumesStrategy"),
				).GeneratePayload()
				Expect(err).ToNot(HaveOccurred())
				vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, p, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return controller.NewVirtualMachineConditionManager().HasCondition(
						vm, virtv1.VirtualMachineManualRecoveryRequired)
				}).WithTimeout(120 * time.Second).WithPolling(time.Second).Should(BeFalse())

				By("Starting the VM")
				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return controller.NewVirtualMachineConditionManager().HasCondition(
						vm, virtv1.VirtualMachineManualRecoveryRequired)
				}).WithTimeout(120 * time.Second).WithPolling(time.Second).Should(BeFalse())
				Eventually(matcher.ThisVMI(vmi), 360*time.Second, time.Second).Should(matcher.BeInPhase(v1.Running))
			})
		})
	})

	Context("Hotplug volumes", func() {
		var fgDisabled bool
		BeforeEach(func() {
			fgDisabled = !checks.HasFeature(featuregate.HotplugVolumesGate)
			if fgDisabled {
				config.EnableFeatureGate(featuregate.HotplugVolumesGate)
			}

		})
		AfterEach(func() {
			if fgDisabled {
				config.DisableFeatureGate(featuregate.HotplugVolumesGate)
			}
		})

		waitForHotplugVol := func(vmName, ns, volName string) {
			Eventually(func() string {
				updatedVMI, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vmName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, volumeStatus := range updatedVMI.Status.VolumeStatus {
					if volumeStatus.Name == volName && volumeStatus.HotplugVolume != nil {
						return volumeStatus.Target
					}
				}
				return ""
			}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).ShouldNot(Equal(""))
		}

		DescribeTable("should be able to add and remove a volume with the volume migration feature gate enabled", func(persist bool) {
			const volName = "vol0"
			ns := testsuite.GetTestNamespace(nil)
			dv := createBlankDV(virtClient, ns, "1Gi")
			vmi := libvmifact.NewCirros(
				libvmi.WithNamespace(ns),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
			)
			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
			)
			vm, err := virtClient.VirtualMachine(ns).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
			libwait.WaitForSuccessfulVMIStart(vmi)

			volumeSource := &virtv1.HotplugVolumeSource{
				DataVolume: &virtv1.DataVolumeSource{
					Name: dv.Name,
				},
			}

			// Add the volume
			addOpts := &virtv1.AddVolumeOptions{
				Name: volName,
				Disk: &virtv1.Disk{
					DiskDevice: virtv1.DiskDevice{
						Disk: &virtv1.DiskTarget{Bus: virtv1.DiskBusSCSI},
					},
					Serial: volName,
				},
				VolumeSource: volumeSource,
			}
			if persist {
				Expect(virtClient.VirtualMachine(ns).AddVolume(context.Background(), vm.Name, addOpts)).ToNot(HaveOccurred())
			} else {
				Expect(virtClient.VirtualMachineInstance(ns).AddVolume(context.Background(), vm.Name, addOpts)).ToNot(HaveOccurred())
			}
			Eventually(func() string {
				updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, volumeStatus := range updatedVMI.Status.VolumeStatus {
					if volumeStatus.Name == volName && volumeStatus.HotplugVolume != nil {
						return volumeStatus.Target
					}
				}
				return ""
			}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).ShouldNot(Equal(""))

			// Remove the volume
			removeOpts := &virtv1.RemoveVolumeOptions{
				Name: volName,
			}
			if persist {
				Expect(virtClient.VirtualMachine(ns).RemoveVolume(context.Background(), vm.Name, removeOpts)).ToNot(HaveOccurred())
			} else {
				Expect(virtClient.VirtualMachineInstance(ns).RemoveVolume(context.Background(), vm.Name, removeOpts)).ToNot(HaveOccurred())
			}
			Eventually(func() []v1.VolumeStatus {
				updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return updatedVMI.Status.VolumeStatus
			}).WithTimeout(120 * time.Second).WithPolling(2 * time.Second).ShouldNot(
				ContainElement(HaveField("Name", Equal(volName))),
			)
		},
			Entry("with a persistent volume", true),
			Entry("with an ephemeral volume", false),
		)

		Context("should be able to volume migrate a VM", func() {
			addVolume := func(vmName, ns, volName, dvName string) {
				volumeSource := &v1.HotplugVolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: dvName,
					},
				}
				// Add the volume
				addOpts := &v1.AddVolumeOptions{
					Name: volName,
					Disk: &v1.Disk{
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{Bus: v1.DiskBusSCSI},
						},
					},
					VolumeSource: volumeSource,
				}
				Expect(virtClient.VirtualMachine(ns).AddVolume(context.Background(), vmName, addOpts)).ToNot(HaveOccurred())
				waitForHotplugVol(vmName, ns, volName)
			}
			getIndexVol := func(vmName, ns, volName string) int {
				var index int
				// Loop if the hotplug volume has been added with the imperative API it might be still not present on the VM
				Eventually(func() int {
					vm, err := virtClient.VirtualMachine(ns).Get(context.Background(), vmName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					for i, v := range vm.Spec.Template.Spec.Volumes {
						if v.Name == volName {
							index = i
							return i
						}
					}
					return -1
				}).WithTimeout(60 * time.Second).WithPolling(2 * time.Second).Should(BeNumerically(">", -1))
				return index
			}

			createFileOnHotpluggedVol := func(vmi *v1.VirtualMachineInstance, volName string) {
				var device string
				Eventually(func() string {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, v := range vmi.Status.VolumeStatus {
						if v.HotplugVolume == nil {
							continue
						}
						if v.Name == volName {
							device = v.Target
							return v.Target
						}
					}
					return ""
				}).WithTimeout(60 * time.Second).WithPolling(2 * time.Second).ShouldNot(BeEmpty())

				Expect(console.LoginToCirros(vmi)).To(Succeed())
				Expect(console.RunCommand(vmi, "sudo mkfs.ext3 /dev/sda", 30*time.Second)).To(Succeed())
				Expect(console.RunCommand(vmi, "mkdir test", 30*time.Second)).To(Succeed())
				Expect(console.RunCommand(vmi, fmt.Sprintf("sudo mount -t ext3 /dev/%s /home/cirros/test", device), 30*time.Second)).To(Succeed())
				Expect(console.RunCommand(vmi, "sudo chmod 777 /home/cirros/test", 30*time.Second)).To(Succeed())
				Expect(console.RunCommand(vmi, "sudo chown cirros:cirros /home/cirros/test", 30*time.Second)).To(Succeed())
				Expect(console.RunCommand(vmi, "printf 'test' &> /home/cirros/test/test", 30*time.Second)).To(Succeed())
			}
			checkFileOnHotpluggedVol := func(vmi *v1.VirtualMachineInstance) {
				Expect(console.LoginToCirros(vmi)).To(Succeed())
				Expect(console.RunCommand(vmi, "cat /home/cirros/test/test |grep test", 60*time.Second)).To(Succeed())
			}

			It("with a containerdisk and a hotplugged volume", func() {
				const volName = "vol0"
				ns := testsuite.GetTestNamespace(nil)
				dv := createBlankDV(virtClient, ns, "2G")
				vmi := libvmifact.NewCirros(
					libvmi.WithNamespace(ns),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
				)
				vm := libvmi.NewVirtualMachine(vmi,
					libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
				)
				vm, err := virtClient.VirtualMachine(ns).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
				libwait.WaitForSuccessfulVMIStart(vmi)

				addVolume(vm.Name, vm.Namespace, volName, dv.Name)
				createFileOnHotpluggedVol(vmi, volName)

				dvDst := createBlankDV(virtClient, vm.Namespace, "2Gi")
				By("Update volumes")
				index := getIndexVol(vm.Name, vm.Namespace, volName)
				p, err := patch.New(
					patch.WithReplace(fmt.Sprintf("/spec/template/spec/volumes/%d/dataVolume/name", index), dvDst.Name),
					patch.WithReplace("/spec/updateVolumesStrategy", virtv1.UpdateVolumesStrategyMigration),
				).GeneratePayload()
				Expect(err).ToNot(HaveOccurred())
				vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, p, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())
				waitForMigrationToSucceed(virtClient, vm.Name, vm.Namespace)
				checkFileOnHotpluggedVol(vmi)
			})

			DescribeTable("with a datavolume and an hotplugged datavolume migrating", func(srcBlock, dstBlock bool) {
				ns := testsuite.GetTestNamespace(nil)
				rootVolName := "root"
				hpVolName := "hp"
				var sc string
				var exist bool
				var volumeMode k8sv1.PersistentVolumeMode
				if srcBlock {
					sc, exist = libstorage.GetRWOBlockStorageClass()
					volumeMode = k8sv1.PersistentVolumeBlock
				} else {
					sc, exist = libstorage.GetRWOFileSystemStorageClass()
					volumeMode = k8sv1.PersistentVolumeFilesystem
				}
				Expect(exist).To(BeTrue())
				rootDV := libdv.NewDataVolume(
					libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
					libdv.WithStorage(libdv.StorageWithStorageClass(sc),
						libdv.StorageWithVolumeSize("1Gi"),
						libdv.StorageWithVolumeMode(volumeMode),
						libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce),
					),
				)
				vmi := libvmi.New(
					libvmi.WithNamespace(ns),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(virtv1.DefaultPodNetwork()),
					libvmi.WithResourceMemory("128Mi"),
					libvmi.WithDataVolume(rootVolName, rootDV.Name),
					libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
				)
				vm := libvmi.NewVirtualMachine(vmi,
					libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
					libvmi.WithDataVolumeTemplate(rootDV),
				)
				vm, err := virtClient.VirtualMachine(ns).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
				libwait.WaitForSuccessfulVMIStart(vmi)

				hpDV := libdv.NewDataVolume(
					libdv.WithBlankImageSource(),
					libdv.WithStorage(libdv.StorageWithStorageClass(sc),
						libdv.StorageWithVolumeSize("1G"),
						libdv.StorageWithVolumeMode(volumeMode),
						libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce),
					),
				)
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(ns).Create(context.Background(),
					hpDV, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				addVolume(vm.Name, vm.Namespace, hpVolName, hpDV.Name)
				createFileOnHotpluggedVol(vmi, hpVolName)

				indexRoot := getIndexVol(vm.Name, vm.Namespace, rootVolName)
				indexHp := getIndexVol(vm.Name, vm.Namespace, hpVolName)

				if dstBlock {
					sc, exist = libstorage.GetRWOBlockStorageClass()
					volumeMode = k8sv1.PersistentVolumeBlock
					Expect(exist).To(BeTrue())
				} else {
					sc = testSc
					volumeMode = k8sv1.PersistentVolumeFilesystem
				}
				dvRootDst := libdv.NewDataVolume(
					libdv.WithBlankImageSource(),
					libdv.WithStorage(libdv.StorageWithStorageClass(sc),
						libdv.StorageWithVolumeSize("2Gi"),
						libdv.StorageWithVolumeMode(volumeMode),
						libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce),
					),
				)
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(ns).Create(context.Background(),
					dvRootDst, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				dvHpDst := libdv.NewDataVolume(
					libdv.WithBlankImageSource(),
					libdv.WithStorage(libdv.StorageWithStorageClass(sc),
						libdv.StorageWithVolumeSize("2Gi"),
						libdv.StorageWithVolumeMode(volumeMode),
						libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce),
					),
				)
				_, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(ns).Create(context.Background(),
					dvHpDst, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				p, err := patch.New(
					patch.WithReplace("/spec/dataVolumeTemplates", []virtv1.DataVolumeTemplateSpec{}),
					patch.WithReplace(fmt.Sprintf("/spec/template/spec/volumes/%d/dataVolume/name", indexRoot), dvRootDst.Name),
					patch.WithReplace(fmt.Sprintf("/spec/template/spec/volumes/%d/dataVolume/name", indexHp), dvHpDst.Name),
					patch.WithReplace("/spec/updateVolumesStrategy", virtv1.UpdateVolumesStrategyMigration),
				).GeneratePayload()
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("Update root DV %s with %s and hotplug dv %s with %s", rootDV.Name, dvRootDst.Name, hpDV.Name, dvHpDst.Name))
				vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, p, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())
				waitForMigrationToSucceed(virtClient, vm.Name, vm.Namespace)
				checkFileOnHotpluggedVol(vmi)
			},
				Entry("from filesystem to filesystem", false, false),
				Entry("from filesystem to block", false, true),
				Entry("from block to filesystem", true, false),
				Entry("from block to block", true, true),
			)
		})
	})

}))

func createUnschedulablePVC(name, namespace, size string) *k8sv1.PersistentVolumeClaim {
	pvc := libstorage.NewPVC(name, size, "dontexist")
	pvc.Spec.VolumeMode = pointer.P(k8sv1.PersistentVolumeFilesystem)
	virtCli := kubevirt.Client()
	createdPvc, err := virtCli.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	return createdPvc
}

// createSmallImageForDestinationMigration creates a smaller raw image on the destination PVC and the PVC is bound to another node then the running
// virt-launcher in order to allow the migration.
func createSmallImageForDestinationMigration(vm *virtv1.VirtualMachine, name, size string) {
	const volName = "vol"
	const dir = "/disks"
	virtCli := kubevirt.Client()
	vmi, err := virtCli.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
	Expect(err).ShouldNot(HaveOccurred())
	libstorage.CreateFSPVC(name, vmi.Namespace, size, nil)
	vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	Expect(err).ShouldNot(HaveOccurred())
	volume := k8sv1.Volume{
		Name: volName,
		VolumeSource: k8sv1.VolumeSource{
			PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: name,
			},
		}}
	q := resource.MustParse(size)
	q.Sub(resource.MustParse("0.5Gi"))
	smallerSize := q.AsApproximateFloat64()
	Expect(smallerSize).Should(BeNumerically(">", 0))
	securityContext := k8sv1.SecurityContext{
		Privileged:               pointer.P(false),
		RunAsUser:                pointer.P(int64(util.NonRootUID)),
		AllowPrivilegeEscalation: pointer.P(false),
		RunAsNonRoot:             pointer.P(true),
		SeccompProfile: &k8sv1.SeccompProfile{
			Type: k8sv1.SeccompProfileTypeRuntimeDefault,
		},
		Capabilities: &k8sv1.Capabilities{
			Drop: []k8sv1.Capability{"ALL"},
		},
	}
	cont := k8sv1.Container{
		Name:       "create",
		Image:      vmiPod.Spec.Containers[0].Image,
		Command:    []string{"qemu-img", "create", "disk.img", strconv.FormatFloat(smallerSize, 'f', -1, 64)},
		WorkingDir: dir,
		VolumeMounts: []k8sv1.VolumeMount{{
			Name:      volName,
			MountPath: dir,
		}},
		SecurityContext: &securityContext,
	}
	affinity := k8sv1.Affinity{
		PodAntiAffinity: &k8sv1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
				{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							virtv1.CreatedByLabel: string(vmi.UID),
						},
					},
					TopologyKey: k8sv1.LabelHostname,
				},
			},
		},
	}
	podSecurityContext := k8sv1.PodSecurityContext{
		FSGroup: pointer.P(int64(util.NonRootUID)),
	}
	pod := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "create-img-",
			Namespace:    vmi.Namespace,
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy:   k8sv1.RestartPolicyNever,
			Volumes:         []k8sv1.Volume{volume},
			Containers:      []k8sv1.Container{cont},
			Affinity:        &affinity,
			SecurityContext: &podSecurityContext,
		},
	}
	p, err := virtCli.CoreV1().Pods(vmi.Namespace).Create(context.Background(), &pod, metav1.CreateOptions{})
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(matcher.ThisPod(p)).WithTimeout(120 * time.Second).WithPolling(time.Second).Should(matcher.HaveSucceeded())
}

func waitMigrationToExist(virtClient kubecli.KubevirtClient, vmiName, ns string) {
	Eventually(func() bool {
		ls := labels.Set{
			virtv1.VolumesUpdateMigration: vmiName,
		}
		migList, err := virtClient.VirtualMachineInstanceMigration(ns).List(context.Background(),
			metav1.ListOptions{
				LabelSelector: ls.String(),
			})
		Expect(err).ToNot(HaveOccurred())
		if len(migList.Items) < 0 {
			return false
		}
		return true

	}, 120*time.Second, time.Second).Should(BeTrue())
}

func waitForMigrationToSucceed(virtClient kubecli.KubevirtClient, vmiName, ns string) {
	waitMigrationToExist(virtClient, vmiName, ns)
	Eventually(func() bool {
		vmi, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vmiName,
			metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		if vmi.Status.MigrationState == nil {
			return false
		}
		if !vmi.Status.MigrationState.Completed {
			return false
		}

		Expect(vmi.Status.MigrationState.Failed).To(BeFalse())
		return true
	}, 120*time.Second, time.Second).Should(BeTrue())
}
