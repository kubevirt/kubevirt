package tests_test

import (
	"context"
	"fmt"
	"strconv"
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

const (
	mdevBusPath               = "/sys/class/mdev_bus/"
	mdevSupportedTypesDirName = "mdev_supported_types"
)

var _ = Describe("[sig-compute]MediatedDevices", Serial, decorators.VGPU, decorators.SigCompute, func() {
	var err error
	var testNodeName string
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		// There should be only one node in this lane
		nodes := libnode.GetAllSchedulableNodes(virtClient).Items
		Expect(nodes).To(HaveLen(1))
		testNodeName = nodes[0].Name
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

	checkAllMDEVCreated := func(mdevTypeName string, expectedInstancesCount uint) func() (*k8sv1.Pod, error) {
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
		var expectedInstancesNum uint
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
			By("Determining the expected amount of mediated device instances used for the test")
			expectedInstancesNum = getNumOfInstancesOnNodeByMdevType(testNodeName, desiredMdevTypeName)

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
		var expectedInstancesNum uint
		var config v1.KubeVirtConfiguration
		var mdevTestLabel = "mdevTestLabel1"

		BeforeEach(func() {
			By("Determining the expected amount of mediated device instances used for the test")
			expectedInstancesNum = getNumOfInstancesOnNodeByMdevType(testNodeName, desiredMdevTypeName)

			By("Creating a configuration for mediated devices")
			kv := libkubevirt.GetCurrentKv(virtClient)
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
	})
})

// Note: this may not work for non-NVIDIA devices as it relies on `<type-id>/description` which is optional
//
//	NVIDIA driver implements the `max_instance` attribute via `description`
//	$ cat /sys/class/mdev_bus/0000:ab:00.0/mdev_supported_types/nvidia-222/description
//	num_heads=4, frl_config=45, framebuffer=1024M, max_resolution=5120x2880, max_instance=16
func getNumOfInstancesOnNodeByMdevType(nodeName string, typeID string) uint {
	cmd := fmt.Sprintf(`num=0
	for dev in %s*/%[2]s/%[3]s; do
	  ins=$(/usr/bin/grep -m1 -oPs '\bmax_instance=\K\d+\b' ${dev}/description)
	  if [[ -n $ins ]]; then
	    ((num+=$ins))
	  fi
	done
	echo $num`, mdevBusPath, mdevSupportedTypesDirName, typeID)
	args := []string{"bash", "-x", "-c", cmd}
	stdout, err := libnode.ExecuteCommandInVirtHandlerPod(nodeName, args)
	Expect(err).ToNot(HaveOccurred())

	num, err := strconv.ParseUint(strings.TrimSpace(stdout), 10, 0)
	Expect(err).ToNot(HaveOccurred())
	Expect(num).To(BeNumerically(">", 0))
	return uint(num)
}
