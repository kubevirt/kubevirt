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
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	failedDeleteVMI = "Failed to delete VMI"
	cmdNumberUSBs   = "dmesg | grep -c idVendor=46f4"
)

var _ = Describe("[sig-compute][USB] [QUARANTINE] host USB Passthrough", Serial,
	decorators.Quarantine, decorators.SigCompute, decorators.USB, func() {
		var virtClient kubecli.KubevirtClient
		var config v1.KubeVirtConfiguration
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			virtClient = kubevirt.Client()
			kv := libkubevirt.GetCurrentKv(virtClient)
			config = kv.Spec.Configuration

			nodeName := libnode.GetNodeNameWithHandler()
			Expect(nodeName).ToNot(BeEmpty())

			stdout, err := libnode.ExecuteCommandInVirtHandlerPod(nodeName, []string{"dmesg"})
			Expect(err).ToNot(HaveOccurred())
			if strings.Count(stdout, "idVendor=46f4") == 0 {
				Fail("No emulated USB devices present for functional test.")
			}

			vmi = libvmifact.NewAlpine()
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
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToAlpine)

				By("Making sure the usb is present inside the VMI")
				const expectTimeout = 15
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: fmt.Sprintf("%s\n", cmdNumberUSBs)},
					&expect.BExp{R: console.RetValue(fmt.Sprintf("%d", len(deviceNames)))},
				}, expectTimeout)).To(Succeed(), "Device not found")

				By("Trying to read and write to each USB device inside the guest")

				testScript := `for bus in /dev/bus/usb/*; do for dev in "$bus"/*; do echo -n X > "$dev"; cat "$dev"; done; done; echo USB_TEST_DONE`
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: testScript + "\n"},
					&expect.BExp{R: "USB_TEST_DONE"},
				}, expectTimeout)).To(Succeed(), "Could not access USB devices from inside the guest")
			},
				Entry("Should successfully passthrough 1 emulated USB device", []string{"slow-storage"}),
				Entry("Should successfully passthrough 2 emulated USB devices", []string{"fast-storage", "low-storage"}),
			)
		})
	})
