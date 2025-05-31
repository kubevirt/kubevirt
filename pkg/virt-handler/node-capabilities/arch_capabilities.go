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
 * Copyright the KubeVirt Authors.
 *
 */

package nodecapabilities

import (
	"runtime"
)

const (
	amd64 = "amd64"
	arm64 = "arm64"
	s390x = "s390x"
)

// Ensure that there is a compile error should the struct not implement the archCapabilities interface anymore.
var _ = archCapabilities(&defaultArchCapabilities{})

type archCapabilities interface {
	defaultVendor() string
	requirePolicy(policy string) bool
	hasHostSupportedFeatures() bool
	supportsHostModel() bool
	supportsNamedModels() bool
	arch() string
}

func NewArchCapabilities(arch string) archCapabilities {
	switch arch {
	case amd64:
		return archCapabilitiesAMD64{}
	case arm64:
		return archCapabilitiesARM64{}
	case s390x:
		return archCapabilitiesS390X{}
	default:
		return defaultArchCapabilities{}
	}
}

type defaultArchCapabilities struct{}

func (defaultArchCapabilities) defaultVendor() string {
	return ""
}

func (defaultArchCapabilities) requirePolicy(policy string) bool {
	return policy == RequirePolicy
}
func (defaultArchCapabilities) hasHostSupportedFeatures() bool {
	return false
}

func (defaultArchCapabilities) supportsHostModel() bool {
	return false
}

func (defaultArchCapabilities) supportsNamedModels() bool {
	return false
}

func (defaultArchCapabilities) arch() string {
	return runtime.GOARCH
}
