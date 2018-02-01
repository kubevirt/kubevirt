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

package tests_test

import (
	"flag"
	"io"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Vmlifecycle", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("New VM with a vnc connection given", func() {
		It("should allow accessing the vnc device on the VM", func(done Done) {
			vm := tests.NewRandomVM()
			Expect(virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Error()).To(Succeed())
			tests.WaitForSuccessfulVMStart(vm)

			tests.WaitForSuccessfulVMStart(vm)

			pipeInReader, _ := io.Pipe()
			pipeOutReader, pipeOutWriter := io.Pipe()

			k8ResChan := make(chan error)
			readStop := make(chan string)

			go func() {
				err := virtClient.VM(vm.ObjectMeta.Namespace).VNC(vm.ObjectMeta.Name, pipeInReader, pipeOutWriter)
				k8ResChan <- err
			}()
			// write to FD <- pipeOutReader
			go func() {
				buf := make([]byte, 1024, 1024)
				// reading qemu vnc server
				n, err := pipeOutReader.Read(buf)
				if err != nil && err != io.EOF {
					log.Log.Reason(err).Error("error while reading from vnc socket.")
					return
				}
				if n == 0 && err == io.EOF {
					log.Log.Error("zero bytes read from vnc socket.")
					return
				}
				readStop <- strings.TrimSpace(string(buf[0:n]))
			}()

			response := ""

			select {
			case response = <-readStop:
			case err = <-k8ResChan:
			}

			// This is the response capture by wireshark when the VNC server is contacted.
			// This verifies that the test is able to establish a connection with VNC and
			// communicate.
			Expect(response).To(Equal("RFB 003.008"))
			Expect(err).To(BeNil())
			close(done)
		}, 45)
	})
})
