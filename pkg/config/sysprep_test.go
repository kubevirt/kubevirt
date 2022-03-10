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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"
)

func createFiles(filenames []string) {
	for _, f := range filenames {
		f, err := os.OpenFile(filepath.Join(SysprepSourceDir, "sysprep-volume", f), os.O_RDONLY|os.O_CREATE, 0666)
		Expect(err).NotTo(HaveOccurred())
		if f != nil {
			f.Close()
		}
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

	DescribeTable("Assert successful sysprep ISO creation with CreateSysprepDisks",
		func(vmi *v1.VirtualMachineInstance, filenames []string) {
			createFiles(filenames)
			err := CreateSysprepDisks(vmi, false)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(SysprepDisksDir, "sysprep-volume.iso"))
			Expect(err).NotTo(HaveOccurred())
		},
		Entry("Should pass when using a configMap and finding valid filenames", vmiConfigMap, []string{"AutounattenD.xml", "UnattenD.xml"}),
		Entry("Should pass when using a configMap and finding only autoattend.xml", vmiConfigMap, []string{"Autounattend.xml"}),
		Entry("Should pass when using a configMap and finding only unattend.xml", vmiConfigMap, []string{"Unattend.xml"}),
		Entry("Should pass when using a secret and finding valid filenames", vmiSecret, []string{"AutounattenD.xml", "UnattenD.xml"}),
		Entry("Should pass when using a secret and finding only autoattend.xml", vmiSecret, []string{"Autounattend.xml"}),
		Entry("Should pass when using a secret and finding only unattend.xml", vmiSecret, []string{"Unattend.xml"}),
	)

	DescribeTable("Assert failures when creating sysprep ISO with CreateSysprepDisks",
		func(vmi *v1.VirtualMachineInstance, filenames []string) {
			createFiles(filenames)
			err := CreateSysprepDisks(vmi, false)
			Expect(err).To(HaveOccurred())
		},
		Entry("Should fail when using a configMap and finding no filenames", vmiConfigMap, []string{}),
		Entry("Should fail when using a configMap and finding incorrect filenames", vmiConfigMap, []string{"wrongname.xml", "foobar.xml"}),
		Entry("Should fail when using a secret and finding no filenames", vmiSecret, []string{}),
		Entry("Should fail when using a secret and finding incorrect filenames", vmiSecret, []string{"wrongname.xml", "foobar.xml"}),
	)
})
