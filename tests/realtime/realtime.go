package realtime

import (
	"strconv"
	"strings"

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

var _ = Describe("[sig-compute-realtime][Serial]Realtime", func() {

	var (
		vmi        *v1.VirtualMachineInstance
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		checks.SkipTestIfNoFeatureGate(virtconfig.NUMAFeatureGate)
		checks.SkipTestIfNoFeatureGate(virtconfig.CPUManager)
		checks.SkipTestIfNotRealtimeCapable()
		tests.BeforeTestCleanup()

	})

	It("should start the realtime VM when no mask is specified", func() {
		vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskFedoraRealtime), tuneAdminRealtimeCloudInitData)
		byConfiguringTheVMIForRealtime(vmi, "")
		byStartingTheVMI(vmi, virtClient)
		By("Validating VCPU scheduler placement information")
		pod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
		psOutput, err := tests.ExecuteCommandOnPod(
			virtClient,
			pod,
			"compute",
			[]string{"/usr/bin/bash", "-c", "ps -u qemu -L -o policy,rtprio,psr|grep FF| awk '{print $2}'"},
		)
		Expect(err).ToNot(HaveOccurred())
		slice := strings.Split(strings.TrimSpace(psOutput), "\n")
		Expect(len(slice)).To(Equal(2))
		for _, l := range slice {
			Expect(parsePriority(l)).To(BeEquivalentTo(1))
		}
		By("Validating that the memory lock limits are higher than the memory requested")
		psOutput, err = tests.ExecuteCommandOnPod(
			virtClient,
			pod,
			"compute",
			[]string{"/usr/bin/bash", "-c", "grep 'locked memory' /proc/$(ps -u qemu -o pid --noheader|xargs)/limits |tr -s ' '| awk '{print $4\" \"$5}'"},
		)
		Expect(err).ToNot(HaveOccurred())
		limits := strings.Split(strings.TrimSpace(psOutput), " ")
		softLimit, err := strconv.ParseInt(limits[0], 10, 64)
		Expect(err).ToNot(HaveOccurred())
		hardLimit, err := strconv.ParseInt(limits[1], 10, 64)
		Expect(err).ToNot(HaveOccurred())
		Expect(softLimit).To(Equal(hardLimit))
		requested, canConvert := memoryRequest.AsInt64()
		Expect(canConvert).To(BeTrue())
		Expect(hardLimit).To(BeNumerically(">", requested))
		By("checking if the guest is still running")
		vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
		Expect(console.LoginToFedora(vmi)).To(Succeed())
	})

	It("should start the realtime VM when realtime mask is specified", func() {
		vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskFedoraRealtime), tuneAdminRealtimeCloudInitData)
		byConfiguringTheVMIForRealtime(vmi, "0-1,^1")
		byStartingTheVMI(vmi, virtClient)
		pod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
		By("Validating VCPU scheduler placement information")
		psOutput, err := tests.ExecuteCommandOnPod(
			virtClient,
			pod,
			"compute",
			[]string{"/usr/bin/bash", "-c", "ps -u qemu -L -o policy,rtprio,psr|grep FF| awk '{print $2}'"},
		)
		Expect(err).ToNot(HaveOccurred())
		slice := strings.Split(strings.TrimSpace(psOutput), "\n")
		Expect(len(slice)).To(Equal(1))
		Expect(parsePriority(slice[0])).To(BeEquivalentTo(1))

		By("Validating the VCPU mask matches the scheduler profile for all cores")
		psOutput, err = tests.ExecuteCommandOnPod(
			virtClient,
			pod,
			"compute",
			[]string{"/usr/bin/bash", "-c", "ps -cT -u qemu  |grep -i cpu |awk '{print $3\" \" $8}'"},
		)
		Expect(err).ToNot(HaveOccurred())
		slice = strings.Split(strings.TrimSpace(psOutput), "\n")
		Expect(len(slice)).To(Equal(2))
		Expect(slice[0]).To(Equal("FF 0/KVM"))
		Expect(slice[1]).To(Equal("TS 1/KVM"))

		By("checking if the guest is still running")
		vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.Phase).To(Equal(v1.Running))
		Expect(console.LoginToFedora(vmi)).To(Succeed())
	})

})

func parsePriority(psLine string) int64 {
	s := strings.Split(psLine, " ")
	Expect(len(s)).To(Equal(1))
	prio, err := strconv.ParseInt(s[0], 10, 8)
	Expect(err).NotTo(HaveOccurred())
	return prio
}
