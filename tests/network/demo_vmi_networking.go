package network

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libwait"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/testsuite"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"

	"kubevirt.io/kubevirt/tests/libnet"

	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
)

var _ = SIGDescribe("[rfe_id:694][crit:medium][vendor:cnv-qe@redhat.com][level:component]Networking", decorators.Networking, func() {

	Describe("Multiple virtual machines connectivity using bridge binding interface", func() {
		var inboundVMI *v1.VirtualMachineInstance
		var outboundVMI *v1.VirtualMachineInstance
		var inboundVMIWithPodNetworkSet *v1.VirtualMachineInstance
		var inboundVMIWithCustomMacAddress *v1.VirtualMachineInstance

		BeforeEach(func() {
			libnet.SkipWhenClusterNotSupportIpv4()
		})
		Context("with a test outbound VMI", Ordered, decorators.SkipGlobalCleanup, func() {
			BeforeAll(func() {
				DeferCleanup(func() {
					testsuite.CleanNamespaces()
					libnode.CleanNodes()
					config.UpdateKubeVirtConfigValueAndWait(testsuite.KubeVirtDefaultConfig)
					testsuite.EnsureKubevirtReady()
				})
				var err error
				virtClient := kubevirt.Client()

				inboundVMI = libvmifact.NewCirros()
				inboundVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).
					Create(context.Background(), inboundVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				inboundVMIWithPodNetworkSet = vmiWithPodNetworkSet()
				inboundVMIWithPodNetworkSet, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).
					Create(context.Background(), inboundVMIWithPodNetworkSet, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				inboundVMIWithCustomMacAddress = vmiWithCustomMacAddress("de:ad:00:00:be:af")
				inboundVMIWithCustomMacAddress, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).
					Create(context.Background(), inboundVMIWithCustomMacAddress, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				outboundVMI = libvmifact.NewCirros()
				outboundVMI = runVMI(outboundVMI)
			})

			DescribeTable("should be able to reach", func(vmiRef **v1.VirtualMachineInstance) {
				vmi := libwait.WaitUntilVMIReady(*vmiRef, console.LoginToCirros)
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
					&expect.BExp{R: console.PromptExpression},
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
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: cmdCheck},
					&expect.BExp{R: console.PromptExpression},
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
				Entry("[test_id:1540]the Inbound VirtualMachineInstance with pod network connectivity explicitly set", &inboundVMIWithPodNetworkSet),
				Entry("[test_id:1541]the Inbound VirtualMachineInstance with custom MAC address", &inboundVMIWithCustomMacAddress),
			)
		})

	})

})
