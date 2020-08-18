package libvmi

import (
	"time"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
)

func SetupVMI(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Should initialize KubeVirt client")

	vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "VMI should be successfully created")

	vmi = tests.WaitUntilVMIReady(vmi, tests.LoggedInCirrosExpecter)

	return vmi
}

func CleanupVMI(vmi *v1.VirtualMachineInstance) {
	if vmi == nil {
		return
	}

	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Should initialize KubeVirt client")

	ExpectWithOffset(1, virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.GetName(), &metav1.DeleteOptions{})).To(Succeed())

	EventuallyWithOffset(1, func() error {
		_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.GetName(), &metav1.GetOptions{})
		return err
	}, 2*time.Minute, time.Second).Should(SatisfyAll(HaveOccurred(), WithTransform(errors.IsNotFound, BeTrue())), "The VMI should be gone within the given timeout")
}
