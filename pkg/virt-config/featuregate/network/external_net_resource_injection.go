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

package network

import (
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

// ExternalNetResourceInjection disables the VMI controller query of NetworkAttachmentDefinition objects and
// the deployment of related RBAC rules by virt-operator.
// Owner: SIG network
// Beta: v1.8.0
const ExternalNetResourceInjection = "ExternalNetResourceInjection"

func init() {
	featuregate.RegisterFeatureGate(featuregate.FeatureGate{Name: ExternalNetResourceInjection, State: featuregate.Beta})
}

// ExternalNetResourceInjectionEnabled returns true when the ExternalNetResourceInjection feature gate is enabled.
func (g NetworkFeatureGates) ExternalNetResourceInjectionEnabled() bool {
	return featuregate.GateEnabled(ExternalNetResourceInjection, g.ConfigReader)
}
