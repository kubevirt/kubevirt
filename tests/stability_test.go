package tests_test

import (
	. "github.com/onsi/ginkgo/v2"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libvmi"
)

// Replace PDescribe with FDescribe in order to measure if your changes made
// VMI startup any worse
var _ = PDescribe("Ensure stable functionality", func() {

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Measure("by repeately starting vmis many times without issues", func(b Benchmarker) {
		b.Time("from_start_to_ready", func() {
			tests.RunVMIAndExpectLaunch(libvmi.NewCirros(), 30)
		})
	}, 15)
})
