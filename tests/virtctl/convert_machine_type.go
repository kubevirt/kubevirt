package virtctl

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gcustom"
	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	machineTypeNeedsUpdate = "pc-q35-rhel8.2.0"
	machineTypeNoUpdate    = "pc-q35-rhel9.0.0"
	machineTypeGlob        = "*rhel8.*"
	update                 = "update"
	machineTypes           = "machine-types"
	namespaceFlag          = "namespace"
	labelSelectorFlag      = "label-selector"
	forceRestartFlag       = "--restart-now"
	testLabel              = "testing-label="
)

var _ = Describe("[sig-compute][virtctl] mass machine type transition", decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("should update the machine types of outdated VMs", func() {
		var vmList []*v1.VirtualMachine
		var job *batchv1.Job

		BeforeEach(func() {
			vmList = []*v1.VirtualMachine{}
		})

		AfterEach(func() {
			for _, vm := range vmList {

				err = virtClient.VirtualMachine(vm.Namespace).Delete(context.Background(), vm.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
			deleteJob(virtClient, job)
		})

		createVM := func(machineType, namespace string, hasLabel, running bool) *v1.VirtualMachine {
			template := libvmi.New(
				libvmi.WithResourceMemory(("32Mi")),
				libvmi.WithNamespace(namespace),
				withMachineType(machineType),
			)

			vm := tests.NewRandomVirtualMachine(template, running)
			if hasLabel {
				if vm.Labels == nil {
					vm.Labels = map[string]string{}
				}
				vm.Labels["testing-label"] = ""
			}

			vm, err = virtClient.VirtualMachine(namespace).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())

			if running {
				tests.StartVirtualMachine(vm)
			}

			vmList = append(vmList, vm)
			return vm
		}
		It("no optional arguments are passed to virtctl command", Label("virtctl-update"), func() {
			vmNeedsUpdateStopped := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, false, false)
			vmNeedsUpdateRunning := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, false, true)
			vmNoUpdate := createVM(machineTypeNoUpdate, util.NamespaceTestDefault, false, false)

			err := clientcmd.NewRepeatableVirtctlCommand(update, machineTypes, machineTypeGlob)()
			Expect(err).ToNot(HaveOccurred())

			job = expectJobExists(virtClient)

			Eventually(ThisVM(vmNeedsUpdateStopped), time.Minute, time.Second).Should(haveDefaultMachineType())
			Eventually(ThisVM(vmNeedsUpdateRunning), time.Minute, time.Second).Should(haveDefaultMachineType())
			Eventually(ThisVM(vmNoUpdate), time.Minute, time.Second).Should(haveOriginalMachineType(machineTypeNoUpdate))

			Eventually(ThisVM(vmNeedsUpdateRunning), time.Minute, time.Second).Should(haveRestartRequiredStatus())

			Consistently(thisJob(virtClient, job), time.Minute, time.Second).ShouldNot(haveCompletionTime())
		})

		It("Example with namespace flag", func() {
			vmNamespaceDefaultStopped := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, false, false)
			vmNamespaceDefaultRunning := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, false, true)
			vmNamespaceOtherStopped := createVM(machineTypeNeedsUpdate, metav1.NamespaceDefault, false, false)
			vmNamespaceOtherRunning := createVM(machineTypeNeedsUpdate, metav1.NamespaceDefault, false, true)

			err := clientcmd.NewRepeatableVirtctlCommand(update, machineTypes, machineTypeGlob,
				setFlag(namespaceFlag, util.NamespaceTestDefault))()
			Expect(err).ToNot(HaveOccurred())

			job = expectJobExists(virtClient)

			Eventually(ThisVM(vmNamespaceDefaultStopped), time.Minute, time.Second).Should(haveDefaultMachineType())
			Eventually(ThisVM(vmNamespaceDefaultRunning), time.Minute, time.Second).Should(haveDefaultMachineType())
			Eventually(ThisVM(vmNamespaceOtherStopped), time.Minute, time.Second).Should(haveOriginalMachineType(machineTypeNeedsUpdate))
			Eventually(ThisVM(vmNamespaceOtherRunning), time.Minute, time.Second).Should(haveOriginalMachineType(machineTypeNeedsUpdate))

			Eventually(ThisVM(vmNamespaceDefaultRunning), time.Minute, time.Second).Should(haveRestartRequiredStatus())
			Eventually(ThisVM(vmNamespaceOtherRunning), time.Minute, time.Second).ShouldNot(haveRestartRequiredStatus())

			Consistently(thisJob(virtClient, job), time.Minute, time.Second).ShouldNot(haveCompletionTime())
		})

		It("Example with label-selector flag", func() {
			vmWithLabelStopped := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, true, false)
			vmWithLabelRunning := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, true, true)
			vmNoLabelStopped := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, false, false)
			vmNoLabelRunning := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, false, true)

			err := clientcmd.NewRepeatableVirtctlCommand(update, machineTypes, machineTypeGlob,
				setFlag(labelSelectorFlag, testLabel))()
			Expect(err).ToNot(HaveOccurred())

			job = expectJobExists(virtClient)

			Eventually(ThisVM(vmWithLabelStopped), time.Minute, time.Second).Should(haveDefaultMachineType())
			Eventually(ThisVM(vmWithLabelRunning), time.Minute, time.Second).Should(haveDefaultMachineType())
			Eventually(ThisVM(vmNoLabelStopped), time.Minute, time.Second).Should(haveOriginalMachineType(machineTypeNeedsUpdate))
			Eventually(ThisVM(vmNoLabelRunning), time.Minute, time.Second).Should(haveOriginalMachineType(machineTypeNeedsUpdate))

			Eventually(ThisVM(vmWithLabelRunning)).Should(haveRestartRequiredStatus())
			Eventually(ThisVM(vmNoLabelRunning)).ShouldNot(haveRestartRequiredStatus())

			Consistently(thisJob(virtClient, job), time.Minute, time.Second).ShouldNot(haveCompletionTime())
		})

		It("Example with force-restart flag", func() {
			By("Creating a stopped VM and a running VM that require machine type update.")
			vmNeedsUpdateStopped := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, false, false)
			vmNeedsUpdateRunning := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, false, true)
			vmiNeedsUpdateRunning, err := virtClient.VirtualMachineInstance(vmNeedsUpdateRunning.Namespace).Get(context.Background(), vmNeedsUpdateRunning.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Sending virtctl update machine type cmd with automatic restart flag.")
			err = clientcmd.NewRepeatableVirtctlCommand(update, machineTypes, machineTypeGlob, forceRestartFlag)()
			Expect(err).ToNot(HaveOccurred())

			job = expectJobExists(virtClient)

			By("Ensuring the machine types of both VMs have been updated to the default.")
			Eventually(ThisVM(vmNeedsUpdateStopped), time.Minute, time.Second).Should(haveDefaultMachineType())
			Eventually(ThisVM(vmNeedsUpdateRunning), time.Minute, time.Second).Should(haveDefaultMachineType())

			By("Ensuring the VM has been restarted and the VMI has the default machine type.")
			Eventually(ThisVMI(vmiNeedsUpdateRunning), 120*time.Second, time.Second).Should(beRestarted(vmiNeedsUpdateRunning.UID))
			Eventually(ThisVMI(vmiNeedsUpdateRunning)).Should(haveDefaultMachineType())

			By("Ensuring the job terminates since there are no VMs pending restart.")
			Eventually(thisJob(virtClient, job), time.Minute, time.Second).Should(haveCompletionTime())
		})

		It("Complex example", func() {
			vmNamespaceDefaultStopped := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, false, false)
			vmNamespaceDefaultRunning := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, false, true)
			vmNamespaceOtherStopped := createVM(machineTypeNeedsUpdate, metav1.NamespaceDefault, false, false)
			vmNamespaceOtherRunning := createVM(machineTypeNeedsUpdate, metav1.NamespaceDefault, false, true)
			vmNamespaceDefaultWithLabelStopped := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, true, false)
			vmNamespaceDefaultWithLabelRunning := createVM(machineTypeNeedsUpdate, util.NamespaceTestDefault, true, true)
			vmNamespaceOtherWithLabelStopped := createVM(machineTypeNeedsUpdate, metav1.NamespaceDefault, true, false)
			vmNamespaceOtherWithLabelRunning := createVM(machineTypeNeedsUpdate, metav1.NamespaceDefault, true, true)
			vmNoUpdateStopped := createVM(machineTypeNoUpdate, util.NamespaceTestDefault, true, false)

			vmiNamespaceDefaultWithLabelRunning, err := virtClient.VirtualMachineInstance(vmNamespaceDefaultWithLabelRunning.Namespace).Get(context.Background(), vmNamespaceDefaultWithLabelRunning.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = clientcmd.NewRepeatableVirtctlCommand(update, machineTypes, machineTypeGlob, forceRestartFlag,
				setFlag(namespaceFlag, util.NamespaceTestDefault),
				setFlag(labelSelectorFlag, testLabel))()
			Expect(err).ToNot(HaveOccurred())

			job = expectJobExists(virtClient)

			// the only VMs that should be updated are the ones in the default test namespace and with the test label
			// with force-restart flag, the restart-vm-required label should not be applied to the running VM
			Eventually(ThisVM(vmNamespaceDefaultWithLabelStopped), time.Minute, time.Second).Should(haveDefaultMachineType())
			Eventually(ThisVM(vmNamespaceDefaultWithLabelRunning), time.Minute, time.Second).Should(haveDefaultMachineType())
			Eventually(ThisVM(vmNamespaceDefaultWithLabelRunning), time.Minute, time.Second).ShouldNot(haveRestartRequiredStatus())

			// all other VM machine types should remain unchanged
			Eventually(ThisVM(vmNamespaceDefaultStopped)).Should(haveOriginalMachineType(machineTypeNeedsUpdate))
			Eventually(ThisVM(vmNamespaceDefaultRunning)).Should(haveOriginalMachineType(machineTypeNeedsUpdate))
			Eventually(ThisVM(vmNamespaceOtherStopped)).Should(haveOriginalMachineType(machineTypeNeedsUpdate))
			Eventually(ThisVM(vmNamespaceOtherRunning)).Should(haveOriginalMachineType(machineTypeNeedsUpdate))
			Eventually(ThisVM(vmNamespaceOtherWithLabelStopped)).Should(haveOriginalMachineType(machineTypeNeedsUpdate))
			Eventually(ThisVM(vmNamespaceOtherWithLabelRunning)).Should(haveOriginalMachineType(machineTypeNeedsUpdate))
			Eventually(ThisVM(vmNoUpdateStopped)).Should(haveOriginalMachineType(machineTypeNoUpdate))

			Eventually(ThisVMI(vmiNamespaceDefaultWithLabelRunning), 120*time.Second, time.Second).Should(beRestarted(vmiNamespaceDefaultWithLabelRunning.UID))
			Eventually(ThisVMI(vmiNamespaceDefaultWithLabelRunning), time.Minute, time.Second).Should(haveDefaultMachineType())

			Eventually(thisJob(virtClient, job), time.Minute, time.Second).Should(haveCompletionTime())
		})
	})
})

