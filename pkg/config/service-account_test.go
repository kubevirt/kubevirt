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

package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

type sAVolumeInfoTest struct {
	path string
}

func (i sAVolumeInfoTest) isValidType(v *v1.Volume) bool {
	return v.ServiceAccount != nil
}
func (i sAVolumeInfoTest) getSourcePath(_ *v1.Volume) string {
	return i.path
}
func (i sAVolumeInfoTest) getIsoPath(_ *v1.Volume) string {
	return GetServiceAccountDiskPath()
}
func (i sAVolumeInfoTest) getLabel(v *v1.Volume) string {
	return ""
}

var _ = Describe("ServiceAccount", func() {
	var (
		saSourceDir string
		volInfo     sAVolumeInfoTest
	)

	BeforeEach(func() {
		var err error

		saSourceDir = GinkgoT().TempDir()
		volInfo = sAVolumeInfoTest{path: saSourceDir}
		Expect(err).NotTo(HaveOccurred())
		os.MkdirAll(saSourceDir, 0755)
		os.OpenFile(filepath.Join(saSourceDir, "token"), os.O_RDONLY|os.O_CREATE, 0666)
		os.OpenFile(filepath.Join(saSourceDir, "namespace"), os.O_RDONLY|os.O_CREATE, 0666)

		ServiceAccountDiskDir = GinkgoT().TempDir()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(saSourceDir)
		os.RemoveAll(ServiceAccountDiskDir)
	})

	It("Should create a new service account iso disk", func() {
		vmi := libvmi.New(
			libvmi.WithServiceAccountDisk("testaccount"),
		)

		err := createIsoDisksForConfigVolumes(vmi, false, volInfo)
		Expect(err).NotTo(HaveOccurred())
		_, err = os.Stat(filepath.Join(ServiceAccountDiskDir, ServiceAccountDiskName))
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should create a new service account iso disk without a Disk device", func() {
		vmi := libvmi.New()
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "serviceaccount-volume",
			VolumeSource: v1.VolumeSource{
				ServiceAccount: &v1.ServiceAccountVolumeSource{
					ServiceAccountName: "testaccount",
				},
			},
		})

		err := createIsoDisksForConfigVolumes(vmi, false, volInfo)
		Expect(err).NotTo(HaveOccurred())
		files, _ := os.ReadDir(ServiceAccountDiskDir)
		Expect(files).To(BeEmpty())
	})
})
