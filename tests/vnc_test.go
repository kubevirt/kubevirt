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

	"time"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VNC", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("A new VirtualMachineInstance", func() {
		Context("with VNC connection", func() {
			It("should allow accessing the VNC device", func() {
				By("Starting a VirtualMachineInstance")
				vmi := tests.NewRandomVMI()
				Expect(virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()).To(Succeed())
				tests.WaitForSuccessfulVMIStart(vmi)

				pipeInReader, _ := io.Pipe()
				pipeOutReader, pipeOutWriter := io.Pipe()
				defer pipeInReader.Close()
				defer pipeOutReader.Close()

				k8ResChan := make(chan error)
				readStop := make(chan string)

				go func() {
					GinkgoRecover()
					vnc, err := virtClient.VirtualMachineInstance(vmi.ObjectMeta.Namespace).VNC(vmi.ObjectMeta.Name)
					if err != nil {
						k8ResChan <- err
						return
					}

					k8ResChan <- vnc.Stream(kubecli.StreamOptions{
						In:  pipeInReader,
						Out: pipeOutWriter,
					})
				}()
				// write to FD <- pipeOutReader
				By("Reading from the VNC socket")
				go func() {
					GinkgoRecover()
					buf := make([]byte, 1024, 1024)
					// reading qemu vnc server
					n, err := pipeOutReader.Read(buf)
					if err != nil && err != io.EOF {
						Expect(err).ToNot(HaveOccurred())
						return
					}
					if n == 0 && err == io.EOF {
						log.Log.Info("zero bytes read from vnc socket.")
						return
					}
					readStop <- strings.TrimSpace(string(buf[0:n]))
				}()

				response := ""

				select {
				case response = <-readStop:
				case err = <-k8ResChan:
					Expect(err).ToNot(HaveOccurred())
				case <-time.After(45 * time.Second):
					Fail("Timout reached while waiting for valid VNC server response")
				}

				// This is the response capture by wireshark when the VNC server is contacted.
				// This verifies that the test is able to establish a connection with VNC and
				// communicate.
				By("Checking the response from VNC server")
				Expect(response).To(Equal("RFB 003.008"))
				Expect(err).To(BeNil())
			})
		})
	})
})
