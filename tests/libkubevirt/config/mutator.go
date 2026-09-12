package config

import (
	v1 "kubevirt.io/api/core/v1"
)

// ConfigMutator applies a KubeVirtConfiguration to the cluster.
//
// The default implementation patches the KubeVirt CR directly.
// Alternative implementations (e.g., for HCO-managed environments)
// patch the managing operator's CR instead, then wait for
// reconciliation to propagate the change to the KubeVirt CR.
//
// The active implementation is selected at compile time via Go
// build tags. See mutator_default.go and mutator_hco.go.
type ConfigMutator interface {
	Apply(config v1.KubeVirtConfiguration) (*v1.KubeVirt, error)
}

var mutator ConfigMutator

func init() {
	mutator = newConfigMutator()
}
