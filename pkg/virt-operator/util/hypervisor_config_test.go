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
 */

package util

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

var _ = Describe("KubeVirtDeploymentConfig hypervisor methods", func() {
	Context("GetHypervisorName", func() {
		It("should return KVM by default when no hypervisor is set", func() {
			config := &KubeVirtDeploymentConfig{
				AdditionalProperties: make(map[string]string),
			}

			hypervisor := config.GetHypervisorName()
			Expect(hypervisor).To(Equal(v1.KvmHypervisorName))
		})

		It("should return configured hypervisor when set", func() {
			config := &KubeVirtDeploymentConfig{
				AdditionalProperties: map[string]string{
					AdditionalPropertiesHypervisorName: v1.HyperVLayeredHypervisorName,
				},
			}

			hypervisor := config.GetHypervisorName()
			Expect(hypervisor).To(Equal(v1.HyperVLayeredHypervisorName))
		})

		It("should default to KVM when hypervisor is empty string", func() {
			config := &KubeVirtDeploymentConfig{
				AdditionalProperties: map[string]string{
					AdditionalPropertiesHypervisorName: "",
				},
			}

			hypervisor := config.GetHypervisorName()
			Expect(hypervisor).To(Equal(v1.KvmHypervisorName))
		})
	})

	Context("GetTargetConfigFromKVWithEnvVarManager", func() {
		var envManager *EnvVarManagerImpl

		BeforeEach(func() {
			envManager = &EnvVarManagerImpl{}
		})

		It("should extract hypervisor configuration when ConfigurableHypervisor feature gate is enabled", func() {
			kv := &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							FeatureGates: []string{featuregate.ConfigurableHypervisor},
						},
						HypervisorConfiguration: &v1.HypervisorConfiguration{
							Name: v1.HyperVLayeredHypervisorName,
						},
					},
				},
			}

			config := GetTargetConfigFromKVWithEnvVarManager(kv, envManager)

			Expect(config.GetHypervisorName()).To(Equal(v1.HyperVLayeredHypervisorName))
		})

		It("should default to KVM when ConfigurableHypervisor feature gate is enabled but no hypervisor config", func() {
			kv := &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							FeatureGates: []string{featuregate.ConfigurableHypervisor},
						},
						// No HypervisorConfiguration
					},
				},
			}

			config := GetTargetConfigFromKVWithEnvVarManager(kv, envManager)

			Expect(config.GetHypervisorName()).To(Equal(v1.KvmHypervisorName))
		})

		It("should default to KVM when ConfigurableHypervisor feature gate is not enabled", func() {
			kv := &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							FeatureGates: []string{}, // No ConfigurableHypervisor
						},
						HypervisorConfiguration: &v1.HypervisorConfiguration{
							Name: v1.HyperVLayeredHypervisorName,
						},
					},
				},
			}

			config := GetTargetConfigFromKVWithEnvVarManager(kv, envManager)

			Expect(config.GetHypervisorName()).To(Equal(v1.KvmHypervisorName))
		})
	})
})
