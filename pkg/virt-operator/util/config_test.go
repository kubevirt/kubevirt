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
 * Copyright 2018 Red Hat, Inc.
 *
 */
package util

import (
	"os"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
)

var _ = Describe("Operator Config", func() {

	getConfig := func(registry, version string) *KubeVirtDeploymentConfig {
		return &KubeVirtDeploymentConfig{
			Registry:        registry,
			KubeVirtVersion: version,
		}
	}

	table.DescribeTable("Parse image", func(image string, config *KubeVirtDeploymentConfig) {
		os.Setenv(OperatorImageEnvName, image)
		parsedConfig, err := GetConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())

		Expect(parsedConfig.GetImageRegistry()).To(Equal(config.GetImageRegistry()), "registry should match")
		Expect(parsedConfig.GetKubeVirtVersion()).To(Equal(config.GetKubeVirtVersion()), "tag should match")
	},
		table.Entry("without registry", "kubevirt/virt-operator:v123", getConfig("kubevirt", "v123")),
		table.Entry("with registry", "reg/kubevirt/virt-operator:v123", getConfig("reg/kubevirt", "v123")),
		table.Entry("with registry with port", "reg:1234/kubevirt/virt-operator:latest", getConfig("reg:1234/kubevirt", "latest")),
		table.Entry("without tag", "kubevirt/virt-operator", getConfig("kubevirt", "latest")),
		table.Entry("with shasum", "kubevirt/virt-operator@sha256:abcdef", getConfig("kubevirt", "latest")),
	)

	getConfigWithShas := func(apiSha, controllerSha, handlerSha, launcherSha, version string) *KubeVirtDeploymentConfig {
		return &KubeVirtDeploymentConfig{
			KubeVirtVersion:   version,
			VirtApiSha:        apiSha,
			VirtControllerSha: controllerSha,
			VirtHandlerSha:    handlerSha,
			VirtLauncherSha:   launcherSha,
		}
	}

	getFullConfig := func(registry, operatorSha, apiSha, controllerSha, handlerSha, launcherSha, version string) *KubeVirtDeploymentConfig {
		return &KubeVirtDeploymentConfig{
			Registry:          registry,
			KubeVirtVersion:   version,
			VirtOperatorSha:   operatorSha,
			VirtApiSha:        apiSha,
			VirtControllerSha: controllerSha,
			VirtHandlerSha:    handlerSha,
			VirtLauncherSha:   launcherSha,
		}
	}

	table.DescribeTable("Read shasums", func(image string, envVersions *KubeVirtDeploymentConfig, expectedConfig *KubeVirtDeploymentConfig, useShasums bool) {
		os.Setenv(OperatorImageEnvName, image)

		os.Setenv(VirtApiShasumEnvName, envVersions.VirtApiSha)
		os.Setenv(VirtControllerShasumEnvName, envVersions.VirtControllerSha)
		os.Setenv(VirtHandlerShasumEnvName, envVersions.VirtHandlerSha)
		os.Setenv(VirtLauncherShasumEnvName, envVersions.VirtLauncherSha)
		os.Setenv(KubeVirtVersionEnvName, envVersions.KubeVirtVersion)

		parsedConfig, err := GetConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())

		Expect(parsedConfig.GetImageRegistry()).To(Equal(expectedConfig.GetImageRegistry()), "registry should match")
		Expect(parsedConfig.GetKubeVirtVersion()).To(Equal(expectedConfig.GetKubeVirtVersion()), "tag / shasums should match")

		if useShasums {
			Expect(parsedConfig.GetApiVersion()).To(Equal(expectedConfig.GetApiVersion()), "api shasums should match")
			Expect(parsedConfig.GetControllerVersion()).To(Equal(expectedConfig.GetControllerVersion()), "controller shasums should match")
			Expect(parsedConfig.GetHandlerVersion()).To(Equal(expectedConfig.GetHandlerVersion()), "handler shasums should match")
			Expect(parsedConfig.GetLauncherVersion()).To(Equal(expectedConfig.GetLauncherVersion()), "launcher shasums should match")
		} else {
			Expect(parsedConfig.GetApiVersion()).To(Equal(expectedConfig.GetKubeVirtVersion()), "api version should be tag")
			Expect(parsedConfig.GetControllerVersion()).To(Equal(expectedConfig.GetKubeVirtVersion()), "controller version should be tag")
			Expect(parsedConfig.GetHandlerVersion()).To(Equal(expectedConfig.GetKubeVirtVersion()), "handler version should be tag")
			Expect(parsedConfig.GetLauncherVersion()).To(Equal(expectedConfig.GetKubeVirtVersion()), "launcher version should be tag")
		}

	},
		table.Entry("with no shasum given", "kubevirt/virt-operator:v123",
			&KubeVirtDeploymentConfig{},
			getConfig("kubevirt", "v123"),
			false),
		table.Entry("with all shasums given", "kubevirt/virt-operator@sha256:operator",
			getConfigWithShas("sha256:api", "sha256:controller", "sha256:handler", "sha256:launcher", "v234"),
			getFullConfig("kubevirt", "sha256:operator", "sha256:api", "sha256:controller", "sha256:handler", "sha256:launcher", "v234"),
			true),
		table.Entry("with all shasums given", "kubevirt/virt-operator:v123",
			getConfigWithShas("sha256:api", "sha256:controller", "", "", ""),
			getConfig("kubevirt", "v123"),
			false),
	)

	Describe("Config json from env var", func() {
		It("should be parsed", func() {
			json := `{"id":"9ca7273e4d5f1bee842f64a8baabc15cbbf1ce59","namespace":"kubevirt","registry":"registry:5000/kubevirt","kubeVirtVersion":"devel","additionalProperties":{"ImagePullPolicy":"IfNotPresent"}}`
			os.Setenv(TargetDeploymentConfig, json)
			parsedConfig, err := GetConfigFromEnv()
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedConfig.GetDeploymentID()).To(Equal("9ca7273e4d5f1bee842f64a8baabc15cbbf1ce59"))
			Expect(parsedConfig.GetNamespace()).To(Equal("kubevirt"))
			Expect(parsedConfig.GetImageRegistry()).To(Equal("registry:5000/kubevirt"))
			Expect(parsedConfig.GetKubeVirtVersion()).To(Equal("devel"))
			Expect(parsedConfig.GetImagePullPolicy()).To(Equal(k8sv1.PullIfNotPresent))
		})
	})

})
