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
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[rfe_id:694][crit:medium][vendor:cnv-qe@redhat.com][level:component]Networking", func() {

	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var inboundVMI *v1.VirtualMachineInstance
	var inboundVMIWithPodNetworkSet *v1.VirtualMachineInstance
	var inboundVMIWithCustomMacAddress *v1.VirtualMachineInstance
	var outboundVMI *v1.VirtualMachineInstance

	const testPort = 1500

	logPodLogs := func(pod *v12.Pod) {
		defer GinkgoRecover()

		var s int64 = 500
		logs := virtClient.CoreV1().Pods(inboundVMI.Namespace).GetLogs(pod.Name, &v12.PodLogOptions{SinceSeconds: &s})
		rawLogs, err := logs.DoRaw()
		Expect(err).ToNot(HaveOccurred())
		log.Log.Infof("%s", string(rawLogs))
	}

	waitForPodToFinish := func(pod *v12.Pod) v12.PodPhase {
		Eventually(func() v12.PodPhase {
			j, err := virtClient.CoreV1().Pods(inboundVMI.ObjectMeta.Namespace).Get(pod.ObjectMeta.Name, v13.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return j.Status.Phase
		}, 90*time.Second, 1*time.Second).Should(Or(Equal(v12.PodSucceeded), Equal(v12.PodFailed)))
		j, err := virtClient.CoreV1().Pods(inboundVMI.ObjectMeta.Namespace).Get(pod.ObjectMeta.Name, v13.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		logPodLogs(pod)
		return j.Status.Phase
	}

	checkMacAddress := func(vmi *v1.VirtualMachineInstance, expectedMacAddress string, prompt string) {
		err := tests.CheckForTextExpecter(vmi, []expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: prompt},
			&expect.BSnd{S: "cat /sys/class/net/eth0/address\n"},
			&expect.BExp{R: expectedMacAddress},
		}, 15)
		Expect(err).ToNot(HaveOccurred())
	}

	checkNetworkVendor := func(vmi *v1.VirtualMachineInstance, expectedVendor string, prompt string) {
		err := tests.CheckForTextExpecter(vmi, []expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: prompt},
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
		tests.UpdateClusterConfigValueAndWait("permitBridgeInterfaceOnPodNetwork", fmt.Sprintf("%t", enable))
	}

	Describe("Multiple virtual machines connectivity using bridge binding interface", func() {
		tests.BeforeAll(func() {
			tests.BeforeTestCleanup()

			// Prepare inbound and outbound VMI definitions

			// inboundVMI expects implicitly to be added to the pod network
			inboundVMI = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			inboundVMI.Labels = map[string]string{"expose": "me"}
			inboundVMI.Spec.Subdomain = "myvmi"
			inboundVMI.Spec.Hostname = "my-subdomain"
			// Remove the masquerade interface to use the default bridge one
			inboundVMI.Spec.Domain.Devices.Interfaces = nil
			inboundVMI.Spec.Networks = nil

			// outboundVMI is used to connect to other vms
			outboundVMI = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			// Remove the masquerade interface to use the default bridge one
			outboundVMI.Spec.Domain.Devices.Interfaces = nil
			outboundVMI.Spec.Networks = nil

			// inboudnVMIWithPodNetworkSet adds itself in an explicit fashion to the pod network
			inboundVMIWithPodNetworkSet = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			// Remove the masquerade interface to use the default bridge one
			inboundVMIWithPodNetworkSet.Spec.Domain.Devices.Interfaces = nil
			inboundVMIWithPodNetworkSet.Spec.Networks = nil
			v1.SetDefaults_NetworkInterface(inboundVMIWithPodNetworkSet)
			Expect(inboundVMIWithPodNetworkSet.Spec.Domain.Devices.Interfaces).NotTo(BeEmpty())

			// inboundVMIWithCustomMacAddress specifies a custom MAC address
			inboundVMIWithCustomMacAddress = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
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
				[]string{"ip", "address", "show", "k6t-eth0"},
			)
			log.Log.Infof("%v", output)
			Expect(err).ToNot(HaveOccurred())

			// The MTU of the Pod network varies, depending on the environment in use. We want to
			// verify that the actual MTU is from the possible range, which is {1500, 1450}, minus
			//  50 bytes for vxlan overhead.
			possibleMtus := []int{1400, 1450}
			mtuMatch := false
			expectedMtuString := ""

			for _, expectedMtu := range possibleMtus {
				payloadSize = expectedMtu - ipHeaderSize

				// the following substring is part of 'ip address show' output
				expectedMtuString = fmt.Sprintf("mtu %d", expectedMtu)
				mtuMatch = strings.Contains(output, expectedMtuString)
				if mtuMatch == true {
					break
				}
			}
			Expect(mtuMatch).To(BeTrue())

			By("checking eth0 MTU inside the VirtualMachineInstance")
			expecter, err := tests.LoggedInCirrosExpecter(outboundVMI)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			addrShow = "ip address show eth0\n"
			resp, err := expecter.ExpectBatch([]expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: addrShow},
				&expect.BExp{R: fmt.Sprintf(".*%s.*\n", expectedMtuString)},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
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
			err = tests.CheckForTextExpecter(outboundVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: cmdCheck},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
			}, 180)
			Expect(err).ToNot(HaveOccurred())

			By("checking the VirtualMachineInstance can fetch via HTTP")
			err = tests.CheckForTextExpecter(outboundVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "curl --silent http://kubevirt.io > /dev/null\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
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

			//TODO if node count 1, skip whe nv12.NodeSelectorOpOut
			nodes, err := virtClient.CoreV1().Nodes().List(v13.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(nodes.Items).ToNot(BeEmpty())
			if len(nodes.Items) == 1 && op == v12.NodeSelectorOpNotIn {
				Skip("Skip network test that requires multiple nodes when only one node is present.")
			}

			// Run netcat and give it one second to ghet "Hello World!" back from the VM
			job := tests.NewHelloWorldJob(ip, strconv.Itoa(testPort))
			job.Spec.Affinity = &v12.Affinity{
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
			job.Spec.HostNetwork = hostNetwork

			job, err = virtClient.CoreV1().Pods(inboundVMI.ObjectMeta.Namespace).Create(job)
			Expect(err).ToNot(HaveOccurred())
			phase := waitForPodToFinish(job)
			Expect(phase).To(Equal(v12.PodSucceeded))
		},
			table.Entry("[test_id:1543]on the same node from Pod", v12.NodeSelectorOpIn, false),
			table.Entry("[test_id:1544]on a different node from Pod", v12.NodeSelectorOpNotIn, false),
			table.Entry("[test_id:1545]on the same node from Node", v12.NodeSelectorOpIn, true),
			table.Entry("[test_id:1546]on a different node from Node", v12.NodeSelectorOpNotIn, true),
		)

		Context("with a service matching the vmi exposed", func() {
			BeforeEach(func() {
				service := &v12.Service{
					ObjectMeta: v13.ObjectMeta{
						Name: "myservice",
					},
					Spec: v12.ServiceSpec{
						Selector: map[string]string{
							"expose": "me",
						},
						Ports: []v12.ServicePort{
							{Protocol: v12.ProtocolTCP, Port: testPort, TargetPort: intstr.FromInt(testPort)},
						},
					},
				}

				_, err := virtClient.CoreV1().Services(inboundVMI.Namespace).Create(service)
				Expect(err).ToNot(HaveOccurred())

			})
			It("[test_id:1547] should be able to reach the vmi based on labels specified on the vmi", func() {

				By("starting a pod which tries to reach the vmi via the defined service")
				job := tests.NewHelloWorldJob(fmt.Sprintf("%s.%s", "myservice", inboundVMI.Namespace), strconv.Itoa(testPort))
				job, err = virtClient.CoreV1().Pods(inboundVMI.Namespace).Create(job)
				Expect(err).ToNot(HaveOccurred())

				By("waiting for the pod to report a successful connection attempt")
				phase := waitForPodToFinish(job)
				Expect(phase).To(Equal(v12.PodSucceeded))
			})
			It("[test_id:1548]should fail to reach the vmi if an invalid servicename is used", func() {

				By("starting a pod which tries to reach the vmi via a non-existent service")
				job := tests.NewHelloWorldJob(fmt.Sprintf("%s.%s", "wrongservice", inboundVMI.Namespace), strconv.Itoa(testPort))
				job, err = virtClient.CoreV1().Pods(inboundVMI.Namespace).Create(job)
				Expect(err).ToNot(HaveOccurred())
				By("waiting for the pod to report an  unsuccessful connection attempt")
				phase := waitForPodToFinish(job)
				Expect(phase).To(Equal(v12.PodFailed))
			})

			AfterEach(func() {
				Expect(virtClient.CoreV1().Services(inboundVMI.Namespace).Delete("myservice", &v13.DeleteOptions{})).To(Succeed())
			})
		})

		Context("with a subdomain and a headless service given", func() {
			BeforeEach(func() {
				service := &v12.Service{
					ObjectMeta: v13.ObjectMeta{
						Name: inboundVMI.Spec.Subdomain,
					},
					Spec: v12.ServiceSpec{
						ClusterIP: v12.ClusterIPNone,
						Selector: map[string]string{
							"expose": "me",
						},
						/* Normally ports are not required on headless services, but there is a bug in kubedns:
						https://github.com/kubernetes/kubernetes/issues/55158
						*/
						Ports: []v12.ServicePort{
							{Protocol: v12.ProtocolTCP, Port: testPort, TargetPort: intstr.FromInt(testPort)},
						},
					},
				}
				_, err := virtClient.CoreV1().Services(inboundVMI.Namespace).Create(service)
				Expect(err).ToNot(HaveOccurred())
			})

			It("[test_id:1549]should be able to reach the vmi via its unique fully qualified domain name", func() {
				By("starting a pod which tries to reach the vm via the defined service")
				job := tests.NewHelloWorldJob(fmt.Sprintf("%s.%s.%s", inboundVMI.Spec.Hostname, inboundVMI.Spec.Subdomain, inboundVMI.Namespace), strconv.Itoa(testPort))
				job, err = virtClient.CoreV1().Pods(inboundVMI.Namespace).Create(job)
				Expect(err).ToNot(HaveOccurred())

				By("waiting for the pod to report a successful connection attempt")
				phase := waitForPodToFinish(job)
				Expect(phase).To(Equal(v12.PodSucceeded))
			})

			AfterEach(func() {
				Expect(virtClient.CoreV1().Services(inboundVMI.Namespace).Delete(inboundVMI.Spec.Subdomain, &v13.DeleteOptions{})).To(Succeed())
			})
		})

		Context("VirtualMachineInstance with default interface model", func() {
			// Unless an explicit interface model is specified, the default interface model is virtio.
			It("[test_id:1550]should expose the right device type to the guest", func() {
				By("checking the device vendor in /sys/class")

				// Taken from https://wiki.osdev.org/Virtio#Technical_Details
				virtio_vid := "0x1af4"

				for _, networkVMI := range []*v1.VirtualMachineInstance{inboundVMI, outboundVMI} {
					// as defined in https://vendev.org/pci/ven_1af4/
					checkNetworkVendor(networkVMI, virtio_vid, "\\$ ")
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
			checkNetworkVendor(e1000VMI, "0x8086", "localhost:~#")
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
			checkMacAddress(deadbeafVMI, deadbeafVMI.Spec.Domain.Devices.Interfaces[0].MacAddress, "localhost:~#")
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
			checkMacAddress(beafdeadVMI, "be:af:00:00:de:ad", "localhost:~#")
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
		BeforeEach(func() {
			tests.BeforeTestCleanup()
			tests.UpdateClusterConfigValueAndWait("permitSlirpInterface", "true")
		})
		AfterEach(func() {
			tests.UpdateClusterConfigValueAndWait("permitSlirpInterface", "false")
		})

		It("[test_id:1773]should configure custom MAC address", func() {
			By("checking eth0 MAC address")
			deadbeafVMI := tests.NewRandomVMIWithSlirpInterfaceEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskAlpine), "#!/bin/bash\necho 'hello'\n", []v1.Port{})
			deadbeafVMI.Spec.Domain.Devices.Interfaces[0].MacAddress = "de:ad:00:00:be:af"
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(deadbeafVMI)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitUntilVMIReady(deadbeafVMI, tests.LoggedInAlpineExpecter)
			checkMacAddress(deadbeafVMI, deadbeafVMI.Spec.Domain.Devices.Interfaces[0].MacAddress, "localhost:~#")
		})
	})

	Context("VirtualMachineInstance with disabled automatic attachment of interfaces", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		It("[test_id:1774]should not configure any external interfaces", func() {
			By("checking loopback is the only guest interface")
			autoAttach := false
			detachedVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			// Remove the masquerade interface to use the default bridge one
			detachedVMI.Spec.Domain.Devices.Interfaces = nil
			detachedVMI.Spec.Networks = nil
			detachedVMI.Spec.Domain.Devices.AutoattachPodInterface = &autoAttach

			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(detachedVMI)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitUntilVMIReady(detachedVMI, tests.LoggedInCirrosExpecter)

			err := tests.CheckForTextExpecter(detachedVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "ls /sys/class/net/ | wc -l\n"},
				&expect.BExp{R: "1"},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:1775]should not request a tun device", func() {
			By("Creating random VirtualMachineInstance")
			autoAttach := false
			vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
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

					netAdminCap := false
					caps := container.SecurityContext.Capabilities
					for _, cap := range caps.Add {
						if cap == "NET_ADMIN" {
							netAdminCap = true
						}
					}
					Expect(netAdminCap).To(BeFalse(), "Compute container should not have NET_ADMIN capability")
				}
			}

			Expect(foundContainer).To(BeTrue(), "Did not find 'compute' container in pod")
		})
	})

	Context("VirtualMachineInstance with custom PCI address", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		checkPciAddress := func(vmi *v1.VirtualMachineInstance, expectedPciAddress string, prompt string) {
			err := tests.CheckForTextExpecter(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: prompt},
				&expect.BSnd{S: "grep INTERFACE /sys/bus/pci/devices/" + expectedPciAddress + "/*/net/eth0/uevent|awk -F= '{ print $2 }'\n"},
				&expect.BExp{R: "eth0"},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
		}

		It("[test_id:1776]should configure custom Pci address", func() {
			By("checking eth0 Pci address")
			testVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			tests.AddExplicitPodNetworkInterface(testVMI)
			testVMI.Spec.Domain.Devices.Interfaces[0].PciAddress = "0000:81:00.1"
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(testVMI)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitUntilVMIReady(testVMI, tests.LoggedInCirrosExpecter)
			checkPciAddress(testVMI, testVMI.Spec.Domain.Devices.Interfaces[0].PciAddress, "\\$")
		})
	})

	Context("VirtualMachineInstance with learning disabled on pod interface", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		It("[test_id:1777]should disable learning on pod iface", func() {
			By("checking learning flag")
			learningDisabledVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskAlpine), "#!/bin/bash\necho 'hello'\n")
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
			dhcpVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskFedora), userData)
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

			err = tests.CheckForTextExpecter(dhcpVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "#"},
				&expect.BSnd{S: "sudo dhclient -1 -r -d eth0\n"},
				&expect.BExp{R: "#"},
				&expect.BSnd{S: "sudo dhclient -1 -sf /usr/bin/env --request-options subnet-mask,broadcast-address,time-offset,routers,domain-search,domain-name,domain-name-servers,host-name,nis-domain,nis-servers,ntp-servers,interface-mtu,tftp-server-name,bootfile-name eth0 | tee /dhcp-env\n"},
				&expect.BExp{R: "#"},
				&expect.BSnd{S: "cat /dhcp-env\n"},
				&expect.BExp{R: "new_tftp_server_name=tftp.kubevirt.io"},
				&expect.BExp{R: "#"},
				&expect.BSnd{S: "cat /dhcp-env\n"},
				&expect.BExp{R: "new_bootfile_name=config"},
				&expect.BExp{R: "#"},
				&expect.BSnd{S: "cat /dhcp-env\n"},
				&expect.BExp{R: "new_ntp_servers=127.0.0.1 127.0.0.2"},
				&expect.BExp{R: "new_unknown_240=private.options.kubevirt.io"},
				&expect.BExp{R: "#"},
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
			dnsVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), userData)

			dnsVMI.Spec.DNSPolicy = "None"
			dnsVMI.Spec.DNSConfig = &k8sv1.PodDNSConfig{
				Nameservers: []string{"8.8.8.8", "4.2.2.1"},
				Searches:    []string{"example.com"},
			}
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(dnsVMI)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitUntilVMIReady(dnsVMI, tests.LoggedInCirrosExpecter)
			err = tests.CheckForTextExpecter(dnsVMI, []expect.Batcher{
				&expect.BSnd{S: "\n\n"},
				&expect.BExp{R: "$"},
				&expect.BSnd{S: "cat /etc/resolv.conf\n"},
				&expect.BExp{R: "search example.com"},
				&expect.BExp{R: "$"},
				&expect.BSnd{S: "cat /etc/resolv.conf\n"},
				&expect.BExp{R: "nameserver 8.8.8.8"},
				&expect.BExp{R: "$"},
				&expect.BSnd{S: "cat /etc/resolv.conf\n"},
				&expect.BExp{R: "nameserver 4.2.2.1"},
				&expect.BExp{R: "$"},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("VirtualMachineInstance with masquerade binding mechanism", func() {
		var serverVMI *v1.VirtualMachineInstance
		var clientVMI *v1.VirtualMachineInstance

		masqueradeVMI := func(containerImage string, userData string, Ports []v1.Port) *v1.VirtualMachineInstance {
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(containerImage, userData)
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", Ports: Ports, InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			return vmi
		}

		table.DescribeTable("[test_id:1780]should allow regular network connection", func(ports []v1.Port) {
			// Create the client only one time
			if clientVMI == nil {
				clientVMI = masqueradeVMI(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n", []v1.Port{})
				_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(clientVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitUntilVMIReady(clientVMI, tests.LoggedInCirrosExpecter)
			}

			serverVMI = masqueradeVMI(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n", ports)
			serverVMI.Labels = map[string]string{"expose": "server"}
			_, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(serverVMI)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitUntilVMIReady(serverVMI, tests.LoggedInCirrosExpecter)

			serverVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(serverVMI.Name, &v13.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(serverVMI.Status.Interfaces)).To(Equal(1))

			By("checking ping to google")
			pingVirtualMachine(serverVMI, "8.8.8.8", "\\$ ")
			pingVirtualMachine(clientVMI, "google.com", "\\$ ")

			By("starting a tcp server")
			err = tests.CheckForTextExpecter(serverVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "screen -d -m sudo nc -klp 8080 -e echo -e 'Hello World!'\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
			}, 30)
			Expect(err).ToNot(HaveOccurred())

			By("Connecting from the client vm")
			err = tests.CheckForTextExpecter(clientVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: fmt.Sprintf("echo test | nc %s 8080 -i 1 -w 1 1> /dev/null\n", serverVMI.Status.Interfaces[0].IP)},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
			}, 30)
			Expect(err).ToNot(HaveOccurred())

			By("Rejecting the connection from the client to unregistered port")
			err = tests.CheckForTextExpecter(clientVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: fmt.Sprintf("echo test | nc %s 8081 -i 1 -w 1 1> /dev/null\n", serverVMI.Status.Interfaces[0].IP)},
				&expect.BExp{R: "\\$ "},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "1"},
			}, 30)
			Expect(err).ToNot(HaveOccurred())

			err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(serverVMI.Name, &v13.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

		}, table.Entry("with a specific port number", []v1.Port{{Name: "http", Port: 8080}}),
			table.Entry("without a specific port number", []v1.Port{}),
		)
	})

	Context("VirtualMachineInstance with TX offload disabled", func() {
		BeforeEach(func() {
			tests.BeforeTestCleanup()
		})

		It("[test_id:1781]should get turned off for interfaces that serve dhcp", func() {
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskAlpine), "#!/bin/bash\necho")
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
			tests.AddEphemeralDisk(vmi, "disk0", "virtio", tests.ContainerDiskFor(tests.ContainerDiskCirros))

			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Bridge interface is not enabled in kubevirt-config"))
		})
	})
})

func waitUntilVMIReady(vmi *v1.VirtualMachineInstance, expecterFactory tests.VMIExpecterFactory) *v1.VirtualMachineInstance {
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
	vmi := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
	tests.AddExplicitPodNetworkInterface(vmi)
	vmi.Spec.Domain.Devices.Interfaces[0].Model = "gibberish"
	return vmi
}
