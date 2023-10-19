package testsuite

import (
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/util"
)

var (
	TestRunConfiguration RunConfiguration
)

type RunConfiguration struct {
	WarningToIgnoreList []string
}

func initRunConfiguration(virtClient kubecli.KubevirtClient) {
	kv := util.GetCurrentKv(virtClient)
	runConfig := RunConfiguration{}
	if kv.Spec.Configuration.EvictionStrategy != nil &&
		*kv.Spec.Configuration.EvictionStrategy == v1.EvictionStrategyLiveMigrate {
		runConfig.WarningToIgnoreList = append(runConfig.WarningToIgnoreList, "EvictionStrategy is set but vmi is not migratable")
	}

	TestRunConfiguration = runConfig
}
