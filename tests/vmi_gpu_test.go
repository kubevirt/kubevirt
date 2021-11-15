package tests_test

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	hwutil "kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libvmi"
)

func parseDeviceAddress(addrString string) []string {
	addrs := strings.Split(addrString, ",")
	naddrs := len(addrs)
	if naddrs > 0 {
		if addrs[naddrs-1] == "" {
			addrs = addrs[:naddrs-1]
		}
	}

	for index, element := range addrs {
		addrs[index] = strings.TrimSpace(element)
	}
	return addrs
}

func checkGPUDevice(vmi *v1.VirtualMachineInstance, gpuName string) {
	cmdCheck := fmt.Sprintf("lspci -m %s\n", gpuName)
	err := console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: console.RetValue("0")},
	}, 15)
	Expect(err).ToNot(HaveOccurred(), "GPU device %q was not found in the VMI %s within the given timeout", gpuName, vmi.Name)
}

var _ = Describe("[Serial][sig-compute]GPU", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	Context("with ephemeral disk", func() {
		It("[test_id:4607]Should create a valid VMI but pod should not go to running state", func() {
			gpuName := "random.com/gpu"
			randomVMI := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			gpus := []v1.GPU{
				{
					Name:       "gpu1",
					DeviceName: gpuName,
				},
			}
			randomVMI.Spec.Domain.Devices.GPUs = gpus
			vmi, apiErr := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(randomVMI)
			Expect(apiErr).ToNot(HaveOccurred())

			pod := libvmi.GetPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
			Expect(pod.Status.Phase).To(Equal(k8sv1.PodPending))
			Expect(pod.Status.Conditions[0].Type).To(Equal(k8sv1.PodScheduled))
			Expect(strings.Contains(pod.Status.Conditions[0].Message, "Insufficient "+gpuName)).To(Equal(true))
			Expect(pod.Status.Conditions[0].Reason).To(Equal("Unschedulable"))
		})

		It("[test_id:4608]Should create a valid VMI and appropriate libvirt domain", func() {
			nodesList, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			var gpuName = ""
			for _, item := range nodesList.Items {
				resourceList := item.Status.Allocatable
				for k, v := range resourceList {
					if strings.HasPrefix(string(k), "nvidia.com") {
						if v.Value() >= 1 {
							gpuName = string(k)
							break
						}
					}
				}
			}
			Expect(gpuName).ToNot(Equal(""))
			randomVMI := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			gpus := []v1.GPU{
				{
					Name:       "gpu1",
					DeviceName: gpuName,
				},
			}
			randomVMI.Spec.Domain.Devices.GPUs = gpus
			vmi, apiErr := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(randomVMI)
			Expect(apiErr).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)
			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())

			readyPod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)

			gpuOutput, err := tests.ExecuteCommandOnPod(
				virtClient,
				readyPod,
				"compute",
				[]string{"printenv", "GPU_PASSTHROUGH_DEVICES_NVIDIA"},
			)
			log.Log.Infof("%v", gpuOutput)
			Expect(err).ToNot(HaveOccurred())
			addrList := parseDeviceAddress(gpuOutput)

			Expect(len(addrList)).To(Equal(len(domSpec.Devices.HostDevices)))
			for n, addr := range addrList {
				Expect(domSpec.Devices.HostDevices[n].Type).To(Equal("pci"))
				Expect(domSpec.Devices.HostDevices[n].Managed).To(Equal("yes"))
				Expect(domSpec.Devices.HostDevices[n].Mode).To(Equal("subsystem"))
				dbsfFields, err := hwutil.ParsePciAddress(addr)
				Expect(err).ToNot(HaveOccurred())
				Expect(domSpec.Devices.HostDevices[n].Source.Address.Domain).To(Equal("0x" + dbsfFields[0]))
				Expect(domSpec.Devices.HostDevices[n].Source.Address.Bus).To(Equal("0x" + dbsfFields[1]))
				Expect(domSpec.Devices.HostDevices[n].Source.Address.Slot).To(Equal("0x" + dbsfFields[2]))
				Expect(domSpec.Devices.HostDevices[n].Source.Address.Function).To(Equal("0x" + dbsfFields[3]))
			}

			checkGPUDevice(randomVMI, "10de")
		})
	})
})
