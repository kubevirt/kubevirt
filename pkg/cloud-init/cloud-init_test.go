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
	"os/exec"
	"os/user"
	"strings"
	"time"

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

	BeforeEach(func() {
		SetIsoCreationFunction(isoCreationFunc)
	})

	AfterSuite(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("CloudInit Nocloud datasource", func() {
		Context("nocloud", func() {
			It("Verify no cloudinit data exec timeout works", func() {

				timedOut := false
				customCreationFunc := func(isoOutFile string, inFiles []string) error {
					var args []string

					args = append(args, "10")
					cmd := exec.Command("sleep", args...)

					err := cmd.Start()
					if err != nil {
						return err
					}

					done := make(chan error)
					go func() { done <- cmd.Wait() }()

					timeout := time.After(1 * time.Second)

					for {
						select {
						case <-timeout:
							cmd.Process.Kill()
							timedOut = true
						case err := <-done:
							if err != nil {
								return err
							}
							return nil
						}
					}
				}
				SetIsoCreationFunction(customCreationFunc)

				namespace := "fake-namespace"
				domain := "fake-domain"
				userData := "fake\nuser\ndata\n"
				cloudInitData := &v1.CloudInitNoCloudSource{
					UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
				}
				err := GenerateLocalData(domain, namespace, cloudInitData)
				Expect(err).To(HaveOccurred())
				Expect(timedOut).To(Equal(true))
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
				cloudInitData := &v1.CloudInitNoCloudSource{
					UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
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

			It("Verify listing VMs based on local nocloud data", func() {

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

				vms, err := ListVmWithLocalData()
				for _, vm := range vms {
					namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
					domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

					isNamespace := strings.Contains(namespace, "fakens")
					isDomain := strings.Contains(domain, "fakedomain")
					Expect(isNamespace).To(Equal(true))
					Expect(isDomain).To(Equal(true))
				}

				Expect(len(vms)).To(Equal(len(domains)))
				Expect(err).ToNot(HaveOccurred())

				// verify a vm from each namespace is present
			})
		})
	})
})
