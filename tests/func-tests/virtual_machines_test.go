package tests_test

import (
	"time"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testscore "kubevirt.io/kubevirt/tests"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
)

const timeout = 360 * time.Second
const pollingInterval = 5 * time.Second

var _ = Describe("Virtual Machines", func() {
	tests.FlagParse()
	client, err := kubecli.GetKubevirtClient()
	testscore.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeEach()
	})

	Context("vmi testing", func() {
		It("should create verify and delete a vmi", func() {
			vmi := testscore.NewRandomVMI()
			vmiName := vmi.Name
			Eventually(func() error {
				_, err := client.VirtualMachineInstance(testscore.NamespaceTestDefault).Create(vmi)
				return err
			}, timeout, pollingInterval).Should(Not(HaveOccurred()), "failed to create a vmi")
			Eventually(func() bool {
				vmi, err = client.VirtualMachineInstance(testscore.NamespaceTestDefault).Get(vmiName, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.Phase == kubevirtv1.Running
			}, timeout, pollingInterval).Should(BeTrue(), "failed to get the vmi Running")
			Eventually(func() error {
				err := client.VirtualMachineInstance(testscore.NamespaceTestDefault).Delete(vmiName, &k8smetav1.DeleteOptions{})
				return err
			}, timeout, pollingInterval).Should(Not(HaveOccurred()), "failed to delete a vmi")
		})
	})
})
