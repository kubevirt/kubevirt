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

package compute

import (
	"context"
	"io"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const startupTimeout = 30

var _ = SIGDescribe("[rfe_id:127][posneg:negative][crit:medium][vendor:cnv-qe@redhat.com][level:component]Console", func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	expectConsoleOutput := func(vmi *v1.VirtualMachineInstance, expected string) {
		By("Checking that the console output equals to expected one")
		ExpectWithOffset(1, console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: expected},
		}, 120)).To(Succeed())
	}

	Describe("[rfe_id:127][posneg:negative][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		Context("with a serial console", func() {
			It("[test_id:1588]should return OS login", func() {
				vmi := libvmi.NewCirros()
				vmi = tests.RunVMIAndExpectLaunch(vmi, startupTimeout)
				expectConsoleOutput(
					vmi,
					"login as 'cirros' user",
				)
			})
			It("[test_id:1590]should be able to reconnect to console multiple times", func() {
				vmi := libvmi.NewAlpine()
				vmi = tests.RunVMIAndExpectLaunch(vmi, startupTimeout)

				for i := 0; i < 5; i++ {
					expectConsoleOutput(vmi, "login")
				}
			})

			It("[test_id:1591]should close console connection when new console connection is opened", func() {
				vmi := tests.RunVMIAndExpectLaunch(libvmi.NewAlpine(), startupTimeout)

				By("opening 1st console connection")
				stream, err := virtClient.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, &kubecli.SerialConsoleOptions{})
				Expect(err).ToNot(HaveOccurred())
				defer stream.AsConn().Close()

				firstConsoleErrChan := make(chan error, 1)
				outReader, outWriter := io.Pipe()
				inReader, _ := io.Pipe()
				go func() {
					io.Copy(io.Discard, outReader)
				}()
				go func() {
					firstConsoleErrChan <- stream.Stream(kvcorev1.StreamOptions{
						In:  inReader,
						Out: outWriter,
					})
				}()

				By("opening 2nd console connection")
				expectConsoleOutput(vmi, "login")

				By("expecting error on 1st console connection")
				Eventually(firstConsoleErrChan, 1*time.Minute, 1*time.Second).Should(Receive(MatchError(ContainSubstring("EOF"))))
			})

			It("[test_id:1592]should wait until the virtual machine is in running state and return a stream interface", func() {
				vmi := libvmi.NewAlpine()
				By("Creating a new VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("and connecting to it very quickly. Hopefully the VM is not yet up")
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, &kubecli.SerialConsoleOptions{ConnectionTimeout: 30 * time.Second})
				Expect(err).ToNot(HaveOccurred())
			})

			It("[test_id:1593]should not be connected if scheduled to non-existing host", func() {
				vmi := libvmi.NewAlpine(
					libvmi.WithNodeAffinityFor("nonexistent"),
				)

				By("Creating a new VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, &kubecli.SerialConsoleOptions{ConnectionTimeout: 30 * time.Second})
				Expect(err).To(MatchError("Timeout trying to connect to the virtual machine instance"))
			})
		})

		Context("without a serial console", func() {
			It("[test_id:4118]should run but not be connectable via the serial console", func() {
				vmi := libvmi.NewAlpine(libvmi.WithoutSerialConsole())
				vmi = tests.RunVMIAndExpectLaunch(vmi, startupTimeout)

				By("failing to connect to serial console")
				_, err := virtClient.VirtualMachineInstance(vmi.ObjectMeta.Namespace).SerialConsole(vmi.ObjectMeta.Name, &kubecli.SerialConsoleOptions{})
				Expect(err).To(MatchError("No serial consoles are present."), "serial console should not connect if there are no serial consoles present")
			})
		})
	})
})
