package numa

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/k8s"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]NUMA", Serial, decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient
	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("[test_id:7299] topology should be mapped to the guest and hugepages should be allocated",
		decorators.RequiresNodeWithCPUManager, decorators.RequiresHugepages2Mi, func() {
			var err error
			cpuVMI := libvmifact.NewCirros()
			cpuVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("128Mi")
			cpuVMI.Spec.Domain.CPU = &v1.CPU{
				Cores:                 3,
				DedicatedCPUPlacement: true,
				NUMA:                  &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}},
			}
			cpuVMI.Spec.Domain.Memory = &v1.Memory{
				Hugepages: &v1.Hugepages{PageSize: "2Mi"},
			}

			By("Starting a VirtualMachineInstance")
			cpuVMI, err = virtClient.VirtualMachineInstance(
				testsuite.NamespaceTestDefault).Create(context.Background(), cpuVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			cpuVMI = libwait.WaitForSuccessfulVMIStart(cpuVMI)
			By("Fetching the numa memory mapping")
			handler, err := libnode.GetVirtHandlerPod(k8s.Client(), cpuVMI.Status.NodeName)
			Expect(err).ToNot(HaveOccurred())
			pid := getQEMUPID(handler, cpuVMI)

			By("Checking if the pinned numa memory chunks match the VMI memory size")
			scanner := bufio.NewScanner(strings.NewReader(getNUMAMapping(virtClient, handler, pid)))
			rex := regexp.MustCompile(`bind:(\d+) .+memfd:.+N(\d+)=(\d+).+kernelpagesize_kB=(\d+)`)
			mappings := map[int]mapping{}
			for scanner.Scan() {
				if findings := rex.FindStringSubmatch(scanner.Text()); findings != nil {
					mappings[mustAtoi(findings[1])] = mapping{
						BindNode:           mustAtoi(findings[1]),
						AllocationNode:     mustAtoi(findings[2]),
						Pages:              mustAtoi(findings[3]),
						PageSizeAsQuantity: toKi(mustAtoi(findings[4])),
						PageSize:           mustAtoi(findings[4]),
					}
				}
			}

			sum := 0
			requestedPageSize := resource.MustParse(cpuVMI.Spec.Domain.Memory.Hugepages.PageSize)
			requestedMemory := cpuVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]
			for _, m := range mappings {
				Expect(m.PageSizeAsQuantity.Equal(requestedPageSize)).To(BeTrue())
				Expect(m.BindNode).To(Equal(m.AllocationNode))
				sum += m.Pages
			}
			const memoryFactor = 2048
			Expect(resource.MustParse(fmt.Sprintf("%dKi", sum*memoryFactor)).Equal(requestedMemory)).To(BeTrue())

			By("Fetching the domain XML")
			domSpec, err := libdomain.GetRunningVMIDomainSpec(cpuVMI)
			Expect(err).ToNot(HaveOccurred())

			By("checking that we really deal with a domain with numa configured")
			Expect(domSpec.CPU.NUMA.Cells).ToNot(BeEmpty())

			By("Checking if number of memory chunkgs matches the number of nodes on the VM")
			Expect(mappings).To(HaveLen(len(domSpec.MemoryBacking.HugePages.HugePage)))
			Expect(mappings).To(HaveLen(len(domSpec.CPU.NUMA.Cells)))
			Expect(mappings).To(HaveLen(len(domSpec.NUMATune.MemNodes)))

			By("checking if the guest came up and is healthy")
			Expect(console.LoginToCirros(cpuVMI)).To(Succeed())
		})
})

func getQEMUPID(handlerPod *k8sv1.Pod, vmi *v1.VirtualMachineInstance) string {
	var stdout, stderr string
	const (
		expectedProcesses    = 2
		expectedPathElements = 4
	)
	// Using `ps` here doesn't work reliably. Grepping /proc instead.
	// The "[g]" prevents grep from finding its own process
	Eventually(func() (err error) {
		stdout, stderr, err = exec.ExecuteCommandOnPodWithResults(handlerPod, "virt-handler",
			[]string{
				"/bin/bash",
				"-c",
				fmt.Sprintf("grep -l '[g]uest=%s_%s' /proc/*/cmdline", vmi.Namespace, vmi.Name),
			})
		return err
	}, 3*time.Second, 500*time.Millisecond).Should(Succeed(), stderr, stdout)

	strs := strings.Split(stdout, "\n")
	Expect(strs).To(HaveLen(expectedProcesses),
		"more (or less?) than one matching process was found")
	path := strings.Split(strs[0], "/")
	Expect(path).To(HaveLen(expectedPathElements), "the cmdline path is invalid")

	return path[2]
}

func getNUMAMapping(virtClient kubecli.KubevirtClient, pod *k8sv1.Pod, pid string) string {
	stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(pod, "virt-handler",
		[]string{
			"/bin/bash",
			"-c",
			fmt.Sprintf("trap '' URG && cat /proc/%v/numa_maps", pid),
		})
	Expect(err).ToNot(HaveOccurred(), stderr)
	return stdout
}

type mapping struct {
	BindNode           int
	AllocationNode     int
	Pages              int
	PageSizeAsQuantity resource.Quantity
	PageSize           int
}

func mustAtoi(str string) int {
	i, err := strconv.Atoi(str)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return i
}

func toKi(value int) resource.Quantity {
	return resource.MustParse(fmt.Sprintf("%dKi", value))
}
