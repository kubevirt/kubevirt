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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = SIGDescribe("Slirp", decorators.Networking, func() {

	BeforeEach(libnet.SkipWhenClusterNotSupportIpv4)

	It("VMI with SLIRP interface, custom mac and port is configured correctly", func() {
		vmi := libvmifact.NewCirros(
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(
				libvmi.InterfaceDeviceWithSlirpBinding(v1.DefaultPodNetwork().Name, v1.Port{Name: "http", Port: 80}),
			),
		)
		vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de:ad:00:00:be:af"
		var err error
		vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		libwait.WaitForSuccessfulVMIStart(vmi,
			libwait.WithFailOnWarnings(false),
			libwait.WithTimeout(180),
		)

		tests.GenerateHelloWorldServer(vmi, 80, "tcp", console.LoginToCirros, true)

		By("have containerPort in the pod manifest")
		vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).NotTo(HaveOccurred())

		var container k8sv1.Container
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
			vmiPod,
			vmiPod.Spec.Containers[0].Name,
			[]string{"nc", "127.0.0.1", "80", "--recv-only"},
		)
		log.Log.Infof("%v", output)
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.Contains(output, "Hello World!")).To(BeTrue())

		By("reject connecting to localhost and port different than 80")
		output, err = exec.ExecuteCommandOnPod(
			vmiPod,
			vmiPod.Spec.Containers[1].Name,
			[]string{"curl", "127.0.0.1:9080"},
		)
		log.Log.Infof("%v", output)
		Expect(err).To(HaveOccurred())
	})
})
