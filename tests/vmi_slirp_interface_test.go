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
	"fmt"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("[Serial]Slirp Networking", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	var genericVmi *v1.VirtualMachineInstance
	var deadbeafVmi *v1.VirtualMachineInstance
	var container k8sv1.Container
	setSlirpEnabled := func(enable bool) {
		tests.UpdateClusterConfigValueAndWait("permitSlirpInterface", fmt.Sprintf("%t", enable))
	}

	setDefaultNetworkInterface := func(iface string) {
		tests.UpdateClusterConfigValueAndWait("default-network-interface", fmt.Sprintf("%s", iface))
	}

	tests.BeforeAll(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		setSlirpEnabled(true)
		ports := []v1.Port{{Name: "http", Port: 80}}
		genericVmi = tests.NewRandomVMIWithSlirpInterfaceEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n", ports)
		deadbeafVmi = tests.NewRandomVMIWithSlirpInterfaceEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n", ports)
		deadbeafVmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de:ad:00:00:be:af"

		for _, vmi := range []*v1.VirtualMachineInstance{genericVmi, deadbeafVmi} {
			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStartIgnoreWarnings(vmi)
			tests.GenerateHelloWorldServer(vmi, 80, "tcp")
		}
	})
	AfterEach(func() {
		setSlirpEnabled(false)
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
			vmiPod.Spec.Containers[0].Name,
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
			vmiPod.Spec.Containers[0].Name,
			[]string{"nc", "127.0.0.1", "80", "--recv-only"},
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

	Context("vmi with default slirp interface", func() {
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
			vmi := v1.NewMinimalVMIWithNS(tests.NamespaceTestDefault, "testvmi"+rand.String(48))
			vmi.Spec.TerminationGracePeriodSeconds = &t
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")
			tests.AddEphemeralDisk(vmi, "disk0", "virtio", cd.ContainerDiskFor(cd.ContainerDiskCirros))

			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(HaveOccurred())
		})
	})
})
