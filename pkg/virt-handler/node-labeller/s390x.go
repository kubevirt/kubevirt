/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package nodelabeller

import "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"

// Ensure that there is a compile error should the struct not implement the archLabeller interface anymore.
var _ = archLabeller(&archLabellerS390X{})

type archLabellerS390X struct{}

func (archLabellerS390X) defaultVendor() string {
	// On s390x the xml does not include a CPU Vendor, however there is only one company selling them anyway.
	return "IBM"
}

func (archLabellerS390X) requirePolicy(policy string) bool {
	// On s390x, the policy is not set
	return policy == util.RequirePolicy || policy == ""
}

func (archLabellerS390X) hasHostSupportedFeatures() bool {
	return true
}

func (archLabellerS390X) supportsHostModel() bool {
	return true
}

func (archLabellerS390X) supportsNamedModels() bool {
	return true
}

func (archLabellerS390X) arch() string {
	return s390x
}
