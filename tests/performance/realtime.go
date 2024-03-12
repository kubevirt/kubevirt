package performance

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/util"
)

const tuneAdminRealtimeCloudInitData = `#cloud-config
password: fedora
chpasswd: { expire: False }
bootcmd:
   - sudo tuned-adm profile realtime
`

func byStartingTheVMI(vmi *v1.VirtualMachineInstance, virtClient kubecli.KubevirtClient) {
	By("Starting a VirtualMachineInstance")
	var err error
	vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	libwait.WaitForSuccessfulVMIStart(vmi)
}

var _ = SIGDescribe("CPU latency tests for measuring realtime VMs performance", decorators.RequiresTwoWorkerNodesWithCPUManager, func() {

	var (
		vmi        *v1.VirtualMachineInstance
		virtClient kubecli.KubevirtClient
		err        error
	)

	BeforeEach(func() {
		skipIfNoRealtimePerformanceTests()
		virtClient = kubevirt.Client()
		checks.SkipTestIfNoFeatureGate(virtconfig.NUMAFeatureGate)
		checks.SkipTestIfNotEnoughNodesWithCPUManagerWith2MiHugepages(1)
	})

	It("running cyclictest and collecting results directly from VM", func() {
		const memory = "512Mi"
		const noMask = ""
		vmi = libvmi.New(
			libvmi.WithRng(),
			libvmi.WithContainerDisk("disk0", cd.ContainerDiskFor(cd.ContainerDiskFedoraRealtime)),
			libvmi.WithCloudInitNoCloudEncodedUserData(tuneAdminRealtimeCloudInitData),
			libvmi.WithResourceCPU("2"),
			libvmi.WithLimitCPU("2"),
			libvmi.WithResourceMemory(memory),
			libvmi.WithLimitMemory(memory),
			libvmi.WithCPUModel(v1.CPUModeHostPassthrough),
			libvmi.WithDedicatedCPUPlacement(),
			libvmi.WithRealtimeMask(noMask),
			libvmi.WithNUMAGuestMappingPassthrough(),
			libvmi.WithHugepages("2Mi"),
			libvmi.WithGuestMemory(memory),
		)
		byStartingTheVMI(vmi, virtClient)
		By("validating VMI is up and running")
		vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(context.Background(), vmi.Name, k8smetav1.GetOptions{})
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
		Expect(res).To(HaveLen(1))
		sout := strings.Split(res[0].Output, "\r\n")
		Expect(sout).To(HaveLen(3))
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
