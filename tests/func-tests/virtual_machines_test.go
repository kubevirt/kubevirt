package tests

import (
	"flag"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
)

const timeout = 240 * time.Second
const pollingInterval = 5 * time.Second

var _ = Describe("Virtual Machines", func() {

	kubecli.Init()
	flag.Parse()
	client, err := kubecli.GetKubevirtClient()
	PanicOnError(err)

	var vmi *kubevirtv1.VirtualMachineInstance
	var vmiRandName = vmiName + rand.String(48)
	vmi = kubevirtv1.NewMinimalVMIWithNS(testNamespace, vmiRandName)
	jobType := GetJobTypeEnvVar()

	Context("vmi testing", func() {
		It("should enable software emulation for prow job", func() {
			if jobType == "prow" {
				kubevirtCfg, err := client.CoreV1().ConfigMaps(testNamespace).Get(kubevirtCfgMap, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				kubevirtCfg.Data["debug.useEmulation"] = "true"
				_, err = client.CoreV1().ConfigMaps(testNamespace).Update(kubevirtCfg)
				Expect(err).ToNot(HaveOccurred())
			} else {
				Skip("Software emulation should not be enabled for this job")
			}
		})
		It("should create verify and delete a vmi", func() {
			Eventually(func() error {
				_, err := client.VirtualMachineInstance(testNamespace).Create(vmi)
				return err
			}, timeout, pollingInterval).Should(Not(HaveOccurred()), "failed to create a vmi")
			Eventually(func() bool {
				vmi, err = client.VirtualMachineInstance(testNamespace).Get(vmiRandName, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.Phase == kubevirtv1.Running
			}, timeout, pollingInterval).Should(BeTrue(), "failed to get the vmi Running")
			Eventually(func() error {
				err := client.VirtualMachineInstance(testNamespace).Delete(vmiRandName, &k8smetav1.DeleteOptions{})
				return err
			}, timeout, pollingInterval).Should(Not(HaveOccurred()), "failed to delete a vmi")
		})
	})
})
