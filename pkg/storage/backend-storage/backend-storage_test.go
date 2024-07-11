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

package backendstorage

import (
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Backend Storage", func() {
	var backendStorage *BackendStorage
	var config *virtconfig.ClusterConfig
	var kvStore cache.Store
	var storageClassInformer cache.SharedIndexInformer
	var storageProfileInformer cache.SharedIndexInformer

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		kubevirtFakeConfig := &virtv1.KubeVirtConfiguration{}
		config, _, kvStore = testutils.NewFakeClusterConfigUsingKVConfig(kubevirtFakeConfig)
		storageClassInformer, _ = testutils.NewFakeInformerFor(&storagev1.StorageClass{})
		storageProfileInformer, _ = testutils.NewFakeInformerFor(&v1beta1.StorageProfile{})
		pvcInformer, _ := testutils.NewFakeInformerFor(&v1.PersistentVolumeClaim{})

		backendStorage = NewBackendStorage(virtClient, config, storageClassInformer.GetStore(), storageProfileInformer.GetStore(), pvcInformer.GetIndexer())
	})

	Context("Storage class", func() {
		It("Should return VMStateStorageClass and RWX when set", func() {
			By("Setting a VM state storage class in the CR")
			kvCR := testutils.GetFakeKubeVirtClusterConfig(kvStore)
			kvCR.Spec.Configuration.VMStateStorageClass = "myfave"
			testutils.UpdateFakeKubeVirtClusterConfig(kvStore, kvCR)

			By("Expecting GetStorageClass() to return that one")
			sc, err := backendStorage.GetStorageClass()
			Expect(err).NotTo(HaveOccurred())
			Expect(sc).To(Equal("myfave"))

			By("Expecting GetModes() to return RWX")
			accessMode, volumeMode := backendStorage.GetModes(sc)
			Expect(accessMode).To(Equal(v1.ReadWriteMany))
			Expect(volumeMode).To(Equal(v1.PersistentVolumeFilesystem))
		})

		It("Should return the default storage class when VMStateStorageClass is not set", func() {
			By("Creating 5 storage classes with one default")
			for i := 0; i < 5; i++ {
				sc := storagev1.StorageClass{
					ObjectMeta: k8smetav1.ObjectMeta{
						Name: fmt.Sprintf("sc%d", i),
					},
				}
				if i == 3 {
					sc.Annotations = map[string]string{"storageclass.kubernetes.io/is-default-class": "true"}
				}
				err := storageClassInformer.GetStore().Add(&sc)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Expecting GetStorageClass() to return the default one")
			sc, err := backendStorage.GetStorageClass()
			Expect(err).NotTo(HaveOccurred())
			Expect(sc).To(Equal("sc3"))

			By("Expecting GetModes() to return RWO")
			accessMode, volumeMode := backendStorage.GetModes(sc)
			Expect(accessMode).To(Equal(v1.ReadWriteOnce))
			Expect(volumeMode).To(Equal(v1.PersistentVolumeBlock))
		})
	})

	Context("Access mode", func() {
		BeforeEach(func() {
			By("Creating a storage profile with no access/volume mode")
			sp := &v1beta1.StorageProfile{
				ObjectMeta: k8smetav1.ObjectMeta{
					Name: "nomode",
				},
				Spec: v1beta1.StorageProfileSpec{},
				Status: v1beta1.StorageProfileStatus{
					ClaimPropertySets: []v1beta1.ClaimPropertySet{},
				},
			}
			err := storageProfileInformer.GetStore().Add(sp)
			Expect(err).NotTo(HaveOccurred())

			By("Creating a storage profile with RWO FS as its only mode")
			sp = sp.DeepCopy()
			sp.Name = "onlyrwo"
			sp.Status.ClaimPropertySets = []v1beta1.ClaimPropertySet{{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				VolumeMode:  pointer.P(v1.PersistentVolumeFilesystem),
			}}
			err = storageProfileInformer.GetStore().Add(sp)
			Expect(err).NotTo(HaveOccurred())

			By("Creating a storage profile that supports FS in both RWO and RWX")
			sp = sp.DeepCopy()
			sp.Name = "both"
			sp.Status.ClaimPropertySets = []v1beta1.ClaimPropertySet{{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany, v1.ReadWriteOnce},
				VolumeMode:  pointer.P(v1.PersistentVolumeFilesystem),
			}}
			err = storageProfileInformer.GetStore().Add(sp)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should default to RWO when no storage profile is defined", func() {
			accessMode, volumeMode := backendStorage.GetModes("doesntexist")
			Expect(accessMode).To(Equal(v1.ReadWriteOnce))
			Expect(volumeMode).To(Equal(v1.PersistentVolumeBlock))
		})

		It("Should default to RWO when the storage profile doesn't have any access mode", func() {
			accessMode, volumeMode := backendStorage.GetModes("nomode")
			Expect(accessMode).To(Equal(v1.ReadWriteOnce))
			Expect(volumeMode).To(Equal(v1.PersistentVolumeBlock))
		})

		It("Should pick RWX when both RWX and RWO are available", func() {
			accessMode, volumeMode := backendStorage.GetModes("both")
			Expect(accessMode).To(Equal(v1.ReadWriteMany))
			Expect(volumeMode).To(Equal(v1.PersistentVolumeFilesystem))
		})

		It("Should pick RWO when RWX isn't possible", func() {
			accessMode, volumeMode := backendStorage.GetModes("onlyrwo")
			Expect(accessMode).To(Equal(v1.ReadWriteOnce), fmt.Sprintf("%#v", storageProfileInformer.GetStore().ListKeys()))
			Expect(volumeMode).To(Equal(v1.PersistentVolumeFilesystem))
		})
	})
})
