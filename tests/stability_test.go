package tests_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gmeasure"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/libvmi"
)

// Replace PDescribe with FDescribe in order to measure if your changes made
// VMI startup any worse
var _ = PDescribe("Ensure stable functionality", func() {
	It("by repeately starting vmis many times without issues", func() {
		experiment := gmeasure.NewExperiment("VMs creation")
		AddReportEntry(experiment.Name, experiment)

		experiment.Sample(func(idx int) {
			experiment.MeasureDuration("Create VM", func() {
				tests.RunVMIAndExpectLaunch(libvmi.NewCirros(), 60)
			})
		}, gmeasure.SamplingConfig{N: 15, Duration: 10 * time.Minute})
	})
})
