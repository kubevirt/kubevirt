package testsuite

import (
	v1 "kubevirt.io/api/core/v1"
)

var (
	TestRunConfiguration RunConfiguration
)

type RunConfiguration struct {
	WarningToIgnoreList []string
}

func InitRunConfiguration() {
	runConfig := RunConfiguration{}
	if KubeVirtDefaultConfig.EvictionStrategy != nil &&
		*KubeVirtDefaultConfig.EvictionStrategy == v1.EvictionStrategyLiveMigrate {
		runConfig.WarningToIgnoreList = append(runConfig.WarningToIgnoreList, "EvictionStrategy is set but vmi is not migratable")
	}

	TestRunConfiguration = runConfig
}
