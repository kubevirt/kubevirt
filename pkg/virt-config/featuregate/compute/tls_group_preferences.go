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

package compute

import (
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// Owner: sig-compute / @bmordeha
// Alpha: v1.10.0
//
// TLSGroupPreferences enables the Groups field on TLSConfiguration,
// allowing cluster administrators to configure TLS supported groups
// (elliptic curves) negotiated during TLS handshakes across all virt
// pod endpoints. When disabled, the Groups field is ignored and Go's
// default curve preferences apply.
const TLSGroupPreferences = "TLSGroupPreferences"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: TLSGroupPreferences, State: featuregate.Alpha})
}

func (g ComputeFeatureGates) TLSGroupPreferencesEnabled() bool {
	return featuregate.GateEnabled(TLSGroupPreferences, g.ConfigReader)
}
