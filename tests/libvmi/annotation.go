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

package libvmi

import (
	"fmt"

	"kubevirt.io/kubevirt/tests/flags"
)

const (
	HookSidecarImage                   = "example-hook-sidecar"
	annotationKeyHookSideCars          = "hooks.kubevirt.io/hookSidecars"
	annotationKeyBaseBoardManufacturer = "smbios.vm.kubevirt.io/baseBoardManufacturer"
)

func NewAnnotations(opts ...AnnotationsRendererOption) map[string]string {
	r := &AnnotationsRenderer{
		entries: map[string]string{},
	}
	for _, option := range opts {
		option(r)
	}
	return r.Render()
}

type AnnotationsRenderer struct {
	entries map[string]string
}

func (s *AnnotationsRenderer) Render() map[string]string {
	result := map[string]string{}
	for k, v := range s.entries {
		result[k] = v
	}
	return result
}

type AnnotationsRendererOption func(s *AnnotationsRenderer)

func WithEntry(key, value string) AnnotationsRendererOption {
	return func(s *AnnotationsRenderer) {
		s.entries[key] = value
	}
}

func WithBaseBoardManufacturer() AnnotationsRendererOption {
	return WithEntry(
		annotationKeyBaseBoardManufacturer,
		"Radical Edward",
	)
}

func WithHookSideCar(value string) AnnotationsRendererOption {
	return WithEntry(annotationKeyHookSideCars, value)
}

func WithExampleHookSideCarAndVersion(version string) AnnotationsRendererOption {
	return WithHookSideCar(
		fmt.Sprintf(
			`[{"args": ["--version", %q],"image": "%s/%s:%s", "imagePullPolicy": "IfNotPresent"}]`,
			version,
			flags.KubeVirtUtilityRepoPrefix,
			HookSidecarImage,
			flags.KubeVirtUtilityVersionTag,
		),
	)
}

func WithExampleHookSideCarAndNoVersion() AnnotationsRendererOption {
	return WithHookSideCar(
		fmt.Sprintf(
			`[{"image": "%s/%s:%s", "imagePullPolicy": "IfNotPresent"}]`,
			flags.KubeVirtUtilityRepoPrefix,
			HookSidecarImage,
			flags.KubeVirtUtilityVersionTag,
		),
	)
}
