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

package render

import v1 "kubevirt.io/api/core/v1"

// OnlineRenderer prepares VMIs for creation through the API server. Its
// dependencies provide the controller's configuration and instancetype lookup.
// Admission defaults and mutations are left to the admission webhooks.
type OnlineRenderer struct {
	renderer vmiRenderer
}

// NewOnlineRenderer uses the supplied cluster configuration and instancetype
// handler. Both dependencies must be non-nil.
func NewOnlineRenderer(config networkConfig, instancetype instancetypeHandler) *OnlineRenderer {
	return &OnlineRenderer{renderer: vmiRenderer{config: config, instancetype: instancetype}}
}

// VMIFromVM runs the shared preparation flow without changing the input VM.
// Runtime state, revision management and VMI creation remain with the caller.
func (r *OnlineRenderer) VMIFromVM(vm *v1.VirtualMachine) (*v1.VirtualMachineInstance, error) {
	return r.renderer.vmiFromVM(vm)
}
