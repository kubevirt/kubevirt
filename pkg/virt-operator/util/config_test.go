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
		Expect(parsedConfig.ImageTag).To(Equal(config.ImageTag), "tag should match")
	},
		table.Entry("without registry", "kubevirt/virt-operator:v123", KubeVirtDeploymentConfig{"kubevirt", "v123"}),
		table.Entry("with registry", "reg/kubevirt/virt-operator:v123", KubeVirtDeploymentConfig{"reg/kubevirt", "v123"}),
		table.Entry("with registry with port", "reg:1234/kubevirt/virt-operator:latest", KubeVirtDeploymentConfig{"reg:1234/kubevirt", "latest"}),
		table.Entry("without tag", "kubevirt/virt-operator", KubeVirtDeploymentConfig{"kubevirt", "latest"}),
	)

})
