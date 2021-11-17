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
	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"
)

func createFile(fullPath string) {
	f, err := os.OpenFile(fullPath, os.O_RDONLY|os.O_CREATE, 0666)
	Expect(err).NotTo(HaveOccurred())
	if f != nil {
		f.Close()
	}
}

var _ = Describe("SysprepConfigMap", func() {

	BeforeEach(func() {
		var err error

		SysprepSourceDir, err = ioutil.TempDir("", "sysprep")
		Expect(err).NotTo(HaveOccurred())
		os.MkdirAll(filepath.Join(SysprepSourceDir, "sysprep-volume"), 0755)

		SysprepDisksDir, err = ioutil.TempDir("", "sysprep-disks")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(SysprepSourceDir)
		os.RemoveAll(SysprepDisksDir)
	})

	vmiConfigMap := api.NewMinimalVMI("fake-vmi")
	vmiConfigMap.Spec.Volumes = append(vmiConfigMap.Spec.Volumes, v1.Volume{
		Name: "sysprep-volume",
		VolumeSource: v1.VolumeSource{
			Sysprep: &v1.SysprepSource{
				ConfigMap: &k8sv1.LocalObjectReference{
					Name: "test-config",
				},
			},
		},
	})

	vmiSecret := api.NewMinimalVMI("fake-vmi")
	vmiSecret.Spec.Volumes = append(vmiSecret.Spec.Volumes, v1.Volume{
		Name: "sysprep-volume",
		VolumeSource: v1.VolumeSource{
			Sysprep: &v1.SysprepSource{
				Secret: &k8sv1.LocalObjectReference{
					Name: "secret-config",
				},
			},
		},
	})

	Describe("With invalid file name", func() {
		BeforeEach(func() {
			createFile(filepath.Join(SysprepSourceDir, "sysprep-volume", "wrongname.xml"))
		})

		It("Should fail on creating config map iso disk", func() {
			err := CreateSysprepDisks(vmiConfigMap, false)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("With valid file name (autounattend.xml)", func() {
		BeforeEach(func() {
			// Check case-insensitivity (should accept anything Windows accepts).
			createFile(filepath.Join(SysprepSourceDir, "sysprep-volume", "AutounattenD.xml"))
		})

		It("Should create a new config map iso disk", func() {
			err := CreateSysprepDisks(vmiConfigMap, false)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(SysprepDisksDir, "sysprep-volume.iso"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should create a new secret iso disk", func() {
			err := CreateSysprepDisks(vmiSecret, false)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(SysprepDisksDir, "sysprep-volume.iso"))
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
