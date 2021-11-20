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
 * Copyright 2021
 *
 */

package launchsecurity

import (
	v1 "kubevirt.io/api/core/v1"
)

const (
	// Guest policy bits as defined in AMD SEV API specification
	SEVPolicyNoDebug        uint = (1 << 0)
	SEVPolicyNoKeysSharing  uint = (1 << 1)
	SEVPolicyEncryptedState uint = (1 << 2)
	SEVPolicyNoSend         uint = (1 << 3)
	SEVPolicyDomain         uint = (1 << 4)
	SEVPolicySEV            uint = (1 << 5)
)

func SEVPolicyToBits(policy *v1.SEVPolicy) uint {
	// NoDebug is always true
	bits := uint(SEVPolicyNoDebug)

	if policy != nil {
		if policy.NoKeysSharing != nil && *policy.NoKeysSharing {
			bits = bits | SEVPolicyNoKeysSharing
		}
		if policy.EncryptedState != nil && *policy.EncryptedState {
			bits = bits | SEVPolicyEncryptedState
		}
		if policy.NoSend != nil && *policy.NoSend {
			bits = bits | SEVPolicyNoSend
		}
		if policy.Domain != nil && *policy.Domain {
			bits = bits | SEVPolicyDomain
		}
		if policy.SEV != nil && *policy.SEV {
			bits = bits | SEVPolicySEV
		}
	}

	return bits
}
