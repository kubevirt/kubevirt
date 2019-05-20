package tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = PDescribe("Ensure stable functionality", func() {

	tests.FlagParse()

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
})
