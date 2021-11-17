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

var _ = Describe("Secret", func() {

	BeforeEach(func() {
		var err error

		SecretSourceDir, err = ioutil.TempDir("", "secret")
		Expect(err).NotTo(HaveOccurred())
		os.MkdirAll(filepath.Join(SecretSourceDir, "secret-volume", "test-dir"), 0755)
		os.OpenFile(filepath.Join(SecretSourceDir, "secret-volume", "test-dir", "test-file1"), os.O_RDONLY|os.O_CREATE, 0666)
		os.OpenFile(filepath.Join(SecretSourceDir, "secret-volume", "test-file2"), os.O_RDONLY|os.O_CREATE, 0666)

		SecretDisksDir, err = ioutil.TempDir("", "secret-disks")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(SecretSourceDir)
		os.RemoveAll(SecretDisksDir)
	})

	It("Should create a new secret iso disk", func() {
		vmi := api.NewMinimalVMI("fake-vmi")
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "secret-volume",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: "test-secret",
				},
			},
		})

		err := CreateSecretDisks(vmi, false)
		Expect(err).NotTo(HaveOccurred())
		_, err = os.Stat(filepath.Join(SecretDisksDir, "secret-volume.iso"))
		Expect(err).NotTo(HaveOccurred())
	})

})
