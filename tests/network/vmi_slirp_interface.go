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
	"fmt"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = SIGDescribe("[Serial]Passt Networking", func() {

	var err error
	var virtClient kubecli.KubevirtClient
	var currentConfiguration v1.KubeVirtConfiguration

	var genericVmi *v1.VirtualMachineInstance
	//	var deadbeafVmi *v1.VirtualMachineInstance
	//var container k8sv1.Container
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

	tests.BeforeAll(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		kv := util.GetCurrentKv(virtClient)
		currentConfiguration = kv.Spec.Configuration

		setSlirpEnabled(true)
		//ports := []v1.Port{{Name: "http", Port: 80}}
		slirpIface := v1.Interface{
			Name: "default",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{
				Slirp: &v1.InterfaceSlirp{},
			},
		}
		genericVmi = libvmi.NewFedora(
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(slirpIface),
		)
		//genericVmi = tests.NewRandomVMIWithSlirpInterfaceEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n", ports)
		genericVmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("2048M")

		//	deadbeafVmi = tests.NewRandomVMIWithSlirpInterfaceEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n", ports)
		//	deadbeafVmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de:ad:00:00:be:af"

		genericVmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(genericVmi)
		Expect(err).ToNot(HaveOccurred())

		//genericVmi = util.WaitUntilVMIReady(genericVmi, libnet.WithIPv6(console.LoginToFedora))
		warningsIgnoreList := []string{"unknown error encountered sending command SyncVMI: rpc error: code = DeadlineExceeded desc = context deadline exceeded",
			"unknown error encountered sending command SyncVMI: rpc error: code = Unavailable desc = transport is closing"}
		genericVmi = tests.WaitUntilVMIReadyIgnoreSelectedWarnings(genericVmi, libnet.WithIPv6(console.LoginToFedora), warningsIgnoreList)

		generateHelloWorldServer(genericVmi, 80, "tcp")
	})
	AfterEach(func() {
		setSlirpEnabled(false)
	})

	table.DescribeTable("should be able to", func(vmiRef **v1.VirtualMachineInstance) {
		By("have containerPort in the pod manifest")
		vmi := *vmiRef
		vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
		//for _, containerSpec := range vmiPod.Spec.Containers {
		//	if containerSpec.Name == "compute" {
		//		container = containerSpec
		//		break
		//	}
		//}
		//Expect(container.Name).ToNot(Equal(""))
		//Expect(container.Ports).ToNot(Equal(nil))
		//Expect(container.Ports[0].Name).To(Equal("http"))
		//Expect(container.Ports[0].Protocol).To(Equal(k8sv1.Protocol("TCP")))
		//Expect(container.Ports[0].ContainerPort).To(Equal(int32(80)))

		By("start the virtual machine with slirp interface")

		// Commented out since passt is bound to all free ports

		//output, err := tests.ExecuteCommandOnPod(
		//	virtClient,
		//	vmiPod,
		//	vmiPod.Spec.Containers[0].Name,
		//	[]string{"cat", "/proc/net/tcp"},
		//)
		//log.Log.Infof("%v", output)
		//Expect(err).ToNot(HaveOccurred())
		//// :0050 is port 80, 0A is listening
		//Expect(strings.Contains(output, "0: 00000000:0050 00000000:0000 0A")).To(BeTrue())

		By("return \"Hello World!\" when connecting to localhost on port 80")
		output, err := tests.ExecuteCommandOnPod(
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
	},
		table.Entry("VirtualMachineInstance with slirp interface", &genericVmi),
		//table.Entry("VirtualMachineInstance with slirp interface with custom MAC address", &deadbeafVmi),
	)

	table.DescribeTable("[outside_connectivity]should be able to communicate with the outside world", func(vmiRef **v1.VirtualMachineInstance) {
		vmi := *vmiRef
		dns := "google.com"
		if flags.ConnectivityCheckDNS != "" {
			dns = flags.ConnectivityCheckDNS
		}

		Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: fmt.Sprintf("curl -o /dev/null -s -w \"%%{http_code}\\n\" -k https://%s\n", dns)},
			&expect.BExp{R: "301"},
		}, 180)).To(Succeed())
	},
		table.Entry("VirtualMachineInstance with slirp interface", &genericVmi),
		// passt poc doesn't support custom MAC address
		//table.Entry("VirtualMachineInstance with slirp interface with custom MAC address", &deadbeafVmi),
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
			vmi := v1.NewMinimalVMIWithNS(util.NamespaceTestDefault, libvmi.RandName(libvmi.DefaultVmiName))
			vmi.Spec.TerminationGracePeriodSeconds = &t
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("128Mi")
			tests.AddEphemeralDisk(vmi, "disk0", "virtio", cd.ContainerDiskFor(cd.ContainerDiskCirros))

			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).To(HaveOccurred())
		})
	})
})

func generateHelloWorldServer(vmi *v1.VirtualMachineInstance, testPort int, protocol string) {
	//Expect(console.RunCommand(vmi, "sudo yum -y install screen", 240*time.Second)).To(Succeed())
	Expect(console.RunCommand(vmi, "sudo yum -y install nc", 360*time.Second)).To(Succeed())

	//serverCommand := fmt.Sprintf("screen -d -m sudo nc -klp %d -e echo -e 'Hello World!'\n", testPort)
	serverCommand := fmt.Sprintf("echo -e 'Hello World!' | sudo nc -klp %d &\n", testPort)

	//if protocol == "udp" {
	//	// nc has to be in a while loop in case of UDP, since it exists after one message
	//	serverCommand = fmt.Sprintf("screen -d -m sh -c \"while true; do nc -uklp %d -e echo -e 'Hello UDP World!';done\"\n", testPort)
	//}
	Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: serverCommand},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: console.RetValue("0")},
	}, 60)).To(Succeed())
}
