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
	"net"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	netutils "k8s.io/utils/net"

	"kubevirt.io/kubevirt/tests/libnet/cluster"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
	"kubevirt.io/kubevirt/tests/libnet/job"
	"kubevirt.io/kubevirt/tests/libnet/vmnetserver"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("[rfe_id:694][crit:medium][vendor:cnv-qe@redhat.com][level:component]Networking", decorators.Networking, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	checkLearningState := func(vmi *v1.VirtualMachineInstance, expectedValue string) {
		output := libpod.RunCommandOnVmiPod(vmi, []string{"cat", "/sys/class/net/eth0-nic/brport/learning"})
		ExpectWithOffset(1, strings.TrimSpace(output)).To(Equal(expectedValue))
	}

	Describe("Multiple virtual machines connectivity using bridge binding interface", func() {
		var inboundVMI *v1.VirtualMachineInstance
		var outboundVMI *v1.VirtualMachineInstance
		var inboundVMIWithCustomMacAddress *v1.VirtualMachineInstance
		var inboundVMIWithMultiQueueSingleCPU *v1.VirtualMachineInstance

		BeforeEach(func() {
			libnet.SkipWhenClusterNotSupportIpv4()
		})
		Context("with a test outbound VMI", func() {
			BeforeEach(func() {
				inboundVMI = libvmifact.NewCirros()
				outboundVMI = libvmifact.NewCirros()
				inboundVMIWithCustomMacAddress = vmiWithCustomMacAddress("de:ad:00:00:be:af")
				inboundVMIWithMultiQueueSingleCPU = vmiWithMultiQueue()

				outboundVMI = runVMI(outboundVMI)
			})

			DescribeTable("should be able to reach", func(vmiRef **v1.VirtualMachineInstance) {
				vmi := runVMI(*vmiRef)
				addr := vmi.Status.Interfaces[0].IP

				payloadSize := 0
				ipHeaderSize := 28 // IPv4 specific

				vmiPod, err := libpod.GetPodByVirtualMachineInstance(outboundVMI, outboundVMI.Namespace)
				Expect(err).NotTo(HaveOccurred())

				Expect(libnet.ValidateVMIandPodIPMatch(outboundVMI, vmiPod)).To(Succeed(), "Should have matching IP/s between pod and vmi")

				var mtu int
				for _, ifaceName := range []string{"k6t-eth0", "tap0"} {
					By(fmt.Sprintf("checking %s MTU inside the pod", ifaceName))
					output, err := exec.ExecuteCommandOnPod(
						vmiPod,
						"compute",
						[]string{"cat", fmt.Sprintf("/sys/class/net/%s/mtu", ifaceName)},
					)
					log.Log.Infof("%s mtu is %v", ifaceName, output)
					Expect(err).ToNot(HaveOccurred())

					output = strings.TrimSuffix(output, "\n")
					mtu, err = strconv.Atoi(output)
					Expect(err).ToNot(HaveOccurred())

					Expect(mtu).To(BeNumerically(">", 1000))

					payloadSize = mtu - ipHeaderSize
				}
				expectedMtuString := fmt.Sprintf("mtu %d", mtu)

				By("checking eth0 MTU inside the VirtualMachineInstance")
				Expect(console.LoginToCirros(outboundVMI)).To(Succeed())

				addrShow := "ip address show eth0\n"
				Expect(console.SafeExpectBatch(outboundVMI, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: ""},
					&expect.BSnd{S: addrShow},
					&expect.BExp{R: fmt.Sprintf(".*%s.*\n", expectedMtuString)},
					&expect.BSnd{S: console.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("0")},
				}, 180)).To(Succeed())

				By("checking the VirtualMachineInstance can send MTU sized frames to another VirtualMachineInstance")
				// NOTE: VirtualMachineInstance is not directly accessible from inside the pod because
				// we transferred its IP address under DHCP server control, so the
				// only thing we can validate is connectivity between VMIs
				//
				// NOTE: cirros ping doesn't support -M do that could be used to
				// validate end-to-end connectivity with Don't Fragment flag set
				cmdCheck := fmt.Sprintf("ping %s -c 1 -w 5 -s %d\n", addr, payloadSize)
				err = console.SafeExpectBatch(outboundVMI, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: ""},
					&expect.BSnd{S: cmdCheck},
					&expect.BExp{R: ""},
					&expect.BSnd{S: console.EchoLastReturnValue},
					&expect.BExp{R: console.RetValue("0")},
				}, 180)
				Expect(err).ToNot(HaveOccurred())

				By("checking the MAC address of eth0 is inline with vmi status")
				if vmiHasCustomMacAddress(vmi) {
					Expect(vmi.Status.Interfaces).NotTo(BeEmpty())
					Expect(vmi.Status.Interfaces[0].MAC).To(Equal(vmi.Spec.Domain.Devices.Interfaces[0].MacAddress))
				}
				Expect(libnet.CheckMacAddress(vmi, "eth0", vmi.Status.Interfaces[0].MAC)).To(Succeed())
			},
				Entry("[test_id:1539]the Inbound VirtualMachineInstance with default (implicit) binding", &inboundVMI),
				Entry("[test_id:1541]the Inbound VirtualMachineInstance with custom MAC address", &inboundVMIWithCustomMacAddress),
				Entry("[test_id:1542]the Inbound VirtualMachineInstance with muti-queue and a single CPU", &inboundVMIWithMultiQueueSingleCPU),
			)
		})

		It("clients should be able to reach VM workload, with propagated IP from a pod", func() {
			inboundVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), libvmifact.NewCirros(), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			inboundVMI = libwait.WaitUntilVMIReady(inboundVMI, console.LoginToCirros)
			const testPort = 1500
			vmnetserver.StartTCPServer(inboundVMI, testPort, console.LoginToCirros)

			ip := inboundVMI.Status.Interfaces[0].IP

			By("start connectivity job on the same node as the VM")
			localNodeTCPJob := job.NewHelloWorldJobTCP(ip, strconv.Itoa(testPort))
			localNodeTCPJob.Spec.Template.Spec.Affinity = &k8sv1.Affinity{NodeAffinity: newNodeAffinity(k8sv1.NodeSelectorOpIn, inboundVMI.Status.NodeName)}
			localNodeTCPJob, err = virtClient.BatchV1().Jobs(inboundVMI.ObjectMeta.Namespace).Create(context.Background(), localNodeTCPJob, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("start connectivity job on different node")
			remoteNodeTCPJob := job.NewHelloWorldJobTCP(ip, strconv.Itoa(testPort))
			remoteNodeTCPJob.Spec.Template.Spec.Affinity = &k8sv1.Affinity{NodeAffinity: newNodeAffinity(k8sv1.NodeSelectorOpNotIn, inboundVMI.Status.NodeName)}
			remoteNodeTCPJob, err = virtClient.BatchV1().Jobs(inboundVMI.ObjectMeta.Namespace).Create(context.Background(), remoteNodeTCPJob, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(job.WaitForJobToSucceed(localNodeTCPJob, 90*time.Second)).To(Succeed(), "should be able to reach VM workload from a pod on the same node")
			Expect(job.WaitForJobToSucceed(remoteNodeTCPJob, 90*time.Second)).To(Succeed(), "should be able to reach VM workload from a pod on different node")
		})
	})

	Context("VirtualMachineInstance with custom and default interface models", func() {
		const nadName = "simple-bridge"

		BeforeEach(func() {
			const bridgeName = "br10"
			netAttachDef := libnet.NewBridgeNetAttachDef(nadName, bridgeName)
			_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(nil), netAttachDef)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:1770]should expose the right device type to the guest", func() {
			By("checking the device vendor in /sys/class")
			// Create a machine with e1000 interface model
			// Use alpine because cirros dhcp client starts prematurely before link is ready
			e1000ModelIface := libvmi.InterfaceDeviceWithMasqueradeBinding()
			e1000ModelIface.Model = "e1000"
			e1000ModelIface.PciAddress = "0000:02:01.0"

			const secondaryNetName = "secondary-net"
			defaultModelIface := libvmi.InterfaceDeviceWithBridgeBinding(secondaryNetName)
			defaultModelIface.PciAddress = "0000:03:00.0"
			vmi := libvmifact.NewAlpine(
				libvmi.WithInterface(e1000ModelIface),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(defaultModelIface),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName, nadName)),
			)

			var err error
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

			By("verifying vendors for respective PCI devices")
			const (
				vendorCmd = "cat /sys/bus/pci/devices/%s/vendor\n"
				// https://admin.pci-ids.ucw.cz/read/PC/8086
				intelVendorID = "0x8086"
				// https://admin.pci-ids.ucw.cz/read/PC/1af4
				redhatVendorID = "0x1af4"
			)

			err = console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: fmt.Sprintf(vendorCmd, e1000ModelIface.PciAddress)},
				&expect.BExp{R: intelVendorID},
				&expect.BSnd{S: fmt.Sprintf(vendorCmd, defaultModelIface.PciAddress)},
				&expect.BExp{R: redhatVendorID},
			}, 40)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	It("[test_id:1774]should not configure any external interfaces when a VMI has no networks and auto attachment is disabled", func() {
		vmi := libvmifact.NewAlpine(libvmi.WithAutoAttachPodInterface(false))

		var err error
		vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

		Expect(vmi.Spec.Domain.Devices.Interfaces).To(BeEmpty())

		By("checking that loopback is the only guest interface")
		err = console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: ""},
			&expect.BSnd{S: "ls /sys/class/net/ | wc -l\n"},
			&expect.BExp{R: "1"},
		}, 15)
		Expect(err).ToNot(HaveOccurred())
	})

	It("VMI with an interface that has ACPI Index set", func() {
		const acpiIndex = 101
		const pciAddress = "0000:01:00.0"
		iface := *v1.DefaultMasqueradeNetworkInterface()
		iface.ACPIIndex = acpiIndex
		iface.PciAddress = pciAddress
		testVMI := libvmifact.NewAlpine(
			libvmi.WithInterface(iface),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		var err error
		testVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), testVMI, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		libwait.WaitUntilVMIReady(testVMI, console.LoginToAlpine)

		err = console.SafeExpectBatch(testVMI, []expect.Batcher{
			&expect.BSnd{S: "\n"},
			&expect.BExp{R: ""},
			&expect.BSnd{S: "ls /sys/bus/pci/devices/" + pciAddress + "/virtio0/net\n"},
			&expect.BExp{R: "eth0"},
			&expect.BSnd{S: "cat /sys/bus/pci/devices/" + pciAddress + "/acpi_index\n"},
			&expect.BExp{R: strconv.Itoa(acpiIndex)},
		}, 15)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("VirtualMachineInstance with learning disabled on pod interface", func() {
		It("[test_id:1777]should disable learning on pod iface", func() {
			libnet.SkipWhenClusterNotSupportIpv4()
			By("checking learning flag")
			learningDisabledVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), libvmifact.NewAlpine(), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			libwait.WaitUntilVMIReady(learningDisabledVMI, console.LoginToAlpine)
			checkLearningState(learningDisabledVMI, "0")
		})
	})

	Context("VirtualMachineInstance with dhcp options", func() {
		It("[test_id:1778]should offer extra dhcp options to pod iface", func() {
			libnet.SkipWhenClusterNotSupportIpv4()
			dhcpVMI := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())

			// This IPv4 address tests backwards compatibility of the "DHCPOptions.NTPServers" field.
			// The leading zero is intentional.
			// For more details please see: https://github.com/kubevirt/kubevirt/issues/6498
			const NTPServerWithLeadingZeros = "0127.0.0.3"

			dhcpVMI.Spec.Domain.Devices.Interfaces[0].DHCPOptions = &v1.DHCPOptions{
				BootFileName:   "config",
				TFTPServerName: "tftp.kubevirt.io",
				NTPServers:     []string{"127.0.0.1", "127.0.0.2", NTPServerWithLeadingZeros},
				PrivateOptions: []v1.DHCPPrivateOptions{{Option: 240, Value: "private.options.kubevirt.io"}},
			}

			dhcpVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(dhcpVMI)).Create(context.Background(), dhcpVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			dhcpVMI = libwait.WaitUntilVMIReady(dhcpVMI, console.LoginToFedora)

			err = console.SafeExpectBatch(dhcpVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "dhclient -1 -r -d eth0\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "dhclient -1 -sf /usr/bin/env --request-options subnet-mask,broadcast-address,time-offset,routers,domain-search,domain-name,domain-name-servers,host-name,nis-domain,nis-servers,ntp-servers,interface-mtu,tftp-server-name,bootfile-name eth0 | tee /dhcp-env\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "grep -q 'new_tftp_server_name=tftp.kubevirt.io' /dhcp-env; echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "grep -q 'new_bootfile_name=config' /dhcp-env; echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "grep -q 'new_ntp_servers=127.0.0.1 127.0.0.2 127.0.0.3' /dhcp-env; echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "grep -q 'new_unknown_240=private.options.kubevirt.io' /dhcp-env; echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
			}, 15)

			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("VirtualMachineInstance with custom dns", func() {
		It("[test_id:1779]should have custom resolv.conf", func() {
			libnet.SkipWhenClusterNotSupportIpv4()
			dnsVMI := libvmifact.NewCirros()

			dnsVMI.Spec.DNSPolicy = "None"

			// This IPv4 address tests backwards compatibility of the "DNSConfig.Nameservers" field.
			// The leading zero is intentional.
			// For more details please see: https://github.com/kubevirt/kubevirt/issues/6498
			const DNSNameserverWithLeadingZeros = "01.1.1.1"
			dnsVMI.Spec.DNSConfig = &k8sv1.PodDNSConfig{
				Nameservers: []string{"8.8.8.8", "4.2.2.1", DNSNameserverWithLeadingZeros},
				Searches:    []string{"example.com"},
			}
			dnsVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(dnsVMI)).Create(context.Background(), dnsVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			dnsVMI = libwait.WaitUntilVMIReady(dnsVMI, console.LoginToCirros)
			const catResolvConf = "cat /etc/resolv.conf\n"
			err = console.SafeExpectBatch(dnsVMI, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: catResolvConf},
				&expect.BExp{R: "search example.com"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: catResolvConf},
				&expect.BExp{R: "nameserver 8.8.8.8"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: catResolvConf},
				&expect.BExp{R: "nameserver 4.2.2.1"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "cat /etc/resolv.conf\n"},
				&expect.BExp{R: "nameserver 1.1.1.1"},
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: ""},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("VirtualMachineInstance with masquerade binding mechanism", func() {
		const LibvirtDirectMigrationPort = 49152

		masqueradeVMI := func(ports []v1.Port, ipv4NetworkCIDR string) *v1.VirtualMachineInstance {
			net := v1.DefaultPodNetwork()
			if ipv4NetworkCIDR != "" {
				net.NetworkSource.Pod.VMNetworkCIDR = ipv4NetworkCIDR
			}
			return libvmifact.NewCirros(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding(ports...)),
				libvmi.WithNetwork(net),
			)
		}

		fedoraMasqueradeVMI := func(ports []v1.Port, ipv6NetworkCIDR string) (*v1.VirtualMachineInstance, error) {
			if ipv6NetworkCIDR == "" {
				ipv6NetworkCIDR = cloudinit.DefaultIPv6CIDR
			}

			isClusterDualStack, err := cluster.DualStack()
			Expect(err).NotTo(HaveOccurred())
			var networkData string
			networkDataParams := []cloudinit.NetworkDataInterfaceOption{
				cloudinit.WithAddresses(ipv6NetworkCIDR),
				cloudinit.WithGateway6(gatewayIPFromCIDR(ipv6NetworkCIDR)),
				cloudinit.WithNameserverFromCluster(),
			}
			if isClusterDualStack {
				networkDataParams = append(networkDataParams, cloudinit.WithDHCP4Enabled())
			}
			networkData, err = cloudinit.NewNetworkData(
				cloudinit.WithEthernet("eth0", networkDataParams...),
			)
			if err != nil {
				return nil, err
			}

			net := v1.DefaultPodNetwork()
			net.Pod.VMIPv6NetworkCIDR = ipv6NetworkCIDR
			vmi := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding(ports...)),
				libvmi.WithNetwork(net),
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
			)

			return vmi, nil
		}

		portsUsedByLiveMigration := func() []v1.Port {
			const LibvirtBlockMigrationPort = 49153
			return []v1.Port{
				{Port: LibvirtDirectMigrationPort},
				{Port: LibvirtBlockMigrationPort},
			}
		}

		Context("[test_id:1780][label:masquerade_binding_connectivity]should allow regular network connection", decorators.Conformance, func() {
			// This CIDR tests backwards compatibility of the "vmNetworkCIDR" field.
			// The leading zero is intentional.
			// For more details please see: https://github.com/kubevirt/kubevirt/issues/6498
			const cidrWithLeadingZeros = "10.10.010.0/24"

			verifyClientServerConnectivity := func(clientVMI, serverVMI *v1.VirtualMachineInstance, tcpPort int, ipFamily k8sv1.IPFamily) error {
				serverIP := libnet.GetVmiPrimaryIPByFamily(serverVMI, ipFamily)
				err := libnet.PingFromVMConsole(clientVMI, serverIP)
				if err != nil {
					return err
				}

				By("Connecting from the client vm")
				err = console.SafeExpectBatch(clientVMI, createExpectConnectToServer(serverIP, tcpPort, true), 30)
				if err != nil {
					return err
				}

				By("Rejecting the connection from the client to unregistered port")
				err = console.SafeExpectBatch(clientVMI, createExpectConnectToServer(serverIP, tcpPort+1, false), 30)
				if err != nil {
					return err
				}

				return nil
			}

			DescribeTable("ipv4", func(ports []v1.Port, tcpPort int, networkCIDR string) {
				libnet.SkipWhenClusterNotSupportIpv4()

				clientVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(
					context.Background(), masqueradeVMI([]v1.Port{}, networkCIDR), metav1.CreateOptions{},
				)
				Expect(err).ToNot(HaveOccurred())
				clientVMI = libwait.WaitUntilVMIReady(clientVMI, console.LoginToCirros)

				serverVMI := masqueradeVMI(ports, networkCIDR)
				serverVMI.Labels = map[string]string{"expose": "server"}
				serverVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), serverVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				serverVMI = libwait.WaitUntilVMIReady(serverVMI, console.LoginToCirros)
				Expect(serverVMI.Status.Interfaces).To(HaveLen(1))
				Expect(serverVMI.Status.Interfaces[0].IPs).NotTo(BeEmpty())

				By("starting a tcp server")
				vmnetserver.StartTCPServer(serverVMI, tcpPort, console.LoginToCirros)

				if networkCIDR == "" {
					networkCIDR = api.DefaultVMCIDR
				}

				By("Checking ping (IPv4) to gateway")
				ipAddr := gatewayIPFromCIDR(networkCIDR)
				Expect(libnet.PingFromVMConsole(serverVMI, ipAddr)).To(Succeed())

				Expect(verifyClientServerConnectivity(clientVMI, serverVMI, tcpPort, k8sv1.IPv4Protocol)).To(Succeed())
			},
				Entry("with a specific port number [IPv4]", []v1.Port{{Name: "http", Port: 8080}}, 8080, ""),
				Entry("with a specific port used by live migration", portsUsedByLiveMigration(), LibvirtDirectMigrationPort, ""),
				Entry("without a specific port number [IPv4]", []v1.Port{}, 8080, ""),
				Entry("with custom CIDR [IPv4] containing leading zeros", []v1.Port{}, 8080, cidrWithLeadingZeros),
			)

			It("should be able to reach the outside world [IPv4]", Label("RequiresOutsideConnectivity"), func() {
				libnet.SkipWhenClusterNotSupportIpv4()
				ipv4Address := "8.8.8.8"
				if flags.IPV4ConnectivityCheckAddress != "" {
					ipv4Address = flags.IPV4ConnectivityCheckAddress
				}
				dns := "google.com"
				if flags.ConnectivityCheckDNS != "" {
					dns = flags.ConnectivityCheckDNS
				}

				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(
					context.Background(), masqueradeVMI([]v1.Port{}, ""), metav1.CreateOptions{},
				)
				Expect(err).ToNot(HaveOccurred())
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				By("Checking ping (IPv4)")
				Expect(libnet.PingFromVMConsole(vmi, ipv4Address, "-c 5", "-w 15")).To(Succeed())
				Expect(libnet.PingFromVMConsole(vmi, dns, "-c 5", "-w 15")).To(Succeed())
			})

			DescribeTable("IPv6", func(ports []v1.Port, tcpPort int, networkCIDR string) {
				libnet.SkipWhenClusterNotSupportIpv6()

				clientVMI, err := fedoraMasqueradeVMI([]v1.Port{}, networkCIDR)
				Expect(err).ToNot(HaveOccurred())
				clientVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				clientVMI = libwait.WaitUntilVMIReady(clientVMI, console.LoginToFedora)

				serverVMI, err := fedoraMasqueradeVMI(ports, networkCIDR)
				Expect(err).ToNot(HaveOccurred())

				serverVMI.Labels = map[string]string{"expose": "server"}
				serverVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), serverVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				serverVMI = libwait.WaitUntilVMIReady(serverVMI, console.LoginToFedora)

				Expect(serverVMI.Status.Interfaces).To(HaveLen(1))
				Expect(serverVMI.Status.Interfaces[0].IPs).NotTo(BeEmpty())

				By("starting a http server")
				vmnetserver.StartPythonHTTPServer(serverVMI, tcpPort)

				Expect(verifyClientServerConnectivity(clientVMI, serverVMI, tcpPort, k8sv1.IPv6Protocol)).To(Succeed())
			},
				Entry("with a specific port number [IPv6]", []v1.Port{{Name: "http", Port: 8080}}, 8080, ""),
				Entry("with a specific port used by live migration", portsUsedByLiveMigration(), LibvirtDirectMigrationPort, ""),
				Entry("without a specific port number [IPv6]", []v1.Port{}, 8080, ""),
				Entry("with custom CIDR [IPv6]", []v1.Port{}, 8080, "fd10:10:10::2/120"),
			)

			It("should be able to reach the outside world", Label("RequiresOutsideConnectivity", "IPv6"), func() {
				libnet.SkipWhenClusterNotSupportIpv6()
				// Cluster nodes subnet (docker network gateway)
				// Docker network subnet cidr definition:
				// https://github.com/kubevirt/project-infra/blob/master/github/ci/shared-deployments/files/docker-daemon-mirror.conf#L5
				ipv6Address := "2001:db8:1::1"
				if flags.IPV6ConnectivityCheckAddress != "" {
					ipv6Address = flags.IPV6ConnectivityCheckAddress
				}

				vmi, err := fedoraMasqueradeVMI([]v1.Port{}, "")
				Expect(err).ToNot(HaveOccurred())
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

				By("Checking ping (IPv6) from vmi to cluster nodes gateway")
				Expect(libnet.PingFromVMConsole(vmi, ipv6Address)).To(Succeed())
			})
		})

		When("performing migration", decorators.RequiresTwoSchedulableNodes, func() {
			var vmi *v1.VirtualMachineInstance

			ping := func(ipAddr string) error {
				return libnet.PingFromVMConsole(vmi, ipAddr, "-c 1", "-w 2")
			}

			DescribeTable("preserves connectivity - IPv4", decorators.Conformance, func(ports []v1.Port) {
				libnet.SkipWhenClusterNotSupportIpv4()

				var err error

				By("Create client pod")
				clientPod := libpod.RenderPod("test-conn", []string{"/bin/sh", "-c", "sleep 360"}, []string{})
				clientPod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Create VMI")
				vmi = masqueradeVMI(ports, "")

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				Eventually(matcher.ThisPod(clientPod)).WithTimeout(120 * time.Second).WithPolling(time.Second).Should(matcher.BeRunning())
				clientPod, err = virtClient.CoreV1().Pods(clientPod.Namespace).Get(context.Background(), clientPod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Check connectivity")
				podIP := libnet.GetPodIPByFamily(clientPod, k8sv1.IPv4Protocol)
				Expect(ping(podIP)).To(Succeed())

				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.Phase).To(Equal(v1.Running))

				Expect(ping(podIP)).To(Succeed())

				By("Initiating DHCP client request after migration")

				Expect(console.RunCommand(vmi, "sudo cirros-dhcpc down eth0\n", time.Second*time.Duration(15))).To(Succeed(), "failed to release dhcp client")
				Expect(console.RunCommand(vmi, "sudo cirros-dhcpc up eth0\n", time.Second*time.Duration(15))).To(Succeed(), "failed to run dhcp client")

				Expect(ping(podIP)).To(Succeed())
			},
				Entry("without a specific port number", []v1.Port{}),
				Entry("with explicit ports used by live migration", portsUsedByLiveMigration()),
			)

			It("should preserve connectivity - IPv6", decorators.Conformance, func() {
				libnet.SkipWhenClusterNotSupportIpv6()

				var err error

				By("Create client pod")
				clientPod := libpod.RenderPod("test-conn", []string{"/bin/sh", "-c", "sleep 360"}, []string{})
				clientPod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(nil)).Create(context.Background(), clientPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Create VMI")
				vmi, err = fedoraMasqueradeVMI([]v1.Port{}, "")
				Expect(err).ToNot(HaveOccurred())

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

				Eventually(matcher.ThisPod(clientPod)).WithTimeout(120 * time.Second).WithPolling(time.Second).Should(matcher.BeRunning())
				clientPod, err = virtClient.CoreV1().Pods(clientPod.Namespace).Get(context.Background(), clientPod.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Check connectivity")
				podIP := libnet.GetPodIPByFamily(clientPod, k8sv1.IPv6Protocol)
				Expect(ping(podIP)).To(Succeed())

				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.Phase).To(Equal(v1.Running))

				Expect(ping(podIP)).To(Succeed())
			})
		})

		Context("MTU verification", func() {
			var vmi *v1.VirtualMachineInstance
			var anotherVmi *v1.VirtualMachineInstance

			getMtu := func(pod *k8sv1.Pod, ifaceName string) int {
				output, err := exec.ExecuteCommandOnPod(
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

			configureIpv6 := func(vmi *v1.VirtualMachineInstance) error {
				networkCIDR := api.DefaultVMIpv6CIDR

				err := console.RunCommand(vmi, "dhclient -6 eth0", 30*time.Second)
				if err != nil {
					return err
				}
				err = console.RunCommand(vmi, "ip -6 route add "+networkCIDR+" dev eth0", 5*time.Second)
				if err != nil {
					return err
				}
				gateway := gatewayIPFromCIDR(networkCIDR)
				err = console.RunCommand(vmi, "ip -6 route add default via "+gateway, 5*time.Second)
				if err != nil {
					return err
				}
				return nil
			}

			BeforeEach(func() {
				var err error

				By("Create masquerade VMI")
				networkData, err := cloudinit.NewNetworkData(
					cloudinit.WithEthernet("eth0",
						cloudinit.WithDHCP4Enabled(),
						cloudinit.WithDHCP6Enabled(),
						cloudinit.WithAddresses(""), // This is a workaround o make fedora client to configure local IPv6
					),
				)
				Expect(err).ToNot(HaveOccurred())

				vmi = libvmifact.NewFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData)),
				)

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Create another VMI")
				anotherVmi = libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())
				anotherVmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), anotherVmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Wait for VMIs to be ready")
				anotherVmi = libwait.WaitUntilVMIReady(anotherVmi, console.LoginToAlpine)

				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
			})

			DescribeTable("should have the correct MTU", func(ipFamily k8sv1.IPFamily) {
				libnet.SkipWhenClusterNotSupportIPFamily(ipFamily)

				if ipFamily == k8sv1.IPv6Protocol {
					// IPv6 address is configured via DHCP6 in this test and not via cloud-init
					Expect(configureIpv6(vmi)).To(Succeed(), "failed to configure ipv6  on server vmi")
				}

				By("checking k6t-eth0 MTU inside the pod")
				vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())

				bridgeMtu := getMtu(vmiPod, "k6t-eth0")
				primaryIfaceMtu := getMtu(vmiPod, "eth0")

				Expect(bridgeMtu).To(Equal(primaryIfaceMtu), "k6t-eth0 bridge mtu should equal eth0 interface mtu")

				By("checking the tap device - tap0 - MTU inside the pod")
				tapDeviceMTU := getMtu(vmiPod, "tap0")
				Expect(tapDeviceMTU).To(Equal(primaryIfaceMtu), "tap0 mtu should equal eth0 interface mtu")

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
				addr := libnet.GetVmiPrimaryIPByFamily(anotherVmi, ipFamily)
				Expect(libnet.PingFromVMConsole(vmi, addr, "-c 1", "-w 5", fmt.Sprintf("-s %d", payloadSize), "-M do")).To(Succeed())

				By("checking the VirtualMachineInstance cannot send bigger than MTU sized frames to another VirtualMachineInstance")
				Expect(libnet.PingFromVMConsole(vmi, addr, "-c 1", "-w 5", fmt.Sprintf("-s %d", payloadSize+1), "-M do")).ToNot(Succeed())
			},
				Entry("IPv4", k8sv1.IPv4Protocol),
				Entry("IPv6", k8sv1.IPv6Protocol),
			)
		})
	})

	Context("VirtualMachineInstance with TX offload disabled", func() {
		It("[test_id:1781]should have tx checksumming disabled on interface serving dhcp", func() {
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(
				context.Background(),
				libvmifact.NewAlpine(libvmi.WithMemoryRequest("1024M")),
				metav1.CreateOptions{},
			)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)
			output := libpod.RunCommandOnVmiPod(
				vmi,
				[]string{"/bin/bash", "-c", "/usr/sbin/ethtool -k k6t-eth0|grep tx-checksumming|awk '{ printf $2 }'"},
			)
			ExpectWithOffset(1, strings.TrimSpace(output)).To(Equal("off"))
		})
	})
}))

