package hotplug

import (
	"context"
	"fmt"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]VM Hotplug PCI Port Allocation", decorators.SigCompute, func() {
	const (
		pciRootPortID = "1b36:000c"
		// 1. Network
		// 2. SCSI controller
		// 3. USB controller
		// 4. Serial controller
		// 5. Memory Balloon
		// 6. Root Disk
		// 7. Cloudinit Disk
		cirrosDefaultPortsUsed = 7
	)

	var (
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	DescribeTable("should allocate the appropriate number of free ports",
		func(memory string, additionalDevs, expectedFreePorts int) {
			options := []libvmi.Option{
				libvmi.WithResourceMemory(memory),
			}
			for i := 1; i <= additionalDevs; i++ {
				options = append(options,
					libvmi.WithEmptyDisk(fmt.Sprintf("emptydisk%d", i), v1.VirtIO, resource.MustParse("10Mi")),
				)
			}
			vmi := libvmifact.NewCirros(options...)
			vmi, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			totalPorts := cirrosDefaultPortsUsed + additionalDevs + expectedFreePorts
			By(fmt.Sprintf("Expecting VM to have %d total ports", totalPorts))

			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

			err = console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("lspci | grep %s | wc -l\n", pciRootPortID)},
				&expect.BExp{R: console.RetValue(fmt.Sprintf("%d", totalPorts))},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
		},
		// min required free ports for <= 2G memory is 3
		Entry("with 1Gi memory and 0 additional devs", "1Gi", 0, 3),
		// 16 total ports default for > 2G and that will leave 9 free
		Entry("with 2.1Gi memory and 0 additional devs", "2.1Gi", 0, 9),
		// min required free ports for > 2G memory is 6
		Entry("with 2.1Gi memory and 6 additional devs", "2.1Gi", 6, 6),
	)
})
