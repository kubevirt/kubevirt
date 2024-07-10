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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package libvmifact

var (
	architectureProvider archProvider = defaultMemProvider{}

	archProviders = map[string]archProvider{
		"arm64": arm64MemProvider{},
	}
)

func RegisterArchitecture(arch string) {
	if provider, found := archProviders[arch]; found {
		architectureProvider = provider
	}
}

// GetMinimalMemory returns the minimal required VMI memory according to the architecture
func GetMinimalMemory() string {
	return architectureProvider.getMinimalMemory()
}

// GetQemuMinimalMemory returns the minimal required QEMU memory according to the architecture
func GetQemuMinimalMemory() string {
	return architectureProvider.getQemuMinimalMemory()
}

type archProvider interface {
	minimalMemoryProvider
}

// minimalMemoryProvider is an interface to return the minimal memory needed for a VMI, according to the architecture
type minimalMemoryProvider interface {
	getMinimalMemory() string
	getQemuMinimalMemory() string
}

type arm64MemProvider struct{}

const (
	arm64MinimalMemory = "256Mi"

	// required to start qemu on ARM with UEFI firmware
	// https://github.com/kubevirt/kubevirt/pull/11366#issuecomment-1970247448
	armMinimalBootableMemory = "128Mi"
)

func (arm64MemProvider) getMinimalMemory() string {
	return arm64MinimalMemory
}

func (arm64MemProvider) getQemuMinimalMemory() string {
	return armMinimalBootableMemory
}

type defaultMemProvider struct{}

const (
	defaultMinimalMemory         = "128Mi"
	defaultMinimalBootableMemory = "1Mi"
)

func (defaultMemProvider) getMinimalMemory() string {
	return defaultMinimalMemory
}
func (defaultMemProvider) getQemuMinimalMemory() string {
	return defaultMinimalBootableMemory
}