func createExpectConnectToServer(serverIP string, tcpPort int, expectSuccess bool) []expect.Batcher {
	expectResult := console.ShellFail
	if expectSuccess {
		expectResult = console.ShellSuccess
	}

	var clientCommand string

	if netutils.IsIPv6String(serverIP) {
		clientCommand = fmt.Sprintf("curl %s\n", net.JoinHostPort(serverIP, strconv.Itoa(tcpPort)))
	} else {
		clientCommand = fmt.Sprintf("echo test | nc %s %d -i 1 -w 1 1> /dev/null\n", serverIP, tcpPort)
	}
	return []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: ""},
		&expect.BSnd{S: clientCommand},
		&expect.BExp{R: ""},
		&expect.BSnd{S: console.EchoLastReturnValue},
		&expect.BExp{R: expectResult},
	}
}

// gatewayIpFromCIDR returns the first address of a network.
func gatewayIPFromCIDR(cidr string) string {
	// ParseCIDRSloppy is intentionally used to test backwards compatibility of the "vmNetworkCIDR" field with leading zeros.
	// For more details please see: https://github.com/kubevirt/kubevirt/issues/6498
	ip, ipnet, _ := netutils.ParseCIDRSloppy(cidr)
	ip = ip.Mask(ipnet.Mask)
	oct := len(ip) - 1
	ip[oct]++
	return ip.String()
}

