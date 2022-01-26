package tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/framework/framework"

	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = PDescribe("Ensure stable functionality", func() {

	f := framework.NewDefaultFramework("stability")

	Measure("by repeately starting vmis many times without issues", func(b Benchmarker) {
		b.Time("from_start_to_ready", func() {
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
			vmi, err := f.KubevirtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Create VMI successfully")
			tests.WaitForSuccessfulVMIStart(vmi)
		})
	}, 15)
})
