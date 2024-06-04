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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libkvconfig"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	util2 "kubevirt.io/kubevirt/tests/util"
)

var _ = SIGDescribe("[Serial]Volumes update with migration", Serial, func() {
	var virtClient kubecli.KubevirtClient
	BeforeEach(func() {
		checks.SkipIfMigrationIsNotPossible()
		virtClient = kubevirt.Client()
		originalKv := util2.GetCurrentKv(virtClient)
		updateStrategy := &virtv1.KubeVirtWorkloadUpdateStrategy{
			WorkloadUpdateMethods: []virtv1.WorkloadUpdateMethod{virtv1.WorkloadUpdateMethodLiveMigrate},
		}
		rolloutStrategy := pointer.P(virtv1.VMRolloutStrategyLiveUpdate)
		libkvconfig.PatchWorkloadUpdateMethodAndRolloutStrategy(originalKv.Name, virtClient, updateStrategy, rolloutStrategy,
			[]string{virtconfig.VMLiveUpdateFeaturesGate, virtconfig.VolumesUpdateStrategy, virtconfig.VolumeMigration})

		currentKv := util2.GetCurrentKv(virtClient)
		tests.WaitForConfigToBePropagatedToComponent(
			"kubevirt.io=virt-controller",
			currentKv.ResourceVersion,
			tests.ExpectResourceVersionToBeLessEqualThanConfigVersion,
			time.Minute)
	})

	Describe("Update volumes with the migration updateVolumesStrategy", func() {
		var (
			ns      string
			destPVC string
		)
		const (
			fsPVC    = "filesystem"
			blockPVC = "block"
			size     = "1Gi"
		)

		waitMigrationToExist := func(vmiName, ns string) {
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

		waitForMigrationToSucceed := func(vmiName, ns string) {
			waitMigrationToExist(vmiName, ns)
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
		createDV := func() *cdiv1.DataVolume {
			sc, exist := libstorage.GetRWOFileSystemStorageClass()
			Expect(exist).To(BeTrue())
			dv := libdv.NewDataVolume(
				libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
				libdv.WithPVC(libdv.PVCWithStorageClass(sc),
					libdv.PVCWithVolumeSize(size),
				),
			)
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(ns).Create(context.Background(),
				dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			return dv
		}
		createBlankDV := func() *cdiv1.DataVolume {
			sc, exist := libstorage.GetRWOFileSystemStorageClass()
			Expect(exist).To(BeTrue())
			dv := libdv.NewDataVolume(
				libdv.WithBlankImageSource(),
				libdv.WithPVC(libdv.PVCWithStorageClass(sc),
					libdv.PVCWithVolumeSize(size),
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
				libvmi.WithCloudInitNoCloudEncodedUserData(("#!/bin/bash\necho hello\n")),
			)
			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithRunning(),
				libvmi.WithDataVolumeTemplate(dv),
			)
			vm, err := virtClient.VirtualMachine(ns).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
			libwait.WaitForSuccessfulVMIStart(vmi)

			return vm
		}

		updateVMWithPVC := func(vmName, volName, claim string) {
			var replacedIndex int
			vm, err := virtClient.VirtualMachine(ns).Get(context.Background(), vmName, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			// Remove datavolume templates
			vm.Spec.DataVolumeTemplates = []virtv1.DataVolumeTemplateSpec{}
			// Replace dst pvc
			for i, v := range vm.Spec.Template.Spec.Volumes {
				if v.Name == volName {
					By(fmt.Sprintf("Replacing volume %s with PVC %s", volName, claim))
					vm.Spec.Template.Spec.Volumes[i].VolumeSource.PersistentVolumeClaim = &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: claim,
						},
					}
					vm.Spec.Template.Spec.Volumes[i].VolumeSource.DataVolume = nil
					replacedIndex = i
					break
				}
			}
			vm.Spec.UpdateVolumesStrategy = pointer.P(virtv1.UpdateVolumesStrategyMigration)
			vm, err = virtClient.VirtualMachine(ns).Update(context.Background(), vm, metav1.UpdateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(vm.Spec.Template.Spec.Volumes[replacedIndex].VolumeSource.PersistentVolumeClaim.
				PersistentVolumeClaimVolumeSource.ClaimName).To(Equal(claim))

		}
		// TODO: right now, for simplicity, this function assumes the DV in the first position in the datavolumes templata list. Otherwise, we need
		// to pass the old name of the DV to be replaces.
		updateVMWithDV := func(vmName, volName, name string) {
			var replacedIndex int
			vm, err := virtClient.VirtualMachine(ns).Get(context.Background(), vmName, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			vm.Spec.DataVolumeTemplates[0].Name = name
			for i, v := range vm.Spec.Template.Spec.Volumes {
				if v.Name == volName {
					vm.Spec.Template.Spec.Volumes[i].VolumeSource.DataVolume = &virtv1.DataVolumeSource{
						Name: name,
					}
					replacedIndex = i
					break
				}
			}
			vm.Spec.UpdateVolumesStrategy = pointer.P(virtv1.UpdateVolumesStrategyMigration)
			vm, err = virtClient.VirtualMachine(ns).Update(context.Background(), vm, metav1.UpdateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(vm.Spec.Template.Spec.Volumes[replacedIndex].VolumeSource.DataVolume.Name).To(Equal(name))
		}

		BeforeEach(func() {
			ns = testsuite.GetTestNamespace(nil)
			destPVC = "dest-" + rand.String(5)

		})

		DescribeTable("should migrate the source volume from a source DV to a destination PVC", func(mode string) {
			volName := "disk0"
			vm := createVMWithDV(createDV(), volName)
			// Create dest PVC
			switch mode {
			case fsPVC:
				libstorage.CreateFSPVC(destPVC, ns, size, nil)
			case blockPVC:
				libstorage.CreateBlockPVC(destPVC, ns, size)
			default:
				Fail("Unrecognized mode")
			}
			By("Update volumes")
			updateVMWithPVC(vm.Name, volName, destPVC)
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vm.Name,
					metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				claim := storagetypes.PVCNameFromVirtVolume(&vmi.Spec.Volumes[0])
				return claim == destPVC
			}, 120*time.Second, time.Second).Should(BeTrue())
			waitForMigrationToSucceed(vm.Name, ns)
		},
			Entry("to a filesystem volume", fsPVC),
			Entry("to a block volume", blockPVC),
		)

		It("should migrate the source volume from a source DV to a destination DV", func() {
			volName := "disk0"
			vm := createVMWithDV(createDV(), volName)
			destDV := createBlankDV()
			By("Update volumes")
			updateVMWithDV(vm.Name, volName, destDV.Name)
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vm.Name,
					metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				claim := storagetypes.PVCNameFromVirtVolume(&vmi.Spec.Volumes[0])
				return claim == destDV.Name
			}, 120*time.Second, time.Second).Should(BeTrue())
			waitForMigrationToSucceed(vm.Name, ns)
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
				libvmi.WithRunning(),
			)
			vm, err := virtClient.VirtualMachine(ns).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Update volumes")
			updateVMWithPVC(vm.Name, volName, destPVC)
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
			waitForMigrationToSucceed(vm.Name, ns)
		})

		It("should cancel the migration by the reverting to the source volume", func() {
			volName := "volume"
			dv := createDV()
			vm := createVMWithDV(dv, volName)
			// Create dest PVC
			createUnschedulablePVC(destPVC, ns, size)
			By("Update volumes")
			updateVMWithPVC(vm.Name, volName, destPVC)
			waitMigrationToExist(vm.Name, ns)
			waitVMIToHaveVolumeChangeCond(vm.Name, ns)
			By("Cancel the volume migration")
			updateVMWithPVC(vm.Name, volName, dv.Name)
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
	})
})

func createUnschedulablePVC(name, namespace, size string) *k8sv1.PersistentVolumeClaim {
	pvc := libstorage.NewPVC(name, size, "dontexist")
	pvc.Spec.VolumeMode = pointer.P(k8sv1.PersistentVolumeFilesystem)
	virtCli := kubevirt.Client()
	createdPvc, err := virtCli.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	Expect(err).ShouldNot(HaveOccurred())

	return createdPvc
}
