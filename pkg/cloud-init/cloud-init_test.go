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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package cloudinit

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/precond"
)

var _ = Describe("CloudInit", func() {

	tmpDir, _ := ioutil.TempDir("", "cloudinittest")

	owner, err := user.Current()
	if err != nil {
		panic(err)
	}
	isoCreationFunc := func(isoOutFile string, inFiles []string) error {
		if isoOutFile == "noCloud" && len(inFiles) != 2 {
			return errors.New("Unexpected number of files for noCloud")
		}

		// fake creating the iso
		_, err := os.Create(isoOutFile)

		return err
	}

	BeforeSuite(func() {
		err := SetLocalDirectory(tmpDir)
		if err != nil {
			panic(err)
		}
		SetLocalDataOwner(owner.Username)
		SetIsoCreationFunction(isoCreationFunc)
	})

	AfterSuite(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("CloudInit Nocloud datasource", func() {
		Context("nocloud", func() {
			It("Verify no cloudinit data is a domain xml no-op", func() {
				vm := v1.NewMinimalVM("fake-vm-nocloud")

				vm, err := InjectDomainData(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(vm.Spec.Domain.Devices.Disks)).To(Equal(0))
			})
			It("Verify nocloud disk domain xml", func() {
				vm := v1.NewMinimalVM("fake-vm-nocloud")
				namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
				domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

				userData := "fake\nuser\ndata\n"
				metaData := "fake\nmeta\ndata\n"
				vm.Spec.CloudInit = &v1.CloudInitSpec{
					DataSource: "noCloud",
					NoCloudData: &v1.CloudInitDataSourceNoCloud{
						DiskTarget:     "vdb",
						UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
						MetaDataBase64: base64.StdEncoding.EncodeToString([]byte(metaData)),
					},
				}

				vm, err := InjectDomainData(vm)
				Expect(err).ToNot(HaveOccurred())
				disk := vm.Spec.Domain.Devices.Disks[0]

				expectedIso := fmt.Sprintf("%s/%s/%s/noCloud.iso", tmpDir, namespace, domain)
				Expect(disk.Type).To(Equal("file"))
				Expect(disk.Device).To(Equal("disk"))
				Expect(disk.Driver.Type).To(Equal("raw"))
				Expect(disk.Driver.Name).To(Equal("qemu"))
				Expect(disk.Source.File).To(Equal(expectedIso))
				Expect(disk.Target.Device).To(Equal("vdb"))
				Expect(disk.Target.Bus).To(Equal("virtio"))

			})
			It("delete non-existent local Nocloud data.", func() {
				namespace := "fake-namespace"
				domain := "fake-domain"
				err = RemoveLocalData(domain, namespace)
				Expect(err).ToNot(HaveOccurred())
			})
			It("define vm with Nocloud datasource.", func() {
				namespace := "fake-namespace"
				domain := "fake-domain"
				userData := "fake\nuser\ndata\n"
				metaData := "fake\nmeta\ndata\n"
				cloudInitData := &v1.CloudInitSpec{
					DataSource: "noCloud",
					NoCloudData: &v1.CloudInitDataSourceNoCloud{
						DiskTarget:     "vdb",
						UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
						MetaDataBase64: base64.StdEncoding.EncodeToString([]byte(metaData)),
					},
				}
				err := GenerateLocalData(domain, namespace, cloudInitData)
				Expect(err).ToNot(HaveOccurred())

				// verify iso is created
				_, err = os.Stat(fmt.Sprintf("%s/%s/%s/noCloud.iso", tmpDir, namespace, domain))
				Expect(err).ToNot(HaveOccurred())

				err = RemoveLocalData(domain, namespace)
				Expect(err).ToNot(HaveOccurred())

				// verify iso and entire dir is deleted
				_, err = os.Stat(fmt.Sprintf("%s/%s/%s", tmpDir, namespace, domain))
				if os.IsNotExist(err) {
					err = nil
				}
				Expect(err).ToNot(HaveOccurred())
			})
			It("Verify no cloudinit metadata auto-gen is no-op when metadata exists already", func() {
				vm := v1.NewMinimalVM("fake-vm-nocloud")

				userData := "fake\nuser\ndata\n"
				metaData := "fake\nmeta\ndata\n"
				metaData64 := base64.StdEncoding.EncodeToString([]byte(metaData))
				vm.Spec.CloudInit = &v1.CloudInitSpec{
					DataSource: "noCloud",
					NoCloudData: &v1.CloudInitDataSourceNoCloud{
						DiskTarget:     "vdb",
						UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
						MetaDataBase64: metaData64,
					},
				}

				ApplyMetadata(vm)
				Expect(vm.Spec.CloudInit.NoCloudData.MetaDataBase64).To(Equal(metaData64))
			})

			It("Verify no cloudinit metadata auto-generated when metadata does not exist", func() {
				vm := v1.NewMinimalVM("fake-vm-nocloud")

				userData := "fake\nuser\ndata\n"
				vm.Spec.CloudInit = &v1.CloudInitSpec{
					DataSource: "noCloud",
					NoCloudData: &v1.CloudInitDataSourceNoCloud{
						DiskTarget:     "vdb",
						UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
						MetaDataBase64: "",
					},
				}

				ApplyMetadata(vm)
				Expect(vm.Spec.CloudInit.NoCloudData.MetaDataBase64).ToNot(Equal(""))
			})
		})
	})
})
