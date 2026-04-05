/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package nodelabeller

// Ensure that there is a compile error should the struct not implement the archLabeller interface anymore.
var _ = archLabeller(&archLabellerAMD64{})

type archLabellerAMD64 struct {
	defaultArchLabeller
}

func (archLabellerAMD64) hasHostSupportedFeatures() bool {
	return true
}

func (archLabellerAMD64) supportsHostModel() bool {
	return true
}

func (archLabellerAMD64) supportsNamedModels() bool {
	return true
}

func (archLabellerAMD64) arch() string {
	return amd64
}
