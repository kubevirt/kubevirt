package tests_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testscore "kubevirt.io/kubevirt/tests"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
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
	testscore.PanicOnError(err)

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
	vmi := testscore.NewRandomVMI()
	EventuallyWithOffset(1, func() error {
		_, err := client.VirtualMachineInstance(testscore.NamespaceTestDefault).Create(vmi)
		return err
	}, timeout, pollingInterval).Should(Not(HaveOccurred()), "failed to create a vmi")
	return vmi.Name
}

func verifyVMIRunning(client kubecli.KubevirtClient, vmiName string) *kubevirtv1.VirtualMachineInstance {
	By("Verifying VMI is running")
	var vmi *kubevirtv1.VirtualMachineInstance
	EventuallyWithOffset(1, func() bool {
		var err error
		vmi, err = client.VirtualMachineInstance(testscore.NamespaceTestDefault).Get(vmiName, &k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vmi.Status.Phase == kubevirtv1.Running
	}, timeout, pollingInterval).Should(BeTrue(), "failed to get the vmi Running")

	return vmi
}

func verifyVMIDeletion(client kubecli.KubevirtClient, vmiName string) {
	By("Verifying node placement of VMI")
	EventuallyWithOffset(1, func() error {
		err := client.VirtualMachineInstance(testscore.NamespaceTestDefault).Delete(vmiName, &k8smetav1.DeleteOptions{})
		return err
	}, timeout, pollingInterval).Should(Not(HaveOccurred()), "failed to delete a vmi")
}
