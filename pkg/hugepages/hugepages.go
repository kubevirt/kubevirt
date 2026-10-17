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

package hugepages

import (
	v1 "kubevirt.io/api/core/v1"
)

// EffectiveMode returns the hugepages mode for the VMI.
// When unset, the mode is static (default k8s behavior).
func EffectiveMode(hugepages *v1.Hugepages) v1.HugepagesMode {
	if hugepages == nil || hugepages.Mode == nil || *hugepages.Mode == "" {
		return v1.HugepagesModeStatic
	}
	return *hugepages.Mode
}

// EffectivePolicy returns the THP collapse policy.
// When unset, the policy is bestEffort. Only meaningful when mode is transparent.
func EffectivePolicy(hugepages *v1.Hugepages) v1.HugepagesPolicy {
	if hugepages == nil || hugepages.Policy == nil || *hugepages.Policy == "" {
		return v1.HugepagesPolicyBestEffort
	}
	return *hugepages.Policy
}

// IsTransparent reports whether hugepages are backed by transparent huge pages.
func IsTransparent(hugepages *v1.Hugepages) bool {
	return EffectiveMode(hugepages) == v1.HugepagesModeTransparent
}

// ForbidsPostCopy reports whether post-copy live migration must be rejected.
// Guaranteed THP coverage requires full preallocation at domain start.
func ForbidsPostCopy(hugepages *v1.Hugepages) bool {
	return IsTransparent(hugepages) && EffectivePolicy(hugepages) == v1.HugepagesPolicyGuaranteed
}
