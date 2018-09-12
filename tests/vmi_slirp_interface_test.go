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

package tests_test

import (
	"flag"
	"strings"
	"time"

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Slirp", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var genericVmi *v1.VirtualMachineInstance
	var deadbeafVmi *v1.VirtualMachineInstance
	var container k8sv1.Container

	tests.BeforeAll(func() {
		ports := []v1.Port{{Name: "http", Port: 80}}
		genericVmi = tests.NewRandomVMIWithSlirpInterfaceEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), "#!/bin/bash\necho 'hello'\n", ports)
		deadbeafVmi = tests.NewRandomVMIWithSlirpInterfaceEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), "#!/bin/bash\necho 'hello'\n", ports)
		deadbeafVmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de:ad:00:00:be:af"

		for _, vmi := range []*v1.VirtualMachineInstance{genericVmi, deadbeafVmi} {
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
			generateHelloWorldServer(vmi, virtClient, 80, "tcp")
		}
	})

	table.DescribeTable("should be able to", func(vmiRef **v1.VirtualMachineInstance) {
		By("have containerPort in the pod manifest")
		vmi := *vmiRef
		vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
		for _, containerSpec := range vmiPod.Spec.Containers {
			if containerSpec.Name == "compute" {
				container = containerSpec
				break
			}
		}
		Expect(container.Name).ToNot(Equal(""))
		Expect(container.Ports).ToNot(Equal(nil))
		Expect(container.Ports[0].Name).To(Equal("http"))
		Expect(container.Ports[0].Protocol).To(Equal(k8sv1.Protocol("TCP")))
		Expect(container.Ports[0].ContainerPort).To(Equal(int32(80)))

		By("start the virtual machine with slirp interface")
		output, err := tests.ExecuteCommandOnPod(
			virtClient,
			vmiPod,
			vmiPod.Spec.Containers[1].Name,
			[]string{"cat", "/proc/net/tcp"},
		)
		log.Log.Infof("%v", output)
		Expect(err).ToNot(HaveOccurred())
		// :0050 is port 80, 0A is listening
		Expect(strings.Contains(output, "0: 00000000:0050 00000000:0000 0A")).To(BeTrue())

		By("return \"Hello World!\" when connecting to localhost on port 80")
		output, err = tests.ExecuteCommandOnPod(
			virtClient,
			vmiPod,
			vmiPod.Spec.Containers[1].Name,
			[]string{"curl", "-s", "--retry", "30", "--retry-delay", "30", "127.0.0.1"},
		)
		log.Log.Infof("%v", output)
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.Contains(output, "Hello World!")).To(BeTrue())

		By("reject connecting to localhost and port different than 80")
		output, err = tests.ExecuteCommandOnPod(
			virtClient,
			vmiPod,
			vmiPod.Spec.Containers[1].Name,
			[]string{"curl", "127.0.0.1:9080"},
		)
		log.Log.Infof("%v", output)
		Expect(err).To(HaveOccurred())

		By("communicate with the outside world")
		expecter, _, err := tests.NewConsoleExpecter(virtClient, vmi, 10*time.Second)
		defer expecter.Close()
		Expect(err).ToNot(HaveOccurred())

		out, err := expecter.ExpectBatch([]expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: "\\$ "},
			&expect.BSnd{S: "curl -o /dev/null -s -w \"%{http_code}\\n\" -k https://google.com\n"},
			&expect.BExp{R: "301"},
		}, 180*time.Second)
		log.Log.Infof("%v", out)
		Expect(err).ToNot(HaveOccurred())
	},
		table.Entry("VirtualMachineInstance with slirp interface", &genericVmi),
		table.Entry("VirtualMachineInstance with slirp interface with custom MAC address", &deadbeafVmi),
	)
})
