/*
 * This file is part of the kubevirt project
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

package network

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = SIGDescribe("Slirp Networking", decorators.Networking, func() {

	var err error
	var virtClient kubecli.KubevirtClient
	var currentConfiguration v1.KubeVirtConfiguration
	var container k8sv1.Container

	setSlirpEnabled := func(enable bool) {
		if currentConfiguration.NetworkConfiguration == nil {
			currentConfiguration.NetworkConfiguration = &v1.NetworkConfiguration{}
		}

		currentConfiguration.NetworkConfiguration.PermitSlirpInterface = pointer.BoolPtr(enable)
		kv := tests.UpdateKubeVirtConfigValueAndWait(currentConfiguration)
		currentConfiguration = kv.Spec.Configuration
	}

	setDefaultNetworkInterface := func(iface string) {
		if currentConfiguration.NetworkConfiguration == nil {
			currentConfiguration.NetworkConfiguration = &v1.NetworkConfiguration{}
		}

		currentConfiguration.NetworkConfiguration.NetworkInterface = iface
		kv := tests.UpdateKubeVirtConfigValueAndWait(currentConfiguration)
		currentConfiguration = kv.Spec.Configuration
	}

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		libnet.SkipWhenClusterNotSupportIpv4()

		kv := util.GetCurrentKv(virtClient)
		currentConfiguration = kv.Spec.Configuration
	})

	Context("slirp is not the default interface", func() {
		var (
			genericVmi  *v1.VirtualMachineInstance
			deadbeafVmi *v1.VirtualMachineInstance
			ports       []v1.Port
		)

		BeforeEach(func() {
			ports = []v1.Port{{Name: "http", Port: 80}}
			genericVmi = libvmi.NewCirros(
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(
					libvmi.InterfaceDeviceWithSlirpBinding(libvmi.DefaultInterfaceName, ports...)))
			deadbeafVmi = libvmi.NewCirros(
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(
					libvmi.InterfaceDeviceWithSlirpBinding(libvmi.DefaultInterfaceName, ports...)))
			deadbeafVmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de:ad:00:00:be:af"
		})

		DescribeTable("should be able to", func(vmiRef **v1.VirtualMachineInstance) {
			vmi := *vmiRef
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithFailOnWarnings(false),
				libwait.WithTimeout(180),
			)
			tests.GenerateHelloWorldServer(vmi, 80, "tcp", console.LoginToCirros, true)

			By("have containerPort in the pod manifest")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
			for _, containerSpec := range vmiPod.Spec.Containers {
				if containerSpec.Name == "compute" {
					container = containerSpec
					break
				}
			}
			Expect(container.Name).ToNot(Equal(""))
			Expect(container.Ports).ToNot(BeNil())
			Expect(container.Ports[0].Name).To(Equal("http"))
			Expect(container.Ports[0].Protocol).To(Equal(k8sv1.Protocol("TCP")))
			Expect(container.Ports[0].ContainerPort).To(Equal(int32(80)))

			By("start the virtual machine with slirp interface")
			output, err := exec.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[0].Name,
				[]string{"cat", "/proc/net/tcp"},
			)
			log.Log.Infof("%v", output)
			Expect(err).ToNot(HaveOccurred())
			// :0050 is port 80, 0A is listening
			Expect(strings.Contains(output, "0: 00000000:0050 00000000:0000 0A")).To(BeTrue())
			By("return \"Hello World!\" when connecting to localhost on port 80")
			output, err = exec.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[0].Name,
				[]string{"nc", "127.0.0.1", "80", "--recv-only"},
			)
			log.Log.Infof("%v", output)
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Contains(output, "Hello World!")).To(BeTrue())

			By("reject connecting to localhost and port different than 80")
			output, err = exec.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[1].Name,
				[]string{"curl", "127.0.0.1:9080"},
			)
			log.Log.Infof("%v", output)
			Expect(err).To(HaveOccurred())
		},
			Entry("VirtualMachineInstance with slirp interface", &genericVmi),
			Entry("VirtualMachineInstance with slirp interface with custom MAC address", &deadbeafVmi),
		)

		DescribeTable("[outside_connectivity]should be able to communicate with the outside world", func(vmiRef **v1.VirtualMachineInstance) {
			vmi := *vmiRef
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithFailOnWarnings(false),
				libwait.WithTimeout(180),
			)
			Expect(console.LoginToCirros(vmi)).To(Succeed())

			dns := "google.com"
			if flags.ConnectivityCheckDNS != "" {
				dns = flags.ConnectivityCheckDNS
			}

			Eventually(func() error {
				return console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: fmt.Sprintf("curl -o /dev/null -s -w \"%%{http_code}\\n\" -k https://%s\n", dns)},
					&expect.BExp{R: "301"},
				}, 60)
			}, 180*time.Second, time.Second).Should(Succeed(), "Failed to establish a successful connection to the outside network")
		},
			Entry("VirtualMachineInstance with slirp interface", &genericVmi),
			Entry("VirtualMachineInstance with slirp interface with custom MAC address", &deadbeafVmi),
		)
	})

	Context("[Serial]slirp is the default interface", Serial, func() {
		BeforeEach(func() {
			setSlirpEnabled(false)
			setDefaultNetworkInterface("slirp")
		})
		AfterEach(func() {
			setSlirpEnabled(true)
			setDefaultNetworkInterface("bridge")
		})
		It("should reject VMIs with default interface slirp when it's not permitted", func() {
			var t int64 = 0
			vmi := tests.NewRandomVMI()
			vmi.Spec.TerminationGracePeriodSeconds = &t
			// Reset memory, devices and networks
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("128Mi")
			vmi.Spec.Domain.Devices = v1.Devices{}
			vmi.Spec.Networks = nil
			tests.AddEphemeralDisk(vmi, "disk0", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
			Expect(err).To(HaveOccurred())
		})
	})
})
