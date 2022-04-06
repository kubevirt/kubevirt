package tests_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kvtests "kubevirt.io/kubevirt/tests"
	kvtutil "kubevirt.io/kubevirt/tests/util"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

const (
	timeout         = 360 * time.Second
	pollingInterval = 5 * time.Second
	vmiSamplingSize = 5
)

var _ = Describe("[rfe_id:273][crit:critical][vendor:cnv-qe@redhat.com][level:system]Virtual Machine", func() {
	tests.FlagParse()
	client, err := kubecli.GetKubevirtClient()
	kvtutil.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeEach()
	})

	It("[test_id:5696] should create, verify and delete VMIs", func() {
		for i := 0; i < vmiSamplingSize; i++ {
			fmt.Fprintf(GinkgoWriter, "Run %d/%d\n", i+1, vmiSamplingSize)
			vmiName := verifyVMICreation(client)
			verifyVMIRunning(client, vmiName)
			verifyVMIDeletion(client, vmiName)
		}
	})
})

func verifyVMICreation(client kubecli.KubevirtClient) string {
	By("Creating VMI...")
	vmi := kvtests.NewRandomVMI()
	EventuallyWithOffset(1, func() error {
		_, err := client.VirtualMachineInstance(kvtutil.NamespaceTestDefault).Create(vmi)
		return err
	}, timeout, pollingInterval).Should(Not(HaveOccurred()), "failed to create a vmi")
	return vmi.Name
}

func verifyVMIRunning(client kubecli.KubevirtClient, vmiName string) *kubevirtcorev1.VirtualMachineInstance {
	By("Verifying VMI is running")
	var vmi *kubevirtcorev1.VirtualMachineInstance
	EventuallyWithOffset(1, func() bool {
		var err error
		vmi, err = client.VirtualMachineInstance(kvtutil.NamespaceTestDefault).Get(vmiName, &k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vmi.Status.Phase == kubevirtcorev1.Running
	}, timeout, pollingInterval).Should(BeTrue(), "failed to get the vmi Running")

	return vmi
}

func verifyVMIDeletion(client kubecli.KubevirtClient, vmiName string) {
	By("Verifying node placement of VMI")
	EventuallyWithOffset(1, func() error {
		err := client.VirtualMachineInstance(kvtutil.NamespaceTestDefault).Delete(vmiName, &k8smetav1.DeleteOptions{})
		return err
	}, timeout, pollingInterval).Should(Not(HaveOccurred()), "failed to delete a vmi")
}
