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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package network

import (
	"context"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = SIGDescribe("Primary Pod Network", func() {
	var virtClient kubecli.KubevirtClient
	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Describe("Status", func() {
		AssertReportedIP := func(vmi *v1.VirtualMachineInstance) {
			By("Getting pod of the VMI")
			vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())

			By("Making sure IP/s reported on the VMI matches the ones on the pod")
			Expect(libnet.ValidateVMIandPodIPMatch(vmi, vmiPod)).To(Succeed(), "Should have matching IP/s between pod and vmi")
		}

		Context("VMI connected to the pod network using the default (implicit) binding", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = setupVMI(virtClient, libvmifact.NewAlpine())
			})

			It("should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
		})

		Context("VMI connected to the pod network using bridge binding", func() {
			When("Guest Agent exists", func() {
				var (
					vmi   *v1.VirtualMachineInstance
					vmiIP = func() string {
						var err error
						vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
						ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success retrieving VMI to get IP")
						return vmi.Status.Interfaces[0].IP
					}

					vmiIPs = func() []string {
						var err error
						vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
						ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success retrieving VMI to get IPs")
						return vmi.Status.Interfaces[0].IPs
					}
				)
				BeforeEach(func() {
					libnet.SkipWhenClusterNotSupportIpv4()
					var err error

					vmi, err = newFedoraWithGuestAgentAndDefaultInterface(libvmi.InterfaceDeviceWithBridgeBinding(v1.DefaultPodNetwork().Name))
					Expect(err).NotTo(HaveOccurred())

					vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())

					libwait.WaitForSuccessfulVMIStart(vmi)
					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				})

				It("should report PodIP/s on interface status", func() {
					vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
					Expect(err).NotTo(HaveOccurred())

					Eventually(vmiIP, 2*time.Minute, 5*time.Second).Should(Equal(vmiPod.Status.PodIP), "should contain VMI Status IP as Pod status ip")
					var podIPs []string
					for _, ip := range vmiPod.Status.PodIPs {
						podIPs = append(podIPs, ip.IP)
					}
					Eventually(vmiIPs, 2*time.Minute, 5*time.Second).Should(Equal(podIPs), "should contain VMI Status IP as Pod status IPs")
				})
			})
			When("no Guest Agent exists", func() {
				var vmi *v1.VirtualMachineInstance

				BeforeEach(func() {
					vmi = setupVMI(
						virtClient,
						libvmifact.NewAlpine(
							libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
							libvmi.WithNetwork(v1.DefaultPodNetwork()),
						),
					)
				})

				It("should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
			})
		})

		Context("VMI connected to the pod network using masquerade binding", func() {
			When("Guest Agent exists", func() {
				var vmi *v1.VirtualMachineInstance

				BeforeEach(func() {
					tmpVmi, err := newFedoraWithGuestAgentAndDefaultInterface(libvmi.InterfaceDeviceWithMasqueradeBinding())
					Expect(err).NotTo(HaveOccurred())

					tmpVmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), tmpVmi, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
					vmi = libwait.WaitUntilVMIReady(tmpVmi, console.LoginToFedora)

					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
				})

				It("[test_id:4153]should report PodIP/s as its own on interface status", func() {
					vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
					Expect(err).NotTo(HaveOccurred())

					Consistently(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
						if err != nil {
							return err
						}
						return libnet.ValidateVMIandPodIPMatch(vmi, vmiPod)
					}, 5*time.Second, time.Second).Should(Succeed())
				})

			})

			When("no Guest Agent exists", func() {
				var vmi *v1.VirtualMachineInstance

				BeforeEach(func() {
					vmi = setupVMI(
						virtClient,
						libvmifact.NewAlpine(
							libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
							libvmi.WithNetwork(v1.DefaultPodNetwork()),
						),
					)
				})

				It("should report PodIP as its own on interface status", decorators.Conformance, func() { AssertReportedIP(vmi) })
			})
		})
	})
})

func setupVMI(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
	By("Creating the VMI")
	var err error
	vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred(), "VMI should be successfully created")

	By("Waiting until the VMI gets ready")
	vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

	return vmi
}

func newFedoraWithGuestAgentAndDefaultInterface(iface v1.Interface) (*v1.VirtualMachineInstance, error) {
	networkData := cloudinit.CreateDefaultCloudInitNetworkData()

	vmi := libvmifact.NewFedora(
		libvmi.WithInterface(iface),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithCloudInitNoCloudNetworkData(networkData),
	)
	return vmi, nil
}
