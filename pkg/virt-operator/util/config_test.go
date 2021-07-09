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
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("Operator Config", func() {

	getConfig := func(registry, version string) *KubeVirtDeploymentConfig {
		return &KubeVirtDeploymentConfig{
			Registry:        registry,
			KubeVirtVersion: version,
		}
	}

	table.DescribeTable("Parse image", func(image string, config *KubeVirtDeploymentConfig, valid bool) {
		os.Setenv(OperatorImageEnvName, image)

		err := VerifyEnv()
		if valid {
			Expect(err).ToNot(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
		}

		parsedConfig, err := GetConfigFromEnv()
		Expect(err).ToNot(HaveOccurred())

		Expect(parsedConfig.GetImageRegistry()).To(Equal(config.GetImageRegistry()), "registry should match")
		Expect(parsedConfig.GetKubeVirtVersion()).To(Equal(config.GetKubeVirtVersion()), "tag should match")
	},
		table.Entry("without registry", "kubevirt/virt-operator:v123", getConfig("kubevirt", "v123"), true),
		table.Entry("with registry", "reg/kubevirt/virt-operator:v123", getConfig("reg/kubevirt", "v123"), true),
		table.Entry("with registry with port", "reg:1234/kubevirt/virt-operator:latest", getConfig("reg:1234/kubevirt", "latest"), true),
		table.Entry("without tag", "kubevirt/virt-operator", getConfig("kubevirt", "latest"), true),
		table.Entry("with shasum", "kubevirt/virt-operator@sha256:abcdef", getConfig("kubevirt", "latest"), true),
		table.Entry("without shasum, with invalid image", "kubevirt/virt-xxx@sha256:abcdef", getConfig("", ""), false),
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

	table.DescribeTable("Read shasums", func(image string, envVersions *KubeVirtDeploymentConfig, expectedConfig *KubeVirtDeploymentConfig, useShasums, valid bool) {
		os.Setenv(OperatorImageEnvName, image)

		os.Setenv(VirtApiShasumEnvName, envVersions.VirtApiSha)
		os.Setenv(VirtControllerShasumEnvName, envVersions.VirtControllerSha)
		os.Setenv(VirtHandlerShasumEnvName, envVersions.VirtHandlerSha)
		os.Setenv(VirtLauncherShasumEnvName, envVersions.VirtLauncherSha)
		os.Setenv(KubeVirtVersionEnvName, envVersions.KubeVirtVersion)

		err := VerifyEnv()
		if valid {
			Expect(err).ToNot(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
		}

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
			false, true),
		table.Entry("with all shasums given", "kubevirt/virt-operator@sha256:operator",
			getConfigWithShas("sha256:api", "sha256:controller", "sha256:handler", "sha256:launcher", "v234"),
			getFullConfig("kubevirt", "sha256:operator", "sha256:api", "sha256:controller", "sha256:handler", "sha256:launcher", "v234"),
			true, true),
		table.Entry("with shasums given should fail if not all are provided", "kubevirt/virt-operator:v123",
			getConfigWithShas("sha256:api", "sha256:controller", "", "", ""),
			getConfig("kubevirt", "v123"),
			false, false),
	)

	Describe("GetPassthroughEnv()", func() {
		It("should eturn environment variables matching the passthrough prefix (and only those vars)", func() {
			realKey := rand.String(10)
			passthroughKey := fmt.Sprintf("%s%s", PassthroughEnvPrefix, realKey)
			otherKey := rand.String(10)
			val := rand.String(10)

			err := os.Setenv(passthroughKey, val)
			Expect(err).ToNot(HaveOccurred())

			err = os.Setenv(otherKey, val)
			Expect(err).ToNot(HaveOccurred())

			envMap := GetPassthroughEnv()

			err = os.Unsetenv(passthroughKey)
			Expect(err).ToNot(HaveOccurred())

			err = os.Unsetenv(otherKey)
			Expect(err).ToNot(HaveOccurred())

			Expect(envMap).To(Equal(map[string]string{realKey: val}))

		})
	})

	Describe("NewEnvVarMap()", func() {
		It("Should convert a map to a list of EnvVar objects", func() {
			key1 := rand.String(10)
			val1 := rand.String(10)
			key2 := rand.String(10)
			val2 := rand.String(10)

			envMap := map[string]string{key1: val1, key2: val2}

			envObjects := NewEnvVarMap(envMap)
			expected := []k8sv1.EnvVar{
				{Name: key1, Value: val1},
				{Name: key2, Value: val2},
			}

			Expect(*envObjects).To(ConsistOf(expected))
		})
	})

	Describe("Config json from env var", func() {
		It("should be parsed", func() {
			json := `{"id":"9ca7273e4d5f1bee842f64a8baabc15cbbf1ce59","namespace":"kubevirt","registry":"registry:5000/kubevirt","imagePrefix":"somePrefix","kubeVirtVersion":"devel","additionalProperties":{"ImagePullPolicy":"IfNotPresent", "MonitorNamespace":"non-default-monitor-namespace", "MonitorAccount":"non-default-prometheus-k8s"}}`
			os.Setenv(TargetDeploymentConfig, json)
			parsedConfig, err := GetConfigFromEnv()
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedConfig.GetDeploymentID()).To(Equal("9ca7273e4d5f1bee842f64a8baabc15cbbf1ce59"))
			Expect(parsedConfig.GetNamespace()).To(Equal("kubevirt"))
			Expect(parsedConfig.GetImageRegistry()).To(Equal("registry:5000/kubevirt"))
			Expect(parsedConfig.GetImagePrefix()).To(Equal("somePrefix"))
			Expect(parsedConfig.GetKubeVirtVersion()).To(Equal("devel"))
			Expect(parsedConfig.GetImagePullPolicy()).To(Equal(k8sv1.PullIfNotPresent))
			Expect(parsedConfig.GetMonitorNamespaces()).To(ConsistOf("non-default-monitor-namespace"))
			Expect(parsedConfig.GetMonitorServiceAccount()).To(Equal("non-default-prometheus-k8s"))
		})
	})

	Describe("Config json with default value", func() {
		It("should be parsed", func() {
			json := `{"id":"9ca7273e4d5f1bee842f64a8baabc15cbbf1ce59","additionalProperties":{"ImagePullPolicy":"IfNotPresent"}}`
			os.Setenv(TargetDeploymentConfig, json)
			parsedConfig, err := GetConfigFromEnv()
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedConfig.GetMonitorNamespaces()).To(ConsistOf("openshift-monitoring", "monitoring"))
			Expect(parsedConfig.GetMonitorServiceAccount()).To(Equal("prometheus-k8s"))
		})
	})

	Describe("parsing ObservedDeploymentConfig", func() {
		It("should retrieve imagePrefix if present", func() {
			prefix := "test-prefix-"
			deploymentConfig := KubeVirtDeploymentConfig{}
			deploymentConfig.ImagePrefix = prefix

			blob, err := deploymentConfig.GetJson()
			Expect(err).ToNot(HaveOccurred())

			result, found, err := getImagePrefixFromDeploymentConfig(blob)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(result).To(Equal(prefix))
		})

		It("should not error if imagePrefix is not present", func() {
			deploymentConfig := KubeVirtDeploymentConfig{}

			blob, err := deploymentConfig.GetJson()
			Expect(err).ToNot(HaveOccurred())

			result, found, err := getImagePrefixFromDeploymentConfig(blob)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
			Expect(result).To(Equal(""))
		})
	})

	Describe("creating config ID", func() {

		var idMissing, idEmpty, idFilled string

		BeforeEach(func() {
			cfgMissing := &KubeVirtDeploymentConfig{}
			cfgMissing.AdditionalProperties = make(map[string]string)
			cfgMissing.generateInstallStrategyID()
			idMissing = cfgMissing.ID

			cfgEmpty := &KubeVirtDeploymentConfig{}
			cfgEmpty.AdditionalProperties = make(map[string]string)
			cfgEmpty.AdditionalProperties[ImagePrefixKey] = ""
			cfgEmpty.generateInstallStrategyID()
			idEmpty = cfgEmpty.ID

			cfgFilled := &KubeVirtDeploymentConfig{}
			cfgFilled.AdditionalProperties = make(map[string]string)
			cfgFilled.AdditionalProperties[ImagePrefixKey] = "something"
			cfgFilled.generateInstallStrategyID()
			idFilled = cfgFilled.ID
		})

		It("should result in same ID with missing or empty image prefix", func() {
			Expect(idMissing).ToNot(BeEmpty())
			Expect(idMissing).To(Equal(idEmpty))
		})

		It("should result in different ID with filled image prefix", func() {
			Expect(idFilled).ToNot(BeEmpty())
			Expect(idFilled).ToNot(Equal(idEmpty))
		})

	})

	Context("Product Names and Versions", func() {
		table.DescribeTable("label validation", func(testVector string, expectedResult bool) {
			Expect(IsValidLabel(testVector)).To(Equal(expectedResult))
		},
			table.Entry("should allow 1 character strings", "a", true),
			table.Entry("should allow 2 character strings", "aa", true),
			table.Entry("should allow 3 character strings", "aaa", true),
			table.Entry("should allow 63 character strings", strings.Repeat("a", 63), true),
			table.Entry("should reject 64 character strings", strings.Repeat("a", 64), false),
			table.Entry("should reject strings that begin with .", ".a", false),
			table.Entry("should reject strings that end with .", "a.", false),
			table.Entry("should reject strings that contain junk characters", `a\a`, false),
			table.Entry("should allow strings that contain dots", "a.a", true),
			table.Entry("should allow strings that contain dashes", "a-a", true),
			table.Entry("should allow strings that contain underscores", "a_a", true),
			table.Entry("should allow empty strings", "", true),
		)
	})

})
