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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package services

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hooks"
)

type SidecarCreatorFunc func(*v1.VirtualMachineInstance, *v1.KubeVirtConfiguration) (hooks.HookSidecarList, error)

func WithSidecarCreator(sidecarCreator SidecarCreatorFunc) templateServiceOption {
	return func(t *templateService) {
		t.sidecarCreators = append(t.sidecarCreators, sidecarCreator)
	}
}
