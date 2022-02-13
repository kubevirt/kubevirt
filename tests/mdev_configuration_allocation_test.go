package tests_test

import (
	"context"
	"fmt"
	"strings"

	"kubevirt.io/kubevirt/tests/framework/cleanup"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
)

var _ = Describe("[Serial][sig-compute]MediatedDevices", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})
	checkAllMDEVCreated := func(mdevTypeName string, expectedInstancesCount int) {
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
		testPod := tests.RenderPod("test-pod", []string{"/bin/bash", "-c"}, []string{check})
		testPod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), testPod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(ThisPod(testPod), 120).Should(BeInPhase(k8sv1.PodSucceeded))
	}
	checkAllMDEVRemoved := func() {
		check := fmt.Sprintf(`set -x
		files_num=$(ls -A /sys/bus/mdev/devices/| wc -l)
		if [[ $files_num != 0 ]] ; then
		  echo "failed, not all mdevs removed"
		  exit 1
		fi
	        exit 0`)
		testPod := tests.RenderPod("test-pod", []string{"/bin/bash", "-c"}, []string{check})
		testPod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), testPod, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(ThisPod(testPod), 120).Should(BeInPhase(k8sv1.PodSucceeded))
		Eventually(func() bool {
			nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, node := range nodes.Items {
				for key, amount := range node.Status.Capacity {
					if strings.HasPrefix(string(key), "nvidia.com/") && !amount.IsZero() {
						return true
					}
				}
			}
			return false
		}, 60).Should(BeFalse(), "wait for the kubelet to stop promoting unconfigured devices")
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
			tests.BeforeTestCleanup()
			kv := util.GetCurrentKv(virtClient)

			By("Creating a configuration for mediated devices")
			config = kv.Spec.Configuration
			config.DeveloperConfiguration.FeatureGates = []string{virtconfig.GPUGate}
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
			checkAllMDEVCreated(desiredMdevTypeName, expectedInstancesNum)
		})

		cleanupConfiguredMdevs := func() {
			By("Deleting the VMI")
			Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI")
			By("Creating a configuration for mediated devices")
			config.MediatedDevicesConfiguration = &v1.MediatedDevicesConfiguration{}
			tests.UpdateKubeVirtConfigValueAndWait(config)
			By("Verifying that an expected amount of devices has been created")
			checkAllMDEVRemoved()
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
			checkAllMDEVCreated(desiredMdevTypeName, expectedInstancesNum)

			By("Adding a mdevTestLabel1 that should trigger mdev config change")
			// There should be only one node in this lane
			singleNode := util.GetAllSchedulableNodes(virtClient).Items[0]
			tests.AddLabelToNode(singleNode.Name, cleanup.TestLabelForNamespace(util.NamespaceTestDefault), mdevTestLabel)

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
			checkAllMDEVCreated(newDesiredMdevTypeName, newExpectedInstancesNum)
		})
	})
})
