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
 * Copyright The KubeVirt Authors
 *
 */

package tests_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	virt_api "kubevirt.io/kubevirt/pkg/virt-api"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe("[sig-compute]virt-handler", decorators.SigCompute, func() {
	// Regression
	It(" multiple HTTP calls should re-use connections and not grow the number of open connections", Serial, func() {

		getHandlerConnectionCount := func(nodeName string) int {
			cmd := []string{"bash", "-c", fmt.Sprintf("ss -ntlap | grep %d | wc -l", virt_api.DefaultConsoleServerPort)}
			stdout, err := libnode.ExecuteCommandInVirtHandlerPod(nodeName, cmd)
			Expect(err).ToNot(HaveOccurred())

			stdout = strings.TrimSpace(stdout)
			stdout = strings.ReplaceAll(stdout, "\n", "")

			handlerCons, err := strconv.Atoi(stdout)
			Expect(err).ToNot(HaveOccurred())

			return handlerCons
		}

		getClientCalls := func(vmi *v1.VirtualMachineInstance) []func() {
			vmiInterface := kubevirt.Client().VirtualMachineInstance(vmi.Namespace)
			expectNoErr := func(err error) {
				ExpectWithOffset(2, err).ToNot(HaveOccurred())
			}

			return []func(){
				func() {
					_, err := vmiInterface.GuestOsInfo(context.Background(), vmi.Name)
					expectNoErr(err)
				},
				func() {
					_, err := vmiInterface.FilesystemList(context.Background(), vmi.Name)
					expectNoErr(err)
				},
				func() {
					_, err := vmiInterface.UserList(context.Background(), vmi.Name)
					expectNoErr(err)
				},
				func() {
					_, err := vmiInterface.VNC(vmi.Name)
					expectNoErr(err)
				},
				func() {
					_, err := vmiInterface.SerialConsole(vmi.Name, &kvcorev1.SerialConsoleOptions{ConnectionTimeout: 30 * time.Second})
					expectNoErr(err)
				},
			}
		}

		By("Running the VMI")
		vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithAutoattachGraphicsDevice(true))
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsTiny)

		By("VMI has the guest agent connected condition")
		Eventually(matcher.ThisVMI(vmi), 240*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected), "should have agent connected condition")
		origHandlerCons := getHandlerConnectionCount(vmi.Status.NodeName)

		By("Making multiple requests")
		const numberOfRequests = 20
		clientCalls := getClientCalls(vmi)
		for i := 0; i < numberOfRequests; i++ {
			for _, clientCallFunc := range clientCalls {
				clientCallFunc()
			}
			time.Sleep(200 * time.Millisecond)
		}

		By("Expecting the number of connections to not grow")
		Expect(getHandlerConnectionCount(vmi.Status.NodeName)-origHandlerCons).To(BeNumerically("<=", len(clientCalls)), "number of connections is not expected to grow")
	})
})
