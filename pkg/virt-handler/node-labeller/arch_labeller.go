/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
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
	shouldLabelNodes() bool
	defaultVendor() string
	requirePolicy(policy string) bool
	hasHostSupportedFeatures() bool
	supportsHostModel() bool
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

func (defaultArchLabeller) shouldLabelNodes() bool {
	return false
}

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

func (defaultArchLabeller) arch() string {
	return runtime.GOARCH
}
