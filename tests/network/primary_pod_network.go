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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/util"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = SIGDescribe("Primary Pod Network", func() {
	var virtClient kubecli.KubevirtClient
	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).NotTo(HaveOccurred(), "Should successfully initialize an API client")
	})

	Describe("Status", func() {
		AssertReportedIP := func(vmi *v1.VirtualMachineInstance) {
			By("Getting pod of the VMI")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)

			By("Making sure IP/s reported on the VMI matches the ones on the pod")
			Expect(libnet.ValidateVMIandPodIPMatch(vmi, vmiPod)).To(Succeed(), "Should have matching IP/s between pod and vmi")
		}

		Context("VMI connected to the pod network using the default (implicit) binding", func() {
			var vmi *v1.VirtualMachineInstance

			BeforeEach(func() {
				vmi = setupVMI(virtClient, vmiWithDefaultBinding())
			})

			AfterEach(func() {
				cleanupVMI(virtClient, vmi)
			})

			It("should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
		})

		Context("VMI connected to the pod network using bridge binding", func() {
			When("Guest Agent exists", func() {
				var (
					vmi   *v1.VirtualMachineInstance
					vmiIP = func() string {
						var err error
						vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
						ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success retrieving VMI to get IP")
						return vmi.Status.Interfaces[0].IP
					}

					vmiIPs = func() []string {
						var err error
						vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
						ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success retrieving VMI to get IPs")
						return vmi.Status.Interfaces[0].IPs
					}
				)
				BeforeEach(func() {
					var err error

					vmi, err = newFedoraWithGuestAgentAndDefaultInterface(libvmi.InterfaceDeviceWithBridgeBinding(libvmi.DefaultInterfaceName))
					Expect(err).NotTo(HaveOccurred())

					vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
					Expect(err).NotTo(HaveOccurred())
					tests.WaitForSuccessfulVMIStart(vmi)
					tests.WaitAgentConnected(virtClient, vmi)
				})

				AfterEach(func() {
					cleanupVMI(virtClient, vmi)
				})

				It("should report PodIP/s IPv4 as its own on interface status", func() {
					vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
					Eventually(vmiIP).Should(Equal(vmiPod.Status.PodIP), "should contain VMI Status IP as Pod status ip")
					Eventually(vmiIPs).Should(ContainElement(vmiPod.Status.PodIP), "should contain IPv4 reported by guest agent")
				})

				It("should report VMIs static IPv6 at interface status", func() {
					Eventually(vmiIPs).Should(ContainElement(libnet.DefaultIPv6Address), "should contain IPv6 address set by cloud-init and reported by guest agent")
				})
			})
			When("no Guest Agent exists", func() {
				var vmi *v1.VirtualMachineInstance

				BeforeEach(func() {
					vmi = setupVMI(virtClient, vmiWithBridgeBinding())
				})

				AfterEach(func() {
					cleanupVMI(virtClient, vmi)
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

					tmpVmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(tmpVmi)
					Expect(err).NotTo(HaveOccurred())
					vmi = tests.WaitUntilVMIReady(tmpVmi, console.LoginToFedora)

					tests.WaitAgentConnected(virtClient, vmi)
				})

				AfterEach(func() {
					cleanupVMI(virtClient, vmi)
				})

				It("[test_id:4153]should report PodIP/s as its own on interface status", func() {
					vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
					Consistently(func() error {
						vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
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
					vmi = setupVMI(virtClient, vmiWithMasqueradeBinding())
				})

				AfterEach(func() {
					cleanupVMI(virtClient, vmi)
				})

				It("[Conformance] should report PodIP as its own on interface status", func() { AssertReportedIP(vmi) })
			})
		})
	})
})

func setupVMI(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
	By("Creating the VMI")
	var err error
	vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
	Expect(err).NotTo(HaveOccurred(), "VMI should be successfully created")

	By("Waiting until the VMI gets ready")
	vmi = tests.WaitUntilVMIReady(vmi, console.LoginToAlpine)

	return vmi
}

func cleanupVMI(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) {
	if vmi != nil {
		By("Deleting the VMI")
		Expect(virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Delete(vmi.GetName(), &metav1.DeleteOptions{})).To(Succeed())

		By("Waiting for the VMI to be gone")
		Eventually(func() error {
			_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.GetName(), &metav1.GetOptions{})
			return err
		}, 2*time.Minute, time.Second).Should(SatisfyAll(HaveOccurred(), WithTransform(errors.IsNotFound, BeTrue())), "The VMI should be gone within the given timeout")
	}
}

func vmiWithDefaultBinding() *v1.VirtualMachineInstance {
	vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
	vmi.Spec.Domain.Devices.Interfaces = nil
	vmi.Spec.Networks = nil
	return vmi
}

func vmiWithBridgeBinding() *v1.VirtualMachineInstance {
	vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	return vmi
}

func vmiWithMasqueradeBinding() *v1.VirtualMachineInstance {
	vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	return vmi
}

func newFedoraWithGuestAgentAndDefaultInterface(iface v1.Interface) (*v1.VirtualMachineInstance, error) {
	networkData, err := libnet.CreateDefaultCloudInitNetworkData()
	if err != nil {
		return nil, err
	}

	vmi := libvmi.NewTestToolingFedora(
		libvmi.WithInterface(iface),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithCloudInitNoCloudNetworkData(networkData, false),
	)
	return vmi, nil
}
