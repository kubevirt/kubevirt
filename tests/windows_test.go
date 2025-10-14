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
 * Copyright The KubeVirt Authors.
 *
 */

package tests_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/kubevirt/tests/libnode"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/network/dns"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
)

const (
	windowsVMIUser     = "Administrator"
	windowsVMIPassword = "Heslo123"
)

const (
	winrmCli    = "winrmcli"
	winrmCliCmd = "winrm-cli"
)

var _ = Describe("[sig-compute]Windows VirtualMachineInstance", Serial, decorators.Windows, decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		const OSWindows = "windows"
		virtClient = kubevirt.Client()
		checks.RecycleImageOrFail(virtClient, libvmifact.WindowsPVCName)
		libstorage.CreatePVC(OSWindows, testsuite.GetTestNamespace(nil), "30Gi", libstorage.Config.StorageClassWindows, true)
	})

	Context("with winrm connection", func() {
		var winrmcliPod *k8sv1.Pod
		var windowsVMI *v1.VirtualMachineInstance

		BeforeEach(func() {
			By("Creating winrm-cli pod for the future use")
			var err error
			winrmcliPod, err = virtClient.CoreV1().Pods(testsuite.NamespaceTestDefault).Create(context.Background(), winRMCliPod(), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		Context("[ref_id:139]VMI is created", func() {

			BeforeEach(func() {
				By("Starting the windows VirtualMachineInstance")
				var err error
				windowsVMI, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), libvmifact.NewWindows(withDefaultE1000Networking()), metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(windowsVMI)
			})

			It("[test_id:240]should have correct UUID", func() {
				command := append(winrmLoginCommand(windowsVMI), "wmic csproduct get \"UUID\"")
				By(fmt.Sprintf("Running \"%s\" command via winrm-cli", command))
				var output string
				Eventually(func() error {
					var err error
					output, err = exec.ExecuteCommandOnPod(
						winrmcliPod,
						winrmcliPod.Spec.Containers[0].Name,
						command,
					)
					return err
				}, time.Minute*5, time.Second*15).ShouldNot(HaveOccurred())
				By("Checking that the Windows VirtualMachineInstance has expected UUID")
				Expect(output).Should(ContainSubstring(strings.ToUpper(libvmifact.WindowsFirmware)))
			})

			It("[test_id:3159]should have default masquerade IP", func() {
				command := append(winrmLoginCommand(windowsVMI), "ipconfig /all")
				By(fmt.Sprintf("Running \"%s\" command via winrm-cli", command))
				var output string
				Eventually(func() error {
					var err error
					output, err = exec.ExecuteCommandOnPod(
						winrmcliPod,
						winrmcliPod.Spec.Containers[0].Name,
						command,
					)
					return err
				}, time.Minute*5, time.Second*15).ShouldNot(HaveOccurred())

				By("Checking that the Windows VirtualMachineInstance has expected IP address")
				Expect(output).Should(ContainSubstring("10.0.2.2"))
			})

			It("[test_id:3160]should have the domain set properly", func() {
				searchDomain := getPodSearchDomain(windowsVMI)
				Expect(searchDomain).To(HavePrefix(windowsVMI.Namespace), "should contain a searchdomain with the namespace of the VMI")

				runCommandAndExpectOutput(
					winrmcliPod,
					winrmLoginCommand(windowsVMI),
					"wmic nicconfig get dnsdomain",
					`DNSDomain[\n\r\t ]+`+searchDomain+`[\n\r\t ]+`)
			})
		})

		Context("VMI with subdomain is created", func() {
			BeforeEach(func() {
				windowsVMI = libvmifact.NewWindows(withDefaultE1000Networking(), libvmi.WithSubdomain("subdomain"))

				By("Starting the windows VirtualMachineInstance with subdomain")
				var err error
				windowsVMI, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), windowsVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(windowsVMI)
			})

			It("should have the domain set properly with subdomain", func() {
				searchDomain := getPodSearchDomain(windowsVMI)
				Expect(searchDomain).To(HavePrefix(windowsVMI.Namespace), "should contain a searchdomain with the namespace of the VMI")

				expectedSearchDomain := windowsVMI.Spec.Subdomain + "." + searchDomain
				runCommandAndExpectOutput(
					winrmcliPod,
					winrmLoginCommand(windowsVMI),
					"wmic nicconfig get dnsdomain",
					`DNSDomain[\n\r\t ]+`+expectedSearchDomain+`[\n\r\t ]+`)
			})
		})

		Context("with bridge binding", func() {

			BeforeEach(func() {
				By("Starting Windows VirtualMachineInstance with bridge binding")
				windowsVMI = libvmifact.NewWindows(libvmi.WithNetwork(v1.DefaultPodNetwork()), libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(v1.DefaultPodNetwork().Name)))
				var err error
				windowsVMI, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), windowsVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				windowsVMI = libwait.WaitForSuccessfulVMIStart(windowsVMI,
					libwait.WithTimeout(420),
				)

			})

			It("should be recognized by other pods in cluster", func() {

				By("Pinging virt-handler Pod from Windows VMI")
				winVmiPod, err := libpod.GetPodByVirtualMachineInstance(windowsVMI, windowsVMI.Namespace)
				Expect(err).NotTo(HaveOccurred())
				nodeName := winVmiPod.Spec.NodeName

				virtHandlerPod, err := libnode.GetVirtHandlerPod(virtClient, nodeName)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to get virt-handler pod on node %s", nodeName))

				virtHandlerPodIP := libnet.GetPodIPByFamily(virtHandlerPod, k8sv1.IPv4Protocol)

				command := append(winrmLoginCommand(windowsVMI), fmt.Sprintf("ping %s", virtHandlerPodIP))

				By(fmt.Sprintf("Running \"%s\" command via winrm-cli", command))
				Eventually(func() error {
					_, err = exec.ExecuteCommandOnPod(
						winrmcliPod,
						winrmcliPod.Spec.Containers[0].Name,
						command,
					)
					return err
				}, time.Minute*1, time.Second*15).Should(Succeed())
			})
		})
	})
})

