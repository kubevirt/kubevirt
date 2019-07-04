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
)

var _ = Describe("Operator Config", func() {

	table.DescribeTable("Parse image", func(image string, config KubeVirtDeploymentConfig) {
		os.Setenv(OperatorImageEnvName, image)
		parsedConfig := GetConfig()
		Expect(parsedConfig.ImageRegistry).To(Equal(config.ImageRegistry), "registry should match")
		Expect(parsedConfig.Versions).To(Equal(config.Versions), "tag should match")
	},
		table.Entry("without registry", "kubevirt/virt-operator:v123", KubeVirtDeploymentConfig{"kubevirt", &Versions{kubeVirtVersion: "v123"}}),
		table.Entry("with registry", "reg/kubevirt/virt-operator:v123", KubeVirtDeploymentConfig{"reg/kubevirt", &Versions{kubeVirtVersion: "v123"}}),
		table.Entry("with registry with port", "reg:1234/kubevirt/virt-operator:latest", KubeVirtDeploymentConfig{"reg:1234/kubevirt", &Versions{kubeVirtVersion: "latest"}}),
		table.Entry("without tag", "kubevirt/virt-operator", KubeVirtDeploymentConfig{"kubevirt", &Versions{kubeVirtVersion: "latest"}}),
		table.Entry("with shasum", "kubevirt/virt-operator@sha256:abcdef", KubeVirtDeploymentConfig{"kubevirt", &Versions{kubeVirtVersion: "latest"}}),
	)

	table.DescribeTable("Read shasums", func(image string, versions Versions, config KubeVirtDeploymentConfig, useShasums bool) {
		os.Setenv(OperatorImageEnvName, image)

		os.Setenv(VirtApiShasumEnvName, versions.virtApiSha)
		os.Setenv(VirtControllerShasumEnvName, versions.virtControllerSha)
		os.Setenv(VirtHandlerShasumEnvName, versions.virtHandlerSha)
		os.Setenv(VirtLauncherShasumEnvName, versions.virtLauncherSha)
		os.Setenv(KubeVirtVersionEnvName, versions.kubeVirtVersion)

		parsedConfig := GetConfig()

		Expect(parsedConfig.ImageRegistry).To(Equal(config.ImageRegistry), "registry should match")
		Expect(parsedConfig.Versions).To(Equal(config.Versions), "tag / shasums should match")

		if useShasums {
			Expect(parsedConfig.Versions.GetApiVersion()).To(Equal(config.Versions.virtApiSha), "api shasums should match")
			Expect(parsedConfig.Versions.GetControllerVersion()).To(Equal(config.Versions.virtControllerSha), "controller shasums should match")
			Expect(parsedConfig.Versions.GetHandlerVersion()).To(Equal(config.Versions.virtHandlerSha), "handler shasums should match")
			Expect(parsedConfig.Versions.GetLauncherVersion()).To(Equal(config.Versions.virtLauncherSha), "launcher shasums should match")
		} else {
			Expect(parsedConfig.Versions.GetApiVersion()).To(Equal(config.Versions.kubeVirtVersion), "api version should be tag")
			Expect(parsedConfig.Versions.GetControllerVersion()).To(Equal(config.Versions.kubeVirtVersion), "controller version should be tag")
			Expect(parsedConfig.Versions.GetHandlerVersion()).To(Equal(config.Versions.kubeVirtVersion), "handler version should be tag")
			Expect(parsedConfig.Versions.GetLauncherVersion()).To(Equal(config.Versions.kubeVirtVersion), "launcher version should be tag")
		}

	},
		table.Entry("with no shasum given", "kubevirt/virt-operator:v123",
			Versions{},
			KubeVirtDeploymentConfig{"kubevirt", &Versions{kubeVirtVersion: "v123"}},
			false),
		table.Entry("with all shasums given", "kubevirt/virt-operator@sha256:operator",
			Versions{virtApiSha: "sha256:api", virtControllerSha: "sha256:controller", virtHandlerSha: "sha256:handler", virtLauncherSha: "sha256:launcher", kubeVirtVersion: "v234"},
			KubeVirtDeploymentConfig{"kubevirt", &Versions{virtOperatorSha: "sha256:operator", virtApiSha: "sha256:api", virtControllerSha: "sha256:controller", virtHandlerSha: "sha256:handler", virtLauncherSha: "sha256:launcher", kubeVirtVersion: "v234"}},
			true),
		table.Entry("with all shasums given", "kubevirt/virt-operator:v123",
			Versions{virtApiSha: "sha256:api", virtControllerSha: "sha256:controller"},
			KubeVirtDeploymentConfig{"kubevirt", &Versions{kubeVirtVersion: "v123"}},
			false),
	)

})
