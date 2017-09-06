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

package configdisk

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/precond"
)

var _ = Describe("ConfigDiskServer", func() {

	tmpDir, _ := ioutil.TempDir("", "configdisktest")
	owner, err := user.Current()
	if err != nil {
		panic(err)
	}
	client := NewConfigDiskClient()
	isoCreationFunc := func(isoOutFile string, inFiles []string) error {
		if isoOutFile == "noCloud" && len(inFiles) != 2 {
			return errors.New("Unexpected number of files for noCloud")
		}

		// fake creating the iso
		_, err := os.Create(isoOutFile)

		return err
	}

	BeforeSuite(func() {
		err := cloudinit.SetLocalDirectory(tmpDir)
		if err != nil {
			panic(err)
		}
		cloudinit.SetLocalDataOwner(owner.Username)
		cloudinit.SetIsoCreationFunction(isoCreationFunc)
	})

	AfterSuite(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("config-disk-server api calls", func() {
		Context("with concrete types", func() {
			It("delete unseen vm ephemeral data", func() {
				var domains []string

				domains = append(domains, "fakens1/fakedomain1")
				domains = append(domains, "fakens1/fakedomain2")
				domains = append(domains, "fakens2/fakedomain1")
				domains = append(domains, "fakens2/fakedomain2")
				domains = append(domains, "fakens3/fakedomain1")
				domains = append(domains, "fakens4/fakedomain1")

				for _, dom := range domains {
					err := os.MkdirAll(fmt.Sprintf("%s/%s/some-other-dir", tmpDir, dom), 0755)
					Expect(err).ToNot(HaveOccurred())
					msg := "fake content"
					bytes := []byte(msg)
					err = ioutil.WriteFile(fmt.Sprintf("%s/%s/some-file", tmpDir, dom), bytes, 0644)
					Expect(err).ToNot(HaveOccurred())
				}

				vmStore := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, cache.Indexers{})

				err := vmStore.Add(v1.NewVMReferenceFromNameWithNS("fakens1", "fakedomain1"))
				Expect(err).ToNot(HaveOccurred())

				err = client.UndefineUnseen(vmStore)
				Expect(err).ToNot(HaveOccurred())

				// expect this domain to still exist
				_, err = os.Stat(fmt.Sprintf("%s/fakens1/fakedomain1", tmpDir))
				Expect(err).ToNot(HaveOccurred())

				// expect these domains to not exist with local cloud init config disk data
				for idx, dom := range domains {
					exists := true
					if idx == 0 {
						continue
					}
					_, err = os.Stat(fmt.Sprintf("%s/%s", tmpDir, dom))
					if os.IsNotExist(err) {
						exists = false
					}
					Expect(exists).To(Equal(false))
				}

			})
			It("delete non existent VM ephemeral data", func() {
				vm := v1.NewMinimalVM("never-started-vm")
				err := client.Undefine(vm)
				Expect(err).ToNot(HaveOccurred())
			})
			It("define vm without config disk data.", func() {
				vm := v1.NewMinimalVM("never-started-vm")
				namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
				domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

				i := 1
				for ; i <= 10; i++ {
					isPending, err := client.Define(vm)
					Expect(err).ToNot(HaveOccurred())

					if isPending {
						time.Sleep(2 * time.Second)
					} else {
						break
					}
				}
				Expect(i).ToNot(Equal(10))

				// no config disk directory should not exist for this vm
				_, err = os.Stat(fmt.Sprintf("%s/%s/%s", tmpDir, namespace, domain))
				if os.IsNotExist(err) {
					err = nil
				}

				Expect(err).ToNot(HaveOccurred())
			})
			It("define vm with cloud init data.", func() {
				vm := v1.NewMinimalVM("never-started-vm")
				namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
				domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

				userData := "fake\nuser\ndata\n"
				metaData := "fake\nmeta\ndata\n"
				spec := &v1.CloudInitSpec{
					NoCloudData: &v1.CloudInitDataSourceNoCloud{
						UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
						MetaDataBase64: base64.StdEncoding.EncodeToString([]byte(metaData)),
					},
				}
				newDisk := v1.Disk{}
				newDisk.Type = "file"
				newDisk.Target = v1.DiskTarget{
					Device: "vdb",
				}
				newDisk.CloudInit = spec

				vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, newDisk)

				i := 1
				for ; i <= 10; i++ {
					isPending, err := client.Define(vm)
					Expect(err).ToNot(HaveOccurred())

					if isPending {
						time.Sleep(2 * time.Second)
					} else {
						break
					}
				}
				Expect(i).ToNot(Equal(10))

				Expect(err).ToNot(HaveOccurred())

				// verify iso is created
				_, err = os.Stat(fmt.Sprintf("%s/%s/%s/noCloud.iso", tmpDir, namespace, domain))
				Expect(err).ToNot(HaveOccurred())

				err = client.Undefine(vm)
				Expect(err).ToNot(HaveOccurred())

				// verify iso and entire dir is deleted
				_, err = os.Stat(fmt.Sprintf("%s/%s/%s", tmpDir, namespace, domain))
				if os.IsNotExist(err) {
					err = nil
				}
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

})
