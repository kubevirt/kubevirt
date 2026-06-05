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

package types

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	virtv1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

var _ = Describe("CDI utils test", func() {
	It("should return 0 with block volume mode", func() {
		volumeMode := corev1.PersistentVolumeBlock
		storageClass := "sc"
		cdiConfig := &cdiv1.CDIConfig{
			Status: cdiv1.CDIConfigStatus{
				FilesystemOverhead: &cdiv1.FilesystemOverhead{
					Global: "10",
				},
			},
		}
		overhead, err := GetFilesystemOverhead(&volumeMode, &storageClass, cdiConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(overhead).To(Equal(virtv1.Percent("0")))
	})

	It("should return error with uninitialized cdi config", func() {
		volumeMode := corev1.PersistentVolumeFilesystem
		storageClass := "sc"
		cdiConfig := &cdiv1.CDIConfig{}
		overhead, err := GetFilesystemOverhead(&volumeMode, &storageClass, cdiConfig)
		Expect(err).To(HaveOccurred())
		Expect(overhead).To(Equal(virtv1.Percent("0")))
	})

	It("should return global overhead with nil storage class", func() {
		volumeMode := corev1.PersistentVolumeFilesystem
		cdiConfig := &cdiv1.CDIConfig{
			Status: cdiv1.CDIConfigStatus{
				FilesystemOverhead: &cdiv1.FilesystemOverhead{
					Global: "10",
				},
			},
		}
		overhead, err := GetFilesystemOverhead(&volumeMode, nil, cdiConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(overhead).To(Equal(virtv1.Percent("10")))
	})

	It("should return global overhead with unknown storage class", func() {
		volumeMode := corev1.PersistentVolumeFilesystem
		storageClass := "sc"
		cdiConfig := &cdiv1.CDIConfig{
			Status: cdiv1.CDIConfigStatus{
				FilesystemOverhead: &cdiv1.FilesystemOverhead{
					Global: "10",
				},
			},
		}
		overhead, err := GetFilesystemOverhead(&volumeMode, &storageClass, cdiConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(overhead).To(Equal(virtv1.Percent("10")))
	})

	It("should return storage class overhead", func() {
		volumeMode := corev1.PersistentVolumeFilesystem
		storageClass := "sc"
		cdiConfig := &cdiv1.CDIConfig{
			Status: cdiv1.CDIConfigStatus{
				FilesystemOverhead: &cdiv1.FilesystemOverhead{
					Global:       "10",
					StorageClass: map[string]cdiv1.Percent{"sc": "20"},
				},
			},
		}
		overhead, err := GetFilesystemOverhead(&volumeMode, &storageClass, cdiConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(overhead).To(Equal(virtv1.Percent("20")))
	})
})
