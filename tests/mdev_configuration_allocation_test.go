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
 */

package tests_test

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]MediatedDevices", Serial, decorators.VGPU, decorators.SigCompute, func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	waitForPod := func(outputPod *k8sv1.Pod, fetchPod func() (*k8sv1.Pod, error)) wait.ConditionFunc {
		return func() (bool, error) {

			latestPod, err := fetchPod()
			if err != nil {
				return false, err
			}
			*outputPod = *latestPod

			return latestPod.Status.Phase == k8sv1.PodFailed || latestPod.Status.Phase == k8sv1.PodSucceeded, nil
		}
	}

	checkAllMDEVCreated := func(mdevTypeName string, expectedInstancesCount int) func() (*k8sv1.Pod, error) {
		return func() (*k8sv1.Pod, error) {
			By(fmt.Sprintf("Checking the number of created mdev types, should be %d of %s type ", expectedInstancesCount, mdevTypeName))
			check := fmt.Sprintf(`set -x
		files_num=$(ls -A /sys/bus/mdev/devices/| wc -l)
		if [[ $files_num != %d ]] ; then
		  echo "failed, not enough mdevs of type %[2]s has been created"
		  exit 1
		fi
		for x in $(ls -A /sys/bus/mdev/devices/); do
		  type_name=$(basename "$(readlink -f /sys/bus/mdev/devices/$x/mdev_type)")
		  if [[ "$type_name" != "%[2]s" ]]; then
		     echo "failed, not all mdevs of type %[2]s"
		     exit 1
		  fi
		  exit 0
		done`, expectedInstancesCount, mdevTypeName)
			testPod := libpod.RenderPod("test-all-mdev-created", []string{"/bin/bash", "-c"}, []string{check})
			testPod, err = virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Create(context.Background(), testPod, metav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			var latestPod k8sv1.Pod
			err := virtwait.PollImmediately(5*time.Second, 1*time.Minute, waitForPod(&latestPod, ThisPod(testPod)).WithContext())
			return &latestPod, err
		}

	}

	checkAllMDEVRemoved := func() (*k8sv1.Pod, error) {
		check := `set -x
		files_num=$(ls -A /sys/bus/mdev/devices/| wc -l)
		if [[ $files_num != 0 ]] ; then
		  echo "failed, not all mdevs removed"
		  exit 1
		fi
	        exit 0`
		testPod := libpod.RenderPod("test-all-mdev-removed", []string{"/bin/bash", "-c"}, []string{check})
		testPod, err = virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Create(context.Background(), testPod, metav1.CreateOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		var latestPod k8sv1.Pod
		err := virtwait.PollImmediately(time.Second, 1*time.Minute, waitForPod(&latestPod, ThisPod(testPod)).WithContext())
		return &latestPod, err
	}

	noGPUDevicesAreAvailable := func() {
		EventuallyWithOffset(2, checkAllMDEVRemoved, 2*time.Minute, 10*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))

		EventuallyWithOffset(2, func() int64 {
			nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
			ExpectWithOffset(3, err).ToNot(HaveOccurred())
			for _, node := range nodes.Items {
				for key, amount := range node.Status.Capacity {
					if strings.HasPrefix(string(key), "nvidia.com/") {
						ret, ok := amount.AsInt64()
						ExpectWithOffset(3, ok).To(BeTrue())
						return ret
					}
				}
			}
			return 0
		}, 2*time.Minute, 5*time.Second).Should(BeZero(), "wait for the kubelet to stop promoting unconfigured devices")
	}

	Context("with externally provided mediated devices", func() {
		var deviceName = "nvidia.com/GRID_T4-1B"
		var mdevSelector = "GRID T4-1B"
		var desiredMdevTypeName = "nvidia-222"
		var expectedInstancesNum = 16
		var config v1.KubeVirtConfiguration
		var originalFeatureGates []string

		addMdevsConfiguration := func() {
			By("Creating a configuration for mediated devices")
			kv := libkubevirt.GetCurrentKv(virtClient)
			config = kv.Spec.Configuration
			originalFeatureGates = append(originalFeatureGates, config.DeveloperConfiguration.FeatureGates...)
			config.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{
				MediatedDeviceTypes: []string{desiredMdevTypeName},
			}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)
		}

		cleanupConfiguredMdevs := func() {
			By("restoring the mdevs handling to allow cleanup")
			config.DeveloperConfiguration.FeatureGates = originalFeatureGates
			By("Removing the configuration of mediated devices")
			config.PermittedHostDevices = &v1.PermittedHostDevices{}
			config.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)
			By("Verifying that an expected amount of devices has been created")
			noGPUDevicesAreAvailable()
		}
		BeforeEach(func() {
			addMdevsConfiguration()

			By("Verifying that an expected amount of devices has been created")
			Eventually(checkAllMDEVCreated(desiredMdevTypeName, expectedInstancesNum), 3*time.Minute, 15*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))
		})

		AfterEach(func() {
			cleanupConfiguredMdevs()
		})
		It("Should make sure that mdevs listed with ExternalResourceProvider are not removed", func() {

			By("Listing the created mdevs as externally provided ")
			config.PermittedHostDevices = &v1.PermittedHostDevices{
				MediatedDevices: []v1.MediatedHostDevice{
					{
						MDEVNameSelector:         mdevSelector,
						ResourceName:             deviceName,
						ExternalResourceProvider: true,
					},
				},
			}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)

			By("Removing the mediated devices configuration and expecting no devices being removed")
			config.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)
			Eventually(checkAllMDEVCreated(desiredMdevTypeName, expectedInstancesNum), 3*time.Minute, 15*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))
		})

		It("Should make sure that no mdev is removed if the feature is gated", func() {

			By("Adding feature gate to disable mdevs handling")

			config.DeveloperConfiguration.FeatureGates = append(config.DeveloperConfiguration.FeatureGates, featuregate.DisableMediatedDevicesHandling)
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)

			By("Removing the mediated devices configuration and expecting no devices being removed")
			config.PermittedHostDevices = &v1.PermittedHostDevices{}
			config.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)
			Eventually(checkAllMDEVCreated(desiredMdevTypeName, expectedInstancesNum), 3*time.Minute, 15*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))
		})
	})

	Context("with mediated devices configuration", func() {
		var vmi *v1.VirtualMachineInstance
		var deviceName = "nvidia.com/GRID_T4-1B"
		var mdevSelector = "GRID T4-1B"
		var updatedDeviceName = "nvidia.com/GRID_T4-2B"
		var updatedMdevSelector = "GRID T4-2B"
		var parentDeviceID = "10de:1eb8"
		var desiredMdevTypeName = "nvidia-222"
		var expectedInstancesNum = 16
		var config v1.KubeVirtConfiguration
		var mdevTestLabel = "mdevTestLabel1"

		BeforeEach(func() {
			kv := libkubevirt.GetCurrentKv(virtClient)

			By("Creating a configuration for mediated devices")
			config = kv.Spec.Configuration
			config.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{
				MediatedDeviceTypes: []string{desiredMdevTypeName},
			}
			config.PermittedHostDevices = &v1.PermittedHostDevices{
				MediatedDevices: []v1.MediatedHostDevice{
					{
						MDEVNameSelector: mdevSelector,
						ResourceName:     deviceName,
					},
					{
						MDEVNameSelector: updatedMdevSelector,
						ResourceName:     updatedDeviceName,
					},
				},
			}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)

			By("Verifying that an expected amount of devices has been created")
			Eventually(checkAllMDEVCreated(desiredMdevTypeName, expectedInstancesNum), 3*time.Minute, 15*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))
		})

		cleanupConfiguredMdevs := func() {
			By("Deleting the VMI")
			ExpectWithOffset(1, virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI")
			By("Creating a configuration for mediated devices")
			config.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)
			By("Verifying that an expected amount of devices has been created")
			noGPUDevicesAreAvailable()
		}

		AfterEach(func() {
			cleanupConfiguredMdevs()
		})

		It("Should successfully passthrough a mediated device", func() {

			By("Creating a Fedora VMI")
			vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithAutoattachGraphicsDevice(true))
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1G")
			vGPUs := []v1.GPU{
				{
					Name:       "gpu1",
					DeviceName: deviceName,
				},
			}
			vmi.Spec.Domain.Devices.GPUs = vGPUs
			createdVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = createdVmi
			libwait.WaitForSuccessfulVMIStart(vmi)
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Making sure the device is present inside the VMI")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "grep -c " + strings.Replace(parentDeviceID, ":", "", 1) + " /proc/bus/pci/devices\n"},
				&expect.BExp{R: console.RetValue("1")},
			}, 250)).To(Succeed(), "Device not found")

			domXml, err := libdomain.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			// make sure that one mdev has display and ramfb on
			By("Maiking sure that a boot display is enabled")
			Expect(domXml).To(MatchRegexp(`<hostdev .*display=.?on.?`), "Display should be on")
			Expect(domXml).To(MatchRegexp(`<hostdev .*ramfb=.?on.?`), "RamFB should be on")
		})
		It("Should successfully passthrough a mediated device with a disabled display", func() {
			_false := false
			By("Creating a Fedora VMI")
			vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1G")
			vGPUs := []v1.GPU{
				{
					Name:       "gpu2",
					DeviceName: deviceName,
					VirtualGPUOptions: &v1.VGPUOptions{
						Display: &v1.VGPUDisplayOptions{
							Enabled: &_false,
						},
					},
				},
			}
			vmi.Spec.Domain.Devices.GPUs = vGPUs
			createdVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = createdVmi
			libwait.WaitForSuccessfulVMIStart(vmi)

			domXml, err := libdomain.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			// make sure that another mdev explicitly turned off its display
			By("Maiking sure that a boot display is disabled")
			Expect(domXml).ToNot(MatchRegexp(`<hostdev .*display=.?on.?`), "Display should not be enabled")
			Expect(domXml).ToNot(MatchRegexp(`<hostdev .*ramfb=.?on.?`), "RamFB should not be enabled")
		})
		It("Should override default mdev configuration on a specific node", func() {
			newDesiredMdevTypeName := "nvidia-223"
			newExpectedInstancesNum := 8
			By("Creating a configuration for mediated devices")
			config.MediatedDevicesConfiguration.NodeMediatedDeviceTypes = []v1.NodeMediatedDeviceTypesConfig{
				{
					NodeSelector: map[string]string{
						cleanup.TestLabelForNamespace(testsuite.GetTestNamespace(vmi)): mdevTestLabel,
					},
					MediatedDeviceTypes: []string{
						"nvidia-223",
					},
				},
			}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)
			By("Verify that the default mdev configuration didn't change")
			Eventually(checkAllMDEVCreated(desiredMdevTypeName, expectedInstancesNum), 3*time.Minute, 15*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))

			By("Adding a mdevTestLabel1 that should trigger mdev config change")
			// There should be only one node in this lane
			singleNode := libnode.GetAllSchedulableNodes(virtClient).Items[0]
			libnode.AddLabelToNode(singleNode.Name, cleanup.TestLabelForNamespace(testsuite.GetTestNamespace(vmi)), mdevTestLabel)

			By("Creating a Fedora VMI")
			vmi = libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithAutoattachGraphicsDevice(true))
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1G")
			vGPUs := []v1.GPU{
				{
					Name:       "gpu1",
					DeviceName: updatedDeviceName,
				},
			}
			vmi.Spec.Domain.Devices.GPUs = vGPUs
			createdVmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = createdVmi
			libwait.WaitForSuccessfulVMIStart(vmi)
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Verifying that an expected amount of devices has been created")
			Eventually(checkAllMDEVCreated(newDesiredMdevTypeName, newExpectedInstancesNum), 3*time.Minute, 15*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))

		})
	})
	Context("with generic mediated devices", func() {
		const mdevBusPath = "/sys/class/mdev_bus/"
		const findMdevCapableDevices = "ls -df1 " + mdevBusPath + "0000* | head -1"
		const findSupportedTypeFmt = "ls -df1 " + mdevBusPath + "%s/mdev_supported_types/* | head -1"
		const deviceNameFmt = mdevBusPath + "%s/mdev_supported_types/%s/name"
		const unbindCmdFmt = "echo %s > %s/unbind"
		const bindCmdFmt = "echo %s > %s/bind"
		const uuidRegex = "????????-????-????-????-????????????"
		const mdevUUIDPathFmt = "/sys/class/mdev_bus/%s/%s"
		const mdevTypePathFmt = "/sys/class/mdev_bus/%s/%s/mdev_type"

		var node string
		var driverPath string
		var rootPCIId string

		runBashCmd := func(cmd string) (string, error) {
			args := []string{"bash", "-x", "-c", cmd}
			stdout, err := libnode.ExecuteCommandInVirtHandlerPod(node, args)
			stdout = strings.TrimSpace(stdout)
			return stdout, err
		}

		runBashCmdRw := func(cmd string) error {
			// On kind, virt-handler seems to have /sys mounted as read-only.
			// This uses a privileged pod with /sys explitly mounted in read/write mode.
			testPod := libpod.RenderPrivilegedPod("test-rw-sysfs", []string{"bash", "-x", "-c"}, []string{cmd})
			testPod.Spec.Volumes = append(testPod.Spec.Volumes, k8sv1.Volume{
				Name: "sys",
				VolumeSource: k8sv1.VolumeSource{
					HostPath: &k8sv1.HostPathVolumeSource{Path: "/sys"},
				},
			})
			testPod.Spec.Containers[0].VolumeMounts = append(testPod.Spec.Containers[0].VolumeMounts, k8sv1.VolumeMount{
				Name:      "sys",
				ReadOnly:  false,
				MountPath: "/sys",
			})
			testPod, err = virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Create(context.Background(), testPod, metav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			var latestPod k8sv1.Pod
			err := virtwait.PollImmediately(time.Second, 2*time.Minute, waitForPod(&latestPod, ThisPod(testPod)).WithContext())
			return err
		}

		BeforeEach(func() {
			Skip("Unbinding older NVIDIA GPUs, such as the Tesla T4 found on vgpu lanes, doesn't work reliably")
			nodes := libnode.GetAllSchedulableNodes(virtClient).Items
			Expect(nodes).To(HaveLen(1))
			node = nodes[0].Name
			rootPCIId = "none"
		})

		AfterEach(func() {
			if CurrentSpecReport().Failed() && rootPCIId != "none" && driverPath != "none" {
				// The last test went far enough to un-bind the device and then failed.
				// Make sure we don't leave the device in an unbound state
				_ = runBashCmdRw(fmt.Sprintf(bindCmdFmt, rootPCIId, driverPath))
			}
		})

		It("should create mdevs on devices that appear after CR configuration", func() {
			By("looking for an mdev-compatible PCI device")
			out, err := runBashCmd(findMdevCapableDevices)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ContainSubstring(mdevBusPath))
			pciId := "'" + filepath.Base(out) + "'"

			By("finding the driver")
			driverPath, err = runBashCmd("readlink -e " + mdevBusPath + pciId + "/driver")
			Expect(err).ToNot(HaveOccurred())
			Expect(driverPath).To(ContainSubstring("drivers"))

			By("finding a supported type")
			out, err = runBashCmd(fmt.Sprintf(findSupportedTypeFmt, pciId))
			Expect(err).ToNot(HaveOccurred())
			Expect(out).ToNot(BeEmpty())
			mdevType := filepath.Base(out)

			By("finding the name of the device")
			fileName := fmt.Sprintf(deviceNameFmt, pciId, mdevType)
			deviceName, err := runBashCmd("cat " + fileName)
			Expect(err).ToNot(HaveOccurred())
			Expect(deviceName).ToNot(BeEmpty())

			By("unbinding the device from its driver")
			re := regexp.MustCompile(`[\da-f]{2}\.[\da-f]'$`)
			rootPCIId = re.ReplaceAllString(pciId, "00.0'")
			err = runBashCmdRw(fmt.Sprintf(unbindCmdFmt, rootPCIId, driverPath))
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				_, err = runBashCmd("ls " + mdevBusPath + pciId)
				return err
			}).Should(HaveOccurred(), "failed to disable the VFs on "+rootPCIId)

			By("adding the device to the KubeVirt CR")
			resourceName := filepath.Base(driverPath) + ".com/" + strings.ReplaceAll(deviceName, " ", "_")
			kv := libkubevirt.GetCurrentKv(virtClient)
			config := kv.Spec.Configuration
			config.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{
				MediatedDevicesTypes: []string{mdevType},
			}
			config.PermittedHostDevices = &v1.PermittedHostDevices{
				MediatedDevices: []v1.MediatedHostDevice{
					{
						MDEVNameSelector: deviceName,
						ResourceName:     resourceName,
					},
				},
			}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)

			By("re-binding the device to its driver")
			err = runBashCmdRw(fmt.Sprintf(bindCmdFmt, rootPCIId, driverPath))
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				_, err = runBashCmd("ls " + mdevBusPath + pciId)
				return err
			}).ShouldNot(HaveOccurred(), "failed to re-enable the VFs on "+rootPCIId)

			By("expecting the creation of a mediated device")
			mdevUUIDPath := fmt.Sprintf(mdevUUIDPathFmt, pciId, uuidRegex)
			Eventually(func() error {
				uuidPath, err := runBashCmd("ls -d " + mdevUUIDPath + " | head -1")
				if err != nil {
					return err
				}
				if uuidPath == "" {
					return fmt.Errorf("no UUID found at %s", mdevUUIDPath)
				}
				uuid := strings.TrimSpace(filepath.Base(uuidPath))
				mdevTypePath := fmt.Sprintf(mdevTypePathFmt, pciId, uuid)
				effectiveTypePath, err := runBashCmd("readlink -e " + mdevTypePath)
				if err != nil {
					return err
				}
				if filepath.Base(effectiveTypePath) != mdevType {
					return fmt.Errorf("%s != %s", filepath.Base(effectiveTypePath), mdevType)
				}
				return nil
			}, 5*time.Minute, time.Second).ShouldNot(HaveOccurred())
		})
	})
})
