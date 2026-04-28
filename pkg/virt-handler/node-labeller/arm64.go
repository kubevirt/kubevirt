/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package nodelabeller

// Ensure that there is a compile error should the struct not implement the archLabeller interface anymore.
var _ = archLabeller(&archLabellerARM64{})

type archLabellerARM64 struct {
	defaultArchLabeller
}

func (archLabellerARM64) arch() string {
	return arm64
}
