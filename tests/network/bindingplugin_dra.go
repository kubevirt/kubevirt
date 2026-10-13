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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/libnet"

	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet/vmnetserver"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("VM with DRA network binding plugin", decorators.NetCustomBindingPlugins, decorators.DRANetwork, Serial, func() {
	const (
		bindingName               = "dra-binding"
		networkName               = "dranet"
		resourceClaimTemplateName = "dra-net-claim-template"
		resourceClaimName         = "dra-net-claim"
		requestName               = "netdev"

		agentTimeout    = 5 * time.Minute
		pollingInterval = 5 * time.Second

		draPort = 5201
	)

	var (
		serverVM, clientVM *v1.VirtualMachine
		clientVMI          *v1.VirtualMachineInstance
		serverIP           string
		ns                 string
	)

	newDRAVMI := func(name string) *v1.VirtualMachineInstance {
		return libvmifact.NewAlpineWithTestTooling(
			libvmi.WithName(name),
			libvmi.WithInterface(libvmi.InterfaceWithBindingPlugin(networkName, v1.PluginBinding{Name: bindingName})),
			libvmi.WithNetwork(libvmi.DRANetwork(networkName, resourceClaimName, requestName)),
			libvmi.WithResourceClaim(v1.VirtualMachineInstanceResourceClaim{
				Name:                      resourceClaimName,
				ResourceClaimTemplateName: new(resourceClaimTemplateName),
			}),
		)
	}

	connectClientToServer := func() error {
		return console.RunCommand(clientVMI,
			fmt.Sprintf("echo test | nc %s %d -i 1 -w 1 1> /dev/null", serverIP, draPort),
			15*time.Second)
	}

	BeforeEach(func() {
		libnet.SkipWhenClusterNotSupportIpv4()
		config.EnableFeatureGate(featuregate.NetworkDevicesWithDRAGate)
	})

	BeforeEach(func() {
		sidecarImage := libregistry.GetUtilityImageFromRegistry("test-network-vhostuser-dra-binding")
		err := config.RegisterKubevirtConfigChange(
			config.WithNetBindingPluginIfNotPresent(bindingName, v1.InterfaceBindingPlugin{
				SidecarImage: sidecarImage,
			}),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		const deviceClassName = "vhostuser.net.kubevirt.io"
		resourceClaimTemplate := newResourceClaimTemplate(resourceClaimTemplateName, requestName, deviceClassName)
		_, err := kubevirt.Client().ResourceV1().ResourceClaimTemplates(testsuite.GetTestNamespace(nil)).Create(
			context.Background(),
			&resourceClaimTemplate,
			metav1.CreateOptions{},
		)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		const (
			draServerVMI = "dra-server"
			draClientVMI = "dra-client"
		)
		serverVMISpec := newDRAVMI(draServerVMI)
		clientVMISpec := newDRAVMI(draClientVMI)

		ns = testsuite.GetTestNamespace(nil)

		var err error
		serverVM = libvmi.NewVirtualMachine(serverVMISpec, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		serverVM, err = kubevirt.Client().VirtualMachine(ns).Create(context.Background(), serverVM, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		clientVM = libvmi.NewVirtualMachine(clientVMISpec, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		clientVM, err = kubevirt.Client().VirtualMachine(ns).Create(context.Background(), clientVM, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the guest agent to connect on both VMs")
		Eventually(matcher.ThisVM(serverVM)).WithTimeout(agentTimeout).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		Eventually(matcher.ThisVM(clientVM)).WithTimeout(agentTimeout).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		clientVMI, err = kubevirt.Client().VirtualMachineInstance(ns).Get(context.Background(), clientVM.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(console.LoginToAlpine(clientVMI)).To(Succeed())

		By("Verifying the DRA network interface is reported in the VMI status")
		var serverVMI *v1.VirtualMachineInstance
		Eventually(func(g Gomega) {
			serverVMI, err = kubevirt.Client().VirtualMachineInstance(ns).Get(context.Background(), serverVM.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(serverVMI.Status.Interfaces).To(HaveLen(1))
			g.Expect(serverVMI.Status.Interfaces[0].InfoSource).To(Equal(vmispec.InfoSourceDomainAndGA))
			g.Expect(serverVMI.Status.Interfaces[0].IPs).ToNot(BeEmpty())
		}, agentTimeout, pollingInterval).Should(Succeed())
		serverIP = serverVMI.Status.Interfaces[0].IPs[0]

		By("Starting a TCP server on the server VM")
		vmnetserver.StartTCPServer(serverVMI, draPort, console.LoginToAlpine)
	})

	It("two VMs on a secondary DRA network should communicate across client migration", decorators.RequiresThreeSchedulableNodes, func() {
		By("Connecting from the client VM to the server over the DRA network")
		Eventually(connectClientToServer, 30*time.Second, pollingInterval).Should(Succeed())

		By("Migrating the client VM")
		migration := libmigration.New(clientVMI.Name, clientVMI.Namespace)
		libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

		By("Connecting from the migrated client VM to the server over the DRA network")
		Eventually(connectClientToServer, 30*time.Second, pollingInterval).Should(Succeed())
	})
}))

func newResourceClaimTemplate(templateName, requestName, deviceClassName string) resourcev1.ResourceClaimTemplate {
	return resourcev1.ResourceClaimTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: templateName,
		},
		Spec: resourcev1.ResourceClaimTemplateSpec{
			Spec: resourcev1.ResourceClaimSpec{
				Devices: resourcev1.DeviceClaim{
					Requests: []resourcev1.DeviceRequest{{
						Name: requestName,
						Exactly: &resourcev1.ExactDeviceRequest{
							DeviceClassName: deviceClassName,
						},
					}},
				},
			},
		},
	}
}
