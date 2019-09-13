package main

import (
	"os"
	"time"

	"github.com/onsi/ginkgo/types"
	"github.com/spf13/pflag"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/reporter"
)

func main() {
	var duration time.Duration
	pflag.CommandLine.AddGoFlagSet(kubecli.FlagSet())
	pflag.DurationVarP(&duration, "since", "s", 10*time.Minute, "collection window, defaults to 10 minutes")
	pflag.Parse()

	reporter := reporter.NewKubernetesReporter(os.Getenv("ARTIFACTS"))
	reporter.BeforeSuiteDidRun(nil)
	reporter.SpecDidComplete(&types.SpecSummary{
		State:   types.SpecStateFailed,
		RunTime: duration,
	})
}
