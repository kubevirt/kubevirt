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
	maxFails := pflag.Int("maxfails", 10, "max failed tests to report, defaults to 10")
	pflag.Parse()

	reporter := reporter.NewKubernetesReporter(os.Getenv("ARTIFACTS"), *maxFails)
	reporter.BeforeSuiteDidRun(nil)
	reporter.SpecDidComplete(&types.SpecSummary{
		State:   types.SpecStateFailed,
		RunTime: duration,
	})
}
