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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package tests_test

import (
	"bytes"
	"fmt"
	"io"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[sig-compute]PortForward", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	var (
		LaunchVMI func(*v1.VirtualMachineInstance) *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		tests.BeforeTestCleanup()

		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		LaunchVMI = tests.VMILauncherIgnoreWarnings(virtClient)
	})

	It("should successfully open connection to guest", func() {
		vmi := tests.NewRandomFedoraVMIWithGuestAgent()
		vmi.Namespace = util.NamespaceTestDefault

		LaunchVMI(vmi)

		By("Opening PortForward Tunnel to SSH port")
		var (
			tunnel kubecli.StreamInterface
			err    error
		)
		Eventually(func() error {
			tunnel, err = virtClient.VirtualMachineInstance(vmi.Namespace).PortForward(vmi.Name, 22, "tcp")
			if err != nil {
				return err
			}
			return nil
		}, 12*60*time.Second, 2).ShouldNot(HaveOccurred())

		inReader, in := io.Pipe()
		var out bytes.Buffer
		streamClosed := make(chan struct{})

		By("Sending data on tunnel")
		go func() {
			err := tunnel.Stream(kubecli.StreamOptions{
				In:  inReader,
				Out: &out,
			})
			if err != nil {
				fmt.Fprintln(GinkgoWriter, err)
			}
			close(streamClosed)
		}()
		_, err = in.Write([]byte("test\n"))
		Expect(err).NotTo(HaveOccurred())
		<-streamClosed
		Expect(out.String()).To(ContainSubstring("OpenSSH"))
	})
})
