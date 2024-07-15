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
 * Copyright 2024 The KubeVirt Contributors
 *
 */

package storage

import (
	"context"

	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"

	"kubevirt.io/kubevirt/tests/libnet"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = SIGDescribe("[Serial]Backend Storage", Serial, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	// TODO: create a RequiresRWXFilesystemStorage decorator to remove the skip?
	It("Should use RWO when RWX is not supported", func() {
		var storageClass string

		By("Finding a storage class that only supports filesystem in RWO")
		sps, err := virtClient.CdiClient().CdiV1beta1().StorageProfiles().List(context.Background(), metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred()) // Assumes CDI is present. Should we also add a decorator for that?
		for _, sp := range sps.Items {
			fsrwo := false
			fsrwx := false
			for _, property := range sp.Status.ClaimPropertySets {
				if property.VolumeMode == nil || *property.VolumeMode != corev1.PersistentVolumeFilesystem || property.AccessModes == nil {
					continue
				}
				for _, accessMode := range property.AccessModes {
					switch accessMode {
					case corev1.ReadWriteMany:
						fsrwx = true
					case corev1.ReadWriteOnce:
						fsrwo = true
					}
				}
			}
			if fsrwo && !fsrwx && sp.Status.StorageClass != nil {
				storageClass = *sp.Status.StorageClass
				break
			}
		}
		if storageClass == "" {
			Skip("Failed to find a suitable storage class") // See TODO above
		}

		By("Setting the VM storage class to it")
		kv := util.GetCurrentKv(virtClient)
		kv.Spec.Configuration.VMStateStorageClass = storageClass
		tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

		By("Creating a VMI with persistent TPM")
		vmi := libvmifact.NewCirros(libnet.WithMasqueradeNetworking())
		vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{Persistent: pointer.P(true)}
		vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

		By("Expecting the creation of a backend storage PVC with the right storage class")
		pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Get(context.Background(), backendstorage.PVCForVMI(vmi), metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(pvc.Spec.StorageClassName).NotTo(BeNil())
		Expect(*pvc.Spec.StorageClassName).To(Equal(storageClass))
		Expect(pvc.Status.AccessModes).To(HaveLen(1))
		Expect(pvc.Status.AccessModes[0]).To(Equal(corev1.ReadWriteOnce))

		By("Expecting the VMI to be non-migratable")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		cond := controller.NewVirtualMachineInstanceConditionManager().GetCondition(vmi, v1.VirtualMachineInstanceIsMigratable)
		Expect(cond).NotTo(BeNil(), "LiveMigratable condition not found")
		Expect(cond.Status).To(Equal(corev1.ConditionFalse))
		Expect(cond.Reason).To(Equal(v1.VirtualMachineInstanceReasonDisksNotMigratable))
		Expect(cond.Message).To(ContainSubstring("Backend storage PVC is not RWX"))
	})

	Context("Block storage", func() {
		BeforeEach(func() {
			_ = virtClient.StorageV1().StorageClasses().Delete(context.Background(), "testclass", metav1.DeleteOptions{})
			_ = virtClient.CdiClient().CdiV1beta1().StorageProfiles().Delete(context.Background(), "testclass", metav1.DeleteOptions{})
		})
		AfterEach(func() {
			_ = virtClient.StorageV1().StorageClasses().Delete(context.Background(), "testclass", metav1.DeleteOptions{})
			_ = virtClient.CdiClient().CdiV1beta1().StorageProfiles().Delete(context.Background(), "testclass", metav1.DeleteOptions{})
		})
		DescribeTable("when using block storage", func(withUnsafeMigration, shouldBeMigratable bool) {
			var storageClass string

			By("Finding a storage class that supports block in RWX")
			sps, err := virtClient.CdiClient().CdiV1beta1().StorageProfiles().List(context.Background(), metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			for _, sp := range sps.Items {
				found := false
				for _, property := range sp.Status.ClaimPropertySets {
					if property.VolumeMode == nil || *property.VolumeMode != corev1.PersistentVolumeBlock || property.AccessModes == nil {
						continue
					}
					for _, accessMode := range property.AccessModes {
						if accessMode == corev1.ReadWriteMany {
							found = true
							break
						}
					}
				}
				if found && sp.Status.StorageClass != nil {
					storageClass = *sp.Status.StorageClass
					break
				}
			}
			if storageClass == "" {
				Skip("Failed to find a suitable storage class")
			}

			By("Cloning the storage class")
			sc, err := virtClient.StorageV1().StorageClasses().Get(context.Background(), storageClass, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			newsc := sc.DeepCopy()
			newsc.Name = "testclass"
			newsc.ResourceVersion = ""
			newsc.UID = ""
			_, err = virtClient.StorageV1().StorageClasses().Create(context.Background(), newsc, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Cloning the storage profile reducing modes to only block RWX")
			var sp *v1beta1.StorageProfile
			// Wait for storage profile auto-creation
			Eventually(func() error {
				sp, err = virtClient.CdiClient().CdiV1beta1().StorageProfiles().Get(context.Background(), "testclass", metav1.GetOptions{})
				return err
			}).Should(BeNil())
			newsp := sp.DeepCopy()
			newsp.Spec.ClaimPropertySets = []v1beta1.ClaimPropertySet{{
				VolumeMode:  pointer.P(corev1.PersistentVolumeBlock),
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			}}
			_, err = virtClient.CdiClient().CdiV1beta1().StorageProfiles().Update(context.Background(), newsp, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() []v1beta1.ClaimPropertySet {
				sp, err = virtClient.CdiClient().CdiV1beta1().StorageProfiles().Get(context.Background(), "testclass", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return sp.Status.ClaimPropertySets
			}).Should(Equal(newsp.Spec.ClaimPropertySets))

			By("Setting the VM storage class and feature gate in the CR")
			kv := util.GetCurrentKv(virtClient)
			kv.Spec.Configuration.VMStateStorageClass = "testclass"
			if kv.Spec.Configuration.MigrationConfiguration == nil {
				kv.Spec.Configuration.MigrationConfiguration = pointer.P(v1.MigrationConfiguration{})
			}
			if withUnsafeMigration {
				kv.Spec.Configuration.MigrationConfiguration.UnsafeBlockBackendStorageMigration = pointer.P(true)
			}
			tests.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

			By("Creating a VMI with persistent TPM")
			vmi := libvmifact.NewCirros(libnet.WithMasqueradeNetworking())
			vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{Persistent: pointer.P(true)}
			vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

			By("Expecting the creation of a backend storage PVC with the right storage class")
			pvc, err := virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Get(context.Background(), backendstorage.PVCForVMI(vmi), metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(pvc.Spec.StorageClassName).NotTo(BeNil())
			Expect(*pvc.Spec.StorageClassName).To(Equal("testclass"))
			Expect(pvc.Status.AccessModes).To(HaveLen(1))
			Expect(pvc.Status.AccessModes[0]).To(Equal(corev1.ReadWriteMany))

			By("Expecting the VMI to be migratable only if the feature gate is set")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			cond := controller.NewVirtualMachineInstanceConditionManager().GetCondition(vmi, v1.VirtualMachineInstanceIsMigratable)
			Expect(cond).NotTo(BeNil(), "LiveMigratable condition not found")
			if shouldBeMigratable {
				Expect(cond.Status).To(Equal(corev1.ConditionTrue), "VMI should be migratable")
			} else {
				Expect(cond.Status).To(Equal(corev1.ConditionFalse), "VMI should not be migratable")
				Expect(cond.Reason).To(Equal(v1.VirtualMachineInstanceReasonDisksNotMigratable))
				Expect(cond.Message).To(ContainSubstring("Backend storage PVC is Block"))
			}
		},
			Entry("with the unsafe option is set, the VM should be migratable", true, true),
			Entry("without the unsafe option, the VM should be non-migratable", false, false),
		)
	})
})
