/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package nodelabeller

import (
	"runtime"

	"kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

const (
	amd64 = "amd64"
	arm64 = "arm64"
	s390x = "s390x"
)

// Ensure that there is a compile error should the struct not implement the archLabeller interface anymore.
var _ = archLabeller(&defaultArchLabeller{})

type archLabeller interface {
	defaultVendor() string
	requirePolicy(policy string) bool
	hasHostSupportedFeatures() bool
	supportsHostModel() bool
	supportsNamedModels() bool
	arch() string
}

func newArchLabeller(arch string) archLabeller {
	switch arch {
	case amd64:
		return archLabellerAMD64{}
	case arm64:
		return archLabellerARM64{}
	case s390x:
		return archLabellerS390X{}
	default:
		return defaultArchLabeller{}
	}
}

type defaultArchLabeller struct{}

func (defaultArchLabeller) defaultVendor() string {
	return ""
}

func (defaultArchLabeller) requirePolicy(policy string) bool {
	return policy == util.RequirePolicy
}

func (defaultArchLabeller) hasHostSupportedFeatures() bool {
	return false
}

func (defaultArchLabeller) supportsHostModel() bool {
	return false
}

func (defaultArchLabeller) supportsNamedModels() bool {
	return false
}

func (defaultArchLabeller) arch() string {
	return runtime.GOARCH
}
