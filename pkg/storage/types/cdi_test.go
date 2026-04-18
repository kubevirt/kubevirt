/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