func expectJobExists(virtClient kubecli.KubevirtClient) *batchv1.Job {
	var job *batchv1.Job
	var ok bool

	Eventually(func() bool {
		jobs, err := virtClient.BatchV1().Jobs("kubevirt").List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		job, ok = hasJob(jobs)
		return ok
	}, 120*time.Second, time.Second).Should(BeTrue())

	return job
}

func thisJob(virtClient kubecli.KubevirtClient, job *batchv1.Job) func() (*batchv1.Job, error) {
	return func() (j *batchv1.Job, err error) {
		return virtClient.BatchV1().Jobs(job.Namespace).Get(context.Background(), job.Name, metav1.GetOptions{})
	}
}

func hasJob(jobs *batchv1.JobList) (*batchv1.Job, bool) {
	for _, job := range jobs.Items {
		if strings.Contains(job.Name, "convert-machine-type") {
			return &job, true
		}
	}

	return nil, false
}

func deleteJob(virtClient kubecli.KubevirtClient, job *batchv1.Job) {
	propagationPolicy := metav1.DeletePropagationBackground
	err := virtClient.BatchV1().Jobs(job.Namespace).Delete(context.Background(), job.Name, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	Expect(err).ToNot(HaveOccurred())
}

func haveOriginalMachineType(machineType string) gomegatypes.GomegaMatcher {
	return gcustom.MakeMatcher(func(actualVM *v1.VirtualMachine) (bool, error) {
		machine := actualVM.Spec.Template.Spec.Domain.Machine
		if machine != nil && machine.Type == machineType {
			return true, nil
		}
		return false, nil
	}).WithTemplate("Expected:\n{{.Actual}}\n{{.To}} to have machine type:\n{{.Data}}", machineType)
}

func haveDefaultMachineType() gomegatypes.GomegaMatcher {
	return gcustom.MakeMatcher(func(obj interface{}) (bool, error) {
		var machine *v1.Machine
		expectedMachineType := virtconfig.DefaultAMD64MachineType
		vm, ok := obj.(*v1.VirtualMachine)
		if ok {
			machine = vm.Spec.Template.Spec.Domain.Machine
		} else {
			vmi, ok := obj.(*v1.VirtualMachineInstance)
			if !ok {
				return false, fmt.Errorf("%v is not a VM or VMI", obj)

			}
			machine = vmi.Spec.Domain.Machine
		}

		if machine != nil && machine.Type == expectedMachineType {
			return true, nil
		}
		return false, nil
	}).WithTemplate("Expected: machine type of \n{{.Actual}}\n{{.To}} to be default machine type")
}

func beRestarted(oldUID types.UID) gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"ObjectMeta": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"UID": Not(Equal(oldUID)),
		}),
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Phase": Equal(v1.Running),
		}),
	}))
}

func haveRestartRequiredStatus() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"MachineTypeRestartRequired": BeTrue(),
		}),
	}))
}

func haveCompletionTime() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"CompletionTime": Not(BeNil()),
		}),
	}))
}

func withMachineType(machineType string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Machine = &v1.Machine{Type: machineType}
		vmi.Status.Machine = &v1.Machine{Type: machineType}
	}
}
