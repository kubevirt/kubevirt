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
 * Copyright 2025 Red Hat, Inc.
 *
 */

package network

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/libnet"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/client-go/kubecli"

	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libwait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	. "kubevirt.io/kubevirt/tests/framework/matcher"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const bridgeInterfaceLinkDownName = "linux-bridge"

var _ = SIGDescribe("interface state up/down", func() {

	var virtClient kubecli.KubevirtClient
	var vm *v1.VirtualMachine
	bridgeInterfaceLinkDown := v1.Interface{
		Name: bridgeInterfaceLinkDownName,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			Bridge: &v1.InterfaceBridge{},
		},
		State: v1.InterfaceStateLinkDown,
	}
	bridgeNetwork := v1.Network{
		Name: bridgeInterfaceLinkDownName,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{
				NetworkName: bridgeInterfaceLinkDownName,
			},
		},
	}

	Context("VMI with one link up and another link down", func() {
		BeforeEach(func() {
			virtClient = kubevirt.Client()

			vmi := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(bridgeInterfaceLinkDown),
				libvmi.WithNetwork(&bridgeNetwork),
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(
					cloudinit.CreateNetworkDataWithStaticIPsByIface("eth1", "10.1.1.2/24"),
				)))

			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

			netAttachDef := libnet.NewBridgeNetAttachDef(
				bridgeInterfaceLinkDownName,
				"br02",
			)

			_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil), netAttachDef)
			Expect(err).NotTo(HaveOccurred())
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())

		})

		It("the guest should show one iface up and the other down", func() {
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
			const timeout = time.Second * 5
			Expect(console.RunCommand(vmi, fmt.Sprintf("ip -one link show eth0 | grep -E %s\n", "'state[[:space:]]+UP'"), timeout)).To(Succeed())
			Expect(console.RunCommand(vmi, fmt.Sprintf("ip -one link show eth1 | grep -E %s\n", "'state[[:space:]]+DOWN'"), timeout)).To(Succeed())
		})
	})
})
