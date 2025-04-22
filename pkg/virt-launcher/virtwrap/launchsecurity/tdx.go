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
 * Copyright 2025 Red Hat, Inc.
 *
*/

package launchsecurity

const (
	// Guest TD runs in off-TD debug mode when set
	TDXPolicyNoDebug uint64 = 0
	// Disable EPT violation conversion to #VE on guest TD access of PENDING pages when set
	TDXDisableVEConversion uint64 = 1 << 28
	// 1:27, 29:63 reserved
)

func TDXPolicyToBits(policy uint64) uint64 {
	// NoDebug is always set
	// Currently it returns either 0x0 or 0x10000000
	bits := TDXPolicyNoDebug
	if policy|TDXDisableVEConversion != 0 {
		bits |= TDXDisableVEConversion
	}

	return bits
}