func winrmLoginCommand(windowsVMI *v1.VirtualMachineInstance) []string {
	var err error
	var vmiIp string
	EventuallyWithOffset(1, func() error {
		windowsVMI, err = kubevirt.Client().VirtualMachineInstance(windowsVMI.Namespace).Get(context.Background(), windowsVMI.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if windowsVMI.Status.Interfaces[0].IP != "" && vmispec.ContainsInfoSource(windowsVMI.Status.Interfaces[0].InfoSource, "guest-agent") {
			vmiIp = windowsVMI.Status.Interfaces[0].IP
			return nil
		}
		return fmt.Errorf("waiting from ip from guest-agent")
	}, 30*time.Second, 1*time.Second).Should(Succeed())
	cli := []string{
		winrmCliCmd,
		"-hostname",
		vmiIp,
		"-username",
		windowsVMIUser,
		"-password",
		windowsVMIPassword,
	}

	return cli
}

func getPodSearchDomain(windowsVMI *v1.VirtualMachineInstance) string {
	By("fetching /etc/resolv.conf from the VMI Pod")
	resolvConf := libpod.RunCommandOnVmiPod(windowsVMI, []string{"cat", "/etc/resolv.conf"})

	By("extracting the search domain of the VMI")
	searchDomains, err := dns.ParseSearchDomains(resolvConf)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	searchDomain := ""
	for _, s := range searchDomains {
		if len(searchDomain) < len(s) {
			searchDomain = s
		}
	}

	return searchDomain
}

func runCommandAndExpectOutput(winrmcliPod *k8sv1.Pod, cli []string, command, expectedOutputRegex string) {
	cliCmd := append(cli, command)
	By(fmt.Sprintf("Running \"%s\" command via winrm-cli", cliCmd))
	By("first making sure that we can execute VMI commands")
	EventuallyWithOffset(1, func() error {
		_, err := exec.ExecuteCommandOnPod(
			winrmcliPod,
			winrmcliPod.Spec.Containers[0].Name,
			cliCmd,
		)
		return err
	}, time.Minute*5, time.Second*15).ShouldNot(HaveOccurred())

	By("repeatedly trying to get the search domain, since it may take some time until the domain is set")
	EventuallyWithOffset(1, func() string {
		output, err := exec.ExecuteCommandOnPod(
			winrmcliPod,
			winrmcliPod.Spec.Containers[0].Name,
			cliCmd,
		)
		Expect(err).ToNot(HaveOccurred())
		return output
	}, time.Minute*1, time.Second*10).Should(MatchRegexp(expectedOutputRegex))
}

func isTSCFrequencyExposed(virtClient kubecli.KubevirtClient) bool {
	nodeList, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	for _, node := range nodeList.Items {
		if _, isExposed := node.Labels[topology.TSCFrequencyLabel]; isExposed {
			return true
		}
	}

	return false
}

func removeTSCFrequencyFromNode(node k8sv1.Node) {
	for _, baseLabelToRemove := range []string{topology.TSCFrequencyLabel, topology.TSCFrequencySchedulingLabel} {
		for key := range node.Labels {
			if strings.HasPrefix(key, baseLabelToRemove) {
				libnode.RemoveLabelFromNode(node.Name, key)
			}
		}
	}
}

func withDefaultE1000Networking() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		libvmi.WithNetwork(v1.DefaultPodNetwork())(vmi)
		libvmi.WithInterface(e1000DefaultInterface())(vmi)
	}
}

func e1000DefaultInterface() v1.Interface {
	return v1.Interface{
		Name: "default",
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			Masquerade: &v1.InterfaceMasquerade{},
		},
		Model: "e1000",
	}
}