func vmiHasCustomMacAddress(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.Devices.Interfaces != nil &&
		vmi.Spec.Domain.Devices.Interfaces[0].MacAddress != ""
}

func runVMI(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
	virtClient := kubevirt.Client()

	var err error
	vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)
	return vmi
}

func vmiWithCustomMacAddress(mac string) *v1.VirtualMachineInstance {
	return libvmifact.NewCirros(
		libvmi.WithInterface(*libvmi.InterfaceWithMac(v1.DefaultBridgeNetworkInterface(), mac)),
		libvmi.WithNetwork(v1.DefaultPodNetwork()))
}

func vmiWithMultiQueue() *v1.VirtualMachineInstance {
	return libvmifact.NewCirros(
		libvmi.WithNetworkInterfaceMultiQueue(true),
		libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()))
}

func newNodeAffinity(selector k8sv1.NodeSelectorOperator, nodeName string) *k8sv1.NodeAffinity {
	req := k8sv1.NodeSelectorRequirement{
		Key:      k8sv1.LabelHostname,
		Operator: selector,
		Values:   []string{nodeName},
	}
	term := []k8sv1.NodeSelectorTerm{
		{
			MatchExpressions: []k8sv1.NodeSelectorRequirement{req},
		},
	}
	return &k8sv1.NodeAffinity{RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
		NodeSelectorTerms: term,
	}}
}
