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
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/network/dns"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
)

const (
	windowsDisk        = "windows-disk"
	windowsFirmware    = "5d307ca9-b3ef-428c-8861-06e72d69f223"
	windowsVMIUser     = "Administrator"
	windowsVMIPassword = "Heslo123"
)

const (
	winrmCli    = "winrmcli"
	winrmCliCmd = "winrm-cli"
)

var getWindowsVMISpec = func() v1.VirtualMachineInstanceSpec {
	gracePeriod := int64(0)
	spinlocks := uint32(8191)
	firmware := types.UID(windowsFirmware)
	_false := false
	return v1.VirtualMachineInstanceSpec{
		TerminationGracePeriodSeconds: &gracePeriod,
		Domain: v1.DomainSpec{
			CPU: &v1.CPU{Cores: 2},
			Features: &v1.Features{
				ACPI: v1.FeatureState{},
				APIC: &v1.FeatureAPIC{},
				Hyperv: &v1.FeatureHyperv{
					Relaxed:    &v1.FeatureState{},
					SyNICTimer: &v1.SyNICTimer{Direct: &v1.FeatureState{}},
					VAPIC:      &v1.FeatureState{},
					Spinlocks:  &v1.FeatureSpinlocks{Retries: &spinlocks},
				},
			},
			Clock: &v1.Clock{
				ClockOffset: v1.ClockOffset{UTC: &v1.ClockOffsetUTC{}},
				Timer: &v1.Timer{
					HPET:   &v1.HPETTimer{Enabled: &_false},
					PIT:    &v1.PITTimer{TickPolicy: v1.PITTickPolicyDelay},
					RTC:    &v1.RTCTimer{TickPolicy: v1.RTCTickPolicyCatchup},
					Hyperv: &v1.HypervTimer{},
				},
			},
			Firmware: &v1.Firmware{UUID: firmware},
			Resources: v1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("2048Mi"),
				},
			},
			Devices: v1.Devices{
				Disks: []v1.Disk{
					{
						Name:       windowsDisk,
						DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: "sata"}},
					},
				},
			},
		},
		Volumes: []v1.Volume{
			{
				Name: windowsDisk,
				VolumeSource: v1.VolumeSource{
					Ephemeral: &v1.EphemeralVolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: tests.DiskWindows,
						},
					},
				},
			},
		},
	}

}

