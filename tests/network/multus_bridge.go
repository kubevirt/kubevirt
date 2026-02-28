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
 * Copyright The KubeVirt Authors.
 *
 */

package network

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	bridgeNADName            = "nad-1"
	bridgeName               = "br1"
	bridgeMasterIfaceName    = "eth1"
	minRequiredScheduleNodes = 2
)

var _ = Describe(SIG("Bridge", decorators.RequiresTwoSchedulableNodes, func() {

	BeforeEach(func() {
		_, err := libnet.SetupBridgeAsMaster(bridgeName, bridgeMasterIfaceName)
		Expect(err).NotTo(HaveOccurred())

	})

	BeforeEach(func() {
		netAttachDef1 := libnet.NewBridgeNetAttachDef(bridgeNADName, bridgeName)
		_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil), netAttachDef1)
		Expect(err).NotTo(HaveOccurred())
	})

	It("connectivity over bridge", func() {
		const (
			ip1 = "10.100.0.10"
			ip2 = "10.100.0.20"
		)
		nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
		Expect(len(nodes.Items)).To(BeNumerically(">=", minRequiredScheduleNodes), "at least 2 schedulable nodes are required")

		vmi1, err := newVMIWithIP(ip1+"/24", nodes.Items[0].Name)
		Expect(err).NotTo(HaveOccurred())

		vmi2, err := newVMIWithIP(ip2+"/24", nodes.Items[1].Name)
		Expect(err).NotTo(HaveOccurred())

		vmi1, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), vmi1, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmi2, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(nil)).
			Create(context.Background(), vmi2, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(matcher.ThisVMI(vmi1)).WithTimeout(2 * time.Minute).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		Eventually(matcher.ThisVMI(vmi2)).WithTimeout(2 * time.Minute).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		Expect(console.LoginToAlpine(vmi1)).To(Succeed())
		Expect(libnet.PingFromVMConsole(vmi1, ip2)).To(Succeed())
		Expect(console.LoginToAlpine(vmi2)).To(Succeed())
		Expect(libnet.PingFromVMConsole(vmi2, ip1)).To(Succeed())

	})

}))

func newVMIWithIP(ip, nodeName string) (*v1.VirtualMachineInstance, error) {
	const ifaceName = "secondary"
	networkData1, err := cloudinit.NewNetworkData(
		cloudinit.WithEthernet("eth0",
			cloudinit.WithAddresses(ip),
		),
	)
	if err != nil {
		return nil, err
	}
	vmi := libvmifact.NewAlpineWithTestTooling(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(ifaceName)),
		libvmi.WithNetwork(libvmi.MultusNetwork(ifaceName, bridgeNADName)),
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData1)),
		libvmi.WithNodeAffinityFor(nodeName),
	)
	return vmi, nil
}
