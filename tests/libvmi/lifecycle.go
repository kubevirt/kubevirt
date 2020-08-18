package libvmi

import (
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
)

type VMIPool map[string]*v1.VirtualMachineInstance

func NewVMIPool() VMIPool {
	return map[string]*v1.VirtualMachineInstance{}
}

func (vmis VMIPool) Setup(loggedInExpecterFactory tests.VMIExpecterFactory) {
	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Should initialize KubeVirt client")

	vmis.create(virtClient)
	vmis.waitUntilReady(loggedInExpecterFactory, virtClient)
}

func (vmis VMIPool) create(virtClient kubecli.KubevirtClient) {
	for name, vmi := range vmis {
		vmi, err := virtClient.VirtualMachineInstance(vmi.GetNamespace()).Create(vmi)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "VMI %q should be successfully created", name)
		vmis[name] = vmi
	}
}

func (vmis VMIPool) waitUntilReady(loggedInExpecterFactory tests.VMIExpecterFactory, virtClient kubecli.KubevirtClient) {
	for name, vmi := range vmis {
		vmis[name] = tests.WaitUntilVMIReady(vmi, loggedInExpecterFactory)
	}
}

func (vmis VMIPool) Cleanup() {
	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Should initialize KubeVirt client")

	errs := []error{}
	errs = append(errs, vmis.delete(virtClient)...)
	if err = vmis.waitUntilDisposed(virtClient); err != nil {
		errs = append(errs, err)
	}
	ExpectWithOffset(1, errs).To(BeEmpty(), "Cleanup of all VMIs should succeed")
}

func (vmis VMIPool) delete(virtClient kubecli.KubevirtClient) []error {
	errs := []error{}

	for _, vmi := range vmis {
		if vmi == nil {
			continue
		}

		err := virtClient.VirtualMachineInstance(vmi.GetNamespace()).Delete(vmi.GetName(), &metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (vmis VMIPool) waitUntilDisposed(virtClient kubecli.KubevirtClient) error {
	var runningVMIs []string

	err := wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
		runningVMIs = []string{}

		for name, vmi := range vmis {
			if vmi == nil {
				continue
			}

			_, err := virtClient.VirtualMachineInstance(vmi.GetNamespace()).Get(vmi.GetName(), &metav1.GetOptions{})
			if err == nil {
				runningVMIs = append(runningVMIs, name)
			} else if err != nil && !errors.IsNotFound(err) {
				return false, err
			}
		}

		if len(runningVMIs) > 0 {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return fmt.Errorf("timed out waiting for VMIs %v to dispose: %v", runningVMIs, err)
	}

	return nil
}

func SetupVMI(vmi *v1.VirtualMachineInstance, loggedInExpecterFactory tests.VMIExpecterFactory) *v1.VirtualMachineInstance {
	vmiPool := VMIPool{"vmi": vmi}
	vmiPool.Setup(loggedInExpecterFactory)
	return vmiPool["vmi"]
}

func CleanupVMI(vmi *v1.VirtualMachineInstance) {
	vmiPool := VMIPool{"vmi": vmi}
	vmiPool.Cleanup()
}
