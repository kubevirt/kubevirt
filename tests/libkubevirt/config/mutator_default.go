//go:build !managed_hco

package config

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/testsuite"
)

type directMutator struct{}

func (d *directMutator) Apply(config v1.KubeVirtConfiguration) (*v1.KubeVirt, error) {
	return testsuite.UpdateKubeVirtConfigValue(config), nil
}

func newConfigMutator() ConfigMutator {
	return &directMutator{}
}
