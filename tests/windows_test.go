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
 * Copyright 2018 Red Hat, Inc.
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

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/network/dns"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	windowsVMIUser     = "Administrator"
	windowsVMIPassword = "Heslo123"
)

const (
	winrmCli    = "winrmcli"
	winrmCliCmd = "winrm-cli"
)

var _ = Describe("[Serial][sig-compute]Windows VirtualMachineInstance", Serial, decorators.Windows, decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient
	var windowsVMI *v1.VirtualMachineInstance

	BeforeEach(func() {
		const OSWindows = "windows"
		virtClient = kubevirt.Client()
		checks.SkipIfMissingRequiredImage(virtClient, libvmi.WindowsPVCName)
		libstorage.CreatePVC(OSWindows, testsuite.GetTestNamespace(nil), "30Gi", libstorage.Config.StorageClassWindows, true)
		windowsVMI = libvmi.NewWindows()
		tests.AddExplicitPodNetworkInterface(windowsVMI)
		windowsVMI.Spec.Domain.Devices.Interfaces[0].Model = "e1000"
	})

	Context("with winrm connection", func() {
		var winrmcliPod *k8sv1.Pod
		var cli []string

		BeforeEach(func() {
			By("Creating winrm-cli pod for the future use")
			winrmcliPod = winRMCliPod()

			var err error
			winrmcliPod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), winrmcliPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		Context("[ref_id:139]VMI is created", func() {

			BeforeEach(func() {
				By("Starting the windows VirtualMachineInstance")
				var err error
				windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), windowsVMI)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(windowsVMI)

				cli = winrnLoginCommand(virtClient, windowsVMI)
			})

			It("[test_id:240]should have correct UUID", func() {
				command := append(cli, "wmic csproduct get \"UUID\"")
				By(fmt.Sprintf("Running \"%s\" command via winrm-cli", command))
				var output string
				Eventually(func() error {
					var err error
					output, err = exec.ExecuteCommandOnPod(
						virtClient,
						winrmcliPod,
						winrmcliPod.Spec.Containers[0].Name,
						command,
					)
					return err
				}, time.Minute*5, time.Second*15).ShouldNot(HaveOccurred())
				By("Checking that the Windows VirtualMachineInstance has expected UUID")
				Expect(output).Should(ContainSubstring(strings.ToUpper(libvmi.WindowsFirmware)))
			})

			It("[test_id:3159]should have default masquerade IP", func() {
				command := append(cli, "ipconfig /all")
				By(fmt.Sprintf("Running \"%s\" command via winrm-cli", command))
				var output string
				Eventually(func() error {
					var err error
					output, err = exec.ExecuteCommandOnPod(
						virtClient,
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

				runCommandAndExpectOutput(virtClient,
					winrmcliPod,
					cli,
					"wmic nicconfig get dnsdomain",
					`DNSDomain[\n\r\t ]+`+searchDomain+`[\n\r\t ]+`)
			})
		})

		Context("VMI with subdomain is created", func() {
			BeforeEach(func() {
				windowsVMI.Spec.Subdomain = "subdomain"

				By("Starting the windows VirtualMachineInstance with subdomain")
				var err error
				windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), windowsVMI)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(windowsVMI)

				cli = winrnLoginCommand(virtClient, windowsVMI)
			})

			It("should have the domain set properly with subdomain", func() {
				searchDomain := getPodSearchDomain(windowsVMI)
				Expect(searchDomain).To(HavePrefix(windowsVMI.Namespace), "should contain a searchdomain with the namespace of the VMI")

				expectedSearchDomain := windowsVMI.Spec.Subdomain + "." + searchDomain
				runCommandAndExpectOutput(virtClient,
					winrmcliPod,
					cli,
					"wmic nicconfig get dnsdomain",
					`DNSDomain[\n\r\t ]+`+expectedSearchDomain+`[\n\r\t ]+`)
			})
		})

		Context("with bridge binding", func() {
			BeforeEach(func() {
				By("Starting Windows VirtualMachineInstance with bridge binding")
				windowsVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{libvmi.InterfaceDeviceWithBridgeBinding(libvmi.DefaultInterfaceName)}
				var err error
				windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), windowsVMI)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(windowsVMI,
					libwait.WithTimeout(420),
				)

				cli = winrnLoginCommand(virtClient, windowsVMI)
			})

			It("should be recognized by other pods in cluster", func() {

				By("Pinging virt-handler Pod from Windows VMI")

				var err error
				windowsVMI, err = virtClient.VirtualMachineInstance(windowsVMI.Namespace).Get(context.Background(), windowsVMI.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				getVirtHandlerPod := func() (*k8sv1.Pod, error) {
					winVmiPod := tests.GetRunningPodByVirtualMachineInstance(windowsVMI, windowsVMI.Namespace)
					nodeName := winVmiPod.Spec.NodeName

					pod, err := libnode.GetVirtHandlerPod(virtClient, nodeName)
					if err != nil {
						return nil, fmt.Errorf("failed to get virt-handler pod on node %s: %v", nodeName, err)
					}
					return pod, nil
				}

				virtHandlerPod, err := getVirtHandlerPod()
				Expect(err).ToNot(HaveOccurred())

				virtHandlerPodIP := libnet.GetPodIPByFamily(virtHandlerPod, k8sv1.IPv4Protocol)

				command := append(cli, fmt.Sprintf("ping %s", virtHandlerPodIP))

				By(fmt.Sprintf("Running \"%s\" command via winrm-cli", command))
				Eventually(func() error {
					_, err = exec.ExecuteCommandOnPod(
						virtClient,
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

func winrnLoginCommand(virtClient kubecli.KubevirtClient, windowsVMI *v1.VirtualMachineInstance) []string {
	var err error
	windowsVMI, err = virtClient.VirtualMachineInstance(windowsVMI.Namespace).Get(context.Background(), windowsVMI.Name, &metav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	vmiIp := windowsVMI.Status.Interfaces[0].IP
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
	resolvConf := tests.RunCommandOnVmiPod(windowsVMI, []string{"cat", "/etc/resolv.conf"})

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

func runCommandAndExpectOutput(virtClient kubecli.KubevirtClient, winrmcliPod *k8sv1.Pod, cli []string, command, expectedOutputRegex string) {
	cliCmd := append(cli, command)
	By(fmt.Sprintf("Running \"%s\" command via winrm-cli", cliCmd))
	By("first making sure that we can execute VMI commands")
	EventuallyWithOffset(1, func() error {
		_, err := exec.ExecuteCommandOnPod(
			virtClient,
			winrmcliPod,
			winrmcliPod.Spec.Containers[0].Name,
			cliCmd,
		)
		return err
	}, time.Minute*5, time.Second*15).ShouldNot(HaveOccurred())

	By("repeatedly trying to get the search domain, since it may take some time until the domain is set")
	EventuallyWithOffset(1, func() string {
		output, err := exec.ExecuteCommandOnPod(
			virtClient,
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
		for key, _ := range node.Labels {
			if strings.HasPrefix(key, baseLabelToRemove) {
				libnode.RemoveLabelFromNode(node.Name, key)
			}
		}
	}
}
