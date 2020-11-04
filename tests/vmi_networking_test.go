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
	"net"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	netutils "k8s.io/utils/net"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = Describe("[Serial][rfe_id:694][crit:medium][vendor:cnv-qe@redhat.com][level:component]Networking", func() {

	var err error
	var virtClient kubecli.KubevirtClient
	var currentConfiguration v1.KubeVirtConfiguration

	var inboundVMI *v1.VirtualMachineInstance
	var inboundVMIWithPodNetworkSet *v1.VirtualMachineInstance
	var inboundVMIWithCustomMacAddress *v1.VirtualMachineInstance
	var outboundVMI *v1.VirtualMachineInstance

	const testPort = 1500

	tests.BeforeAll(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		kv := tests.GetCurrentKv(virtClient)
		currentConfiguration = kv.Spec.Configuration
	})

	checkMacAddress := func(vmi *v1.VirtualMachineInstance, expectedMacAddress string) {
		err := console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "cat /sys/class/net/eth0/address\n"},
			&expect.BExp{R: expectedMacAddress},
		}, 15)
		Expect(err).ToNot(HaveOccurred())
	}

	checkNetworkVendor := func(vmi *v1.VirtualMachineInstance, expectedVendor string) {
		err := console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: "cat /sys/class/net/eth0/device/vendor\n"},
			&expect.BExp{R: expectedVendor},
		}, 15)
		Expect(err).ToNot(HaveOccurred())
	}

	checkLearningState := func(vmi *v1.VirtualMachineInstance, expectedValue string) {
		output := tests.RunCommandOnVmiPod(vmi, []string{"cat", "/sys/class/net/eth0/brport/learning"})
		ExpectWithOffset(1, strings.TrimSpace(output)).To(Equal(expectedValue))
	}

	setBridgeEnabled := func(enable bool) {
		if currentConfiguration.NetworkConfiguration == nil {
			currentConfiguration.NetworkConfiguration = &v1.NetworkConfiguration{}
		}

		currentConfiguration.NetworkConfiguration.PermitBridgeInterfaceOnPodNetwork = pointer.BoolPtr(enable)
		kv := tests.UpdateKubeVirtConfigValueAndWait(currentConfiguration)
		currentConfiguration = kv.Spec.Configuration
	}

	Describe("Multiple virtual machines connectivity using bridge binding interface", func() {
		tests.BeforeAll(func() {
			tests.BeforeTestCleanup()

			// Prepare inbound and outbound VMI definitions

			// inboundVMI expects implicitly to be added to the pod network
			inboundVMI = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			// Remove the masquerade interface to use the default bridge one
			inboundVMI.Spec.Domain.Devices.Interfaces = nil
			inboundVMI.Spec.Networks = nil

			// outboundVMI is used to connect to other vms
			outboundVMI = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			// Remove the masquerade interface to use the default bridge one
			outboundVMI.Spec.Domain.Devices.Interfaces = nil
			outboundVMI.Spec.Networks = nil

			// inboudnVMIWithPodNetworkSet adds itself in an explicit fashion to the pod network
			inboundVMIWithPodNetworkSet = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			// Remove the masquerade interface to use the default bridge one
			inboundVMIWithPodNetworkSet.Spec.Domain.Devices.Interfaces = nil
			inboundVMIWithPodNetworkSet.Spec.Networks = nil
			v1.SetDefaults_NetworkInterface(inboundVMIWithPodNetworkSet)
			Expect(inboundVMIWithPodNetworkSet.Spec.Domain.Devices.Interfaces).NotTo(BeEmpty())

			// inboundVMIWithCustomMacAddress specifies a custom MAC address
			inboundVMIWithCustomMacAddress = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			// Remove the masquerade interface to use the default bridge one
			inboundVMIWithCustomMacAddress.Spec.Domain.Devices.Interfaces = nil
			inboundVMIWithCustomMacAddress.Spec.Networks = nil
			v1.SetDefaults_NetworkInterface(inboundVMIWithCustomMacAddress)
			Expect(inboundVMIWithCustomMacAddress.Spec.Domain.Devices.Interfaces).NotTo(BeEmpty())
			inboundVMIWithCustomMacAddress.Spec.Domain.Devices.Interfaces[0].MacAddress = "de:ad:00:00:be:af"

			// Create VMIs
			for _, networkVMI := range []*v1.VirtualMachineInstance{inboundVMI, outboundVMI, inboundVMIWithPodNetworkSet, inboundVMIWithCustomMacAddress} {
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(networkVMI)
				Expect(err).ToNot(HaveOccurred())
			}

			// Wait for VMIs to become ready
			inboundVMI = tests.WaitUntilVMIReady(inboundVMI, tests.LoggedInCirrosExpecter)
			outboundVMI = tests.WaitUntilVMIReady(outboundVMI, tests.LoggedInCirrosExpecter)
			inboundVMIWithPodNetworkSet = tests.WaitUntilVMIReady(inboundVMIWithPodNetworkSet, tests.LoggedInCirrosExpecter)
			inboundVMIWithCustomMacAddress = tests.WaitUntilVMIReady(inboundVMIWithCustomMacAddress, tests.LoggedInCirrosExpecter)

			tests.StartTCPServer(inboundVMI, testPort)
		})

		table.DescribeTable("should be able to reach", func(destination string) {
			var cmdCheck, addrShow, addr string

			if destination == "InboundVMIWithCustomMacAddress" {
				tests.SkipIfOpenShift("Custom MAC addresses on pod networks are not supported")
			}

			switch destination {
			case "Internet":
				addr = "kubevirt.io"
			case "InboundVMI":
				addr = inboundVMI.Status.Interfaces[0].IP
			case "InboundVMIWithPodNetworkSet":
				addr = inboundVMIWithPodNetworkSet.Status.Interfaces[0].IP
			case "InboundVMIWithCustomMacAddress":
				addr = inboundVMIWithCustomMacAddress.Status.Interfaces[0].IP
			}

			payloadSize := 0
			ipHeaderSize := 28 // IPv4 specific

			By("checking k6t-eth0 MTU inside the pod")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(outboundVMI, tests.NamespaceTestDefault)
			output, err := tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				"compute",
				[]string{"cat", "/sys/class/net/k6t-eth0/mtu"},
			)
			log.Log.Infof("k6t-eth0 mtu is %v", output)
			Expect(err).ToNot(HaveOccurred())

			output = strings.TrimSuffix(output, "\n")
			mtu, err := strconv.Atoi(output)
			Expect(err).ToNot(HaveOccurred())

			Expect(mtu > 1000).To(BeTrue())

			payloadSize = mtu - ipHeaderSize
			expectedMtuString := fmt.Sprintf("mtu %d", mtu)

			By("checking eth0 MTU inside the VirtualMachineInstance")
			expecter, err := tests.LoggedInCirrosExpecter(outboundVMI)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			addrShow = "ip address show eth0\n"
			resp, err := console.ExpectBatchWithValidatedSend(expecter, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: addrShow},
				&expect.BExp{R: fmt.Sprintf(".*%s.*\n", expectedMtuString)},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
			}, 180*time.Second)
			log.Log.Infof("%v", resp)
			Expect(err).ToNot(HaveOccurred())

			By("checking the VirtualMachineInstance can send MTU sized frames to another VirtualMachineInstance")
			// NOTE: VirtualMachineInstance is not directly accessible from inside the pod because
			// we transferred its IP address under DHCP server control, so the
			// only thing we can validate is connectivity between VMIs
			//
			// NOTE: cirros ping doesn't support -M do that could be used to
			// validate end-to-end connectivity with Don't Fragment flag set
			cmdCheck = fmt.Sprintf("ping %s -c 1 -w 5 -s %d\n", addr, payloadSize)
			err = console.SafeExpectBatch(outboundVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: cmdCheck},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
			}, 180)
			Expect(err).ToNot(HaveOccurred())

			By("checking the VirtualMachineInstance can fetch via HTTP")
			err = console.SafeExpectBatch(outboundVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "curl --silent http://kubevirt.io > /dev/null\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
		},
			table.Entry("[test_id:1539]the Inbound VirtualMachineInstance", "InboundVMI"),
			table.Entry("[test_id:1540]the Inbound VirtualMachineInstance with pod network connectivity explicitly set", "InboundVMIWithPodNetworkSet"),
			table.Entry("[test_id:1541]the Inbound VirtualMachineInstance with custom MAC address", "InboundVMIWithCustomMacAddress"),
			table.Entry("[test_id:1542]the internet", "Internet"),
		)

		table.DescribeTable("should be reachable via the propagated IP from a Pod", func(op v12.NodeSelectorOperator, hostNetwork bool) {

			ip := inboundVMI.Status.Interfaces[0].IP

			//TODO if node count 1, skip the nv12.NodeSelectorOpOut
			nodes, err := virtClient.CoreV1().Nodes().List(v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(nodes.Items).ToNot(BeEmpty())
			if len(nodes.Items) == 1 && op == v12.NodeSelectorOpNotIn {
				Skip("Skip network test that requires multiple nodes when only one node is present.")
			}

			job := tests.NewHelloWorldJob(ip, strconv.Itoa(testPort))
			job.Spec.Template.Spec.Affinity = &v12.Affinity{
				NodeAffinity: &v12.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &v12.NodeSelector{
						NodeSelectorTerms: []v12.NodeSelectorTerm{
							{
								MatchExpressions: []v12.NodeSelectorRequirement{
									{Key: "kubernetes.io/hostname", Operator: op, Values: []string{inboundVMI.Status.NodeName}},
								},
							},
						},
					},
				},
			}
			job.Spec.Template.Spec.HostNetwork = hostNetwork

			job, err = virtClient.BatchV1().Jobs(inboundVMI.ObjectMeta.Namespace).Create(job)
			Expect(err).ToNot(HaveOccurred())
			Expect(tests.WaitForJobToSucceed(job, 90*time.Second)).To(Succeed())
		},
			table.Entry("[test_id:1543]on the same node from Pod", v12.NodeSelectorOpIn, false),
			table.Entry("[test_id:1544]on a different node from Pod", v12.NodeSelectorOpNotIn, false),
			table.Entry("[test_id:1545]on the same node from Node", v12.NodeSelectorOpIn, true),
			table.Entry("[test_id:1546]on a different node from Node", v12.NodeSelectorOpNotIn, true),
		)

		Context("VirtualMachineInstance with default interface model", func() {
			// Unless an explicit interface model is specified, the default interface model is virtio.
			It("[test_id:1550]should expose the right device type to the guest", func() {
				By("checking the device vendor in /sys/class")

				// Taken from https://wiki.osdev.org/Virtio#Technical_Details
				virtio_vid := "0x1af4"

				for _, networkVMI := range []*v1.VirtualMachineInstance{inboundVMI, outboundVMI} {
					// as defined in https://vendev.org/pci/ven_1af4/
					checkNetworkVendor(networkVMI, virtio_vid)
				}
			})

			It("[test_id:1551]should reject the creation of virtual machine with unsupported interface model", func() {
				// Create a virtual machine with an unsupported interface model
				customIfVMI := NewRandomVMIWithInvalidNetworkInterface()
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(customIfVMI)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("VirtualMachineInstance with custom interface model", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		It("[test_id:1770]should expose the right device type to the guest", func() {
			By("checking the device vendor in /sys/class")
			// Create a machine with e1000 interface model
			e1000VMI := tests.NewRandomVMIWithe1000NetworkInterface()
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(e1000VMI)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitUntilVMIReady(e1000VMI, tests.LoggedInAlpineExpecter)
			// as defined in https://vendev.org/pci/ven_8086/
			checkNetworkVendor(e1000VMI, "0x8086")
		})
	})

	Context("VirtualMachineInstance with custom MAC address", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		It("[test_id:1771]should configure custom MAC address", func() {
			By("checking eth0 MAC address")
			deadbeafVMI := tests.NewRandomVMIWithCustomMacAddress()
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(deadbeafVMI)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitUntilVMIReady(deadbeafVMI, tests.LoggedInAlpineExpecter)
			checkMacAddress(deadbeafVMI, deadbeafVMI.Spec.Domain.Devices.Interfaces[0].MacAddress)
		})
	})

	Context("VirtualMachineInstance with custom MAC address in non-conventional format", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		It("[test_id:1772]should configure custom MAC address", func() {
			By("checking eth0 MAC address")
			beafdeadVMI := tests.NewRandomVMIWithCustomMacAddress()
			beafdeadVMI.Spec.Domain.Devices.Interfaces[0].MacAddress = "BE-AF-00-00-DE-AD"
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(beafdeadVMI)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitUntilVMIReady(beafdeadVMI, tests.LoggedInAlpineExpecter)
			checkMacAddress(beafdeadVMI, "be:af:00:00:de:ad")
		})
	})

	Context("VirtualMachineInstance with invalid MAC addres", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		It("[test_id:700]should failed to start with invalid MAC address", func() {
			By("Start VMI")
			beafdeadVMI := tests.NewRandomVMIWithCustomMacAddress()
			beafdeadVMI.Spec.Domain.Devices.Interfaces[0].MacAddress = "de:00c:00c:00:00:de:abc"
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(beafdeadVMI)
			Expect(err).To(HaveOccurred())
			testErr := err.(*errors.StatusError)
			Expect(testErr.ErrStatus.Reason).To(BeEquivalentTo("Invalid"))
		})
	})

	Context("VirtualMachineInstance with custom MAC address and slirp interface", func() {

		setPermitSlirpInterface := func(enable bool) {
			if currentConfiguration.NetworkConfiguration == nil {
				currentConfiguration.NetworkConfiguration = &v1.NetworkConfiguration{}
			}

			currentConfiguration.NetworkConfiguration.PermitSlirpInterface = pointer.BoolPtr(enable)
			kv := tests.UpdateKubeVirtConfigValueAndWait(currentConfiguration)
			currentConfiguration = kv.Spec.Configuration
		}
		BeforeEach(func() {
			tests.BeforeTestCleanup()
			setPermitSlirpInterface(true)
		})
		AfterEach(func() {
			setPermitSlirpInterface(false)
		})

		It("[test_id:1773]should configure custom MAC address", func() {
			By("checking eth0 MAC address")
			deadbeafVMI := tests.NewRandomVMIWithSlirpInterfaceEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskAlpine), "#!/bin/bash\necho 'hello'\n", []v1.Port{})
			deadbeafVMI.Spec.Domain.Devices.Interfaces[0].MacAddress = "de:ad:00:00:be:af"
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(deadbeafVMI)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitUntilVMIReady(deadbeafVMI, tests.LoggedInAlpineExpecter)
			checkMacAddress(deadbeafVMI, deadbeafVMI.Spec.Domain.Devices.Interfaces[0].MacAddress)
		})
	})

	Context("VirtualMachineInstance with disabled automatic attachment of interfaces", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		It("[test_id:1774]should not configure any external interfaces", func() {
			By("checking loopback is the only guest interface")
			autoAttach := false
			detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			// Remove the masquerade interface to use the default bridge one
			detachedVMI.Spec.Domain.Devices.Interfaces = nil
			detachedVMI.Spec.Networks = nil
			detachedVMI.Spec.Domain.Devices.AutoattachPodInterface = &autoAttach

			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitUntilVMIReady(detachedVMI, tests.LoggedInCirrosExpecter)

			err := console.SafeExpectBatch(detachedVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "ls /sys/class/net/ | wc -l\n"},
				&expect.BExp{R: "1"},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:1775]should not request a tun device", func() {
			By("Creating random VirtualMachineInstance")
			autoAttach := false
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			// Remove the masquerade interface to use the default bridge one
			vmi.Spec.Domain.Devices.Interfaces = nil
			vmi.Spec.Networks = nil
			vmi.Spec.Domain.Devices.AutoattachPodInterface = &autoAttach

			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			waitUntilVMIReady(vmi, tests.LoggedInAlpineExpecter)

			By("Checking that the pod did not request a tun device")
			virtClient, err := kubecli.GetKubevirtClient()
			Expect(err).ToNot(HaveOccurred())

			By("Looking up pod using VMI's label")
			pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(tests.UnfinishedVMIPodSelector(vmi))
			Expect(err).ToNot(HaveOccurred())
			Expect(pods.Items).NotTo(BeEmpty())
			pod := pods.Items[0]

			foundContainer := false
			for _, container := range pod.Spec.Containers {
				if container.Name == "compute" {
					foundContainer = true
					_, ok := container.Resources.Requests[services.TunDevice]
					Expect(ok).To(BeFalse())

					_, ok = container.Resources.Limits[services.TunDevice]
					Expect(ok).To(BeFalse())

					caps := container.SecurityContext.Capabilities

					Expect(caps.Add).To(Not(ContainElement(k8sv1.Capability("NET_ADMIN"))), "Compute container should not have NET_ADMIN capability")
					Expect(caps.Drop).To(ContainElement(k8sv1.Capability("NET_RAW")), "Compute container should drop NET_RAW capability")
				}
			}

			Expect(foundContainer).To(BeTrue(), "Did not find 'compute' container in pod")
		})
	})

	Context("VirtualMachineInstance with custom PCI address", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		checkPciAddress := func(vmi *v1.VirtualMachineInstance, expectedPciAddress string) {
			err := console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "grep INTERFACE /sys/bus/pci/devices/" + expectedPciAddress + "/*/net/eth0/uevent|awk -F= '{ print $2 }'\n"},
				&expect.BExp{R: "eth0"},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
		}

		It("[test_id:1776]should configure custom Pci address", func() {
			By("checking eth0 Pci address")
			testVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			tests.AddExplicitPodNetworkInterface(testVMI)
			testVMI.Spec.Domain.Devices.Interfaces[0].PciAddress = "0000:81:00.1"
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(testVMI)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitUntilVMIReady(testVMI, tests.LoggedInCirrosExpecter)
			checkPciAddress(testVMI, testVMI.Spec.Domain.Devices.Interfaces[0].PciAddress)
		})
	})

	Context("VirtualMachineInstance with learning disabled on pod interface", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		It("[test_id:1777]should disable learning on pod iface", func() {
			By("checking learning flag")
			learningDisabledVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskAlpine), "#!/bin/bash\necho 'hello'\n")
			// Remove the masquerade interface to use the default bridge one
			learningDisabledVMI.Spec.Domain.Devices.Interfaces = nil
			learningDisabledVMI.Spec.Networks = nil
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(learningDisabledVMI)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitUntilVMIReady(learningDisabledVMI, tests.LoggedInAlpineExpecter)
			checkLearningState(learningDisabledVMI, "0")
		})
	})

	Context("VirtualMachineInstance with dhcp options", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		It("[test_id:1778]should offer extra dhcp options to pod iface", func() {
			userData := "#cloud-config\npassword: fedora\nchpasswd: { expire: False }\n"
			dhcpVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskFedora), userData)
			tests.AddExplicitPodNetworkInterface(dhcpVMI)

			dhcpVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceName("memory")] = resource.MustParse("1024M")
			dhcpVMI.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				BootFileName:   "config",
				TFTPServerName: "tftp.kubevirt.io",
				NTPServers:     []string{"127.0.0.1", "127.0.0.2"},
				PrivateOptions: []v1.DHCPPrivateOptions{v1.DHCPPrivateOptions{Option: 240, Value: "private.options.kubevirt.io"}},
			}

			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(dhcpVMI)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitUntilVMIReady(dhcpVMI, tests.LoggedInFedoraExpecter)

			err = console.SafeExpectBatch(dhcpVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "dhclient -1 -r -d eth0\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "dhclient -1 -sf /usr/bin/env --request-options subnet-mask,broadcast-address,time-offset,routers,domain-search,domain-name,domain-name-servers,host-name,nis-domain,nis-servers,ntp-servers,interface-mtu,tftp-server-name,bootfile-name eth0 | tee /dhcp-env\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "grep -q 'new_tftp_server_name=tftp.kubevirt.io' /dhcp-env; echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "grep -q 'new_bootfile_name=config' /dhcp-env; echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "grep -q 'new_ntp_servers=127.0.0.1 127.0.0.2' /dhcp-env; echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "grep -q 'new_unknown_240=private.options.kubevirt.io' /dhcp-env; echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
			}, 15)

			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("VirtualMachineInstance with custom dns", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})
		It("[test_id:1779]should have custom resolv.conf", func() {
			userData := "#cloud-config\n"
			dnsVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), userData)

			dnsVMI.Spec.DNSPolicy = "None"
			dnsVMI.Spec.DNSConfig = &k8sv1.PodDNSConfig{
				Nameservers: []string{"8.8.8.8", "4.2.2.1"},
				Searches:    []string{"example.com"},
			}
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(dnsVMI)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitUntilVMIReady(dnsVMI, tests.LoggedInCirrosExpecter)
			err = console.SafeExpectBatch(dnsVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "cat /etc/resolv.conf\n"},
				&expect.BExp{R: "search example.com"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "cat /etc/resolv.conf\n"},
				&expect.BExp{R: "nameserver 8.8.8.8"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "cat /etc/resolv.conf\n"},
				&expect.BExp{R: "nameserver 4.2.2.1"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("VirtualMachineInstance with masquerade binding mechanism", func() {
		const (
			defaultCIDR = false
			customCIDR  = true
		)

		var serverVMI *v1.VirtualMachineInstance
		var clientVMI *v1.VirtualMachineInstance

		masqueradeVMI := func(Ports []v1.Port, ipv4NetworkCIDR string) *v1.VirtualMachineInstance {
			containerImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
			userData := "#!/bin/bash\necho 'hello'\n"
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(containerImage, userData)
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", Ports: Ports, InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}}
			net := v1.DefaultPodNetwork()
			if ipv4NetworkCIDR != "" {
				net.NetworkSource.Pod.VMNetworkCIDR = ipv4NetworkCIDR
			}
			vmi.Spec.Networks = []v1.Network{*net}

			return vmi
		}

		table.DescribeTable("[Conformance][test_id:1780][label:masquerade_binding_connectivity]should allow regular network connection", func(ports []v1.Port, withCustomCIDR bool) {
			var ipv4NetworkCIDR string

			if withCustomCIDR {
				ipv4NetworkCIDR = "10.10.10.0/24"
			}

			// Create the client only one time
			if clientVMI == nil {
				clientVMI = masqueradeVMI([]v1.Port{}, ipv4NetworkCIDR)
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(clientVMI)
				Expect(err).ToNot(HaveOccurred())
				clientVMI = tests.WaitUntilVMIReady(clientVMI, tests.LoggedInCirrosExpecter)
			}

			serverVMI = masqueradeVMI(ports, ipv4NetworkCIDR)
			serverVMI.Labels = map[string]string{"expose": "server"}
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(serverVMI)
			Expect(err).ToNot(HaveOccurred())
			serverVMI = tests.WaitUntilVMIReady(serverVMI, tests.LoggedInCirrosExpecter)
			Expect(serverVMI.Status.Interfaces).To(HaveLen(1))
			Expect(serverVMI.Status.Interfaces[0].IPs).NotTo(BeEmpty())

			By("starting a tcp server")
			tcpPort := 8080
			tests.StartTCPServer(serverVMI, tcpPort)

			for _, serverIP := range serverVMI.Status.Interfaces[0].IPs {
				if netutils.IsIPv6String(serverIP) {
					By("Checking ping (IPv6) from vmi to cluster nodes gateway")
					// Cluster nodes subnet (docker network gateway)
					// Docker network subnet cidr definition:
					// https://github.com/kubevirt/project-infra/blob/master/github/ci/shared-deployments/files/docker-daemon-mirror.conf#L5
					Expect(libnet.PingFromVMConsole(serverVMI, "2001:db8:1::1")).To(Succeed())
				} else {
					if ipv4NetworkCIDR == "" {
						ipv4NetworkCIDR = api.DefaultVMCIDR
					}
					By("Checking ping (IPv4) to gateway")
					ipAddr := gatewayIPFromCIDR(ipv4NetworkCIDR)
					Expect(libnet.PingFromVMConsole(serverVMI, ipAddr)).To(Succeed())

					By("Checking ping (IPv4) to google")
					Expect(libnet.PingFromVMConsole(serverVMI, "8.8.8.8")).To(Succeed())
					Expect(libnet.PingFromVMConsole(clientVMI, "google.com")).To(Succeed())
				}

				Expect(libnet.PingFromVMConsole(clientVMI, serverIP)).To(Succeed())

				By("Connecting from the client vm")
				err = console.SafeExpectBatch(clientVMI, createExpectConnectToServer(serverIP, tcpPort, true), 30)
				Expect(err).ToNot(HaveOccurred())

				By("Rejecting the connection from the client to unregistered port")
				err = console.SafeExpectBatch(clientVMI, createExpectConnectToServer(serverIP, tcpPort+1, false), 30)
				Expect(err).ToNot(HaveOccurred())
			}

			err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(serverVMI.Name, &v13.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		}, table.Entry("with a specific port number", []v1.Port{{Name: "http", Port: 8080}}, defaultCIDR),
			table.Entry("without a specific port number", []v1.Port{}, defaultCIDR),
			table.Entry("with custom CIDR", []v1.Port{}, customCIDR),
		)

		When("performing migration", func() {
			var vmi *v1.VirtualMachineInstance
			var virtHandlerIPs []k8sv1.PodIP

			ping := func(ipAddr string) error {
				return libnet.PingFromVMConsole(vmi, ipAddr, "-c 1", "-w 2")
			}

			getVirtHandlerPod := func() (*k8sv1.Pod, error) {
				node := vmi.Status.NodeName
				pod, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(node).Pod()
				if err != nil {
					return nil, fmt.Errorf("failed to get virt-handler pod on node %s: %v", node, err)
				}
				return pod, nil
			}

			runMigrationAndExpectCompletion := func(migration *v1.VirtualMachineInstanceMigration, timeout int) {
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() error {
					migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &v13.GetOptions{})
					if err != nil {
						return err
					}

					Expect(migration.Status.Phase).ToNot(Equal(v1.MigrationFailed))

					if migration.Status.Phase == v1.MigrationSucceeded {
						return nil
					}
					return fmt.Errorf("Migration is in phase %s", migration.Status.Phase)

				}, timeout, time.Second).Should(Succeed(), fmt.Sprintf("migration should succeed after %d s", timeout))
			}

			BeforeEach(func() {
				if !tests.HasLiveMigration() {
					Skip("LiveMigration feature gate is not enabled in kubevirt-config")
				}

				var err error

				By("Create VMI")
				vmi = masqueradeVMI([]v1.Port{}, "")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(vmi, tests.LoggedInCirrosExpecter)

				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v13.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				virtHandlerPod, err := getVirtHandlerPod()
				Expect(err).ToNot(HaveOccurred())
				virtHandlerIPs = virtHandlerPod.Status.PodIPs

				By("Check connectivity")
				for _, podIP := range virtHandlerIPs {
					Expect(ping(podIP.IP)).To(Succeed())
				}

				By("Execute migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				runMigrationAndExpectCompletion(migration, migrationWaitTime)

				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v13.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.Phase).To(Equal(v1.Running))
			})

			AfterEach(func() {
				if vmi != nil {
					By("Delete VMI")
					Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &v13.DeleteOptions{})).To(Succeed())

					Eventually(func() error {
						_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v13.GetOptions{})
						return err
					}, time.Minute, time.Second).Should(
						SatisfyAll(HaveOccurred(), WithTransform(errors.IsNotFound, BeTrue())),
						"The VMI should be gone within the given timeout",
					)
				}
			})

			It("[Conformance] preserves connectivity", func() {
				Eventually(func() error {
					for _, podIP := range virtHandlerIPs {
						err := ping(podIP.IP)
						if err != nil {
							return err
						}
					}
					return nil
				}, 120*time.Second).Should(Succeed())
			})
		})

		Context("MTU verification", func() {
			var vmi *v1.VirtualMachineInstance
			var anotherVmi *v1.VirtualMachineInstance

			getMtu := func(pod *k8sv1.Pod, ifaceName string) int {
				output, err := tests.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"cat", fmt.Sprintf("/sys/class/net/%s/mtu", ifaceName)},
				)
				ExpectWithOffset(1, err).ToNot(HaveOccurred())

				output = strings.TrimSuffix(output, "\n")
				mtu, err := strconv.Atoi(output)
				ExpectWithOffset(1, err).ToNot(HaveOccurred())
				return mtu
			}

			BeforeEach(func() {
				var err error

				By("Create VMI")
				vmi = libvmi.NewFedora()

				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())

				By("Create another VMI")
				anotherVmi = masqueradeVMI([]v1.Port{}, "")
				anotherVmi, err = virtClient.VirtualMachineInstance(anotherVmi.Namespace).Create(anotherVmi)
				Expect(err).ToNot(HaveOccurred())

				By("Wait for VMIs to be ready")
				tests.WaitUntilVMIReady(anotherVmi, tests.LoggedInCirrosExpecter)
				anotherVmi, err = virtClient.VirtualMachineInstance(anotherVmi.Namespace).Get(anotherVmi.Name, &v13.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				tests.WaitUntilVMIReady(vmi, tests.LoggedInFedoraExpecter)
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v13.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				if vmi != nil {
					By("Delete VMI")
					Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &v13.DeleteOptions{})).To(Succeed())
				}
			})

			AfterEach(func() {
				if anotherVmi != nil {
					By("Delete another VMI")
					Expect(virtClient.VirtualMachineInstance(anotherVmi.Namespace).Delete(anotherVmi.Name, &v13.DeleteOptions{})).To(Succeed())
				}
			})

			table.DescribeTable("should have the correct MTU", func(ipFamily k8sv1.IPFamily) {
				if ipFamily == k8sv1.IPv6Protocol {
					libnet.SkipWhenNotDualStackCluster(virtClient)
				}

				By("checking k6t-eth0 MTU inside the pod")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				bridgeMtu := getMtu(vmiPod, "k6t-eth0")
				primaryIfaceMtu := getMtu(vmiPod, "eth0")

				Expect(bridgeMtu).To(Equal(primaryIfaceMtu), "k6t-eth0 bridge mtu should equal eth0 interface mtu")

				By("checking k6t-eth0-nic MTU inside the pod")
				bridgePrimaryNicMtu := getMtu(vmiPod, "k6t-eth0-nic")
				Expect(bridgePrimaryNicMtu).To(Equal(primaryIfaceMtu), "k6t-eth0-nic mtu should equal eth0 interface mtu")

				By("checking eth0 MTU inside the VirtualMachineInstance")
				showMtu := "cat /sys/class/net/eth0/mtu\n"
				err = console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: showMtu},
					&expect.BExp{R: console.RetValue(strconv.Itoa(bridgeMtu))},
				}, 180)
				Expect(err).ToNot(HaveOccurred())

				By("checking the VirtualMachineInstance can send MTU sized frames to another VirtualMachineInstance")
				icmpHeaderSize := 8
				var ipHeaderSize int
				if ipFamily == k8sv1.IPv4Protocol {
					ipHeaderSize = 20
				} else {
					ipHeaderSize = 40
				}
				payloadSize := primaryIfaceMtu - ipHeaderSize - icmpHeaderSize
				addr := libnet.GetVmiPrimaryIpByFamily(anotherVmi, ipFamily)
				Expect(libnet.PingFromVMConsole(vmi, addr, "-c 1", "-w 5", fmt.Sprintf("-s %d", payloadSize), "-M do")).To(Succeed())

				By("checking the VirtualMachineInstance cannot send bigger than MTU sized frames to another VirtualMachineInstance")
				Expect(libnet.PingFromVMConsole(vmi, addr, "-c 1", "-w 5", fmt.Sprintf("-s %d", payloadSize+1), "-M do")).ToNot(Succeed())
			},
				table.Entry("IPv4", k8sv1.IPv4Protocol),
				table.Entry("IPv6", k8sv1.IPv6Protocol),
			)
		})
	})

	Context("VirtualMachineInstance with TX offload disabled", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		It("[test_id:1781]should get turned off for interfaces that serve dhcp", func() {
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskAlpine), "#!/bin/bash\necho")
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceName("memory")] = resource.MustParse("1024M")

			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitUntilVMIReady(vmi, tests.LoggedInAlpineExpecter)

			output := tests.RunCommandOnVmiPod(vmi, []string{"python3", "-c", `import array
import fcntl
import socket
import struct
SIOCETHTOOL     = 0x8946
ETHTOOL_GTXCSUM = 0x00000016
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sockfd = sock.fileno()
ecmd = array.array('B', struct.pack('I39s', ETHTOOL_GTXCSUM, b'\x00'*39))
ifreq = struct.pack('16sP', str.encode('k6t-eth0'), ecmd.buffer_info()[0])
fcntl.ioctl(sockfd, SIOCETHTOOL, ifreq)
res = ecmd.tostring()
print(res[4])
sock = None
sockfd = None`})

			ExpectWithOffset(1, strings.TrimSpace(output)).To(Equal("0"))
		})
	})

	Context("vmi with default bridge interface on pod network", func() {
		BeforeEach(func() {
			setBridgeEnabled(false)
		})
		AfterEach(func() {
			setBridgeEnabled(true)
		})
		It("[test_id:2964]should reject VMIs with bridge interface when it's not permitted on pod network", func() {
			var t int64 = 0
			vmi := v1.NewMinimalVMIWithNS(tests.NamespaceTestDefault, "testvmi"+rand.String(48))
			vmi.Spec.TerminationGracePeriodSeconds = &t
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")
			tests.AddEphemeralDisk(vmi, "disk0", "virtio", cd.ContainerDiskFor(cd.ContainerDiskCirros))

			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err.Error()).To(ContainSubstring("Bridge interface is not enabled in kubevirt-config"))
		})
	})
})

