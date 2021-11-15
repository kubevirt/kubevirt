package performance

import (
	"fmt"
	"strconv"
	"strings"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/util"
)

const tuneAdminRealtimeCloudInitData = `#cloud-config
password: fedora
chpasswd: { expire: False }
bootcmd:
   - sudo tuned-adm profile realtime
`

var (
	memoryRequest = resource.MustParse("512Mi")
)

func byStartingTheVMI(vmi *v1.VirtualMachineInstance, virtClient kubecli.KubevirtClient) {
	By("Starting a VirtualMachineInstance")
	var err error
	vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
	Expect(err).ToNot(HaveOccurred())
	tests.WaitForSuccessfulVMIStart(vmi)
}

func byConfiguringTheVMIForRealtime(vmi *v1.VirtualMachineInstance, realtimeMask string) {
	vmi.Spec.Domain.Resources.Limits = k8sv1.ResourceList{
		k8sv1.ResourceMemory: memoryRequest,
		k8sv1.ResourceCPU:    resource.MustParse("2"),
	}
	vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
		k8sv1.ResourceMemory: memoryRequest,
		k8sv1.ResourceCPU:    resource.MustParse("2"),
	}
	vmi.Spec.Domain.CPU = &v1.CPU{
		Model:                 "host-passthrough",
		DedicatedCPUPlacement: true,
		Realtime:              &v1.Realtime{Mask: realtimeMask},
		NUMA:                  &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}},
	}
	vmi.Spec.Domain.Memory = &v1.Memory{
		Hugepages: &v1.Hugepages{PageSize: "2Mi"},
		Guest:     &memoryRequest,
	}
}

var _ = SIGDescribe("CPU latency tests for measuring realtime VMs performance", func() {

	var (
		vmi        *v1.VirtualMachineInstance
		virtClient kubecli.KubevirtClient
		err        error
	)

	BeforeEach(func() {
		skipIfNoRealtimePerformanceTests()
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
		checks.SkipTestIfNoFeatureGate(virtconfig.NUMAFeatureGate)
		checks.SkipTestIfNotEnoughNodesWithCPUManagerWith2MiHugepages(1)
		tests.BeforeTestCleanup()
	})

	It("running cyclictest and collecting results directly from VM", func() {
		vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskFedoraRealtime), tuneAdminRealtimeCloudInitData)
		byConfiguringTheVMIForRealtime(vmi, "")
		byStartingTheVMI(vmi, virtClient)
		By("validating VMI is up and running")
		vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
		Expect(console.LoginToFedora(vmi)).To(Succeed())
		By(fmt.Sprintf("running cyclictest for %d seconds", cyclicTestDurationInSeconds))
		cmd := fmt.Sprintf("sudo cyclictest --policy fifo --priority 95 -i 100 -H 1000 -D %ds -q |grep 'Max Latencies' |awk '{print $4}'\n", cyclicTestDurationInSeconds)
		res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
			&expect.BSnd{S: cmd},
			&expect.BExp{R: console.RetValue("[0-9]+")},
		}, int(5+cyclicTestDurationInSeconds))
		Expect(err).NotTo(HaveOccurred())
		Expect(len(res)).To(Equal(1))
		sout := strings.Split(res[0].Output, "\r\n")
		Expect(len(sout)).To(Equal(3))
		max, err := strconv.ParseInt(sout[1], 10, 64)
		Expect(err).NotTo(HaveOccurred())
		Expect(max).NotTo(BeNumerically(">", realtimeThreshold), fmt.Sprintf("Maximum CPU latency of %d is greater than threshold %d", max, realtimeThreshold))
	})

})

type psOutput struct {
	priority    int64
	processorID int64
}

func newPs(psLine string) psOutput {
	s := strings.Split(psLine, " ")
	Expect(len(s)).To(BeNumerically(">", 1))
	prio, err := strconv.ParseInt(s[0], 10, 8)
	Expect(err).NotTo(HaveOccurred())
	procID, err := strconv.ParseInt(s[1], 10, 8)
	Expect(err).NotTo(HaveOccurred())
	return psOutput{priority: prio, processorID: procID}
}
