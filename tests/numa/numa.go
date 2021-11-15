package numa

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[sig-compute][serial]NUMA", func() {

	var virtClient kubecli.KubevirtClient
	BeforeEach(func() {
		checks.SkipTestIfNoCPUManager()
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		tests.BeforeTestCleanup()
	})

	It("[test_id:7299] topology should be mapped to the guest and hugepages should be allocated", func() {
		checks.SkipTestIfNoFeatureGate(virtconfig.NUMAFeatureGate)
		checks.SkipTestIfNotEnoughNodesWithCPUManagerWith2MiHugepages(1)
		var err error
		cpuVMI := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
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
		cpuVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(cpuVMI)
		Expect(err).ToNot(HaveOccurred())
		tests.WaitForSuccessfulVMIStart(cpuVMI)
		By("Fetching the numa memory mapping")
		cpuVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(cpuVMI.Name, &k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		handler, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(cpuVMI.Status.NodeName).Pod()
		Expect(err).ToNot(HaveOccurred())
		pid := getQEMUPID(virtClient, handler, cpuVMI)

		By("Checking if the pinned numa memory chunks match the VMI memory size")
		scanner := bufio.NewScanner(strings.NewReader(getNUMAMapping(virtClient, handler, pid)))
		rex := regexp.MustCompile(`bind:([0-9]+) .+memfd:.+N([0-9]+)=([0-9]+).+kernelpagesize_kB=([0-9]+)`)
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
		Expect(resource.MustParse(fmt.Sprintf("%dKi", sum*2048)).Equal(requestedMemory)).To(BeTrue())

		By("Fetching the domain XML")
		domSpec, err := tests.GetRunningVMIDomainSpec(cpuVMI)
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

func getQEMUPID(virtClient kubecli.KubevirtClient, handlerPod *k8sv1.Pod, vmi *v1.VirtualMachineInstance) string {
	var stdout, stderr string
	// The retry is a desperate try to cope with URG in case that URG is not catches by the script
	// since URG keep ps failing
	Eventually(func() (err error) {
		stdout, stderr, err = tests.ExecuteCommandOnPodV2(virtClient, handlerPod, "virt-handler",
			[]string{
				"/bin/bash",
				"-c",
				"trap '' URG && ps ax",
			})
		return err
	}, 3*time.Second, 500*time.Millisecond).Should(Succeed(), stderr)

	pid := ""
	for _, str := range strings.Split(stdout, "\n") {
		if !strings.Contains(str, fmt.Sprintf("-name guest=%s_%s", vmi.Namespace, vmi.Name)) {
			continue
		}
		words := strings.Fields(str)

		// verify it is numeric
		_, err := strconv.Atoi(words[0])
		Expect(err).ToNot(HaveOccurred(), "should have found pid for qemu that is numeric")

		pid = words[0]
		break
	}

	Expect(pid).ToNot(Equal(""), "qemu pid not found")
	return pid
}

func getNUMAMapping(virtClient kubecli.KubevirtClient, pod *k8sv1.Pod, pid string) string {
	stdout, stderr, err := tests.ExecuteCommandOnPodV2(virtClient, pod, "virt-handler",
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
