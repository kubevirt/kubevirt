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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/api"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("DownwardAPI", func() {

	BeforeEach(func() {
		var err error

		DownwardAPISourceDir, err = ioutil.TempDir("", "downwardapi")
		Expect(err).NotTo(HaveOccurred())
		os.MkdirAll(filepath.Join(DownwardAPISourceDir, "downwardapi-volume", "test-dir"), 0755)
		os.OpenFile(filepath.Join(DownwardAPISourceDir, "downwardapi-volume", "test-dir", "test-file1"), os.O_RDONLY|os.O_CREATE, 0666)
		os.OpenFile(filepath.Join(DownwardAPISourceDir, "downwardapi-volume", "test-file2"), os.O_RDONLY|os.O_CREATE, 0666)

		DownwardAPIDisksDir, err = ioutil.TempDir("", "downwardapi-disks")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(DownwardAPISourceDir)
		os.RemoveAll(DownwardAPIDisksDir)
	})

	It("Should create a new downwardapi iso disk", func() {
		vmi := api.NewMinimalVMI("fake-vmi")
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "downwardapi-volume",
			VolumeSource: v1.VolumeSource{
				DownwardAPI: &v1.DownwardAPIVolumeSource{
					Fields: []k8sv1.DownwardAPIVolumeFile{
						{
							Path: "labels",
							FieldRef: &k8sv1.ObjectFieldSelector{
								FieldPath: "metadata.labels",
							},
						},
					},
				},
			},
		})

		err := CreateDownwardAPIDisks(vmi, false)
		Expect(err).NotTo(HaveOccurred())
		_, err = os.Stat(filepath.Join(DownwardAPIDisksDir, "downwardapi-volume.iso"))
		Expect(err).NotTo(HaveOccurred())
	})

})