func waitUntilVMIReady(vmi *v1.VirtualMachineInstance, expecterFactory console.VMIExpecterFactory) *v1.VirtualMachineInstance {
	// Wait for VirtualMachineInstance start
	tests.WaitForSuccessfulVMIStart(vmi)

	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	// Fetch the new VirtualMachineInstance with updated status
	vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &v13.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	// Lets make sure that the OS is up by waiting until we can login
	expecter, err := expecterFactory(vmi)
	Expect(err).ToNot(HaveOccurred())
	expecter.Close()
	return vmi
}

func NewRandomVMIWithInvalidNetworkInterface() *v1.VirtualMachineInstance {
	// Use alpine because cirros dhcp client starts prematurily before link is ready
	vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
	tests.AddExplicitPodNetworkInterface(vmi)
	vmi.Spec.Domain.Devices.Interfaces[0].Model = "gibberish"
	return vmi
}

func createExpectConnectToServer(serverIP string, tcpPort int, expectSuccess bool) []expect.Batcher {
	expectResult := "1"
	if expectSuccess {
		expectResult = "0"
	}
	return []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: fmt.Sprintf("echo test | nc %s %d -i 1 -w 1 1> /dev/null\n", serverIP, tcpPort)},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: console.RetValue(expectResult)},
	}
}

// gatewayIpFromCIDR returns the first address of a network.
func gatewayIPFromCIDR(cidr string) string {
	ip, ipnet, _ := net.ParseCIDR(cidr)
	ip = ip.Mask(ipnet.Mask)
	oct := len(ip) - 1
	ip[oct]++
	return ip.String()
}
