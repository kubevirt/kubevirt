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

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("ServiceAccount", func() {

	BeforeEach(func() {
		var err error

		ServiceAccountSourceDir, err = ioutil.TempDir("", "serviceaccount")
		Expect(err).NotTo(HaveOccurred())
		os.MkdirAll(ServiceAccountSourceDir, 0755)
		os.OpenFile(filepath.Join(ServiceAccountSourceDir, "token"), os.O_RDONLY|os.O_CREATE, 0666)
		os.OpenFile(filepath.Join(ServiceAccountSourceDir, "namespace"), os.O_RDONLY|os.O_CREATE, 0666)

		ServiceAccountDiskDir, err = ioutil.TempDir("", "serviceaccount-disk")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(ServiceAccountSourceDir)
		os.RemoveAll(ServiceAccountDiskDir)
	})

	It("Should create a new service account iso disk", func() {
		vmi := api.NewMinimalVMI("fake-vmi")
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "serviceaccount-volume",
			VolumeSource: v1.VolumeSource{
				ServiceAccount: &v1.ServiceAccountVolumeSource{
					ServiceAccountName: "testaccount",
				},
			},
		})

		err := CreateServiceAccountDisk(vmi, false)
		Expect(err).NotTo(HaveOccurred())
		_, err = os.Stat(filepath.Join(ServiceAccountDiskDir, ServiceAccountDiskName))
		Expect(err).NotTo(HaveOccurred())
	})

})
