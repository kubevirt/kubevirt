package tests_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/framework/cleanup"
	"kubevirt.io/kubevirt/tests/testsuite"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
)

var _ = Describe("[Serial][sig-compute]MediatedDevices", Serial, func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
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
			testPod := tests.RenderPod("test-all-mdev-created", []string{"/bin/bash", "-c"}, []string{check})
			testPod, err = virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Create(context.Background(), testPod, metav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			var latestPod k8sv1.Pod
			err := wait.PollImmediate(5*time.Second, 3*time.Minute, waitForPod(&latestPod, ThisPod(testPod)))
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
		testPod := tests.RenderPod("test-all-mdev-removed", []string{"/bin/bash", "-c"}, []string{check})
		testPod, err = virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Create(context.Background(), testPod, metav1.CreateOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		var latestPod k8sv1.Pod
		err := wait.PollImmediate(time.Second, 2*time.Minute, waitForPod(&latestPod, ThisPod(testPod)))
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
	Context("with mediated devices configuration", func() {
		var vmi *v1.VirtualMachineInstance
		var deviceName string = "nvidia.com/GRID_T4-1B"
		var mdevSelector string = "GRID T4-1B"
		var updatedDeviceName string = "nvidia.com/GRID_T4-2B"
		var updatedMdevSelector string = "GRID T4-2B"
		var parentDeviceID string = "10de:1eb8"
		var desiredMdevTypeName string = "nvidia-222"
		var expectedInstancesNum int = 16
		var config v1.KubeVirtConfiguration
		var mdevTestLabel = "mdevTestLabel1"

		BeforeEach(func() {
			kv := util.GetCurrentKv(virtClient)

			By("Creating a configuration for mediated devices")
			config = kv.Spec.Configuration
			config.DeveloperConfiguration.FeatureGates = append(config.DeveloperConfiguration.FeatureGates, virtconfig.GPUGate)
			config.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{
				MediatedDevicesTypes: []string{desiredMdevTypeName},
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
			tests.UpdateKubeVirtConfigValueAndWait(config)

			By("Verifying that an expected amount of devices has been created")
			Eventually(checkAllMDEVCreated(desiredMdevTypeName, expectedInstancesNum), 3*time.Minute, 15*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))
		})

		cleanupConfiguredMdevs := func() {
			By("Deleting the VMI")
			ExpectWithOffset(1, virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI")
			By("Creating a configuration for mediated devices")
			config.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{}
			tests.UpdateKubeVirtConfigValueAndWait(config)
			By("Verifying that an expected amount of devices has been created")
			noGPUDevicesAreAvailable()
		}

		AfterEach(func() {
			cleanupConfiguredMdevs()
		})

		It("Should successfully passthrough a mediated device", func() {

			By("Creating a Fedora VMI")
			vmi = tests.NewRandomFedoraVMIWithGuestAgent()
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1G")
			vGPUs := []v1.GPU{
				{
					Name:       "gpu1",
					DeviceName: deviceName,
				},
			}
			vmi.Spec.Domain.Devices.GPUs = vGPUs
			createdVmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			vmi = createdVmi
			tests.WaitForSuccessfulVMIStart(vmi)
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Making sure the device is present inside the VMI")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "grep -c " + strings.Replace(parentDeviceID, ":", "", 1) + " /proc/bus/pci/devices\n"},
				&expect.BExp{R: console.RetValue("1")},
			}, 250)).To(Succeed(), "Device not found")

			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			// make sure that one mdev has display and ramfb on
			By("Maiking sure that a boot display is enabled")
			Expect(domXml).To(MatchRegexp(`<hostdev .*display=.?on.?`), "Display should be on")
			Expect(domXml).To(MatchRegexp(`<hostdev .*ramfb=.?on.?`), "RamFB should be on")
		})
		It("Should successfully passthrough a mediated device with a disabled display", func() {
			_false := false
			By("Creating a Fedora VMI")
			vmi = tests.NewRandomFedoraVMIWithGuestAgent()
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
			createdVmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			vmi = createdVmi
			tests.WaitForSuccessfulVMIStart(vmi)

			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
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
						cleanup.TestLabelForNamespace(util.NamespaceTestDefault): mdevTestLabel,
					},
					MediatedDevicesTypes: []string{
						"nvidia-223",
					},
				},
			}
			tests.UpdateKubeVirtConfigValueAndWait(config)
			By("Verify that the default mdev configuration didn't change")
			Eventually(checkAllMDEVCreated(desiredMdevTypeName, expectedInstancesNum), 3*time.Minute, 15*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))

			By("Adding a mdevTestLabel1 that should trigger mdev config change")
			// There should be only one node in this lane
			singleNode := libnode.GetAllSchedulableNodes(virtClient).Items[0]
			libnode.AddLabelToNode(singleNode.Name, cleanup.TestLabelForNamespace(util.NamespaceTestDefault), mdevTestLabel)

			By("Creating a Fedora VMI")
			vmi = tests.NewRandomFedoraVMIWithGuestAgent()
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1G")
			vGPUs := []v1.GPU{
				{
					Name:       "gpu1",
					DeviceName: updatedDeviceName,
				},
			}
			vmi.Spec.Domain.Devices.GPUs = vGPUs
			createdVmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			vmi = createdVmi
			tests.WaitForSuccessfulVMIStart(vmi)
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Verifying that an expected amount of devices has been created")
			Eventually(checkAllMDEVCreated(newDesiredMdevTypeName, newExpectedInstancesNum), 3*time.Minute, 15*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))

		})
	})
})
