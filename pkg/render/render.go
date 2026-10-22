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

import (
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

// ManifestRenderer produces a virt-launcher Pod from a prepared VMI.
// *services.TemplateService satisfies it.
type ManifestRenderer interface {
	RenderLaunchManifest(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error)
}

var _ ManifestRenderer = (*services.TemplateService)(nil)

// LaunchManifest renders a virt-launcher Pod from an already-prepared VMI.
// renderer is typically *services.TemplateService; a future virt-launcher
// plugin can implement ManifestRenderer and be passed in instead.
func LaunchManifest(vmi *v1.VirtualMachineInstance, renderer ManifestRenderer) (*k8sv1.Pod, error) {
	return renderer.RenderLaunchManifest(vmi)
}
