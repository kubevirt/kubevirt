package main

import (
	"os"
	"time"

	"github.com/spf13/pflag"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/reporter"
)

func main() {
	var duration time.Duration
	pflag.CommandLine.AddGoFlagSet(kubecli.FlagSet())
	pflag.DurationVarP(&duration, "since", "s", 10*time.Minute, "collection window, defaults to 10 minutes")
	pflag.Parse()

	// Hardcoding maxFails to 1 since the purpouse here is just to dump the state once
	reporter := reporter.NewKubernetesReporter(os.Getenv("ARTIFACTS"), 1)
	reporter.Cleanup()
	reporter.Dump(duration)
}
