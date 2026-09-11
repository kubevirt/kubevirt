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
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("VM with DRA network binding plugin", decorators.NetCustomBindingPlugins, decorators.DRANetwork, Serial, func() {
	const (
		bindingName               = "dra-binding"
		networkName               = "dranet"
		guestIfaceName            = "eth0"
		resourceClaimTemplateName = "dra-net-claim-template"
		resourceClaimName         = "dra-net-claim"
		requestName               = "netdev"

		// deviceClassName must match the DeviceClass published by the test DRA
		// network driver deployed in the cluster.
		deviceClassName = "netdevices.example.com"

		agentTimeout    = 5 * time.Minute
		pollingInterval = 5 * time.Second
	)

	newDRAVMI := func(cidr string, opts ...libvmi.Option) (*v1.VirtualMachineInstance, error) {
		networkData, err := cloudinit.NewNetworkData(
			cloudinit.WithEthernet(guestIfaceName, cloudinit.WithAddresses(cidr)),
		)
		if err != nil {
			return nil, err
		}

		base := []libvmi.Option{
			libvmi.WithInterface(libvmi.InterfaceWithBindingPlugin(networkName, v1.PluginBinding{Name: bindingName})),
			libvmi.WithNetwork(libvmi.DRANetwork(networkName, resourceClaimName, requestName)),
			libvmi.WithResourceClaim(v1.VirtualMachineInstanceResourceClaim{
				Name:                      resourceClaimName,
				ResourceClaimTemplateName: new(resourceClaimTemplateName),
			}),
			libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
		}

		return libvmifact.NewAlpineWithTestTooling(slices.Concat(base, opts)...), nil
	}

	BeforeEach(func() {
		config.EnableFeatureGate(featuregate.NetworkDevicesWithDRAGate)
	})

	BeforeEach(func() {
		// The test network binding plugin sidecar discovers the DRA-provisioned
		// device and attaches it to the domain XML (see VEP #183).
		sidecarImage := libregistry.GetUtilityImageFromRegistry("network-dra-binding")
		err := config.RegisterKubevirtConfigChange(
			config.WithNetBindingPluginIfNotPresent(bindingName, v1.InterfaceBindingPlugin{
				SidecarImage: sidecarImage,
			}),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		resourceClaimTemplate := newResourceClaimTemplate(resourceClaimTemplateName, requestName, deviceClassName)
		_, err := kubevirt.Client().ResourceV1().ResourceClaimTemplates(testsuite.GetTestNamespace(nil)).Create(
			context.Background(),
			&resourceClaimTemplate,
			metav1.CreateOptions{},
		)
		Expect(err).NotTo(HaveOccurred())
	})

	It("two VMs connected to a secondary DRA network should communicate", func() {
		const (
			serverIPAddr = "192.0.2.102"
			serverCIDR   = serverIPAddr + "/24"
			clientCIDR   = "192.0.2.101/24"
			serverLabel  = "dra-net-server"
		)

		serverVMI, err := newDRAVMI(serverCIDR, libvmi.WithLabel(serverLabel, ""))
		Expect(err).ToNot(HaveOccurred())

		// Schedule the client on the same node as the server so both consume the
		// node-local DRA network device and can reach each other.
		sameNodeAsServer := k8sv1.PodAffinityTerm{
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: serverLabel, Operator: metav1.LabelSelectorOpExists},
				},
			},
			TopologyKey: k8sv1.LabelHostname,
		}
		clientVMI, err := newDRAVMI(clientCIDR, libvmi.WithRequiredPodAffinity(sameNodeAsServer))
		Expect(err).ToNot(HaveOccurred())

		ns := testsuite.GetTestNamespace(nil)

		serverVM := libvmi.NewVirtualMachine(serverVMI, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		serverVM, err = kubevirt.Client().VirtualMachine(ns).Create(context.Background(), serverVM, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		clientVM := libvmi.NewVirtualMachine(clientVMI, libvmi.WithRunStrategy(v1.RunStrategyAlways))
		clientVM, err = kubevirt.Client().VirtualMachine(ns).Create(context.Background(), clientVM, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for the guest agent to connect on both VMs")
		Eventually(matcher.ThisVM(serverVM)).WithTimeout(agentTimeout).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		Eventually(matcher.ThisVM(clientVM)).WithTimeout(agentTimeout).WithPolling(pollingInterval).
			Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

		serverVMI, err = kubevirt.Client().VirtualMachineInstance(ns).Get(context.Background(), serverVM.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		clientVMI, err = kubevirt.Client().VirtualMachineInstance(ns).Get(context.Background(), clientVM.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(console.LoginToAlpine(serverVMI)).To(Succeed())
		Expect(console.LoginToAlpine(clientVMI)).To(Succeed())

		By("Verifying the DRA network interface is reported in the VMI status")
		Eventually(func(g Gomega) {
			serverVMI, err = kubevirt.Client().VirtualMachineInstance(ns).Get(context.Background(), serverVM.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(serverVMI.Status.Interfaces).To(HaveLen(1))
			g.Expect(serverVMI.Status.Interfaces[0].InfoSource).To(Equal(vmispec.InfoSourceDomainAndGA))
			g.Expect(serverVMI.Status.Interfaces[0].IPs).To(ContainElement(serverIPAddr))
		}, agentTimeout, pollingInterval).Should(Succeed())

		Expect(libnet.PingFromVMConsole(clientVMI, serverIPAddr)).To(Succeed())
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
