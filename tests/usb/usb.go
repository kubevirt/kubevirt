package usb

import (
	"context"
	"fmt"
	"strings"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	failedDeleteVMI = "Failed to delete VMI"
	cmdNumberUSBs   = "dmesg | grep -c idVendor=46f4"
)

var _ = Describe("[sig-compute][USB] host USB Passthrough", Serial, decorators.SigCompute, decorators.USB, func() {
	var virtClient kubecli.KubevirtClient
	var config v1.KubeVirtConfiguration
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		kv := libkubevirt.GetCurrentKv(virtClient)
		config = kv.Spec.Configuration

		nodeName := libnode.GetNodeNameWithHandler()
		Expect(nodeName).ToNot(BeEmpty())

		vmi = libvmifact.NewCirros()
	})

	AfterEach(func() {
		// Make sure to delete the VMI before ending the test otherwise a device could still be taken
		const deleteTimeout = 180
		err := virtClient.VirtualMachineInstance(
			testsuite.NamespaceTestDefault).Delete(context.Background(), vmi.ObjectMeta.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), failedDeleteVMI)

		libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, deleteTimeout)
	})

	Context("with usb storage", func() {
		DescribeTable("with emulated USB devices", func(deviceNames []string) {
			const resourceName = "kubevirt.io/usb-storage"

			By("Adding the emulated USB device to the permitted host devices")
			config.DeveloperConfiguration = &v1.DeveloperConfiguration{
				FeatureGates: []string{featuregate.HostDevicesGate},
			}
			config.PermittedHostDevices = &v1.PermittedHostDevices{
				USB: []v1.USBHostDevice{
					{
						ResourceName: resourceName,
						Selectors: []v1.USBSelector{
							{
								Vendor:  "46f4",
								Product: "0001",
							},
						},
					},
				},
			}
			kvconfig.UpdateKubeVirtConfigValueAndWait(config)

			By("Creating a Fedora VMI with the usb host device")
			hostDevs := []v1.HostDevice{}
			for i, name := range deviceNames {
				hostDevs = append(hostDevs, v1.HostDevice{
					Name:       fmt.Sprintf("usb-%d-%s", i, name),
					DeviceName: resourceName,
				})
			}

			var err error
			vmi.Spec.Domain.Devices.HostDevices = hostDevs
			vmi, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

			By("Making sure the usb is present inside the VMI")
			const expectTimeout = 15
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("%s\n", cmdNumberUSBs)},
				&expect.BExp{R: console.RetValue(fmt.Sprintf("%d", len(deviceNames)))},
			}, expectTimeout)).To(Succeed(), "Device not found")

			By("Verifying ownership is properly set in the host")
			vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())

			usbBusesRaw, err := exec.ExecuteCommandOnPod(
				vmiPod,
				vmiPod.Spec.Containers[0].Name,
				[]string{"ls", "-1", "/dev/bus/usb/"},
			)
			Expect(err).ToNot(HaveOccurred())

			usbBuses := strings.Split(strings.TrimSpace(usbBusesRaw), "\n")
			Expect(len(usbBuses)).To(HaveLen(1), "Expected exactly one USB bus directory, found: %v", usbBuses)

			usbBus := usbBuses[0]

			// List devices in that bus
			usbDevicesRaw, err := exec.ExecuteCommandOnPod(
				vmiPod,
				vmiPod.Spec.Containers[0].Name,
				[]string{"ls", "-1", fmt.Sprintf("/dev/bus/usb/%s", usbBus)},
			)
			Expect(err).ToNot(HaveOccurred())

			usbDevices := strings.Split(strings.TrimSpace(usbDevicesRaw), "\n")

			for _, dev := range usbDevices {
				fullPath := fmt.Sprintf("/dev/bus/usb/%s/%s", usbBus, dev)
				cmd := []string{"stat", "--printf", `"%u %g"`, fullPath}
				stdout, err := exec.ExecuteCommandOnPod(
					vmiPod,
					vmiPod.Spec.Containers[0].Name,
					cmd,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(stdout).Should(Equal(`"107 107"`))
			}
		},
			Entry("Should successfully passthrough 1 emulated USB device", []string{"slow-storage"}),
			Entry("Should successfully passthrough 2 emulated USB devices", []string{"fast-storage", "low-storage"}),
		)
	})
})