var _ = Describe("[Serial][sig-compute]Windows VirtualMachineInstance", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	var windowsVMI *v1.VirtualMachineInstance

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
		tests.BeforeTestCleanup()
		tests.SkipIfMissingRequiredImage(virtClient, tests.DiskWindows)
		tests.CreatePVC(tests.OSWindows, "30Gi", tests.Config.StorageClassWindows, true)
		windowsVMI = tests.NewRandomVMI()
		windowsVMI.Spec = getWindowsVMISpec()
		tests.AddExplicitPodNetworkInterface(windowsVMI)
		windowsVMI.Spec.Domain.Devices.Interfaces[0].Model = "e1000"
	})

	It("[test_id:487]should succeed to start a vmi", func() {
		vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
		Expect(err).To(BeNil())
		tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 360)
	}, 300)

	It("[test_id:488]should succeed to stop a running vmi", func() {
		By("Starting the vmi")
		vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
		Expect(err).To(BeNil())
		tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 360)

		By("Stopping the vmi")
		err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
		Expect(err).To(BeNil())
	}, 300)

	Context("with winrm connection", func() {
		var winrmcliPod *k8sv1.Pod
		var cli []string
		var output string

		BeforeEach(func() {
			By("Creating winrm-cli pod for the future use")
			winrmcliPod = &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{GenerateName: winrmCli},
				Spec: k8sv1.PodSpec{
					Containers: []k8sv1.Container{
						{
							Name:    winrmCli,
							Image:   fmt.Sprintf("%s/%s:%s", flags.KubeVirtUtilityRepoPrefix, winrmCli, flags.KubeVirtUtilityVersionTag),
							Command: []string{"sleep"},
							Args:    []string{"3600"},
						},
					},
				},
			}
			winrmcliPod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), winrmcliPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		Context("[ref_id:139]VMI is created", func() {

			BeforeEach(func() {
				By("Starting the windows VirtualMachineInstance")
				windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
				Expect(err).To(BeNil())
				tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 360)

				cli = winrnLoginCommand(virtClient, windowsVMI)
			})

			It("[test_id:240]should have correct UUID", func() {
				command := append(cli, "wmic csproduct get \"UUID\"")
				By(fmt.Sprintf("Running \"%s\" command via winrm-cli", command))
				Eventually(func() error {
					output, err = tests.ExecuteCommandOnPod(
						virtClient,
						winrmcliPod,
						winrmcliPod.Spec.Containers[0].Name,
						command,
					)
					return err
				}, time.Minute*5, time.Second*15).ShouldNot(HaveOccurred())
				By("Checking that the Windows VirtualMachineInstance has expected UUID")
				Expect(output).Should(ContainSubstring(strings.ToUpper(windowsFirmware)))
			}, 360)

			It("[test_id:3159]should have default masquerade IP", func() {
				command := append(cli, "ipconfig /all")
				By(fmt.Sprintf("Running \"%s\" command via winrm-cli", command))
				Eventually(func() error {
					output, err = tests.ExecuteCommandOnPod(
						virtClient,
						winrmcliPod,
						winrmcliPod.Spec.Containers[0].Name,
						command,
					)
					return err
				}, time.Minute*5, time.Second*15).ShouldNot(HaveOccurred())

				By("Checking that the Windows VirtualMachineInstance has expected IP address")
				Expect(output).Should(ContainSubstring("10.0.2.2"))
			}, 360)

			It("[test_id:3160]should have the domain set properly", func() {
				searchDomain := getPodSearchDomain(windowsVMI)
				Expect(searchDomain).To(HavePrefix(windowsVMI.Namespace), "should contain a searchdomain with the namespace of the VMI")

				runCommandAndExpectOutput(virtClient,
					winrmcliPod,
					cli,
					"wmic nicconfig get dnsdomain",
					`DNSDomain[\n\r\t ]+`+searchDomain+`[\n\r\t ]+`)
			}, 360)
		})

		Context("VMI with subdomain is created", func() {
			BeforeEach(func() {
				windowsVMI.Spec.Subdomain = "subdomain"

				By("Starting the windows VirtualMachineInstance with subdomain")
				windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 360)

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
			}, 360)
		})
	})

	Context("[ref_id:142]with kubectl command", func() {
		var workDir string
		var yamlFile string
		BeforeEach(func() {
			tests.SkipIfNoCmd("kubectl")
			workDir, err = ioutil.TempDir("", tests.TempDirPrefix+"-")
			Expect(err).ToNot(HaveOccurred())
			yamlFile, err = tests.GenerateVMIJson(windowsVMI, workDir)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			if workDir != "" {
				err = os.RemoveAll(workDir)
				Expect(err).ToNot(HaveOccurred())
				workDir = ""
			}
		})

		It("[test_id:223]should succeed to start a vmi", func() {
			By("Starting the vmi via kubectl command")
			_, _, err = tests.RunCommand("kubectl", "create", "-f", yamlFile)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 360)
		})

		It("[test_id:239]should succeed to stop a vmi", func() {
			By("Starting the vmi via kubectl command")
			_, _, err = tests.RunCommand("kubectl", "create", "-f", yamlFile)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 360)

			podSelector := tests.UnfinishedVMIPodSelector(windowsVMI)
			By("Deleting the vmi via kubectl command")
			_, _, err = tests.RunCommand("kubectl", "delete", "-f", yamlFile)
			Expect(err).ToNot(HaveOccurred())

			By("Checking that the vmi does not exist anymore")
			result := virtClient.RestClient().Get().Resource(tests.VMIResource).Namespace(k8sv1.NamespaceDefault).Name(windowsVMI.Name).Do(context.Background())
			Expect(result).To(testutils.HaveStatusCode(http.StatusNotFound))

			By("Checking that the vmi pod terminated")
			Eventually(func() int {
				pods, err := virtClient.CoreV1().Pods(util.NamespaceTestDefault).List(context.Background(), podSelector)
				Expect(err).ToNot(HaveOccurred())
				return len(pods.Items)
			}, 75, 0.5).Should(Equal(0))
		})
	})
})

func winrnLoginCommand(virtClient kubecli.KubevirtClient, windowsVMI *v1.VirtualMachineInstance) []string {
	var err error
	windowsVMI, err = virtClient.VirtualMachineInstance(windowsVMI.Namespace).Get(windowsVMI.Name, &metav1.GetOptions{})
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
		_, err := tests.ExecuteCommandOnPod(
			virtClient,
			winrmcliPod,
			winrmcliPod.Spec.Containers[0].Name,
			cliCmd,
		)
		return err
	}, time.Minute*5, time.Second*15).ShouldNot(HaveOccurred())

	By("repeatedly trying to get the search domain, since it may take some time until the domain is set")
	EventuallyWithOffset(1, func() string {
		output, err := tests.ExecuteCommandOnPod(
			virtClient,
			winrmcliPod,
			winrmcliPod.Spec.Containers[0].Name,
			cliCmd,
		)
		Expect(err).ToNot(HaveOccurred())
		return output
	}, time.Minute*1, time.Second*10).Should(MatchRegexp(expectedOutputRegex))
}
