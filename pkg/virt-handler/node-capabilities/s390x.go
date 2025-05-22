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

// Ensure that there is a compile error should the struct not implement the archCapabilities interface anymore.
var _ = archCapabilities(&archCapabilitiesS390X{})

type archCapabilitiesS390X struct {
	defaultArchCapabilities
}

func (archCapabilitiesS390X) defaultVendor() string {
	// On s390x the xml does not include a CPU Vendor, however there is only one company selling them anyway.
	return "IBM"
}

func (archCapabilitiesS390X) requirePolicy(policy string) bool {
	// On s390x, the policy is not set
	return policy == RequirePolicy || policy == ""
}

func (archCapabilitiesS390X) hasHostSupportedFeatures() bool {
	return true
}

func (archCapabilitiesS390X) supportsHostModel() bool {
	return true
}

func (archCapabilitiesS390X) supportsNamedModels() bool {
	return true
}

func (archCapabilitiesS390X) arch() string {
	return s390x
}
