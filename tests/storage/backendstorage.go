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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = SIGDescribe("Backend Storage", Serial, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("Should use RWO when RWX is not supported", func() {
		var storageClass string

		By("Finding a storage class that only supports filesystem in RWO")
		sps, err := virtClient.CdiClient().CdiV1beta1().StorageProfiles().List(context.Background(), metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred()) // Assumes CDI is present. Should we also add a decorator for that?
		for _, sp := range sps.Items {
			fsrwo := false
			fsrwx := false
			for _, property := range sp.Status.ClaimPropertySets {
				if property.VolumeMode == nil || *property.VolumeMode != k8sv1.PersistentVolumeFilesystem || property.AccessModes == nil {
					continue
				}
				for _, accessMode := range property.AccessModes {
					switch accessMode {
					case k8sv1.ReadWriteMany:
						fsrwx = true
					case k8sv1.ReadWriteOnce:
						fsrwo = true
					}
				}
			}
			if fsrwo && !fsrwx && sp.Status.StorageClass != nil {
				storageClass = *sp.Status.StorageClass
				break
			}
		}
		Expect(storageClass).NotTo(BeEmpty(), "Failed to find a storage class that only supports filesystem in RWO")

		By("Setting the VM storage class to it")
		kv := libkubevirt.GetCurrentKv(virtClient)
		kv.Spec.Configuration.VMStateStorageClass = storageClass
		config.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

		By("Creating a VMI with persistent TPM")
		vmi := libvmifact.NewCirros(libnet.WithMasqueradeNetworking())
		vmi.Spec.Domain.Devices.TPM = &v1.TPMDevice{Persistent: pointer.P(true)}
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, 60)

		By("Expecting the creation of a backend storage PVC with the right storage class")
		pvcs, err := virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: "persistent-state-for=" + vmi.Name,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(pvcs.Items).To(HaveLen(1))
		pvc := pvcs.Items[0]
		Expect(pvc.Spec.StorageClassName).NotTo(BeNil())
		Expect(*pvc.Spec.StorageClassName).To(Equal(storageClass))
		Expect(pvc.Status.AccessModes).To(HaveLen(1))
		Expect(pvc.Status.AccessModes[0]).To(Equal(k8sv1.ReadWriteOnce))
	})
})
