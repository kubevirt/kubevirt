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

package tests_test

import (
	"flag"
	"io/ioutil"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Virctl command", func() {

	flag.Parse()

	Context("with valid config file", func() {

		It("should try to open VNC", func() {

			tempConfig, err := ioutil.TempFile(os.TempDir(), "kubevirt-test.*.vvv")
			if err != nil {
				Fail("Cannot create the temporary file")
			}
			defer func() {
				err := tempConfig.Close()
				if err != nil {
					Fail("Cannot close the file")
				}
			}()

			data := []byte("vnc " + tests.NamespaceTestDefault + " vmi_test_123")
			_, err = tempConfig.Write(data)

			virctl := tests.NewVirtctlCommand(tempConfig.Name())

			err = virctl()
			Expect(strings.HasPrefix(err.Error(), "Can't access VMI vmi_test_123")).To(BeTrue(), "should not be able to properly connect")
		})
	})

	Context("with normal command", func() {

		It("should work normally", func() {
			r, w, _ := os.Pipe()
			tmp := os.Stdout
			defer func() {
				os.Stdout = tmp
			}()
			os.Stdout = w

			go func() {
				defer GinkgoRecover()

				virtctl := tests.NewVirtctlCommand("help")
				virtctl()

				w.Close()
			}()

			stdout, _ := ioutil.ReadAll(r)
			Expect(strings.HasPrefix(string(stdout), "virtctl controls virtual machine related operations on your kubernetes cluster")).To(BeTrue(), "should run help normally")
		})

	})

	Context("with an invalid file", func() {

		It("should reject wrong file format", func() {
			tempConfig, err := ioutil.TempFile(os.TempDir(), "kubevirt-test.*.vvv")
			if err != nil {
				Fail("Cannot create the temporary file")
			}
			defer func() {
				err := tempConfig.Close()
				if err != nil {
					Fail("Cannot close the file")
				}
			}()

			data := []byte("vnc vmi_test_123")
			_, err = tempConfig.Write(data)

			virctl := tests.NewVirtctlCommand(tempConfig.Name())

			err = virctl()
			Expect(err.Error()).To(Equal("Invalid file format, 3 parameters required, 2 received"), "should reject the invalid file")
		})

		It("should ignore non .vvv file and continue as normal", func() {
			virtctl := tests.NewVirtctlCommand("/tmp/kubevirt-test.233242342.novvv")
			err := virtctl()
			Expect(err.Error()).To(Equal(`unknown command "/tmp/kubevirt-test.233242342.novvv" for "virtctl"`), "should report unknown command")
		})

	})
})
