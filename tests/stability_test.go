package tests_test

import (
	"flag"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = FDescribe("Ensure stable functionality", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Measure("by repeately starting vmis many times without issues", func(b Benchmarker) {
		b.Time("from_start_to_ready", func() {
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Create VMI successfully")
			tests.WaitForSuccessfulVMIStart(vmi)
		})
	}, 15)

	Measure("by repeately starting vmis many times with detailed profiling", func(b Benchmarker) {
		vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
		vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
		Expect(err).To(BeNil(), "Create VMI successfully")
		b.Time("from_start_to_scheduling", func() {
			Eventually(func() v1.VirtualMachineInstancePhase {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).ToNot(BeTrue())
				return vmi.Status.Phase
			}, 30*time.Second, 10*time.Millisecond).Should(Equal(v1.Scheduling))
		})
		b.Time("from_scheduling_to_scheduled", func() {
			Eventually(func() v1.VirtualMachineInstancePhase {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).ToNot(BeTrue())
				return vmi.Status.Phase
			}, 30*time.Second, 10*time.Millisecond).Should(Equal(v1.Scheduled))
		})
		b.Time("from_scheduled_to_running", func() {
			Eventually(func() v1.VirtualMachineInstancePhase {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).ToNot(BeTrue())
				return vmi.Status.Phase
			}, 30*time.Second, 10*time.Millisecond).Should(Equal(v1.Running))
		})
	}, 15)

	Measure("by repeately starting vmis many times with detailed profiling until they are booted", func(b Benchmarker) {
		vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
		vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
		Expect(err).To(BeNil(), "Create VMI successfully")
		b.Time("from_start_to_scheduling", func() {
			Eventually(func() v1.VirtualMachineInstancePhase {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).ToNot(BeTrue())
				return vmi.Status.Phase
			}, 30*time.Second, 10*time.Millisecond).Should(Equal(v1.Scheduling))
		})
		b.Time("from_scheduling_to_scheduled", func() {
			Eventually(func() v1.VirtualMachineInstancePhase {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).ToNot(BeTrue())
				return vmi.Status.Phase
			}, 30*time.Second, 10*time.Millisecond).Should(Equal(v1.Scheduled))
		})
		b.Time("from_scheduled_to_booted", func() {
			_, err := tests.LoggedInCirrosExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
		})
	}, 15)

})
